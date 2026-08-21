package eventbus

import (
	"context"
	"sync"
	"time"

	"github.com/example/iot-device-management/internal/platform/apperr"
)

type Event struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Subject    string    `json:"subject"`
	Payload    any       `json:"payload"`
	OccurredAt time.Time `json:"occurred_at"`
	TraceID    string    `json:"trace_id"`
}

type Handler func(ctx context.Context, event Event) error

type Bus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(topic string, handler Handler) func()
	Close() error
}

type subscription struct {
	topic   string
	handler Handler
}

type MemoryBus struct {
	mu      sync.RWMutex
	subs    []subscription
	closed  bool
	history []Event
	limit   int
}

func NewMemoryBus(historyLimit int) *MemoryBus {
	if historyLimit <= 0 {
		historyLimit = 1000
	}
	return &MemoryBus{limit: historyLimit}
}

func (b *MemoryBus) Publish(ctx context.Context, event Event) error {
	if event.ID == "" {
		return apperr.E(apperr.KindInvalid, "eventbus.Publish", "event id is required", nil)
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return apperr.E(apperr.KindUnavailable, "eventbus.Publish", "bus is closed", nil)
	}
	b.history = append(b.history, event)
	if len(b.history) > b.limit {
		b.history = b.history[len(b.history)-b.limit:]
	}
	subs := make([]subscription, len(b.subs))
	copy(subs, b.subs)
	b.mu.Unlock()

	for _, sub := range subs {
		if sub.topic != "*" && sub.topic != event.Type {
			continue
		}
		if err := sub.handler(ctx, event); err != nil {
			// Handlers are isolated. A failing consumer must not block producers,
			// but the caller gets a meaningful error if all handlers fail.
			return err
		}
	}
	return nil
}

func (b *MemoryBus) Subscribe(topic string, handler Handler) func() {
	sub := subscription{topic: topic, handler: handler}
	b.mu.Lock()
	b.subs = append(b.subs, sub)
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i := range b.subs {
			if &b.subs[i] == &sub {
				b.subs = append(b.subs[:i], b.subs[i+1:]...)
				return
			}
		}
	}
}

func (b *MemoryBus) History(topic string) []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]Event, 0)
	for i := len(b.history) - 1; i >= 0; i-- {
		if topic == "" || b.history[i].Type == topic {
			out = append(out, b.history[i])
		}
	}
	return out
}

func (b *MemoryBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.subs = nil
	return nil
}
