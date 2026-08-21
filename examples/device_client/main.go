package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type message struct {
	DeviceID  string         `json:"device_id"`
	Kind      string         `json:"kind"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

func main() {
	base := flag.String("base", "http://localhost:8080", "platform base URL")
	deviceID := flag.String("device-id", "", "device id")
	credential := flag.String("credential", "", "device credential")
	value := flag.Float64("value", 25, "temperature value")
	flag.Parse()
	if *deviceID == "" || *credential == "" {
		fmt.Fprintln(os.Stderr, "device-id and credential are required")
		os.Exit(2)
	}
	payload := message{
		DeviceID:  *deviceID,
		Kind:      "telemetry",
		Timestamp: time.Now().UTC(),
		Data:      map[string]any{"temperature": *value},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, *base+"/api/v1/telemetry/ingest", bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-ID", *deviceID)
	req.Header.Set("X-Device-Credential", *credential)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Printf("status=%d\n", resp.StatusCode)
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	fmt.Printf("%v\n", out)
}
