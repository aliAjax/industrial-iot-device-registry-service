package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	url := flag.String("url", "ws://localhost:28080/api/v1/telemetry/ws", "WebSocket endpoint")
	deviceID := flag.String("device-id", "", "device id")
	credential := flag.String("credential", "", "device credential")
	value := flag.Float64("value", 24.5, "temperature value")
	flag.Parse()
	if *deviceID == "" || *credential == "" {
		fmt.Fprintln(os.Stderr, "device-id and credential are required")
		os.Exit(2)
	}
	header := http.Header{}
	header.Set("X-Device-ID", *deviceID)
	header.Set("X-Device-Credential", *credential)
	conn, _, err := websocket.DefaultDialer.Dial(*url, header)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	message := map[string]any{
		"kind":      "telemetry",
		"timestamp": time.Now().UTC(),
		"data":      map[string]any{"temperature": *value},
	}
	if err := conn.WriteJSON(message); err != nil {
		panic(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, payload, err := conn.ReadMessage()
	if err != nil {
		panic(err)
	}
	var result map[string]any
	_ = json.Unmarshal(payload, &result)
	fmt.Printf("%v\n", result)
}
