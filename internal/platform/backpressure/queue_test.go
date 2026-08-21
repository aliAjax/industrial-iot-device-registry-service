package backpressure

import (
	"context"
	"sync"
	"testing"
)

func TestQueueCloseRejectsConcurrentEnqueue(t *testing.T) {
	q := NewQueue[int](1)
	closed := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		q.Close()
		close(closed)
	}()
	go func() {
		defer wg.Done()
		<-closed
		_ = q.Enqueue(context.Background(), 1)
	}()
	wg.Wait()
}

func TestClosedQueueRejectsEnqueue(t *testing.T) {
	q := NewQueue[int](1)
	q.Close()
	if err := q.Enqueue(context.Background(), 1); err == nil {
		t.Fatal("expected enqueue into closed queue to fail")
	}
}

func TestClosedQueueTryEnqueueReturnsFalse(t *testing.T) {
	q := NewQueue[int](1)
	q.Close()
	if q.TryEnqueue(1) {
		t.Fatal("expected TryEnqueue into closed queue to return false")
	}
}
