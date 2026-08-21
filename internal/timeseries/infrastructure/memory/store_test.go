package memory

import (
	"context"
	"testing"
	"time"

	"github.com/example/iot-device-management/internal/timeseries/domain"
)

func TestQueryReturnsIndependentSnapshot(t *testing.T) {
	store := NewStore(100)
	now := time.Now().UTC().Truncate(time.Second)
	if err := store.Write(context.Background(), []domain.Point{
		{DeviceID: "dev-1", Property: "temp", Timestamp: now, Value: 1},
		{DeviceID: "dev-1", Property: "temp", Timestamp: now.Add(time.Second), Value: 2},
	}); err != nil {
		t.Fatal(err)
	}
	query := domain.Query{
		DeviceID: "dev-1",
		Property: "temp",
		Start:    now.Add(time.Millisecond),
		End:      now.Add(time.Hour),
		Limit:    1,
	}
	if _, err := store.Query(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	points, err := store.Query(context.Background(), domain.Query{
		DeviceID: "dev-1",
		Property: "temp",
		Start:    now.Add(-time.Hour),
		End:      now.Add(time.Hour),
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 || points[0].Timestamp.Equal(points[1].Timestamp) {
		t.Fatalf("stored series was mutated by query: %+v", points)
	}
}
