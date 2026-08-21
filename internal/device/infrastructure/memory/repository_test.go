package memory

import (
	"context"
	"testing"

	"github.com/example/iot-device-management/internal/device/domain"
)

func TestRepositoryGetDeviceCopiesTags(t *testing.T) {
	repo := NewRepository()
	_ = repo.CreateDevice(context.Background(), domain.Device{
		ID:     "dev-1",
		Status: domain.DevicePending,
		Tags:   map[string]string{"site": "old"},
	})
	got, err := repo.GetDevice(context.Background(), "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	got.Tags["site"] = "mutated"
	again, err := repo.GetDevice(context.Background(), "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if again.Tags["site"] != "old" {
		t.Fatal("GetDevice returned an internal mutable reference")
	}
}

func TestRepositoryListDevicesCopiesItems(t *testing.T) {
	repo := NewRepository()
	_ = repo.CreateDevice(context.Background(), domain.Device{ID: "dev-1", Status: domain.DevicePending, Tags: map[string]string{"site": "old"}})
	items, _, err := repo.ListDevices(context.Background(), domain.DeviceFilter{}, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	items[0].Tags["site"] = "mutated"
	again, _, err := repo.ListDevices(context.Background(), domain.DeviceFilter{}, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if again[0].Tags["site"] != "old" {
		t.Fatal("ListDevices returned internal mutable references")
	}
}
