package domain

import (
	"context"
	"sort"
	"time"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, product Product) error
	UpdateProduct(ctx context.Context, product Product) error
	GetProduct(ctx context.Context, id string) (Product, error)
	ListProducts(ctx context.Context, offset, limit int) ([]Product, int, error)
}

type DeviceRepository interface {
	CreateDevice(ctx context.Context, device Device) error
	UpdateDevice(ctx context.Context, device Device) error
	GetDevice(ctx context.Context, id string) (Device, error)
	DeleteDevice(ctx context.Context, id string) error
	ListDevices(ctx context.Context, filter DeviceFilter, offset, limit int) ([]Device, int, error)
}

type GroupRepository interface {
	CreateGroup(ctx context.Context, group Group) error
	UpdateGroup(ctx context.Context, group Group) error
	GetGroup(ctx context.Context, id string) (Group, error)
	ListGroups(ctx context.Context, offset, limit int) ([]Group, int, error)
}

type DeviceFilter struct {
	ProductID string
	Status    DeviceStatus
	GroupID   string
	Query     string
	Tags      map[string]string
}

func SortDevices(items []Device) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func SortProducts(items []Product) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func ExpiredAt(t time.Time) *time.Time {
	value := t.UTC()
	return &value
}
