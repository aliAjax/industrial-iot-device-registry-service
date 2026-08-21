package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/example/iot-device-management/internal/command/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
	clockpkg "github.com/example/iot-device-management/internal/platform/clock"
	"github.com/example/iot-device-management/internal/platform/eventbus"
	"github.com/example/iot-device-management/internal/platform/id"
	"github.com/example/iot-device-management/internal/platform/trace"
)

type Service struct {
	repo         domain.Repository
	bus          eventbus.Bus
	clock        clockpkg.Clock
	reapInterval time.Duration
}

func NewService(repo domain.Repository, bus eventbus.Bus, clock clockpkg.Clock, reapInterval time.Duration) *Service {
	if clock == nil {
		clock = clockpkg.SystemClock{}
	}
	if reapInterval <= 0 {
		reapInterval = 5 * time.Second
	}
	return &Service{repo: repo, bus: bus, clock: clock, reapInterval: reapInterval}
}

func (s *Service) Enqueue(ctx context.Context, input domain.EnqueueInput) (domain.Command, error) {
	if input.DeviceID == "" || input.Name == "" {
		return domain.Command{}, apperr.E(apperr.KindInvalid, "command.Enqueue", "device id and command name are required", nil)
	}
	if input.IdempotencyKey == "" {
		input.IdempotencyKey = id.New("idem")
	}
	if input.MaxRetries < 0 || input.MaxRetries > 20 {
		input.MaxRetries = 3
	}
	if input.Timeout < 0 {
		input.Timeout = 30 * time.Second
	}
	existing, err := s.repo.GetByIdempotencyKey(ctx, input.DeviceID, input.IdempotencyKey)
	if err == nil {
		if !existing.IsTerminal() {
			return existing, nil
		}
		return existing, apperr.E(apperr.KindConflict, "command.Enqueue", "idempotency key already used", nil)
	}
	if !apperr.IsKind(err, apperr.KindNotFound) {
		return domain.Command{}, apperr.E(apperr.KindInternal, "command.Enqueue", "check idempotency", err)
	}
	now := s.clock.Now()
	timeout := input.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	command := domain.Command{
		ID:             id.New("cmd"),
		DeviceID:       input.DeviceID,
		Name:           strings.TrimSpace(input.Name),
		Payload:        cloneMap(input.Payload),
		IdempotencyKey: input.IdempotencyKey,
		Status:         domain.StatusPending,
		MaxRetries:     input.MaxRetries,
		Timeout:        timeout,
		CreatedAt:      now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(timeout),
		Audit: []domain.AuditEntry{{
			At:      now,
			Status:  domain.StatusPending,
			Message: "command enqueued",
			TraceID: trace.TraceID(ctx),
		}},
	}
	if err := s.repo.Save(ctx, command); err != nil {
		return domain.Command{}, apperr.E(apperr.KindInternal, "command.Enqueue", "save command", err)
	}
	_ = s.publish(ctx, "command.enqueued", command.ID, domain.Delivery{Command: command, Attempt: 1})
	return command, nil
}

func (s *Service) Ack(ctx context.Context, input domain.AckInput) (domain.Command, error) {
	if input.DeviceID == "" || input.CommandID == "" {
		return domain.Command{}, apperr.E(apperr.KindInvalid, "command.Ack", "device id and command id are required", nil)
	}
	command, err := s.repo.Get(ctx, input.CommandID)
	if err != nil {
		return domain.Command{}, apperr.E(apperr.KindNotFound, "command.Ack", "command not found", err)
	}
	if command.DeviceID != input.DeviceID {
		return domain.Command{}, apperr.E(apperr.KindForbidden, "command.Ack", "command does not belong to device", nil)
	}
	if command.IsTerminal() {
		return command, nil
	}
	now := s.clock.Now()
	status := domain.StatusAcked
	message := input.Message
	if message == "" {
		message = "acknowledged"
	}
	if !input.Success {
		status = domain.StatusFailed
		command.FailureReason = message
		if message == "" {
			message = "failed"
		}
	}
	command.Status = status
	command.UpdatedAt = now
	command.AckedAt = &now
	command.Audit = append(command.Audit, domain.AuditEntry{At: now, Status: status, Message: message, TraceID: trace.TraceID(ctx)})
	if err := s.repo.Save(ctx, command); err != nil {
		return domain.Command{}, apperr.E(apperr.KindInternal, "command.Ack", "save command", err)
	}
	_ = s.publish(ctx, "command."+string(status), command.ID, command)
	return command, nil
}

func (s *Service) Get(ctx context.Context, commandID string) (domain.Command, error) {
	command, err := s.repo.Get(ctx, commandID)
	if err != nil {
		return domain.Command{}, apperr.E(apperr.KindNotFound, "command.Get", "command not found", err)
	}
	return command, nil
}

func (s *Service) List(ctx context.Context, filter domain.CommandFilter, offset, limit int) ([]domain.Command, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items, total, err := s.repo.List(ctx, filter, offset, limit)
	if err != nil {
		return nil, 0, apperr.E(apperr.KindInternal, "command.List", "list commands", err)
	}
	return items, total, nil
}

func (s *Service) PendingForDevice(ctx context.Context, deviceID string, limit int) ([]domain.Command, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	return s.repo.PendingForDevice(ctx, deviceID, limit)
}

func (s *Service) StartReaper(ctx context.Context) {
	ticker := time.NewTicker(s.reapInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reapExpired(ctx)
		}
	}
}

func (s *Service) reapExpired(ctx context.Context) {
	commands, err := s.repo.Expired(ctx, s.clock.Now(), 100)
	if err != nil {
		return
	}
	for _, command := range commands {
		if command.IsTerminal() {
			continue
		}
		if command.RetryCount < command.MaxRetries {
			command.RetryCount++
			command.Status = domain.StatusRetrying
			command.ExpiresAt = s.clock.Now().Add(command.Timeout)
			command.UpdatedAt = s.clock.Now()
			command.Audit = append(command.Audit, domain.AuditEntry{At: s.clock.Now(), Status: domain.StatusRetrying, Message: fmt.Sprintf("retry %d", command.RetryCount), TraceID: trace.TraceID(ctx)})
			if err := s.repo.Save(ctx, command); err != nil {
				continue
			}
			_ = s.publish(ctx, "command.retrying", command.ID, domain.Delivery{Command: command, Attempt: command.RetryCount + 1})
			continue
		}
		command.Status = domain.StatusFailed
		command.FailureReason = "command timed out after retries"
		command.UpdatedAt = s.clock.Now()
		command.Audit = append(command.Audit, domain.AuditEntry{At: s.clock.Now(), Status: domain.StatusFailed, Message: command.FailureReason, TraceID: trace.TraceID(ctx)})
		if err := s.repo.Save(ctx, command); err != nil {
			continue
		}
		_ = s.publish(ctx, "command.failed", command.ID, command)
	}
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
		TraceID:    trace.TraceID(ctx),
	})
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
