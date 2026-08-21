package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type jsonCodec struct{}

func (jsonCodec) Name() string { return "json" }
func (jsonCodec) Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}
func (jsonCodec) Unmarshal(data []byte, value any) error {
	return json.Unmarshal(data, value)
}

func main() {
	addr := flag.String("addr", "localhost:29090", "gRPC server address")
	flag.Parse()
	conn, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(jsonCodec{})),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var response map[string]any
	if err := conn.Invoke(ctx, "/iot.v1.IoTService/Health", map[string]any{}, &response); err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", response)
}
