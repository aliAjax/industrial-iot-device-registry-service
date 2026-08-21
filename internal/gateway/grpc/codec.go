package grpc

import (
	"encoding/json"
)

type jsonCodec struct{}

func (jsonCodec) Name() string {
	return "json"
}

func (jsonCodec) Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func (jsonCodec) Unmarshal(data []byte, value any) error {
	return json.Unmarshal(data, value)
}
