package memory

import (
	"context"
	"testing"

	"github.com/example/iot-device-management/internal/command/domain"
)

func TestSaveCopiesCommandPayload(t *testing.T) {
	repo := NewRepository()
	command := domain.Command{
		ID:      "cmd-1",
		Payload: map[string]any{"mode": "eco"},
	}
	if err := repo.Save(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	command.Payload["mode"] = "turbo"
	got, err := repo.Get(context.Background(), "cmd-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Payload["mode"] != "eco" {
		t.Fatalf("stored command aliases caller input: %v", got.Payload["mode"])
	}
}
