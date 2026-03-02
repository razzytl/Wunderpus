package security

import (
	"sync"
	"time"
)

// RateLimiter implements a simple sliding window rate limiter for sessions.
type RateLimiter struct {
	mu          sync.Mutex
	limits      map[string][]time.Time
	window      time.Duration
	maxRequests int
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(window time.Duration, maxRequests int) *RateLimiter {
	return &RateLimiter{
		limits:      make(map[string][]time.Time),
		window:      window,
		maxRequests: maxRequests,
	}
}

// Allow checks if a request from the given sessionID is allowed.
func (rl *RateLimiter) Allow(sessionID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean up old requests
	requests := rl.limits[sessionID]
	var valid []time.Time
	for _, t := range requests {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.maxRequests {
		rl.limits[sessionID] = valid
		return false
	}

	rl.limits[sessionID] = append(valid, now)
	return true
}

// Cleanup periodically removes expired entries from memory.
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	for id, requests := range rl.limits {
		var valid []time.Time
		for _, t := range requests {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.limits, id)
		} else {
			rl.limits[id] = valid
		}
	}
}
