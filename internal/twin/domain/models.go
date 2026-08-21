package domain

import (
	"encoding/json"
	"time"
)

type TwinDocument struct {
	DeviceID      string         `json:"device_id"`
	DesiredState  map[string]any `json:"desired_state"`
	ReportedState map[string]any `json:"reported_state"`
	Version       int64          `json:"version"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type StateDiff struct {
	Property string `json:"property"`
	Path     string `json:"path"`
	Desired  any    `json:"desired"`
	Reported any    `json:"reported"`
}

type MergeResult struct {
	DeviceID      string         `json:"device_id"`
	DesiredState  map[string]any `json:"desired_state"`
	ReportedState map[string]any `json:"reported_state"`
	Version       int64          `json:"version"`
	Diff          []StateDiff    `json:"diff"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (d TwinDocument) Clone() TwinDocument {
	out := d
	out.DesiredState = cloneMap(d.DesiredState)
	out.ReportedState = cloneMap(d.ReportedState)
	return out
}

func (d TwinDocument) MarshalJSON() ([]byte, error) {
	type alias TwinDocument
	return json.Marshal(alias(d))
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneValue(value)
	}
	return out
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneValue(typed[i])
		}
		return out
	default:
		return value
	}
}
