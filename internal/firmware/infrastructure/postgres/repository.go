package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/example/iot-device-management/internal/firmware/domain"
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

func (r *Repository) SaveFirmware(ctx context.Context, firmware domain.Firmware) error {
	data, err := json.Marshal(firmware)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, "INSERT INTO firmware (id, data, updated_at) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, updated_at = EXCLUDED.updated_at", firmware.ID, data, time.Now().UTC())
	return err
}

func (r *Repository) GetFirmware(ctx context.Context, id string) (domain.Firmware, error) {
	row := r.pool.QueryRow(ctx, "SELECT data FROM firmware WHERE id = $1", id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Firmware{}, apperr.E(apperr.KindNotFound, "firmware.postgres.GetFirmware", "firmware not found", err)
		}
		return domain.Firmware{}, err
	}
	var firmware domain.Firmware
	if err := json.Unmarshal(data, &firmware); err != nil {
		return domain.Firmware{}, err
	}
	return firmware, nil
}

func (r *Repository) ListFirmware(ctx context.Context, productID string, offset, limit int) ([]domain.Firmware, int, error) {
	rows, err := r.pool.Query(ctx, "SELECT data FROM firmware ORDER BY data->>'created_at' DESC")
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	all := make([]domain.Firmware, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, 0, err
		}
		var firmware domain.Firmware
		if err := json.Unmarshal(data, &firmware); err != nil {
			return nil, 0, err
		}
		if productID == "" || firmware.ProductID == productID {
			all = append(all, firmware)
		}
	}
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (r *Repository) SaveTask(ctx context.Context, task domain.UpgradeTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, "INSERT INTO firmware_tasks (id, firmware_id, data, updated_at) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET firmware_id = EXCLUDED.firmware_id, data = EXCLUDED.data, updated_at = EXCLUDED.updated_at", task.ID, task.FirmwareID, data, time.Now().UTC())
	return err
}

func (r *Repository) GetTask(ctx context.Context, id string) (domain.UpgradeTask, error) {
	row := r.pool.QueryRow(ctx, "SELECT data FROM firmware_tasks WHERE id = $1", id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UpgradeTask{}, apperr.E(apperr.KindNotFound, "firmware.postgres.GetTask", "task not found", err)
		}
		return domain.UpgradeTask{}, err
	}
	var task domain.UpgradeTask
	if err := json.Unmarshal(data, &task); err != nil {
		return domain.UpgradeTask{}, err
	}
	return task, nil
}

func (r *Repository) ListTasks(ctx context.Context, filter domain.Filter, offset, limit int) ([]domain.UpgradeTask, int, error) {
	rows, err := r.pool.Query(ctx, "SELECT data FROM firmware_tasks ORDER BY data->>'created_at' DESC")
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	all := make([]domain.UpgradeTask, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, 0, err
		}
		var task domain.UpgradeTask
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, 0, err
		}
		all = append(all, task)
	}
	filtered := make([]domain.UpgradeTask, 0)
	for _, task := range all {
		if filter.FirmwareID != "" && task.FirmwareID != filter.FirmwareID {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		filtered = append(filtered, task)
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

func (r *Repository) SaveReceipt(ctx context.Context, receipt domain.DeviceReceipt) error {
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, "INSERT INTO firmware_receipts (id, task_id, device_id, data, updated_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET task_id = EXCLUDED.task_id, device_id = EXCLUDED.device_id, data = EXCLUDED.data, updated_at = EXCLUDED.updated_at", receipt.ID, receipt.TaskID, receipt.DeviceID, data, time.Now().UTC())
	return err
}

func (r *Repository) GetReceipt(ctx context.Context, id string) (domain.DeviceReceipt, error) {
	row := r.pool.QueryRow(ctx, "SELECT data FROM firmware_receipts WHERE id = $1", id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DeviceReceipt{}, apperr.E(apperr.KindNotFound, "firmware.postgres.GetReceipt", "receipt not found", err)
		}
		return domain.DeviceReceipt{}, err
	}
	var receipt domain.DeviceReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return domain.DeviceReceipt{}, err
	}
	return receipt, nil
}

func (r *Repository) ReceiptsForDevice(ctx context.Context, deviceID string, limit int) ([]domain.DeviceReceipt, error) {
	rows, err := r.pool.Query(ctx, "SELECT data FROM firmware_receipts WHERE device_id = $1 ORDER BY data->>'created_at' DESC LIMIT $2", deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.DeviceReceipt, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var receipt domain.DeviceReceipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			return nil, err
		}
		items = append(items, receipt)
	}
	return items, nil
}

func (r *Repository) ActiveReceiptForDevice(ctx context.Context, deviceID string) (domain.DeviceReceipt, error) {
	rows, err := r.pool.Query(ctx, "SELECT data FROM firmware_receipts WHERE device_id = $1", deviceID)
	if err != nil {
		return domain.DeviceReceipt{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return domain.DeviceReceipt{}, err
		}
		var receipt domain.DeviceReceipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			return domain.DeviceReceipt{}, err
		}
		if receipt.Status == domain.StatusRunning || receipt.Status == domain.StatusScheduled {
			return receipt, nil
		}
	}
	return domain.DeviceReceipt{}, apperr.E(apperr.KindNotFound, "firmware.postgres.ActiveReceiptForDevice", "no active receipt", nil)
}
