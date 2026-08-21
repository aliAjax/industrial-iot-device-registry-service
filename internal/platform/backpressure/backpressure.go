package backpressure

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/example/iot-device-management/internal/platform/apperr"
)

var (
	errQueueClosed = errors.New("queue is closed")
	errQueueFull   = errors.New("queue is full")
)

type Queue[T any] struct {
	ch      chan T
	dropped int64
	mu      sync.RWMutex
	closed  bool
}

func NewQueue[T any](size int) *Queue[T] {
	if size <= 0 {
		size = 1
	}
	return &Queue[T]{ch: make(chan T, size)}
}

// Enqueue attempts to deliver item to the queue, blocking until item is
// accepted or ctx is cancelled. It is safe to call concurrently with Close:
// a closed queue rejects the send instead of panicking.
func (q *Queue[T]) Enqueue(ctx context.Context, item T) error {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return apperr.E(apperr.KindUnavailable, "backpressure.Enqueue", "queue is closed", nil)
	}
	defer q.mu.RUnlock()
	select {
	case q.ch <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryEnqueue attempts to deliver item without blocking. It reports false when
// the queue is closed or has no free slot. It is safe to call concurrently
// with Close: a closed queue reports false instead of panicking.
func (q *Queue[T]) TryEnqueue(item T) bool {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return false
	}
	defer q.mu.RUnlock()
	select {
	case q.ch <- item:
		return true
	default:
		q.mu.RUnlock()
		q.mu.Lock()
		q.dropped++
		q.mu.Unlock()
		return false
	}
}

// Recv removes the next item from the queue, blocking until one is available,
// ctx is cancelled, or the queue is drained after being closed. When the queue
// is closed and empty, Recv returns an error so consumers can exit their loop.
func (q *Queue[T]) Recv(ctx context.Context) (T, error) {
	var zero T
	select {
	case item, ok := <-q.ch:
		if !ok {
			return zero, errQueueClosed
		}
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

// Close shuts the queue down. It is idempotent and safe to call concurrently
// with Enqueue/TryEnqueue: the write lock is held across close(ch), and
// senders hold the read lock across the channel send, so the two operations
// are mutually exclusive — a send never lands on a closed channel.
func (q *Queue[T]) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.closed = true
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
		case item, ok := <-b.queue.ch:
			if !ok {
				if len(items) == 0 {
					return nil, errQueueClosed
				}
				return items, nil
			}
			items = append(items, item)
		}
	}
	return items, nil
}
