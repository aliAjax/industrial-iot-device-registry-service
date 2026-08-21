package memory

import (
	"context"
	"sync"
	"time"

	"github.com/example/iot-device-management/internal/auth/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
)

type rateBucket struct {
	window time.Time
	count  int
}

type Repository struct {
	mu         sync.RWMutex
	principals map[string]domain.Principal
	rates      map[string]rateBucket
}

func NewRepository() *Repository {
	return &Repository{
		principals: make(map[string]domain.Principal),
		rates:      make(map[string]rateBucket),
	}
}

func (r *Repository) UpsertPrincipal(_ context.Context, principal domain.Principal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	principal.Scopes = append([]string(nil), principal.Scopes...)
	r.principals[principal.DeviceID] = principal
	return nil
}

func (r *Repository) FindPrincipal(_ context.Context, deviceID string) (domain.Principal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	principal, exists := r.principals[deviceID]
	if !exists {
		return domain.Principal{}, apperr.E(apperr.KindNotFound, "auth.memory.FindPrincipal", "principal not found", nil)
	}
	return principal, nil
}

func (r *Repository) DeletePrincipal(_ context.Context, deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.principals, deviceID)
	delete(r.rates, "auth:"+deviceID)
	return nil
}

func (r *Repository) IncrementRate(_ context.Context, key string, window time.Duration, limit int) (domain.RateRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	bucket := r.rates[key]
	if bucket.window.IsZero() || now.Sub(bucket.window) >= window {
		bucket = rateBucket{window: now, count: 0}
	}
	bucket.count++
	r.rates[key] = bucket
	return domain.RateRecord{Key: key, Window: bucket.window, Count: bucket.count, Limit: limit}, nil
}
