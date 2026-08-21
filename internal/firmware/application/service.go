package application

import (
	"context"
	"github.com/example/iot-device-management/internal/firmware/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
	clockpkg "github.com/example/iot-device-management/internal/platform/clock"
	"github.com/example/iot-device-management/internal/platform/eventbus"
	"github.com/example/iot-device-management/internal/platform/id"
	"sort"
	"strings"
)

type Service struct {
	repo  domain.Repository
	bus   eventbus.Bus
	clock clockpkg.Clock
}

func NewService(repo domain.Repository, bus eventbus.Bus, clock clockpkg.Clock) *Service {
	if clock == nil {
		clock = clockpkg.SystemClock{}
	}
	return &Service{repo: repo, bus: bus, clock: clock}
}

func (s *Service) CreateFirmware(ctx context.Context, firmware domain.Firmware) (domain.Firmware, error) {
	if firmware.ProductID == "" || firmware.Version == "" {
		return domain.Firmware{}, apperr.E(apperr.KindInvalid, "firmware.CreateFirmware", "product id and version are required", nil)
	}
	now := s.clock.Now()
	firmware.ID = id.New("fw")
	firmware.CreatedAt = now
	firmware.UpdatedAt = now
	if err := s.repo.SaveFirmware(ctx, firmware); err != nil {
		return domain.Firmware{}, apperr.E(apperr.KindInternal, "firmware.CreateFirmware", "save firmware", err)
	}
	_ = s.publish(ctx, "firmware.created", firmware.ID, firmware)
	return firmware, nil
}

func (s *Service) ListFirmware(ctx context.Context, productID string, offset, limit int) ([]domain.Firmware, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items, total, err := s.repo.ListFirmware(ctx, productID, offset, limit)
	if err != nil {
		return nil, 0, apperr.E(apperr.KindInternal, "firmware.ListFirmware", "list firmware", err)
	}
	return items, total, nil
}

func (s *Service) CreateTask(ctx context.Context, input domain.TaskInput) (domain.UpgradeTask, error) {
	if input.FirmwareID == "" {
		return domain.UpgradeTask{}, apperr.E(apperr.KindInvalid, "firmware.CreateTask", "firmware id is required", nil)
	}
	if _, err := s.repo.GetFirmware(ctx, input.FirmwareID); err != nil {
		return domain.UpgradeTask{}, apperr.E(apperr.KindNotFound, "firmware.CreateTask", "firmware not found", err)
	}
	now := s.clock.Now()
	task := domain.UpgradeTask{
		ID:         id.New("task"),
		FirmwareID: input.FirmwareID,
		Status:     domain.StatusDraft,
		Rollout:    input.Rollout.Normalized(),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.SaveTask(ctx, task); err != nil {
		return domain.UpgradeTask{}, apperr.E(apperr.KindInternal, "firmware.CreateTask", "save task", err)
	}
	_ = s.publish(ctx, "firmware.task_created", task.ID, task)
	return task, nil
}

func (s *Service) TransitionTask(ctx context.Context, taskID string, target domain.Status) (domain.UpgradeTask, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return domain.UpgradeTask{}, apperr.E(apperr.KindNotFound, "firmware.TransitionTask", "task not found", err)
	}
	if task.Status == domain.StatusCompleted || task.Status == domain.StatusCancelled {
		return task, nil
	}
	now := s.clock.Now()
	task.Status = target
	task.UpdatedAt = now
	if target == domain.StatusRunning && task.StartedAt == nil {
		task.StartedAt = &now
	}
	if target == domain.StatusCompleted {
		task.CompletedAt = &now
	}
	if err := s.repo.SaveTask(ctx, task); err != nil {
		return domain.UpgradeTask{}, apperr.E(apperr.KindInternal, "firmware.TransitionTask", "save task", err)
	}
	_ = s.publish(ctx, "firmware.task_"+string(target), task.ID, task)
	return task, nil
}

