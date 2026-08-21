package memory

import (
	"context"
	"testing"

	"github.com/example/iot-device-management/internal/firmware/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
)

func TestGetTaskMissingReturnsNotFoundKind(t *testing.T) {
	repo := NewRepository()
	_, err := repo.GetTask(context.Background(), "missing")
	if !apperr.IsKind(err, apperr.KindNotFound) {
		t.Fatalf("expected not_found kind, got %s (%v)", apperr.KindOf(err), err)
	}
}

func TestSaveReceiptClonesValue(t *testing.T) {
	repo := NewRepository()
	receipt := domain.DeviceReceipt{ID: "receipt-1", Message: "initial"}
	if err := repo.SaveReceipt(context.Background(), receipt); err != nil {
		t.Fatal(err)
	}
	receipt.Message = "mutated"
	got, err := repo.GetReceipt(context.Background(), "receipt-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Message != "initial" {
		t.Fatal("stored receipt aliases caller input")
	}
}
