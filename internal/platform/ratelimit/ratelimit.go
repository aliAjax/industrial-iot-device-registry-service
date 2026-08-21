package ratelimit

import (
	"context"
	"sync"
	"time"
)

type Limiter interface {
	Allow(ctx context.Context, key string) bool
}

type tokenBucket struct {
	tokens   float64
	last     time.Time
	rate     float64
	capacity float64
}

type MemoryLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rate    float64
	window  time.Duration
}

func NewMemoryLimiter(ratePerWindow int, window time.Duration) *MemoryLimiter {
	return &MemoryLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    float64(ratePerWindow) / window.Seconds(),
		window:  window,
	}
}

func (l *MemoryLimiter) Allow(_ context.Context, key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &tokenBucket{tokens: float64(1), last: now, rate: l.rate, capacity: float64(1)}
		l.buckets[key] = bucket
		return true
	}
	elapsed := now.Sub(bucket.last).Seconds()
	bucket.tokens += elapsed * bucket.rate
	if bucket.tokens > bucket.capacity {
		bucket.tokens = bucket.capacity
	}
	bucket.last = now
	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

func (l *MemoryLimiter) Prune(ctx context.Context, olderThan time.Duration) {
	ticker := time.NewTicker(olderThan)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.mu.Lock()
			cutoff := time.Now().Add(-olderThan)
			for key, bucket := range l.buckets {
				if bucket.last.Before(cutoff) {
					delete(l.buckets, key)
				}
			}
			l.mu.Unlock()
		}
	}
}