func (s *Service) Receive(ctx context.Context, input domain.ReceiptInput) (domain.DeviceReceipt, error) {
	if input.TaskID == "" || input.DeviceID == "" {
		return domain.DeviceReceipt{}, apperr.E(apperr.KindInvalid, "firmware.Receive", "task and device are required", nil)
	}
	task, err := s.repo.GetTask(ctx, input.TaskID)
	if err != nil {
		return domain.DeviceReceipt{}, apperr.E(apperr.KindNotFound, "firmware.Receive", "task not found", err)
	}
	if task.Status != domain.StatusRunning {
		return domain.DeviceReceipt{}, apperr.E(apperr.KindConflict, "firmware.Receive", "task is not running", nil)
	}
	active, err := s.repo.ActiveReceiptForDevice(ctx, input.DeviceID)
	if err == nil && active.ID != "" {
		return active, nil
	}
	if err != nil && !apperr.IsKind(err, apperr.KindNotFound) {
		return domain.DeviceReceipt{}, apperr.E(apperr.KindInternal, "firmware.Receive", "check active receipt", err)
	}
	now := s.clock.Now()
	receipt := domain.DeviceReceipt{
		ID:        id.New("rcpt"),
		TaskID:    input.TaskID,
		DeviceID:  input.DeviceID,
		Status:    domain.StatusRunning,
		Attempt:   1,
		Message:   input.Message,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.SaveReceipt(ctx, receipt); err != nil {
		return domain.DeviceReceipt{}, apperr.E(apperr.KindInternal, "firmware.Receive", "save receipt", err)
	}
	_ = s.publish(ctx, "firmware.device_started", receipt.ID, receipt)
	return receipt, nil
}

func (s *Service) CompleteReceipt(ctx context.Context, receiptID string, success bool, message string) (domain.DeviceReceipt, error) {
	receipt, err := s.repo.GetReceipt(ctx, receiptID)
	if err != nil {
		return domain.DeviceReceipt{}, apperr.E(apperr.KindNotFound, "firmware.CompleteReceipt", "receipt not found", err)
	}
	if receipt.Status == domain.StatusCompleted || receipt.Status == domain.StatusCancelled {
		return receipt, nil
	}
	now := s.clock.Now()
	if success {
		receipt.Status = domain.StatusCompleted
	} else {
		receipt.Status = domain.StatusFailed
		if task, taskErr := s.repo.GetTask(ctx, receipt.TaskID); taskErr == nil && task.Rollout.AutoRollback {
			_ = s.publish(ctx, "firmware.rollback_required", receipt.ID, map[string]any{"receipt_id": receipt.ID, "device_id": receipt.DeviceID})
		}
	}
	receipt.Message = message
	receipt.UpdatedAt = now
	if err := s.repo.SaveReceipt(ctx, receipt); err != nil {
		return domain.DeviceReceipt{}, apperr.E(apperr.KindInternal, "firmware.CompleteReceipt", "save receipt", err)
	}
	_ = s.publish(ctx, "firmware.device_"+string(receipt.Status), receipt.ID, receipt)
	return receipt, nil
}

func (s *Service) ListTasks(ctx context.Context, filter domain.Filter, offset, limit int) ([]domain.UpgradeTask, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items, total, err := s.repo.ListTasks(ctx, filter, offset, limit)
	if err != nil {
		return nil, 0, apperr.E(apperr.KindInternal, "firmware.ListTasks", "list tasks", err)
	}
	return items, total, nil
}

func (s *Service) ReceiptsForDevice(ctx context.Context, deviceID string, limit int) ([]domain.DeviceReceipt, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.ReceiptsForDevice(ctx, deviceID, limit)
}

func (s *Service) publish(ctx context.Context, eventType, subject string, payload any) error {
	if s.bus == nil {
		return nil
	}
	return s.bus.Publish(ctx, eventbus.Event{
		ID:         id.New("evt"),
		Type:       eventType,
		Subject:    subject,
		Payload:    payload,
		OccurredAt: s.clock.Now(),
	})
}

func (s *Service) ValidateRollout(policy domain.RolloutPolicy) error {
	policy = policy.Normalized()
	if policy.Strategy != "percentage" && policy.Strategy != "device_list" && policy.Strategy != "group" {
		return apperr.E(apperr.KindInvalid, "firmware.ValidateRollout", "unsupported rollout strategy", nil)
	}
	if policy.Strategy == "percentage" && (policy.Percentage <= 0 || policy.Percentage > 100) {
		return apperr.E(apperr.KindInvalid, "firmware.ValidateRollout", "percentage must be between 0 and 100", nil)
	}
	if policy.MaxConcurrent < 1 {
		return apperr.E(apperr.KindInvalid, "firmware.ValidateRollout", "max_concurrent must be positive", nil)
	}
	return nil
}

func SortReceipts(items []domain.DeviceReceipt) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return strings.Compare(items[i].ID, items[j].ID) < 0
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}
