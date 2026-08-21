package domain

import (
	"time"
)

type Kind string

const (
	KindTelemetry       Kind = "telemetry"
	KindEvent           Kind = "event"
	KindAttributeChange Kind = "attribute_change"
)

type Message struct {
	ID        string         `json:"id,omitempty"`
	DeviceID  string         `json:"device_id"`
	Kind      Kind           `json:"kind"`
	Timestamp time.Time      `json:"timestamp,omitempty"`
	Data      map[string]any `json:"data"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Result struct {
	MessageID  string    `json:"message_id"`
	DeviceID   string    `json:"device_id"`
	Kind       Kind      `json:"kind"`
	AcceptedAt time.Time `json:"accepted_at"`
	Points     int       `json:"points"`
	Status     string    `json:"status"`
	TraceID    string    `json:"trace_id"`
}

type IngestError struct {
	MessageID string
	DeviceID  string
	Err       error
}

func (e IngestError) Error() string {
	return e.Err.Error()
}

func (e IngestError) Unwrap() error {
	return e.Err
}
