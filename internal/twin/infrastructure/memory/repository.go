package memory

import (
	"context"
	"sync"

	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/twin/domain"
)

type Repository struct {
	mu        sync.RWMutex
	documents map[string]domain.TwinDocument
}

func NewRepository() *Repository {
	return &Repository{documents: make(map[string]domain.TwinDocument)}
}

func (r *Repository) Get(_ context.Context, deviceID string) (domain.TwinDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	document, exists := r.documents[deviceID]
	if !exists {
		return domain.TwinDocument{}, apperr.E(apperr.KindNotFound, "twin.memory.Get", "twin document not found", nil)
	}
	return document.Clone(), nil
}

func (r *Repository) Save(_ context.Context, document domain.TwinDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.documents[document.DeviceID] = document.Clone()
	return nil
}

func (r *Repository) Delete(_ context.Context, deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.documents, deviceID)
	return nil
}
