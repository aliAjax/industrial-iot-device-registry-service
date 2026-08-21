package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/example/iot-device-management/internal/firmware/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
)

type Repository struct {
	mu        sync.RWMutex
	firmwares map[string]domain.Firmware
	tasks     map[string]domain.UpgradeTask
	receipts  map[string]domain.DeviceReceipt
}

func NewRepository() *Repository {
	return &Repository{
		firmwares: make(map[string]domain.Firmware),
		tasks:     make(map[string]domain.UpgradeTask),
		receipts:  make(map[string]domain.DeviceReceipt),
	}
}

func (r *Repository) SaveFirmware(_ context.Context, firmware domain.Firmware) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.firmwares[firmware.ID] = firmware
	return nil
}

func (r *Repository) GetFirmware(_ context.Context, id string) (domain.Firmware, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	firmware, exists := r.firmwares[id]
	if !exists {
		return domain.Firmware{}, apperr.E(apperr.KindNotFound, "firmware.memory.GetFirmware", "firmware not found", nil)
	}
	return firmware, nil
}

func (r *Repository) ListFirmware(_ context.Context, productID string, offset, limit int) ([]domain.Firmware, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Firmware, 0)
	for _, firmware := range r.firmwares {
		if productID != "" && firmware.ProductID != productID {
			continue
		}
		items = append(items, firmware)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	total := len(items)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total, nil
}

func (r *Repository) SaveTask(_ context.Context, task domain.UpgradeTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.ID] = task
	return nil
}

func (r *Repository) GetTask(_ context.Context, id string) (domain.UpgradeTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, exists := r.tasks[id]
	if !exists {
		return domain.UpgradeTask{}, apperr.E(apperr.KindNotFound, "firmware.memory.GetTask", "task not found", nil)
	}
	return task, nil
}

func (r *Repository) ListTasks(_ context.Context, filter domain.Filter, offset, limit int) ([]domain.UpgradeTask, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.UpgradeTask, 0)
	for _, task := range r.tasks {
		if filter.FirmwareID != "" && task.FirmwareID != filter.FirmwareID {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		if filter.DeviceID != "" {
			found := false
			for _, receipt := range r.receipts {
				if receipt.TaskID == task.ID && receipt.DeviceID == filter.DeviceID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		items = append(items, task)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	total := len(items)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total, nil
}

func (r *Repository) SaveReceipt(_ context.Context, receipt domain.DeviceReceipt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.receipts[receipt.ID] = receipt
	return nil
}

func (r *Repository) GetReceipt(_ context.Context, id string) (domain.DeviceReceipt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	receipt, exists := r.receipts[id]
	if !exists {
		return domain.DeviceReceipt{}, apperr.E(apperr.KindNotFound, "firmware.memory.GetReceipt", "receipt not found", nil)
	}
	return receipt, nil
}

func (r *Repository) ReceiptsForDevice(_ context.Context, deviceID string, limit int) ([]domain.DeviceReceipt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.DeviceReceipt, 0)
	for _, receipt := range r.receipts {
		if receipt.DeviceID == deviceID {
			items = append(items, receipt)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *Repository) ActiveReceiptForDevice(_ context.Context, deviceID string) (domain.DeviceReceipt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, receipt := range r.receipts {
		if receipt.DeviceID == deviceID && (receipt.Status == domain.StatusRunning || receipt.Status == domain.StatusScheduled) {
			return receipt, nil
		}
	}
	return domain.DeviceReceipt{}, apperr.E(apperr.KindNotFound, "firmware.memory.ActiveReceiptForDevice", "no active receipt", nil)
}
