package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestLimiterAllowsConfiguredBurst(t *testing.T) {
	limiter := NewMemoryLimiter(5, time.Minute)
	for i := 0; i < 5; i++ {
		if !limiter.Allow(context.Background(), "device-a") {
			t.Fatalf("request %d within configured burst should be allowed", i+1)
		}
	}
	if limiter.Allow(context.Background(), "device-a") {
		t.Fatal("request beyond configured burst should be denied")
	}
}
