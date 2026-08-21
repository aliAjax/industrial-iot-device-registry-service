package application

import (
	"context"
	"testing"
	"time"

	"github.com/example/iot-device-management/internal/telemetry/domain"
)

func TestStartDoesNotCloseQueueBeforeIngest(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("unexpected panic after Start: %v", recovered)
		}
	}()
	svc := NewService(Config{
		QueueSize:    2,
		WorkerCount:  1,
		MaxBatchSize: 10,
		MaxClockSkew: time.Minute,
	}, nil, nil, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)
	if !svc.queue.TryEnqueue(workItem{}) {
		t.Fatal("queue was closed before the service context ended")
	}
	_, err := svc.Ingest(context.Background(), domain.Message{
		DeviceID: "dev-1",
		Kind:     domain.KindTelemetry,
		Data:     map[string]any{"temperature": 21.5},
	})
	if err != nil {
		t.Fatalf("ingest after Start failed: %v", err)
	}
}
