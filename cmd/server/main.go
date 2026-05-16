package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/jmrgjuan/teltonika-tracker/protocol"
	"github.com/jmrgjuan/teltonika-tracker/storage"
)

type packetTask struct {
	Remote     string
	IMEI       string
	Packet     *protocol.Codec8EPacket
	ReceivedAt time.Time
}

var (
	taskQueue   chan packetTask
	workerCount = 4
	queueSize   = 200
)

func main() {
	listenIP := flag.String("listen-ip", "0.0.0.0", "server listen IP")
	listenPort := flag.Int("listen-port", 5000, "server listen port")
	flag.Parse()

	if *listenPort < 1 || *listenPort > 65535 {
		log.Fatalf("[ERROR] Invalid listen port: %d. Must be between 1 and 65535", *listenPort)
	}

	address := fmt.Sprintf("%s:%d", *listenIP, *listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("[FATAL] Failed to listen on %s: %v", address, err)
	}
	defer listener.Close()

	// initialize Influx (if env present)
	if err := storage.InitFromEnv(); err != nil {
		log.Printf("[WARN] failed to initialize influx: %v", err)
	}
	defer storage.Close()

	// start async processing queue
	taskQueue = make(chan packetTask, queueSize)
	for i := 0; i < workerCount; i++ {
		go worker(i, taskQueue)
	}

	log.Printf("[INFO] Server listening on %s", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[WARN] Accept error: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	log.Printf("[INFO] New connection from %s", remote)

	imei, err := protocol.ReadIMEI(conn)
	if err != nil {
		log.Printf("[ERROR] Failed to read IMEI from %s: %v", remote, err)
		return
	}
	log.Printf("[INFO] Received IMEI from %s: %s", remote, imei)

	if err := protocol.SendIMEIResponse(conn, true); err != nil {
		log.Printf("[ERROR] Failed to send IMEI ACK to %s: %v", remote, err)
		return
	}
	log.Printf("[OK] Sent IMEI ACK to %s", remote)

	for {
		packet, err := protocol.Read8EPacket(conn)
		if err != nil {
			if err == io.EOF {
				log.Printf("[INFO] Connection closed by %s", remote)
			} else {
				log.Printf("[ERROR] Failed to read 8E packet from %s: %v", remote, err)
			}
			return
		}

		// enqueue packet for asynchronous processing to avoid blocking the connection
		task := packetTask{
			Remote:     remote,
			IMEI:       imei,
			Packet:     packet,
			ReceivedAt: time.Now(),
		}
		select {
		case taskQueue <- task:
			// enqueued successfully
		default:
			// queue full — spawn a goroutine to enqueue so we don't block the connection
			go func(t packetTask) {
				taskQueue <- t
			}(task)
			log.Printf("[WARN] task queue full, enqueuing in background for %s", remote)
		}

		if _, err := conn.Write([]byte{0x01}); err != nil {
			log.Printf("[ERROR] Failed to send packet response to %s: %v", remote, err)
			return
		}
		log.Printf("[OK] Sent packet response to %s", remote)

		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			log.Printf("[WARN] Failed to update read deadline for %s: %v", remote, err)
		}
	}
}

func worker(id int, q <-chan packetTask) {
	log.Printf("[INFO] worker %d started", id)
	for task := range q {
		remote := task.Remote
		packet := task.Packet
		log.Printf("[INFO] [worker-%d] Processing packet from %s: codec=0x%X, records=%d (recv=%s)", id, remote, packet.CodecID, len(packet.Records), task.ReceivedAt.Format(time.RFC3339))
		for idx, record := range packet.Records {
			log.Printf("[DEBUG] [worker-%d] %s IMEI=%s Record %d: ts=%d priority=%d lon=%.7f lat=%.7f alt=%d speed=%d io_bytes=%d",
				id,
				remote,
				task.IMEI,
				idx,
				record.Timestamp,
				record.Priority,
				protocol.Int32ToDegrees(record.Longitude),
				protocol.Int32ToDegrees(record.Latitude),
				record.Altitude,
				record.Speed,
				len(record.IOData))

			if err := storage.WriteRecord(task.IMEI, record); err != nil {
				log.Printf("[ERROR] [worker-%d] failed writing to influx for %s: %v", id, task.IMEI, err)
			}
		}
	}
}
