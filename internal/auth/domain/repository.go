package domain

import (
	"context"
	"time"
)

type Repository interface {
	UpsertPrincipal(ctx context.Context, principal Principal) error
	FindPrincipal(ctx context.Context, deviceID string) (Principal, error)
	DeletePrincipal(ctx context.Context, deviceID string) error
	IncrementRate(ctx context.Context, key string, window time.Duration, limit int) (RateRecord, error)
}

type DeviceLookup interface {
	FindDevice(ctx context.Context, deviceID string) (DeviceIdentity, error)
}

type DeviceIdentity struct {
	ID        string
	ProductID string
	Enabled   bool
}
