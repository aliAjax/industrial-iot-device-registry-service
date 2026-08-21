package application

import (
	"context"
	"testing"
	"time"

	"github.com/example/iot-device-management/internal/timeseries/domain"
	tsmem "github.com/example/iot-device-management/internal/timeseries/infrastructure/memory"
)

func TestWriteBatchDoesNotAliasCallerMetadata(t *testing.T) {
	store := tsmem.NewStore(100)
	svc := NewService(store, nil)
	now := time.Now().UTC().Truncate(time.Second)
	point := domain.Point{
		DeviceID:  "dev-1",
		Property:  "temp",
		Timestamp: now,
		Value:     20,
		Metadata:  map[string]any{"kind": "original"},
	}
	if err := svc.WriteBatch(context.Background(), []domain.Point{point}); err != nil {
		t.Fatal(err)
	}
	point.Metadata["kind"] = "mutated"
	points, err := svc.Query(context.Background(), domain.Query{
		DeviceID: "dev-1",
		Property: "temp",
		Start:    now.Add(-time.Hour),
		End:      now.Add(time.Hour),
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := points[0].Metadata["kind"]; got != "original" {
		t.Fatalf("stored metadata aliases caller input, got %v", got)
	}
}

func TestDetectGapsDefaultsWindowForward(t *testing.T) {
	store := tsmem.NewStore(100)
	svc := NewService(store, nil)
	if _, err := svc.DetectGaps(context.Background(), domain.GapRequest{
		DeviceID:         "dev-1",
		Property:         "temp",
		ExpectedInterval: time.Minute,
	}); err != nil {
		t.Fatalf("detect gaps with default window failed: %v", err)
	}
}
