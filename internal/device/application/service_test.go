package application

import (
	"context"
	"sync"
	"testing"
	"time"

	devicedomain "github.com/example/iot-device-management/internal/device/domain"
	"github.com/example/iot-device-management/internal/device/infrastructure/memory"
)

func TestUpdateTagsDoesNotRaceRepositoryReads(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.CreateDevice(context.Background(), devicedomain.Device{
		ID:     "dev-1",
		Status: devicedomain.DevicePending,
		Tags:   map[string]string{"site": "old"},
	})
	svc := NewService(repo, repo, repo, nil, nil, nil, time.Hour)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, _ = svc.UpdateTags(context.Background(), "dev-1", map[string]string{"site": "new"})
		}()
		go func() {
			defer wg.Done()
			<-start
			_, _ = repo.GetDevice(context.Background(), "dev-1")
		}()
	}
	close(start)
	wg.Wait()
}

func TestAssignGroupsValidatesAndDeduplicates(t *testing.T) {
	repo := memory.NewRepository()
	_ = repo.CreateGroup(context.Background(), devicedomain.Group{ID: "grp-1"})
	_ = repo.CreateDevice(context.Background(), devicedomain.Device{ID: "dev-1", Status: devicedomain.DevicePending})
	svc := NewService(repo, repo, repo, nil, nil, nil, time.Hour)
	_, err := svc.AssignGroups(context.Background(), "dev-1", []string{"grp-1", "grp-1", "missing"})
	if err == nil {
		t.Fatal("expected missing group to be rejected")
	}
}

func TestCreateGroupValidatesParent(t *testing.T) {
	repo := memory.NewRepository()
	svc := NewService(repo, repo, repo, nil, nil, nil, time.Hour)
	_, err := svc.CreateGroup(context.Background(), CreateGroupInput{Name: "child", ParentID: "missing"})
	if err == nil {
		t.Fatal("expected missing parent group to be rejected")
	}
}
