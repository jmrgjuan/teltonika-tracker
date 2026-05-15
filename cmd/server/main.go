package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/jmrgjuan/teltonika-tracker/protocol"
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

		log.Printf("[INFO] Received 8E packet from %s: payload size=%d, codec=0x%X", remote, len(packet), packet[0])

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
