package application

import (
	"context"
	"testing"
	"time"

	"github.com/example/iot-device-management/internal/command/domain"
	"github.com/example/iot-device-management/internal/command/infrastructure/memory"
)

func TestReapExpiredObservesCancellation(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.Save(context.Background(), domain.Command{
		ID:         "cmd-1",
		DeviceID:   "dev-1",
		Status:     domain.StatusPending,
		MaxRetries: 1,
		Timeout:    time.Second,
		ExpiresAt:  time.Now().Add(-time.Second),
	})
	svc := NewService(repo, nil, nil, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc.reapExpired(ctx)
	got, err := repo.Get(context.Background(), "cmd-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusPending {
		t.Fatalf("expected pending after canceled context, got %s", got.Status)
	}
}

func TestEnqueueDeepClonesPayload(t *testing.T) {
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, time.Second)
	payload := map[string]any{"nested": map[string]any{"mode": "eco"}}
	command, err := svc.Enqueue(context.Background(), domain.EnqueueInput{
		DeviceID:       "dev-1",
		Name:           "set-mode",
		Payload:        payload,
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload["nested"].(map[string]any)["mode"] = "turbo"
	got, err := repo.Get(context.Background(), command.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Payload["nested"].(map[string]any)["mode"] != "eco" {
		t.Fatal("stored payload aliases caller input")
	}
}
