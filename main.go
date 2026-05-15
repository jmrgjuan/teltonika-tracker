package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/jmrgjuan/teltonika-tracker/protocol"
)

func main() {
	// --- CLI mandatory parameters ---
	serverIP := flag.String("server-ip", "", "server IP (required)")
	serverPort := flag.Int("server-port", 0, "server port (required)")
	imei := flag.String("imei", "", "IMEI of the simulated device (required)")

	// --- CLI optional parameters ---
	interval8E := flag.Int("interval-8e", 10, "seconds between 8e packet sends")
	timeoutResponse := flag.Int("timeout-response", 2, "seconds to wait for server response after sending 8e")
	retryConnect := flag.Int("retry-connect", 3, "number of retries before long pause")
	sleepRetry := flag.Int("sleep-retry", 2, "seconds between retries")
	sleepNoConnect := flag.Int("sleep-noconnect", 30, "seconds to wait if retries are exhausted")

	flag.Parse()

	if *serverIP == "" || *serverPort == 0 || *imei == "" {
		log.Fatalf("[ERROR] Missing required flags: -server-ip, -server-port and -imei must be provided")
	}
	if *serverPort < 1 || *serverPort > 65535 {
		log.Fatalf("[ERROR] Invalid server port: %d. Must be between 1 and 65535", *serverPort)
	}

	log.Printf("[INFO] Simulator started. IMEI=%s", *imei)

	address := fmt.Sprintf("%s:%d", *serverIP, *serverPort)

	for {
		log.Printf("[INFO] Trying to connect to %s ...", address)

		var conn net.Conn
		var err error

		// --- Retries ---
		for attempt := 1; attempt <= *retryConnect; attempt++ {
			conn, err = net.Dial("tcp", address)
			if err == nil {
				log.Printf("[OK] Connected to server on attempt %d", attempt)
				break
			}

			log.Printf("[WARN] Connection failed (attempt %d/%d): %v",
				attempt, *retryConnect, err)

			time.Sleep(time.Duration(*sleepRetry) * time.Second)
		}

		// --- Connection failed after retries ---
		if err != nil {
			log.Printf("[ERROR] Could not connect after %d attempts. Waiting %d seconds...",
				*retryConnect, *sleepNoConnect)
			time.Sleep(time.Duration(*sleepNoConnect) * time.Second)
			continue
		}

		// --- Connection established ---
		if err := performInitialHandshake(conn, *imei); err != nil {
			log.Printf("[ERROR] Initial handshake failed: %v", err)
			conn.Close()
			time.Sleep(time.Duration(*sleepNoConnect) * time.Second)
			continue
		}

		handleConnection(conn, *interval8E, *timeoutResponse)

		// If handleConnection returns, the connection was closed
		log.Printf("[INFO] Connection closed. Retrying...")
	}
}

func performInitialHandshake(conn net.Conn, imei string) error {
	if err := protocol.SendIMEI(conn, imei); err != nil {
		return fmt.Errorf("failed to send IMEI: %w", err)
	}

	ok, err := protocol.ReadIMEIResponse(conn)
	if err != nil {
		return fmt.Errorf("failed to read IMEI response: %w", err)
	}
	if !ok {
		return fmt.Errorf("server rejected the IMEI")
	}

	log.Printf("[OK] Handshake completed")
	return nil
}

func handleConnection(conn net.Conn, intervalSec int, timeoutSec int) {
	defer conn.Close()

	log.Printf("[INFO] Active connection with %s", conn.RemoteAddr())
	log.Printf("[INFO] Handshake completed; sending 8E packets every %d seconds and waiting %d seconds for server response", intervalSec, timeoutSec)

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		packet, err := protocol.Build8EPacket(uint64(time.Now().Unix()))
		if err != nil {
			log.Printf("[ERROR] Failed to build 8E packet: %v", err)
			return
		}

		if _, err := conn.Write(packet); err != nil {
			log.Printf("[ERROR] Failed to send 8E packet: %v", err)
			return
		}
		log.Printf("[INFO] Sent 8E packet (%d bytes)", len(packet))

		if err := conn.SetReadDeadline(time.Now().Add(time.Duration(timeoutSec) * time.Second)); err != nil {
			log.Printf("[WARN] Failed to set read deadline: %v", err)
		}

		response := make([]byte, 512)
		n, err := conn.Read(response)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Printf("[WARN] No response within %d seconds, disconnecting", timeoutSec)
				return
			}
			log.Printf("[ERROR] Read failed: %v", err)
			return
		}

		log.Printf("[INFO] Received %d bytes from server", n)
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			log.Printf("[WARN] Failed to clear read deadline: %v", err)
		}

		select {
		case <-ticker.C:
			continue
		}
	}
}
