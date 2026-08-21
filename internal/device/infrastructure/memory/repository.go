package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/example/iot-device-management/internal/device/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
)

type Repository struct {
	mu       sync.RWMutex
	products map[string]domain.Product
	devices  map[string]domain.Device
	groups   map[string]domain.Group
}

func NewRepository() *Repository {
	return &Repository{
		products: make(map[string]domain.Product),
		devices:  make(map[string]domain.Device),
		groups:   make(map[string]domain.Group),
	}
}

func (r *Repository) CreateProduct(_ context.Context, product domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.products[product.ID]; exists {
		return apperr.E(apperr.KindConflict, "memory.CreateProduct", "product already exists", nil)
	}
	r.products[product.ID] = product
	return nil
}

func (r *Repository) UpdateProduct(_ context.Context, product domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.products[product.ID]; !exists {
		return apperr.E(apperr.KindNotFound, "memory.UpdateProduct", "product not found", nil)
	}
	r.products[product.ID] = product
	return nil
}

func (r *Repository) GetProduct(_ context.Context, id string) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	product, exists := r.products[id]
	if !exists {
		return domain.Product{}, apperr.E(apperr.KindNotFound, "memory.GetProduct", "product not found", nil)
	}
	return cloneProduct(product), nil
}

func (r *Repository) ListProducts(_ context.Context, offset, limit int) ([]domain.Product, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Product, 0, len(r.products))
	for _, product := range r.products {
		items = append(items, cloneProduct(product))
	}
	domain.SortProducts(items)
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

func (r *Repository) CreateDevice(_ context.Context, device domain.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.devices[device.ID]; exists {
		return apperr.E(apperr.KindConflict, "memory.CreateDevice", "device already exists", nil)
	}
	r.devices[device.ID] = cloneDevice(device)
	return nil
}

func (r *Repository) UpdateDevice(_ context.Context, device domain.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.devices[device.ID]; !exists {
		return apperr.E(apperr.KindNotFound, "memory.UpdateDevice", "device not found", nil)
	}
	r.devices[device.ID] = cloneDevice(device)
	return nil
}

func (r *Repository) GetDevice(_ context.Context, id string) (domain.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	device, exists := r.devices[id]
	if !exists {
		return domain.Device{}, apperr.E(apperr.KindNotFound, "memory.GetDevice", "device not found", nil)
	}
	return cloneDevice(device), nil
}

func (r *Repository) DeleteDevice(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.devices[id]; !exists {
		return apperr.E(apperr.KindNotFound, "memory.DeleteDevice", "device not found", nil)
	}
	delete(r.devices, id)
	return nil
}

func (r *Repository) ListDevices(_ context.Context, filter domain.DeviceFilter, offset, limit int) ([]domain.Device, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Device, 0)
	for _, device := range r.devices {
		if !matchesFilter(device, filter) {
			continue
		}
		items = append(items, cloneDevice(device))
	}
	domain.SortDevices(items)
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

func (r *Repository) CreateGroup(_ context.Context, group domain.Group) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.groups[group.ID]; exists {
		return apperr.E(apperr.KindConflict, "memory.CreateGroup", "group already exists", nil)
	}
	r.groups[group.ID] = group
	return nil
}

func (r *Repository) UpdateGroup(_ context.Context, group domain.Group) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.groups[group.ID]; !exists {
		return apperr.E(apperr.KindNotFound, "memory.UpdateGroup", "group not found", nil)
	}
	r.groups[group.ID] = group
	return nil
}

func (r *Repository) GetGroup(_ context.Context, id string) (domain.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	group, exists := r.groups[id]
	if !exists {
		return domain.Group{}, apperr.E(apperr.KindNotFound, "memory.GetGroup", "group not found", nil)
	}
	return group, nil
}

func (r *Repository) ListGroups(_ context.Context, offset, limit int) ([]domain.Group, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Group, 0, len(r.groups))
	for _, group := range r.groups {
		items = append(items, group)
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

func matchesFilter(device domain.Device, filter domain.DeviceFilter) bool {
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
		if !strings.Contains(strings.ToLower(device.Name), query) &&
			!strings.Contains(strings.ToLower(device.SerialNumber), query) &&
			!strings.Contains(strings.ToLower(device.ID), query) {
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

func cloneDevice(device domain.Device) domain.Device {
	out := device
	out.GroupIDs = append([]string(nil), device.GroupIDs...)
	out.Tags = make(map[string]string, len(device.Tags))
	for key, value := range device.Tags {
		out.Tags[key] = value
	}
	out.Attributes = make(map[string]any, len(device.Attributes))
	for key, value := range device.Attributes {
		out.Attributes[key] = value
	}
	out.Credentials = append([]domain.Credential(nil), device.Credentials...)
	if device.Location != nil {
		location := *device.Location
		out.Location = &location
	}
	return out
}

func cloneProduct(product domain.Product) domain.Product {
	out := product
	out.Attributes = append([]domain.AttributeSchema(nil), product.Attributes...)
	return out
}
