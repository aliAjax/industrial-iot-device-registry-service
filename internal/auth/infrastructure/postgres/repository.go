package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/example/iot-device-management/internal/auth/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) UpsertPrincipal(ctx context.Context, principal domain.Principal) error {
	data, err := json.Marshal(principal)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, "INSERT INTO auth_principals (id, data, updated_at) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, updated_at = EXCLUDED.updated_at", principal.DeviceID, data, time.Now().UTC())
	return err
}

func (r *Repository) FindPrincipal(ctx context.Context, deviceID string) (domain.Principal, error) {
	row := r.pool.QueryRow(ctx, "SELECT data FROM auth_principals WHERE id = $1", deviceID)
	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Principal{}, apperr.E(apperr.KindNotFound, "auth.postgres.FindPrincipal", "principal not found", err)
		}
		return domain.Principal{}, err
	}
	var principal domain.Principal
	if err := json.Unmarshal(data, &principal); err != nil {
		return domain.Principal{}, err
	}
	return principal, nil
}

func (r *Repository) DeletePrincipal(ctx context.Context, deviceID string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM auth_principals WHERE id = $1", deviceID)
	return err
}

func (r *Repository) IncrementRate(ctx context.Context, key string, window time.Duration, limit int) (domain.RateRecord, error) {
	now := time.Now().UTC()
	cutoff := now.Add(-window)
	var bucketTime time.Time
	var count int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO auth_rates (rate_key, window_start, count)
		VALUES ($1, $2, 1)
		ON CONFLICT (rate_key) DO UPDATE SET
			window_start = CASE WHEN auth_rates.window_start < $3 THEN EXCLUDED.window_start ELSE auth_rates.window_start END,
			count = CASE WHEN auth_rates.window_start < $3 THEN 1 ELSE auth_rates.count + 1 END
		RETURNING window_start, count`,
		key, now, cutoff,
	).Scan(&bucketTime, &count)
	if err != nil {
		return domain.RateRecord{}, err
	}
	return domain.RateRecord{Key: key, Window: bucketTime, Count: count, Limit: limit}, nil
}
