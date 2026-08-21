package memory

import (
	"context"
	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/timeseries/domain"
	"sort"
	"sync"
)

type Store struct {
	mu     sync.RWMutex
	points map[string][]domain.Point
	limit  int
}

func NewStore(retentionSeriesLimit int) *Store {
	if retentionSeriesLimit <= 0 {
		retentionSeriesLimit = 20000
	}
	return &Store{points: make(map[string][]domain.Point), limit: retentionSeriesLimit}
}

func (s *Store) Write(_ context.Context, points []domain.Point) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, point := range points {
		key := seriesKey(point.DeviceID, point.Property)
		series := s.points[key]
		series = append(series, point)
		if len(series) > s.limit {
			series = series[len(series)-s.limit:]
		}
		sort.Slice(series, func(i, j int) bool {
			if series[i].Timestamp.Equal(series[j].Timestamp) {
				return series[i].StringValue < series[j].StringValue
			}
			return series[i].Timestamp.Before(series[j].Timestamp)
		})
		s.points[key] = series
	}
	return nil
}

func (s *Store) Query(_ context.Context, query domain.Query) ([]domain.Point, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := seriesKey(query.DeviceID, query.Property)
	series := s.points[key]
	out := make([]domain.Point, 0, len(series))
	for _, point := range series {
		if point.Timestamp.Before(query.Start) || point.Timestamp.After(query.End) {
			continue
		}
		out = append(out, clonePoint(point))
	}
	if query.Descending {
		sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	}
	if len(out) > query.Limit {
		if query.Descending {
			out = out[len(out)-query.Limit:]
		} else {
			out = out[:query.Limit]
		}
	}
	return out, nil
}

func (s *Store) Aggregate(_ context.Context, request domain.AggregateRequest) ([]domain.AggregateResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := seriesKey(request.DeviceID, request.Property)
	series := s.points[key]
	buckets := map[int64]*domain.AggregateResult{}
	for _, point := range series {
		if point.Timestamp.Before(request.Start) || point.Timestamp.After(request.End) {
			continue
		}
		bucketStart := point.Timestamp.Truncate(request.Bucket)
		bucketID := bucketStart.UnixNano()
		result := buckets[bucketID]
		if result == nil {
			result = &domain.AggregateResult{Start: bucketStart, End: bucketStart.Add(request.Bucket), Value: point.Value, Count: 0}
			buckets[bucketID] = result
		}
		switch request.Kind {
		case domain.AggregateMin:
			if point.Value < result.Value {
				result.Value = point.Value
			}
		case domain.AggregateMax:
			if point.Value > result.Value {
				result.Value = point.Value
			}
		case domain.AggregateSum, domain.AggregateAvg:
			result.Value += point.Value
		case domain.AggregateCount:
			result.Value = float64(result.Count + 1)
		}
		result.Count++
	}
	if request.Kind == domain.AggregateAvg {
		for _, result := range buckets {
			if result.Count > 0 {
				result.Value = result.Value / float64(result.Count)
			}
		}
	}
	results := make([]domain.AggregateResult, 0, len(buckets))
	for _, result := range buckets {
		results = append(results, *result)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Start.Before(results[j].Start) })
	return results, nil
}

func (s *Store) DetectGaps(_ context.Context, request domain.GapRequest) ([]domain.Gap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := seriesKey(request.DeviceID, request.Property)
	series := s.points[key]
	filtered := make([]domain.Point, 0)
	for _, point := range series {
		if point.Timestamp.Before(request.Start) || point.Timestamp.After(request.End) {
			continue
		}
		filtered = append(filtered, point)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Timestamp.Before(filtered[j].Timestamp) })
	gaps := make([]domain.Gap, 0)
	threshold := request.ExpectedInterval * 3 / 2
	for i := 1; i < len(filtered); i++ {
		gapDuration := filtered[i].Timestamp.Sub(filtered[i-1].Timestamp)
		if gapDuration > threshold {
			gaps = append(gaps, domain.Gap{
				DeviceID:         request.DeviceID,
				Property:         request.Property,
				Start:            filtered[i-1].Timestamp,
				End:              filtered[i].Timestamp,
				Duration:         gapDuration,
				ExpectedInterval: request.ExpectedInterval,
			})
		}
	}
	return gaps, nil
}

func seriesKey(deviceID, property string) string {
	return deviceID + "\x00" + property
}

func clonePoint(point domain.Point) domain.Point {
	out := point
	out.Metadata = make(map[string]any, len(point.Metadata))
	for key, value := range point.Metadata {
		out.Metadata[key] = value
	}
	return out
}

func (s *Store) Snapshot() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int, len(s.points))
	for key, points := range s.points {
		out[key] = len(points)
	}
	return out
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.points = map[string][]domain.Point{}
	return nil
}

func IsNotFound(err error) bool {
	return apperr.IsKind(err, apperr.KindNotFound)
}
