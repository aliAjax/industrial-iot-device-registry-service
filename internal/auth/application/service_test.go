package application

import (
	"context"
	"testing"
	"time"

	"github.com/example/iot-device-management/internal/auth/infrastructure/memory"
	"github.com/example/iot-device-management/internal/platform/apperr"
)

func TestAuthenticateUnknownDeviceYieldsUnauthorized(t *testing.T) {
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, time.Hour, 120, time.Minute)
	_, err := svc.Authenticate(context.Background(), "missing-device", "secret", "")
	if !apperr.IsKind(err, apperr.KindUnauthorized) {
		t.Fatalf("expected unauthorized, got %s (%v)", apperr.KindOf(err), err)
	}
}

func TestRegisterDeviceNormalizesCredentialType(t *testing.T) {
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, time.Hour, 120, time.Minute)
	if err := svc.RegisterDevice(context.Background(), "dev-1", "prod-1", "PSK", "secret", nil); err != nil {
		t.Fatal(err)
	}
	principal, err := repo.FindPrincipal(context.Background(), "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if principal.CredentialType != "psk" {
		t.Fatalf("expected normalized psk credential type, got %s", principal.CredentialType)
	}
}

func TestRegisterDeviceNormalizesScopes(t *testing.T) {
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, time.Hour, 120, time.Minute)
	if err := svc.RegisterDevice(context.Background(), "dev-1", "prod-1", "token", "secret", []string{" device:dev-1:telemetry ", "device:dev-1:telemetry", "*"}); err != nil {
		t.Fatal(err)
	}
	principal, err := repo.FindPrincipal(context.Background(), "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(principal.Scopes) != 2 || principal.Scopes[0] != "device:dev-1:telemetry" || principal.Scopes[1] != "*" {
		t.Fatalf("scopes were not normalized: %#v", principal.Scopes)
	}
}
