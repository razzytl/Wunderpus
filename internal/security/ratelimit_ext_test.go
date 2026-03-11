package security

import (
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(time.Second*100, 60)
	if rl == nil {
		t.Error("expected non-nil rate limiter")
	}
}

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(time.Second*5, 5)

	// First 5 should succeed
	for i := 0; i < 5; i++ {
		allowed := rl.Allow("test")
		if !allowed {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// 6th should fail
	allowed := rl.Allow("test")
	if allowed {
		t.Error("6th request should be rate limited")
	}
}

func TestRateLimiterDifferentKeys(t *testing.T) {
	rl := NewRateLimiter(time.Second*2, 60)

	// Different keys should have separate limits
	rl.Allow("key1")
	rl.Allow("key1")

	// key1 exhausted but key2 should still work
	if !rl.Allow("key2") {
		t.Error("key2 should still have allowance")
	}
}
