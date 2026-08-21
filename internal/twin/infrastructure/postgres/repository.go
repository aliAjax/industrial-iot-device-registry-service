package postgres

import (
	"context"
	"errors"

	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/platform/postgres"
	"github.com/example/iot-device-management/internal/twin/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Get(ctx context.Context, deviceID string) (domain.TwinDocument, error) {
	var document domain.TwinDocument
	if err := postgres.GetJSON(ctx, r.pool, "twin_documents", deviceID, &document); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TwinDocument{}, apperr.E(apperr.KindNotFound, "twin.postgres.Get", "twin document not found", err)
		}
		return domain.TwinDocument{}, err
	}
	return document, nil
}

func (r *Repository) Save(ctx context.Context, document domain.TwinDocument) error {
	return postgres.UpsertJSON(ctx, r.pool, "twin_documents", document.DeviceID, document)
}

func (r *Repository) Delete(ctx context.Context, deviceID string) error {
	return postgres.DeleteByID(ctx, r.pool, "twin_documents", deviceID)
}
