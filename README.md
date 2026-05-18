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

This repository includes a `docker-compose.yml` that starts InfluxDB 2 and Grafana. Use it to collect and visualize tracker telemetry. All services are automatically configured and run together.

### Option A: Run everything with docker-compose (recommended for demos)

1. Start all services (InfluxDB, Grafana, server, and tracker):

```bash
docker-compose up -d
```

This will build and start:
- `influxdb` (port 8086) — initializes bucket `teltonika`, token `my-token`, org `my-org`
- `grafana` (port 3000) — admin dashboard
- `server` (port 5000) — TCP server that writes to InfluxDB
- `tracker` (no exposed port) — simulated device sending data to server

2. Wait for services to fully initialize (~10-15 seconds):

```bash
docker-compose ps
```

All services should show `Up`. If any show `Exited`, check logs:

```bash
docker-compose logs server
docker-compose logs tracker
docker-compose logs influxdb
```

3. Access InfluxDB UI at **http://localhost:8086**:
   - **Username**: `admin`
   - **Password**: `adminpass`
   - **Organization**: `my-org`
   - **Token**: `my-token`

   The bucket `teltonika` is pre-created and should already contain data from the tracker.

4. Verify data is being recorded:
   - In InfluxDB UI: go to **Data Explorer** → select bucket `teltonika` → run query
   - You should see measurement `teltonika` with records tagged by `imei` and fields `lat`, `lon`, `alt`, `speed`

5. Access Grafana at **http://localhost:3000**:
   - **Username**: `admin`
   - **Password**: `admin`

   (Dashboard provisioning is left for manual setup later)

6. To modify device count or IMEI, edit `docker-compose.yml`:

```yaml
tracker:
  environment:
    - IMEI=123456789012345
    - NUM_DEVICES=5  # Change this to simulate multiple devices
```

Then rebuild and restart:

```bash
docker-compose up -d --build
```

7. To stop all services:

```bash
docker-compose down
```

To stop and remove volumes (resets InfluxDB data):

```bash
docker-compose down -v
```

### Option B: Run server locally, use InfluxDB + Grafana from docker-compose

1. Start only InfluxDB and Grafana:

```bash
docker-compose up -d influxdb grafana
```

2. Export Influx connection variables (using `localhost` since running from host):

```bash
export INFLUX_URL=http://localhost:8086
export INFLUX_TOKEN=my-token
export INFLUX_ORG=my-org
export INFLUX_BUCKET=teltonika
```

3. Build and run the server locally:

```bash
go build -o bin/teltonika-server ./cmd/server
./bin/teltonika-server --listen-port 5000
```

4. Start one or more simulators (client) to send data to the server:

```bash
go build -o bin/teltonika-tracker main.go
./bin/teltonika-tracker -server-ip 127.0.0.1 -server-port 5000 -imei 123456789012345 -num-devices 1
```

5. Verify data:
- Influx UI: http://localhost:8086
- Grafana: http://localhost:3000

### Notes

- The server writes measurement `teltonika` with tag `imei` and fields `lat`, `lon`, `alt`, `speed`.
- When running with docker-compose, use `INFLUX_URL=http://influxdb:8086` (service name) inside containers.
- When running locally, use `INFLUX_URL=http://localhost:8086`.

## Querying Data

### InfluxDB UI (Data Explorer)

1. Go to **http://localhost:8086** and log in
2. Click **Data Explorer** (left sidebar)
3. Select bucket `teltonika`
4. The UI shows:
   - **Measurements**: `teltonika`
   - **Fields**: `lat`, `lon`, `alt`, `speed`
   - **Tags**: `imei`

5. Quick example query (click **Script Editor** for advanced):
   ```flux
   from(bucket: "teltonika")
     |> range(start: -1h)
     |> filter(fn: (r) => r._measurement == "teltonika")
     |> filter(fn: (r) => r.imei == "123456789012345")
   ```

6. Click **Submit** to see data

### Grafana Dashboard

1. Go to **http://localhost:3000** and log in (admin/admin)
2. **Add Data Source**:
   - Click **Configuration** (gear icon) → **Data Sources**
   - Click **Add data source**
   - Select **InfluxDB**
   - Configure:
     - **URL**: `http://influxdb:8086`
     - **Organization**: `my-org`
     - **Token**: `my-token`
     - **Default Bucket**: `teltonika`
   - Click **Save & Test**

3. **Create a new Dashboard**:
   - Click **+** → **Dashboard**
   - Click **Add new panel**
   - In **Query** section, select your InfluxDB data source
   - Example Flux query to show latest coordinates by IMEI:
     ```flux
     from(bucket: "teltonika")
       |> range(start: -1h)
       |> filter(fn: (r) => r._measurement == "teltonika")
       |> last()
     ```
   - Choose visualization (e.g., **Graph**, **Table**, **Stat**)
   - Click **Save**

4. **Example panel ideas**:
   - **Map**: Plot `lat`/`lon` on a map (use Grafana's native map or external plugin)
   - **Graph**: Show `speed` over time
   - **Table**: Display all records with `imei`, `lat`, `lon`, `alt`, `speed`
   - **Stat**: Show latest GPS coordinates per device

### CLI Query (optional)

If you prefer to query InfluxDB from terminal, use `influx` CLI inside the container:

```bash
docker exec teltonika-influxdb influx query 'from(bucket: "teltonika") |> range(start: -1h)'
```

Or with authentication:

```bash
docker exec teltonika-influxdb influx query \
  --org my-org \
  --token my-token \
  'from(bucket: "teltonika") |> range(start: -1h) |> filter(fn: (r) => r._measurement == "teltonika")'
```
