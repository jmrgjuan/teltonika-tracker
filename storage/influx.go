package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/jmrgjuan/teltonika-tracker/protocol"
)

var (
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	enabled  bool
)

// InitFromEnv initializes InfluxDB client using environment variables.
// Supported env vars: INFLUX_URL, INFLUX_TOKEN, INFLUX_ORG, INFLUX_BUCKET
func InitFromEnv() error {
	url := os.Getenv("INFLUX_URL")
	token := os.Getenv("INFLUX_TOKEN")
	org := os.Getenv("INFLUX_ORG")
	bucket := os.Getenv("INFLUX_BUCKET")

	if url == "" {
		url = "http://localhost:8086"
	}
	if token == "" || org == "" || bucket == "" {
		log.Printf("[WARN] Influx env incomplete; Influx writing disabled unless INFLUX_TOKEN, INFLUX_ORG and INFLUX_BUCKET are set")
		enabled = false
		return nil
	}

	client = influxdb2.NewClient(url, token)
	writeAPI = client.WriteAPIBlocking(org, bucket)
	enabled = true

	// Simple health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.Ready(ctx)
	if err != nil {
		client.Close()
		enabled = false
		return fmt.Errorf("influx not ready: %w", err)
	}

	log.Printf("[INFO] InfluxDB client initialized (org=%s bucket=%s url=%s)", org, bucket, url)
	return nil
}

// Close closes the underlying InfluxDB client.
func Close() {
	if client != nil {
		client.Close()
	}
}

// WriteRecord writes a protocol record as a point into InfluxDB.
func WriteRecord(imei string, record protocol.Codec8ERecord) error {
	if !enabled {
		return nil
	}

	// convert timestamp (assume milliseconds since epoch)
	t := time.Unix(0, int64(record.Timestamp)*int64(time.Millisecond))

	p := influxdb2.NewPointWithMeasurement("teltonika").
		AddTag("imei", imei).
		AddField("lat", protocol.Int32ToDegrees(record.Latitude)).
		AddField("lon", protocol.Int32ToDegrees(record.Longitude)).
		AddField("alt", record.Altitude).
		AddField("speed", record.Speed).
		SetTime(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return writeAPI.WritePoint(ctx, p)
}
