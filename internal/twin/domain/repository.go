package domain

import "context"

type Repository interface {
	Get(ctx context.Context, deviceID string) (TwinDocument, error)
	Save(ctx context.Context, document TwinDocument) error
	Delete(ctx context.Context, deviceID string) error
}
