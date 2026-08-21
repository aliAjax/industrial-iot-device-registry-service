package memory

import (
	"context"
	"testing"

	"github.com/example/iot-device-management/internal/twin/domain"
)

func TestSaveClonesDocument(t *testing.T) {
	repo := NewRepository()
	doc := domain.TwinDocument{
		DeviceID:     "dev-1",
		DesiredState: map[string]any{"mode": "eco"},
		ReportedState: map[string]any{"mode": "normal"},
	}
	if err := repo.Save(context.Background(), doc); err != nil {
		t.Fatal(err)
	}
	doc.DesiredState["mode"] = "turbo"
	got, err := repo.Get(context.Background(), "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.DesiredState["mode"] != "eco" {
		t.Fatalf("stored document aliases caller input: %v", got.DesiredState["mode"])
	}
}
