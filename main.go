package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
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
	numDevices := flag.Int("num-devices", 1, "number of device goroutines to run")

	flag.Parse()

	if *serverIP == "" || *serverPort == 0 || *imei == "" {
		log.Fatalf("[ERROR] Missing required flags: -server-ip, -server-port and -imei must be provided")
	}
	if *serverPort < 1 || *serverPort > 65535 {
		log.Fatalf("[ERROR] Invalid server port: %d. Must be between 1 and 65535", *serverPort)
	}
	if *numDevices < 1 {
		log.Fatalf("[ERROR] Invalid num-devices: %d. Must be at least 1", *numDevices)
	}

	log.Printf("[INFO] Simulator started. IMEI=%s devices=%d", *imei, *numDevices)

	address := fmt.Sprintf("%s:%d", *serverIP, *serverPort)

	for deviceIndex := 0; deviceIndex < *numDevices; deviceIndex++ {
		deviceIMEI, err := computeDeviceIMEI(*imei, deviceIndex)
		if err != nil {
			log.Fatalf("[ERROR] Failed to compute IMEI for device %d: %v", deviceIndex+1, err)
		}

		go runDevice(deviceIndex+1, deviceIMEI, address, *interval8E, *timeoutResponse, *retryConnect, *sleepRetry, *sleepNoConnect)
	}

	select {}
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

	log.Printf("[OK] Handshake completed for IMEI %s", imei)
	return nil
}

func computeDeviceIMEI(baseIMEI string, deviceIndex int) (string, error) {
	if deviceIndex == 0 {
		return baseIMEI, nil
	}

	if len(baseIMEI) == 0 {
		return "", fmt.Errorf("base IMEI is empty")
	}

	for _, r := range baseIMEI {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("IMEI must contain only digits for sequential generation")
		}
	}

	baseValue := baseIMEI
	carry := deviceIndex
	result := make([]byte, len(baseValue))
	for i := len(baseValue) - 1; i >= 0; i-- {
		digit := int(baseValue[i] - '0')
		digit += carry
		carry = digit / 10
		digit = digit % 10
		result[i] = byte('0' + digit)
	}

	if carry != 0 {
		return "", fmt.Errorf("IMEI overflow when generating device %d", deviceIndex+1)
	}

	return string(result), nil
}

func runDevice(deviceNumber int, imei, address string, intervalSec, timeoutSec, retryConnect, sleepRetry, sleepNoConnect int) {
	logPrefix := fmt.Sprintf("[device %d][IMEI %s]", deviceNumber, imei)
	log.Printf("%s starting", logPrefix)

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(deviceNumber)))
	latitude := 40.365279 + rng.Float64()*(40.512379-40.365279)
	longitude := -3.695889 + rng.Float64()*(-3.664330+3.695889)
	movesRemaining := 100
	stayRemaining := 0

	for {
		log.Printf("%s trying to connect to %s", logPrefix, address)

		var conn net.Conn
		var err error
		for attempt := 1; attempt <= retryConnect; attempt++ {
			conn, err = net.Dial("tcp", address)
			if err == nil {
				log.Printf("%s connected on attempt %d", logPrefix, attempt)
				break
			}
			log.Printf("%s connection failed (attempt %d/%d): %v", logPrefix, attempt, retryConnect, err)
			time.Sleep(time.Duration(sleepRetry) * time.Second)
		}

		if err != nil {
			log.Printf("%s could not connect after %d attempts, waiting %d seconds", logPrefix, retryConnect, sleepNoConnect)
			time.Sleep(time.Duration(sleepNoConnect) * time.Second)
			continue
		}

		if err := performInitialHandshake(conn, imei); err != nil {
			log.Printf("%s handshake failed: %v", logPrefix, err)
			conn.Close()
			time.Sleep(time.Duration(sleepNoConnect) * time.Second)
			continue
		}

		handleDeviceConnection(logPrefix, conn, intervalSec, timeoutSec, &latitude, &longitude, &movesRemaining, &stayRemaining, rng)
		log.Printf("%s connection closed, restarting", logPrefix)
	}
}

func handleDeviceConnection(logPrefix string, conn net.Conn, intervalSec int, timeoutSec int, latitude, longitude *float64, movesRemaining, stayRemaining *int, rng *rand.Rand) {
	defer conn.Close()
	log.Printf("%s active", logPrefix)
	log.Printf("%s sending 8E packets every %d seconds", logPrefix, intervalSec)

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		packet, err := protocol.Build8EPacket(uint64(time.Now().Unix()), *latitude, *longitude)
		if err != nil {
			log.Printf("%s failed to build 8E packet: %v", logPrefix, err)
			return
		}

		if _, err := conn.Write(packet); err != nil {
			log.Printf("%s failed to send 8E packet: %v", logPrefix, err)
			return
		}
		log.Printf("%s sent 8E packet (%d bytes)", logPrefix, len(packet))

		if err := conn.SetReadDeadline(time.Now().Add(time.Duration(timeoutSec) * time.Second)); err != nil {
			log.Printf("%s failed to set read deadline: %v", logPrefix, err)
		}

		response := make([]byte, 512)
		n, err := conn.Read(response)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Printf("%s no response within %d seconds, disconnecting", logPrefix, timeoutSec)
				return
			}
			log.Printf("%s read failed: %v", logPrefix, err)
			return
		}

		log.Printf("%s received %d bytes from server", logPrefix, n)
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			log.Printf("%s failed to clear read deadline: %v", logPrefix, err)
		}

		// Update coordinate state after the packet is sent.
		if *stayRemaining > 0 {
			*stayRemaining--
			if *stayRemaining == 0 {
				*movesRemaining = 100
			}
		} else {
			meters := 1.0 + rng.Float64()*4.0
			latStep := meters / 111000.0
			lonStep := meters / 85000.0
			switch rng.Intn(4) {
			case 0:
				*latitude += latStep
			case 1:
				*latitude -= latStep
			case 2:
				*longitude += lonStep
			case 3:
				*longitude -= lonStep
			}

			if *latitude < 40.365279 {
				*latitude = 40.365279
			}
			if *latitude > 40.512379 {
				*latitude = 40.512379
			}
			if *longitude < -3.695889 {
				*longitude = -3.695889
			}
			if *longitude > -3.664330 {
				*longitude = -3.664330
			}

			*movesRemaining--
			if *movesRemaining == 0 {
				*stayRemaining = 100
			}
		}

		select {
		case <-ticker.C:
			continue
		}
	}
}
