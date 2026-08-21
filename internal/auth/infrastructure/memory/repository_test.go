package memory

import (
	"context"
	"testing"

	"github.com/example/iot-device-management/internal/auth/domain"
)

func TestUpsertPrincipalClonesScopes(t *testing.T) {
	repo := NewRepository()
	principal := domain.Principal{
		DeviceID: "dev-1",
		Scopes:   []string{"device:dev-1:telemetry"},
	}
	if err := repo.UpsertPrincipal(context.Background(), principal); err != nil {
		t.Fatal(err)
	}
	principal.Scopes[0] = "device:dev-1:commands"
	got, err := repo.FindPrincipal(context.Background(), "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Scopes[0] != "device:dev-1:telemetry" {
		t.Fatalf("stored principal aliases caller input: %v", got.Scopes[0])
	}
}
