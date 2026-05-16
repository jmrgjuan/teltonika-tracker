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

## InfluxDB + Grafana (docker-compose)

This repository includes a `docker-compose.yml` that starts an InfluxDB 2.x instance and Grafana. Use it to collect and visualize tracker telemetry.

1. Start services:

```bash
docker-compose up -d
```

2. Export Influx connection variables so the server can write points (example values match `docker-compose.yml`):

```bash
export INFLUX_URL=http://localhost:8086
export INFLUX_TOKEN=my-token
export INFLUX_ORG=my-org
export INFLUX_BUCKET=teltonika
```

3. Build and run the server (it will initialize the Influx client from the env variables):

```bash
go build -o bin/teltonika-server ./cmd/server
./bin/teltonika-server --listen-port 5000
```

4. Start one or more simulators (client) to send data to the server:

```bash
go build -o bin/teltonika-tracker main.go
./bin/teltonika-tracker -server-ip 127.0.0.1 -server-port 5000 -imei 123456789012345
```

5. Verify data:
- Influx UI: http://localhost:8086 (use the credentials from `docker-compose.yml` to log in during initial setup)
- Grafana: http://localhost:3000 (Grafana is started but dashboard provisioning is left for later)

Notes:
- The server writes measurement `teltonika` with tag `imei` and fields `lat`, `lon`, `alt`, `speed`.
- If you prefer to containerize the server and client, I can add `Dockerfile` entries and include them as services in `docker-compose.yml`.
