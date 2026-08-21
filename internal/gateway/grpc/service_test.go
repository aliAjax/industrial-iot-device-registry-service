package grpc

import (
	"context"
	"testing"

	v1 "github.com/example/iot-device-management/api/iot/v1"
	twinapp "github.com/example/iot-device-management/internal/twin/application"
	twinmem "github.com/example/iot-device-management/internal/twin/infrastructure/memory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetTwinReturnsCanceledForCanceledContext(t *testing.T) {
	repo := twinmem.NewRepository()
	twinService := twinapp.NewService(repo, nil)
	svc := NewService(nil, nil, nil, twinService, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.GetTwin(ctx, &v1.TwinRequest{DeviceID: "dev-1"})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("expected canceled code, got %s (%v)", status.Code(err), err)
	}
}

func TestGetDeviceReturnsCanceledForCanceledContext(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.GetDevice(ctx, &v1.DeviceRequest{DeviceID: "dev-1"})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("expected canceled code, got %s (%v)", status.Code(err), err)
	}
}

func TestQueryTimeSeriesReturnsCanceledForCanceledContext(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.QueryTimeSeries(ctx, &v1.TimeSeriesRequest{DeviceID: "dev-1"})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("expected canceled code, got %s (%v)", status.Code(err), err)
	}
}
