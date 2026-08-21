package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/example/iot-device-management/internal/rule/domain"
	rulemem "github.com/example/iot-device-management/internal/rule/infrastructure/memory"
	tsdomain "github.com/example/iot-device-management/internal/timeseries/domain"
)

type countingExecutor struct {
	mu    sync.Mutex
	calls int
}

func (e *countingExecutor) Execute(context.Context, domain.Rule, domain.Execution) error {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()
	return nil
}

func TestCooldownPreventsDuplicateActions(t *testing.T) {
	repo := rulemem.NewRepository()
	_ = repo.SaveRule(context.Background(), domain.Rule{
		ID:       "rule-1",
		Name:     "high temp",
		Enabled:  true,
		DeviceID: "dev-1",
		Property: "temp",
		Condition: domain.Condition{
			Operator:   domain.OperatorGT,
			Threshold:  0,
			Aggregate:  domain.AggregateAvg,
			MinSamples: 1,
		},
		Window:   time.Minute,
		Cooldown: time.Hour,
	})
	executor := &countingExecutor{}
	engine := NewEngine(repo, nil, executor, nil)
	point := tsdomain.Point{DeviceID: "dev-1", Property: "temp", Timestamp: time.Now(), Value: 20}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = engine.HandleTelemetry(context.Background(), point)
		}()
	}
	close(start)
	wg.Wait()
	executor.mu.Lock()
	calls := executor.calls
	executor.mu.Unlock()
	if calls != 1 {
		t.Fatalf("expected exactly one action execution, got %d", calls)
	}
}
