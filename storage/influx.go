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

	// Retry logic for connection
	maxRetries := 15
	retryInterval := 2 * time.Second

	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		client = influxdb2.NewClient(url, token)
		writeAPI = client.WriteAPIBlocking(org, bucket)

		// Simple health check
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = client.Ready(ctx)
		cancel()

		if err == nil {
			enabled = true
			log.Printf("[INFO] InfluxDB client initialized (org=%s bucket=%s url=%s)", org, bucket, url)
			return nil
		}

		log.Printf("[WARN] InfluxDB connection failed (attempt %d/%d): %v, retrying in %v...", attempt, maxRetries, err, retryInterval)
		client.Close()
		time.Sleep(retryInterval)
	}

	log.Printf("[ERROR] Failed to initialize InfluxDB after %d attempts: %v", maxRetries, err)
	enabled = false
	return fmt.Errorf("influx not ready after retries: %w", err)
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

	// convert timestamp (record.Timestamp is in seconds since epoch)
	t := time.Unix(int64(record.Timestamp), 0)

	p := influxdb2.NewPointWithMeasurement("teltonika").
		AddTag("imei", imei).
		AddField("lat", protocol.Int32ToDegrees(record.Latitude)).
		AddField("lon", protocol.Int32ToDegrees(record.Longitude)).
		AddField("alt", record.Altitude).
		AddField("speed", record.Speed).
		SetTime(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := writeAPI.WritePoint(ctx, p)
	if err != nil {
		log.Printf("[ERROR] Failed to write point to InfluxDB: %v", err)
		return err
	}
	log.Printf("[DEBUG] Written point to InfluxDB: imei=%s lat=%.7f lon=%.7f", imei, protocol.Int32ToDegrees(record.Latitude), protocol.Int32ToDegrees(record.Longitude))
	return nil
