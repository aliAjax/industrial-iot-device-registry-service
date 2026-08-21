package domain

import (
	"time"
)

type DataType string

const (
	DataTypeNumber DataType = "number"
	DataTypeString DataType = "string"
	DataTypeBool   DataType = "boolean"
)

type Point struct {
	DeviceID    string         `json:"device_id"`
	Property    string         `json:"property"`
	Timestamp   time.Time      `json:"timestamp"`
	Value       float64        `json:"value"`
	StringValue string         `json:"string_value,omitempty"`
	BoolValue   *bool          `json:"bool_value,omitempty"`
	DataType    DataType       `json:"data_type"`
	Unit        string         `json:"unit,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type SeriesKey struct {
	DeviceID string
	Property string
}

type Query struct {
	DeviceID   string
	Property   string
	Start      time.Time
	End        time.Time
	Limit      int
	Descending bool
}

type AggregateKind string

const (
	AggregateAvg   AggregateKind = "avg"
	AggregateMin   AggregateKind = "min"
	AggregateMax   AggregateKind = "max"
	AggregateSum   AggregateKind = "sum"
	AggregateCount AggregateKind = "count"
)

type AggregateRequest struct {
	DeviceID string        `json:"device_id"`
	Property string        `json:"property"`
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Kind     AggregateKind `json:"kind"`
	Bucket   time.Duration `json:"bucket"`
}

type AggregateResult struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Value float64   `json:"value"`
	Count int       `json:"count"`
}

type Gap struct {
	DeviceID         string        `json:"device_id"`
	Property         string        `json:"property"`
	Start            time.Time     `json:"start"`
	End              time.Time     `json:"end"`
	Duration         time.Duration `json:"duration"`
	ExpectedInterval time.Duration `json:"expected_interval"`
}

type GapRequest struct {
	DeviceID         string        `json:"device_id"`
	Property         string        `json:"property"`
	Start            time.Time     `json:"start"`
	End              time.Time     `json:"end"`
	ExpectedInterval time.Duration `json:"expected_interval"`
}
