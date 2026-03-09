package security

import (
	"log/slog"
	"sync"
	"time"
)

// RateLimiter implements a sliding window rate limiter with optional automatic cleanup.
type RateLimiter struct {
	mu          sync.Mutex
	limits      map[string][]time.Time
	window      time.Duration
	maxRequests int

	// Automatic cleanup
	stopCleanup chan struct{}
	wg          sync.WaitGroup
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

// Cleanup manually removes expired entries from memory.
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

// StartAutoCleanup starts a background goroutine that periodically cleans up
// expired entries. The interval parameter specifies how often to run cleanup.
// If interval is 0 or negative, no automatic cleanup is started.
// Call StopAutoCleanup to stop the background goroutine.
func (rl *RateLimiter) StartAutoCleanup(interval time.Duration) {
	if interval <= 0 {
		slog.Debug("rate limiter: automatic cleanup disabled (interval <= 0)")
		return
	}

	rl.stopCleanup = make(chan struct{})
	rl.wg.Add(1)

	go func() {
		defer rl.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		slog.Debug("rate limiter: started automatic cleanup", "interval", interval)
		for {
			select {
			case <-rl.stopCleanup:
				slog.Debug("rate limiter: stopped automatic cleanup")
				return
			case <-ticker.C:
				rl.Cleanup()
				slog.Debug("rate limiter: performed automatic cleanup")
			}
		}
	}()
}

// StopAutoCleanup stops the background cleanup goroutine.
// It waits for the goroutine to finish before returning.
func (rl *RateLimiter) StopAutoCleanup() {
	if rl.stopCleanup == nil {
		return
	}

	close(rl.stopCleanup)
	rl.wg.Wait()
	rl.stopCleanup = nil
	slog.Debug("rate limiter: cleanup goroutine stopped")
}

// Len returns the current number of sessions being tracked.
func (rl *RateLimiter) Len() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.limits)
}
