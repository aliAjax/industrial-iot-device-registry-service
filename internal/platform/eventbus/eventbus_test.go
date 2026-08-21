package eventbus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestUnsubscribeStopsHandler(t *testing.T) {
	bus := NewMemoryBus(10)
	var calls atomic.Int32
	unsubscribe := bus.Subscribe("test.event", func(context.Context, Event) error {
		calls.Add(1)
		return nil
	})
	unsubscribe()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			if err := bus.Publish(context.Background(), Event{ID: fmt.Sprintf("e%d", index), Type: "test.event"}); err != nil {
				t.Errorf("publish after unsubscribe failed: %v", err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if got := calls.Load(); got != 0 {
		t.Fatalf("handler called after unsubscribe: %d", got)
	}
}

func TestHistoryReturnsClonedPayload(t *testing.T) {
	bus := NewMemoryBus(10)
	payload := map[string]any{"mode": "eco"}
	if err := bus.Publish(context.Background(), Event{ID: "e1", Type: "test.event", Payload: payload}); err != nil {
		t.Fatal(err)
	}
	payload["mode"] = "turbo"
	events := bus.History("test.event")
	if events[0].Payload.(map[string]any)["mode"] != "eco" {
		t.Fatal("history payload aliases publisher input")
	}
}
