package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/example/iot-device-management/internal/command/domain"
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

func (r *Repository) Save(ctx context.Context, command domain.Command) error {
	data, err := json.Marshal(command)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO commands (id, device_id, idempotency_key, data, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			device_id = EXCLUDED.device_id,
			idempotency_key = EXCLUDED.idempotency_key,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at`,
		command.ID, command.DeviceID, command.IdempotencyKey, data, time.Now().UTC())
	return err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Command, error) {
	row := r.pool.QueryRow(ctx, "SELECT data FROM commands WHERE id = $1", id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Command{}, apperr.E(apperr.KindNotFound, "command.postgres.Get", "command not found", err)
		}
		return domain.Command{}, err
	}
	var command domain.Command
	if err := json.Unmarshal(data, &command); err != nil {
		return domain.Command{}, err
	}
	return command, nil
}

func (r *Repository) GetByIdempotencyKey(ctx context.Context, deviceID, key string) (domain.Command, error) {
	row := r.pool.QueryRow(ctx, "SELECT data FROM commands WHERE device_id = $1 AND idempotency_key = $2", deviceID, key)
	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Command{}, apperr.E(apperr.KindNotFound, "command.postgres.GetByIdempotencyKey", "command not found", err)
		}
		return domain.Command{}, err
	}
	var command domain.Command
	if err := json.Unmarshal(data, &command); err != nil {
		return domain.Command{}, err
	}
	return command, nil
}

func (r *Repository) List(ctx context.Context, filter domain.CommandFilter, offset, limit int) ([]domain.Command, int, error) {
	rows, err := r.pool.Query(ctx, "SELECT data FROM commands")
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	all := make([]domain.Command, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, 0, err
		}
		var command domain.Command
		if err := json.Unmarshal(data, &command); err != nil {
			return nil, 0, err
		}
		all = append(all, command)
	}
	filtered := make([]domain.Command, 0)
	for _, command := range all {
		if filter.DeviceID != "" && command.DeviceID != filter.DeviceID {
			continue
		}
		if filter.Status != "" && command.Status != filter.Status {
			continue
		}
		if filter.Query != "" {
			query := strings.ToLower(filter.Query)
			if !strings.Contains(strings.ToLower(command.Name), query) && !strings.Contains(strings.ToLower(command.ID), query) {
				continue
			}
		}
		filtered = append(filtered, command)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.After(filtered[j].CreatedAt) })
	total := len(filtered)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (r *Repository) PendingForDevice(ctx context.Context, deviceID string, limit int) ([]domain.Command, error) {
	rows, err := r.pool.Query(ctx, "SELECT data FROM commands WHERE device_id = $1 ORDER BY data->>'created_at' ASC LIMIT $2", deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Command, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var command domain.Command
		if err := json.Unmarshal(data, &command); err != nil {
			return nil, err
		}
		if command.Status == domain.StatusPending {
			items = append(items, command)
		}
	}
	return items, nil
}

func (r *Repository) Expired(ctx context.Context, before time.Time, limit int) ([]domain.Command, error) {
	rows, err := r.pool.Query(ctx, "SELECT data FROM commands WHERE data->>'status' = 'pending' ORDER BY data->>'expires_at' ASC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Command, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var command domain.Command
		if err := json.Unmarshal(data, &command); err != nil {
			return nil, err
		}
		if command.ExpiresAt.Before(before) {
			items = append(items, command)
		}
	}
	return items, nil
}
