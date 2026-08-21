package application

import (
	"context"
	"errors"
	"testing"

	"github.com/example/iot-device-management/internal/twin/domain"
)

type failingSaveRepo struct{}

func (failingSaveRepo) Get(context.Context, string) (domain.TwinDocument, error) {
	return domain.TwinDocument{DeviceID: "dev-1", DesiredState: map[string]any{}, ReportedState: map[string]any{}}, nil
}

func (failingSaveRepo) Save(context.Context, domain.TwinDocument) error {
	return errors.New("save failed")
}

func (failingSaveRepo) Delete(context.Context, string) error {
	return nil
}

func TestUpdateDesiredReturnsSaveError(t *testing.T) {
	svc := NewService(failingSaveRepo{}, nil)
	_, err := svc.UpdateDesired(context.Background(), "dev-1", map[string]any{"mode": "eco"})
	if err == nil {
		t.Fatal("expected save error to be returned")
	}
}

func TestUpdateReportedReturnsSaveError(t *testing.T) {
	svc := NewService(failingSaveRepo{}, nil)
	_, err := svc.UpdateReported(context.Background(), "dev-1", map[string]any{"mode": "eco"})
	if err == nil {
		t.Fatal("expected save error to be returned")
	}
}

func TestComputeDiffFindsNestedProperty(t *testing.T) {
	svc := NewService(failingSaveRepo{}, nil)
	diffs := svc.ComputeDiff(
		map[string]any{"settings": map[string]any{"mode": "eco"}},
		map[string]any{"settings": map[string]any{"mode": "turbo"}},
	)
	found := false
	for _, diff := range diffs {
		if diff.Path == "settings.mode" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected nested settings.mode diff, got %#v", diffs)
	}
}
