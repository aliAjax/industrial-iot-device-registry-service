package application

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/platform/backpressure"
	clockpkg "github.com/example/iot-device-management/internal/platform/clock"
	"github.com/example/iot-device-management/internal/platform/eventbus"
	"github.com/example/iot-device-management/internal/platform/id"
	"github.com/example/iot-device-management/internal/platform/trace"
	"github.com/example/iot-device-management/internal/telemetry/domain"
	tsapp "github.com/example/iot-device-management/internal/timeseries/application"
	tsdomain "github.com/example/iot-device-management/internal/timeseries/domain"
	twinapp "github.com/example/iot-device-management/internal/twin/application"
)

type Config struct {
	QueueSize        int
	WorkerCount      int
	MaxBatchSize     int
	MaxClockSkew     time.Duration
	DefaultRetention time.Duration
}

type workItem struct {
	message domain.Message
	traceID string
}

type Service struct {
	cfg        Config
	timeseries *tsapp.Service
	twin       *twinapp.Service
	bus        eventbus.Bus
	clock      clockpkg.Clock
	logger     *slog.Logger
	queue      *backpressure.Queue[workItem]
}

func NewService(cfg Config, timeseries *tsapp.Service, twin *twinapp.Service, bus eventbus.Bus, clock clockpkg.Clock, logger *slog.Logger) *Service {
	if clock == nil {
		clock = clockpkg.SystemClock{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 4096
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 4
	}
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = 512
	}
	if cfg.MaxClockSkew <= 0 {
		cfg.MaxClockSkew = 5 * time.Minute
	}
	return &Service{
		cfg:        cfg,
		timeseries: timeseries,
		twin:       twin,
		bus:        bus,
		clock:      clock,
		logger:     logger,
		queue:      backpressure.NewQueue[workItem](cfg.QueueSize),
	}
}

func (s *Service) Ingest(ctx context.Context, message domain.Message) (domain.Result, error) {
	if message.DeviceID == "" {
		return domain.Result{}, apperr.E(apperr.KindInvalid, "telemetry.Ingest", "device id is required", nil)
	}
	if message.Kind == "" {
		message.Kind = domain.KindTelemetry
	}
	if message.Kind != domain.KindTelemetry && message.Kind != domain.KindEvent && message.Kind != domain.KindAttributeChange {
		return domain.Result{}, apperr.E(apperr.KindInvalid, "telemetry.Ingest", "unsupported message kind", nil)
	}
	if message.ID == "" {
		message.ID = id.New("msg")
	}
	if message.Data == nil {
		message.Data = map[string]any{}
	}
	now := s.clock.Now()
	message.Timestamp = s.calibrateTimestamp(message.Timestamp, now)
	result := domain.Result{
		MessageID:  message.ID,
		DeviceID:   message.DeviceID,
		Kind:       message.Kind,
		AcceptedAt: now,
		Status:     "accepted",
		TraceID:    trace.TraceID(ctx),
	}
	if err := s.queue.Enqueue(ctx, workItem{message: message, traceID: trace.TraceID(ctx)}); err != nil {
		return domain.Result{}, apperr.E(apperr.KindBackpressure, "telemetry.Ingest", "ingest queue is full", err)
	}
	return result, nil
}

func (s *Service) IngestBatch(ctx context.Context, messages []domain.Message) ([]domain.Result, error) {
	if len(messages) == 0 {
		return []domain.Result{}, nil
	}
	if len(messages) > s.cfg.MaxBatchSize {
		return nil, apperr.E(apperr.KindInvalid, "telemetry.IngestBatch", fmt.Sprintf("batch exceeds maximum size %d", s.cfg.MaxBatchSize), nil)
	}
	results := make([]domain.Result, 0, len(messages))
	for _, message := range messages {
		result, err := s.Ingest(ctx, message)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) Start(ctx context.Context) {
	s.queue.Close()
	for i := 0; i < s.cfg.WorkerCount; i++ {
		go s.worker(ctx, i)
	}
}

func (s *Service) QueueDepth() int {
	return s.queue.Len()
}

func (s *Service) Dropped() int64 {
	return s.queue.Dropped()
}

func (s *Service) worker(ctx context.Context, index int) {
	for {
		item, err := s.queue.Recv(ctx)
		if err != nil {
			return
		}
		if err := s.process(trace.WithTraceID(ctx, item.traceID), item.message); err != nil {
			s.logger.Error("telemetry processing failed", "worker", index, "message_id", item.message.ID, "device_id", item.message.DeviceID, "error", err)
		}
	}
}

func (s *Service) process(ctx context.Context, message domain.Message) error {
	points := normalize(message)
	if message.Kind == domain.KindAttributeChange {
		if s.twin != nil {
			if _, err := s.twin.UpdateReported(ctx, message.DeviceID, message.Data); err != nil {
				return fmt.Errorf("update twin reported state: %w", err)
			}
		}
	}
	if s.timeseries != nil && len(points) > 0 {
		if err := s.timeseries.WriteBatch(ctx, points); err != nil {
			return fmt.Errorf("write timeseries: %w", err)
		}
	}
	if s.bus != nil {
		event := eventbus.Event{
			ID:         id.New("evt"),
			Type:       "telemetry.message_processed",
			Subject:    message.DeviceID,
			Payload:    message,
			OccurredAt: s.clock.Now(),
			TraceID:    trace.TraceID(ctx),
		}
		if err := s.bus.Publish(ctx, event); err != nil {
			return fmt.Errorf("publish message event: %w", err)
		}
		for _, point := range points {
			if err := s.bus.Publish(ctx, eventbus.Event{
				ID:         id.New("evt"),
				Type:       "telemetry.point_ingested",
				Subject:    message.DeviceID,
				Payload:    point,
				OccurredAt: s.clock.Now(),
				TraceID:    trace.TraceID(ctx),
			}); err != nil {
				return fmt.Errorf("publish point event: %w", err)
			}
		}
	}
	return nil
}

func (s *Service) calibrateTimestamp(value time.Time, now time.Time) time.Time {
	if value.IsZero() {
		return now
	}
	skew := now.Sub(value)
	if skew < 0 {
		skew = -skew
	}
	if skew > s.cfg.MaxClockSkew {
		return now
	}
	return value.UTC()
}

func normalize(message domain.Message) []tsdomain.Point {
	points := make([]tsdomain.Point, 0, len(message.Data))
	for property, raw := range message.Data {
		point := tsdomain.Point{
			DeviceID:  message.DeviceID,
			Property:  property,
			Timestamp: message.Timestamp,
			Metadata:  map[string]any{"kind": string(message.Kind)},
		}
		switch value := raw.(type) {
		case float64:
			point.DataType = tsdomain.DataTypeNumber
			point.Value = value
		case int:
			point.DataType = tsdomain.DataTypeNumber
			point.Value = float64(value)
		case int64:
			point.DataType = tsdomain.DataTypeNumber
			point.Value = float64(value)
		case string:
			point.DataType = tsdomain.DataTypeString
			point.StringValue = value
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				point.Value = parsed
			}
		case bool:
			point.DataType = tsdomain.DataTypeBool
			point.BoolValue = &value
			if value {
				point.Value = 1
			}
		default:
			point.DataType = tsdomain.DataTypeString
			point.StringValue = fmt.Sprintf("%v", value)
		}
		points = append(points, point)
	}
	return points
}
