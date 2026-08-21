package application

import (
	"context"
	"testing"

	"github.com/example/iot-device-management/internal/firmware/domain"
	"github.com/example/iot-device-management/internal/firmware/infrastructure/memory"
)

func TestCompleteReceiptReturnsTaskLookupError(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.SaveReceipt(context.Background(), domain.DeviceReceipt{
		ID:       "receipt-1",
		TaskID:   "missing-task",
		DeviceID: "dev-1",
		Status:   domain.StatusRunning,
	})
	svc := NewService(repo, nil, nil)
	_, err := svc.CompleteReceipt(context.Background(), "receipt-1", false, "flash failed")
	if err == nil {
		t.Fatal("expected task lookup error to be returned")
	}
}

func TestValidateRolloutRejectsZeroPercentage(t *testing.T) {
	svc := NewService(memory.NewRepository(), nil, nil)
	err := svc.ValidateRollout(domain.RolloutPolicy{Strategy: "percentage", Percentage: 0, MaxConcurrent: 1})
	if err == nil {
		t.Fatal("expected zero percentage to be rejected")
	}
}
