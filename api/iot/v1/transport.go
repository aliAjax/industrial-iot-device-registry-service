package v1

import (
	"time"

	"github.com/example/iot-device-management/internal/device/domain"
	tsdomain "github.com/example/iot-device-management/internal/timeseries/domain"
)

type HealthRequest struct{}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Time    string `json:"time"`
}

type DeviceRequest struct {
	DeviceID string `json:"device_id"`
}

type DeviceResponse struct {
	Device domain.Device `json:"device"`
}

type IngestRequest struct {
	DeviceID  string         `json:"device_id"`
	Kind      string         `json:"kind"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

type IngestResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

type CommandRequest struct {
	DeviceID       string         `json:"device_id"`
	Name           string         `json:"name"`
	Payload        map[string]any `json:"payload"`
	IdempotencyKey string         `json:"idempotency_key"`
	MaxRetries     int            `json:"max_retries"`
	Timeout        time.Duration  `json:"timeout"`
}

type CommandResponse struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
}

type TwinRequest struct {
	DeviceID string `json:"device_id"`
}

type TwinResponse struct {
	DeviceID      string         `json:"device_id"`
	DesiredState  map[string]any `json:"desired_state"`
	ReportedState map[string]any `json:"reported_state"`
	Version       int64          `json:"version"`
}

type TimeSeriesRequest struct {
	DeviceID string    `json:"device_id"`
	Property string    `json:"property"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Limit    int       `json:"limit"`
}

type TimeSeriesResponse struct {
	Points []tsdomain.Point `json:"points"`
	Count  int              `json:"count"`
}
