package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker")
	topic := flag.String("topic", "devices/demo/telemetry", "publish topic")
	value := flag.Float64("value", 22.5, "temperature value")
	flag.Parse()
	opts := mqtt.NewClientOptions().AddBroker(*broker).SetClientID("example-device")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Fprintln(os.Stderr, token.Error())
		os.Exit(1)
	}
	payload, _ := json.Marshal(map[string]any{
		"kind":      "telemetry",
		"timestamp": time.Now().UTC(),
		"data":      map[string]any{"temperature": *value},
	})
	token := client.Publish(*topic, 1, false, payload)
	token.Wait()
	fmt.Printf("published to %s\n", *topic)
	client.Disconnect(100)
}
