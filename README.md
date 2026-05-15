# teltonika-tracker
A Teltonika tracker simulator that establishes a TCP connection to a server.

This program performs the initial communication of a Teltonika device: TCP connection, IMEI handshake, and then sends periodic 8E packets. For now the simulator only uses 8E data packets after the handshake.

## Getting started

This repository contains a Go executable that connects to a TCP server and keeps the connection open.

### Useful commands

- `go run main.go` — run the program directly
- `go build -o bin/teltonika-tracker main.go` — build the executable
- `./bin/teltonika-tracker` — run the compiled binary

### Available options

- `-server-ip` — server IP (required)
- `-server-port` — server port (required)
- `-imei` — IMEI of the simulated device (required)
- `-interval-8e` — seconds between 8E packet sends (default: `10`)
- `-timeout-response` — seconds to wait for server response after sending 8E (default: `2`)
- `-retry-connect` — number of retries before a long pause (default: `3`)
- `-sleep-retry` — seconds between connection retries (default: `2`)
- `-sleep-noconnect` — seconds to wait if retries are exhausted (default: `30`)

## Execution examples

Run with default values:

```bash
go run main.go
```

Run against a local server on port 5000 with a custom IMEI:

```bash
go run main.go \
  -server-ip 127.0.0.1 \
  -server-port 5000 \
  -imei 123456789012345
```

Build and run the binary:

```bash
go build -o bin/teltonika-tracker main.go
./bin/teltonika-tracker -server-ip 192.168.1.10 -server-port 5000 -imei 123456789012345
```

Increase retries and decrease retry interval:

```bash
go run main.go \
  -retry-connect 5 \
  -sleep-retry 1
```

Run with a 10-second 8E send interval and a 2-second server response timeout:

```bash
go run main.go \
  -server-ip 127.0.0.1 \
  -server-port 5000 \
  -imei 123456789012345 \
  -interval-8e 10 \
  -timeout-response 2
```

## Docker

Build the Docker image from the repository root:

```bash
docker build -t teltonika-tracker .
```

Run the container with required flags:

```bash
docker run --rm teltonika-tracker \
  -server-ip 127.0.0.1 \
  -server-port 5000 \
  -imei 123456789012345 \
  -interval-8e 10 \
  -timeout-response 2
```

If the server is running on the Docker host, use host networking:

```bash
docker run --rm --network host teltonika-tracker \
  -server-ip 127.0.0.1 \
  -server-port 5000 \
  -imei 123456789012345 \
  -interval-8e 10 \
  -timeout-response 2
```
