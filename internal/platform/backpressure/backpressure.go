package backpressure

import (
	"context"
	"sync"
	"time"

	"github.com/example/iot-device-management/internal/platform/apperr"
)

type Queue[T any] struct {
	ch      chan T
	dropped int64
	mu      sync.RWMutex
}

func NewQueue[T any](size int) *Queue[T] {
	if size <= 0 {
		size = 1
	}
	return &Queue[T]{ch: make(chan T, size)}
}

func (q *Queue[T]) Enqueue(ctx context.Context, item T) error {
	select {
	case q.ch <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		q.mu.Lock()
		q.dropped++
		q.mu.Unlock()
		return apperr.E(apperr.KindBackpressure, "backpressure.Enqueue", "queue is full", nil)
	}
}

func (q *Queue[T]) TryEnqueue(item T) bool {
	select {
	case q.ch <- item:
		return true
	default:
		q.mu.Lock()
		q.dropped++
		q.mu.Unlock()
		return false
	}
}

func (q *Queue[T]) Recv(ctx context.Context) (T, error) {
	var zero T
	select {
	case item := <-q.ch:
		return item, nil
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

func (q *Queue[T]) Len() int {
	return len(q.ch)
}

func (q *Queue[T]) Dropped() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.dropped
}

func (q *Queue[T]) Close() {
	close(q.ch)
}

type Batcher[T any] struct {
	queue   *Queue[T]
	size    int
	timeout time.Duration
}

func NewBatcher[T any](queue *Queue[T], size int, timeout time.Duration) *Batcher[T] {
	return &Batcher[T]{queue: queue, size: size, timeout: timeout}
}

func (b *Batcher[T]) Batch(ctx context.Context) ([]T, error) {
	items := make([]T, 0, b.size)
	timeout := time.NewTimer(b.timeout)
	defer timeout.Stop()
	for len(items) < b.size {
		select {
		case <-ctx.Done():
			if len(items) == 0 {
				return nil, ctx.Err()
			}
			return items, nil
		case <-timeout.C:
			return items, nil
		case item := <-b.queue.ch:
			items = append(items, item)
		}
	}
	return items, nil
}
