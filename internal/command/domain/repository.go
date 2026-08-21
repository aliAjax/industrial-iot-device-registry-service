package domain

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, command Command) error
	Get(ctx context.Context, id string) (Command, error)
	GetByIdempotencyKey(ctx context.Context, deviceID, key string) (Command, error)
	List(ctx context.Context, filter CommandFilter, offset, limit int) ([]Command, int, error)
	PendingForDevice(ctx context.Context, deviceID string, limit int) ([]Command, error)
	Expired(ctx context.Context, before time.Time, limit int) ([]Command, error)
}

type CommandFilter struct {
	DeviceID string
	Status   Status
	Query    string
}
