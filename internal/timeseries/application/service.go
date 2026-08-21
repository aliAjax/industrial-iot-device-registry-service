package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/example/iot-device-management/internal/platform/apperr"
	clockpkg "github.com/example/iot-device-management/internal/platform/clock"
	"github.com/example/iot-device-management/internal/timeseries/domain"
)

type Service struct {
	repo  domain.Repository
	clock clockpkg.Clock
}

func NewService(repo domain.Repository, clock clockpkg.Clock) *Service {
	if clock == nil {
		clock = clockpkg.SystemClock{}
	}
	return &Service{repo: repo, clock: clock}
}

func (s *Service) WriteBatch(ctx context.Context, points []domain.Point) error {
	if len(points) == 0 {
		return nil
	}
	normalized := make([]domain.Point, 0, len(points))
	now := s.clock.Now()
	for _, point := range points {
		if point.DeviceID == "" || point.Property == "" {
			return apperr.E(apperr.KindInvalid, "timeseries.WriteBatch", "device and property are required", nil)
		}
		if point.Timestamp.IsZero() {
			point.Timestamp = now
		}
		if point.Timestamp.After(now.Add(time.Hour)) {
			return apperr.E(apperr.KindInvalid, "timeseries.WriteBatch", "timestamp is too far in the future", nil)
		}
		point.Timestamp = point.Timestamp.UTC()
		if point.Metadata != nil {
			point.Metadata = cloneMetadata(point.Metadata)
		}
		normalized = append(normalized, point)
	}
	if err := s.repo.Write(ctx, normalized); err != nil {
		return apperr.E(apperr.KindInternal, "timeseries.WriteBatch", "write points", err)
	}
	return nil
}

func cloneMetadata(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneAny(value)
	}
	return out
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = cloneAny(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			out[index] = cloneAny(item)
		}
		return out
	default:
		return value
	}
}

func (s *Service) Query(ctx context.Context, query domain.Query) ([]domain.Point, error) {
	if query.DeviceID == "" {
		return nil, apperr.E(apperr.KindInvalid, "timeseries.Query", "device id is required", nil)
	}
	if query.Start.IsZero() {
		query.Start = s.clock.Now().Add(-24 * time.Hour)
	}
	if query.End.IsZero() {
		query.End = s.clock.Now()
	}
	if query.End.Before(query.Start) {
		return nil, apperr.E(apperr.KindInvalid, "timeseries.Query", "end must be after start", nil)
	}
	if query.Limit <= 0 || query.Limit > 10000 {
		query.Limit = 1000
	}
	points, err := s.repo.Query(ctx, query)
	if err != nil {
		return nil, apperr.E(apperr.KindInternal, "timeseries.Query", "query points", err)
	}
	return points, nil
}

func (s *Service) Aggregate(ctx context.Context, request domain.AggregateRequest) ([]domain.AggregateResult, error) {
	if request.DeviceID == "" || request.Property == "" {
		return nil, apperr.E(apperr.KindInvalid, "timeseries.Aggregate", "device and property are required", nil)
	}
	if request.Kind == "" {
		request.Kind = domain.AggregateAvg
	}
	switch request.Kind {
	case domain.AggregateAvg, domain.AggregateMin, domain.AggregateMax, domain.AggregateSum, domain.AggregateCount:
	default:
		return nil, apperr.E(apperr.KindInvalid, "timeseries.Aggregate", "unsupported aggregation kind", nil)
	}
	if request.Start.IsZero() {
		request.Start = s.clock.Now().Add(-24 * time.Hour)
	}
	if request.End.IsZero() {
		request.End = s.clock.Now()
	}
	if request.End.Before(request.Start) {
		return nil, apperr.E(apperr.KindInvalid, "timeseries.Aggregate", "end must be after start", nil)
	}
	if request.Bucket < time.Second {
		request.Bucket = time.Minute
	}
	results, err := s.repo.Aggregate(ctx, request)
	if err != nil {
		return nil, apperr.E(apperr.KindInternal, "timeseries.Aggregate", "aggregate points", err)
	}
	return results, nil
}

func (s *Service) DetectGaps(ctx context.Context, request domain.GapRequest) ([]domain.Gap, error) {
	if request.DeviceID == "" || request.Property == "" {
		return nil, apperr.E(apperr.KindInvalid, "timeseries.DetectGaps", "device and property are required", nil)
	}
	if request.ExpectedInterval <= 0 {
		request.ExpectedInterval = time.Minute
	}
	if request.Start.IsZero() {
		request.Start = s.clock.Now().Add(-24 * time.Hour)
	}
	if request.End.IsZero() {
		request.End = s.clock.Now()
	}
	if request.End.Before(request.Start) {
		return nil, apperr.E(apperr.KindInvalid, "timeseries.DetectGaps", "end must be after start", nil)
	}
	gaps, err := s.repo.DetectGaps(ctx, request)
	if err != nil {
		return nil, apperr.E(apperr.KindInternal, "timeseries.DetectGaps", "detect gaps", err)
	}
	return gaps, nil
}

func (s *Service) Downsample(ctx context.Context, deviceID, property string, start, end time.Time, bucket time.Duration) ([]domain.AggregateResult, error) {
	return s.Aggregate(ctx, domain.AggregateRequest{
		DeviceID: deviceID,
		Property: property,
		Start:    start,
		End:      end,
		Kind:     domain.AggregateAvg,
		Bucket:   bucket,
	})
}

func ValidateProperty(value float64, dataType domain.DataType) error {
	switch dataType {
	case domain.DataTypeNumber, domain.DataTypeString, domain.DataTypeBool:
		return nil
	default:
		return fmt.Errorf("unsupported data type %q", dataType)
	}
}

func SortPoints(points []domain.Point) {
	sort.Slice(points, func(i, j int) bool {
		if points[i].Timestamp.Equal(points[j].Timestamp) {
			return points[i].Property < points[j].Property
		}
		return points[i].Timestamp.Before(points[j].Timestamp)
	})
}
