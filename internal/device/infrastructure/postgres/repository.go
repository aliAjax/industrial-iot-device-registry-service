package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/example/iot-device-management/internal/device/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/platform/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateProduct(ctx context.Context, product domain.Product) error {
	return postgres.UpsertJSON(ctx, r.pool, "device_products", product.ID, product)
}

func (r *Repository) UpdateProduct(ctx context.Context, product domain.Product) error {
	if err := r.CreateProduct(ctx, product); err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	var product domain.Product
	if err := postgres.GetJSON(ctx, r.pool, "device_products", id, &product); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Product{}, apperr.E(apperr.KindNotFound, "postgres.GetProduct", "product not found", err)
		}
		return domain.Product{}, err
	}
	return product, nil
}

func (r *Repository) ListProducts(ctx context.Context, offset, limit int) ([]domain.Product, int, error) {
	items, err := postgres.ListJSON[domain.Product](ctx, r.pool, "device_products", "", nil, "data->>'created_at' DESC", offset, limit)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM device_products").Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) CreateDevice(ctx context.Context, device domain.Device) error {
	data, err := json.Marshal(device)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO devices (id, product_id, status, data, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			product_id = EXCLUDED.product_id,
			status = EXCLUDED.status,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at`,
		device.ID, device.ProductID, string(device.Status), data, time.Now().UTC())
	return err
}

func (r *Repository) UpdateDevice(ctx context.Context, device domain.Device) error {
	return r.CreateDevice(ctx, device)
}

func (r *Repository) GetDevice(ctx context.Context, id string) (domain.Device, error) {
	var device domain.Device
	if err := postgres.GetJSON(ctx, r.pool, "devices", id, &device); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Device{}, apperr.E(apperr.KindNotFound, "postgres.GetDevice", "device not found", err)
		}
		return domain.Device{}, err
	}
	return device, nil
}

func (r *Repository) DeleteDevice(ctx context.Context, id string) error {
	if err := postgres.DeleteByID(ctx, r.pool, "devices", id); err != nil {
		return apperr.E(apperr.KindNotFound, "postgres.DeleteDevice", "device not found", err)
	}
	return nil
}

func (r *Repository) ListDevices(ctx context.Context, filter domain.DeviceFilter, offset, limit int) ([]domain.Device, int, error) {
	items, err := postgres.ListJSON[domain.Device](ctx, r.pool, "devices", "", nil, "data->>'created_at' DESC", offset, limit)
	if err != nil {
		return nil, 0, err
	}
	filtered := make([]domain.Device, 0, len(items))
	for _, device := range items {
		if matchesDevice(device, filter) {
			filtered = append(filtered, device)
		}
	}
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM devices").Scan(&total); err != nil {
		return nil, 0, err
	}
	if total > len(filtered) {
		total = len(filtered)
	}
	return filtered, total, nil
}

func (r *Repository) CreateGroup(ctx context.Context, group domain.Group) error {
	return postgres.UpsertJSON(ctx, r.pool, "device_groups", group.ID, group)
}

func (r *Repository) UpdateGroup(ctx context.Context, group domain.Group) error {
	return postgres.UpsertJSON(ctx, r.pool, "device_groups", group.ID, group)
}

func (r *Repository) GetGroup(ctx context.Context, id string) (domain.Group, error) {
	var group domain.Group
	if err := postgres.GetJSON(ctx, r.pool, "device_groups", id, &group); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Group{}, apperr.E(apperr.KindNotFound, "postgres.GetGroup", "group not found", err)
		}
		return domain.Group{}, err
	}
	return group, nil
}

func (r *Repository) ListGroups(ctx context.Context, offset, limit int) ([]domain.Group, int, error) {
	items, err := postgres.ListJSON[domain.Group](ctx, r.pool, "device_groups", "", nil, "data->>'created_at' DESC", offset, limit)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM device_groups").Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func matchesDevice(device domain.Device, filter domain.DeviceFilter) bool {
	if filter.ProductID != "" && device.ProductID != filter.ProductID {
		return false
	}
	if filter.Status != "" && device.Status != filter.Status {
		return false
	}
	if filter.GroupID != "" && !contains(device.GroupIDs, filter.GroupID) {
		return false
	}
	if filter.Query != "" {
		query := strings.ToLower(filter.Query)
		if !strings.Contains(strings.ToLower(device.Name), query) && !strings.Contains(strings.ToLower(device.SerialNumber), query) && !strings.Contains(strings.ToLower(device.ID), query) {
			return false
		}
	}
	for key, value := range filter.Tags {
		if device.Tags[key] != value {
			return false
		}
	}
	return true
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
