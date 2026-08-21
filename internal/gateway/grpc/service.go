package grpc

import (
	"context"
	"time"

	v1 "github.com/example/iot-device-management/api/iot/v1"
	commandapp "github.com/example/iot-device-management/internal/command/application"
	commanddomain "github.com/example/iot-device-management/internal/command/domain"
	deviceapp "github.com/example/iot-device-management/internal/device/application"
	telemetryapp "github.com/example/iot-device-management/internal/telemetry/application"
	telemetrydomain "github.com/example/iot-device-management/internal/telemetry/domain"
	tsapp "github.com/example/iot-device-management/internal/timeseries/application"
	tsdomain "github.com/example/iot-device-management/internal/timeseries/domain"
	twinapp "github.com/example/iot-device-management/internal/twin/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	devices    *deviceapp.Service
	telemetry  *telemetryapp.Service
	commands   *commandapp.Service
	twins      *twinapp.Service
	timeSeries *tsapp.Service
}

func NewService(devices *deviceapp.Service, telemetry *telemetryapp.Service, commands *commandapp.Service, twins *twinapp.Service, timeSeries *tsapp.Service) *Service {
	return &Service{devices: devices, telemetry: telemetry, commands: commands, twins: twins, timeSeries: timeSeries}
}

func (s *Service) Health(ctx context.Context, req *v1.HealthRequest) (*v1.HealthResponse, error) {
	return &v1.HealthResponse{Status: "SERVING", Service: "iot-device-management", Time: time.Now().UTC().Format(time.RFC3339)}, nil
}

func (s *Service) GetDevice(ctx context.Context, req *v1.DeviceRequest) (*v1.DeviceResponse, error) {
	device, err := s.devices.GetDevice(ctx, req.DeviceID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &v1.DeviceResponse{Device: device}, nil
}

func (s *Service) IngestTelemetry(ctx context.Context, req *v1.IngestRequest) (*v1.IngestResponse, error) {
	message := telemetrydomain.Message{
		DeviceID:  req.DeviceID,
		Kind:      telemetrydomain.Kind(req.Kind),
		Timestamp: req.Timestamp,
		Data:      req.Data,
	}
	result, err := s.telemetry.Ingest(ctx, message)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &v1.IngestResponse{MessageID: result.MessageID, Status: result.Status}, nil
}

func (s *Service) EnqueueCommand(ctx context.Context, req *v1.CommandRequest) (*v1.CommandResponse, error) {
	command, err := s.commands.Enqueue(ctx, commanddomain.EnqueueInput{
		DeviceID:       req.DeviceID,
		Name:           req.Name,
		Payload:        req.Payload,
		IdempotencyKey: req.IdempotencyKey,
		MaxRetries:     req.MaxRetries,
		Timeout:        req.Timeout,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &v1.CommandResponse{CommandID: command.ID, Status: string(command.Status)}, nil
}

func (s *Service) GetTwin(ctx context.Context, req *v1.TwinRequest) (*v1.TwinResponse, error) {
	document, err := s.twins.Get(ctx, req.DeviceID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.TwinResponse{
		DeviceID:      document.DeviceID,
		DesiredState:  document.DesiredState,
		ReportedState: document.ReportedState,
		Version:       document.Version,
	}, nil
}

func (s *Service) QueryTimeSeries(ctx context.Context, req *v1.TimeSeriesRequest) (*v1.TimeSeriesResponse, error) {
	points, err := s.timeSeries.Query(ctx, tsdomain.Query{
		DeviceID:   req.DeviceID,
		Property:   req.Property,
		Start:      req.Start,
		End:        req.End,
		Limit:      req.Limit,
		Descending: false,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &v1.TimeSeriesResponse{Points: points, Count: len(points)}, nil
}

func unary[TReq, TResp any](handler func(context.Context, *TReq) (*TResp, error)) func(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error) {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		req := new(TReq)
		if err := dec(req); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return handler(ctx, req)
		}
		info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "iot.v1.IoTService/unknown"}
		call := func(ctx context.Context, req any) (any, error) {
			return handler(ctx, req.(*TReq))
		}
		return interceptor(ctx, req, info, call)
	}
}
