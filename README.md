# teltonika-tracker
A Teltonika tracker simulator that establishes a TCP connection to a server.

This program performs the initial communication of a Teltonika device: TCP connection, IMEI handshake, and then sends periodic 8E packets. For now the simulator only uses 8E data packets after the handshake.

## Getting started

This repository contains two Go programs:

- client: `main.go`
- server: `cmd/server/main.go`

The client connects to a Teltonika-compatible server, sends an IMEI login, receives an ACK, and then sends periodic 8E packets.
The server accepts connections, reads the IMEI login, sends an ACK byte, and receives 8E packets.

## Build locally

Build the client binary:

```bash
go build -o bin/teltonika-tracker main.go
```

Build the server binary:

```bash
go build -o bin/teltonika-server ./cmd/server
```

Run local tests:

```bash
go test ./...
```

## Run locally

### Start the server

```bash
./bin/teltonika-server \
  -listen-ip 0.0.0.0 \
  -listen-port 5000
```

### Start the client

```bash
./bin/teltonika-tracker \
  -server-ip 127.0.0.1 \
  -server-port 5000 \
  -imei 123456789012345 \
  -interval-8e 10 \
  -timeout-response 2
```

## Client flags

- `-server-ip` — server IP (required)
- `-server-port` — server port (required)
- `-imei` — IMEI of the simulated device (required)
- `-interval-8e` — seconds between 8E packet sends (default: `10`)
- `-timeout-response` — seconds to wait for server response after sending 8E (default: `2`)
- `-retry-connect` — number of retries before a long pause (default: `3`)
- `-sleep-retry` — seconds between connection retries (default: `2`)
- `-sleep-noconnect` — seconds to wait if retries are exhausted (default: `30`)

## Server flags

- `-listen-ip` — IP address to listen on (default: `0.0.0.0`)
- `-listen-port` — port to listen on (default: `5000`)

## Docker

Build the Docker image from the repository root:

```bash
docker build -t teltonika-tracker .
```

### Run the server in Docker

```bash
docker run --rm teltonika-tracker /teltonika-server \
  -listen-ip 0.0.0.0 \
  -listen-port 5000
```

### Run the client in Docker

```bash
docker run --rm teltonika-tracker \
  -server-ip 127.0.0.1 \
  -server-port 5000 \
  -imei 123456789012345 \
  -interval-8e 10 \
  -timeout-response 2
```

If the server is on the Docker host, use host networking for the client:

```bash
docker run --rm --network host teltonika-tracker \
  -server-ip 127.0.0.1 \
  -server-port 5000 \
  -imei 123456789012345 \
  -interval-8e 10 \
  -timeout-response 2
```
