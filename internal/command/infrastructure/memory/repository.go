package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/example/iot-device-management/internal/command/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
)

type Repository struct {
	mu            sync.RWMutex
	commands      map[string]domain.Command
	byIdempotency map[string]string
}

func NewRepository() *Repository {
	return &Repository{
		commands:      make(map[string]domain.Command),
		byIdempotency: make(map[string]string),
	}
}

func (r *Repository) Save(_ context.Context, command domain.Command) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Clone on write so the stored entry does not alias the caller's payload
	// map; later mutations to the passed command must not leak into storage.
	r.commands[command.ID] = cloneCommand(command)
	r.byIdempotency[command.DeviceID+"\x00"+command.IdempotencyKey] = command.ID
	return nil
}

func (r *Repository) Get(_ context.Context, id string) (domain.Command, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	command, exists := r.commands[id]
	if !exists {
		return domain.Command{}, apperr.E(apperr.KindNotFound, "command.memory.Get", "command not found", nil)
	}
	return cloneCommand(command), nil
}

func (r *Repository) GetByIdempotencyKey(_ context.Context, deviceID, key string) (domain.Command, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	commandID := r.byIdempotency[deviceID+"\x00"+key]
	command, exists := r.commands[commandID]
	if !exists {
		return domain.Command{}, apperr.E(apperr.KindNotFound, "command.memory.GetByIdempotencyKey", "command not found", nil)
	}
	return cloneCommand(command), nil
}

func (r *Repository) List(_ context.Context, filter domain.CommandFilter, offset, limit int) ([]domain.Command, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Command, 0)
	for _, command := range r.commands {
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
		items = append(items, cloneCommand(command))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
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

func (r *Repository) PendingForDevice(_ context.Context, deviceID string, limit int) ([]domain.Command, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Command, 0)
	for _, command := range r.commands {
		if command.DeviceID != deviceID || command.Status != domain.StatusPending {
			continue
		}
		items = append(items, cloneCommand(command))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *Repository) Expired(_ context.Context, before time.Time, limit int) ([]domain.Command, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Command, 0)
	for _, command := range r.commands {
		if command.Status == domain.StatusPending && command.ExpiresAt.Before(before) {
			items = append(items, cloneCommand(command))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ExpiresAt.Before(items[j].ExpiresAt) })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func cloneCommand(command domain.Command) domain.Command {
	out := command
	out.Payload = cloneMap(command.Payload)
	out.Audit = append([]domain.AuditEntry(nil), command.Audit...)
	return out
}

// cloneMap deep-copies a command payload so the returned command cannot alias
// the in-memory stored entry's nested maps or slices. Mutating a value handed
// back to a caller must not leak into the persisted record.
func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneValue(value)
	}
	return out
}

func cloneValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneMap(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneValue(item)
		}
		return out
	default:
		return value
	}
}
