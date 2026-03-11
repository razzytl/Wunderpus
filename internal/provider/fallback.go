package provider

//go:generate stringer -type=FailoverReason -linecomment

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/logging"
)

type FailoverReason int

const (
	FailoverReasonNone FailoverReason = iota
	FailoverReasonRateLimit
	FailoverReasonTimeout
	FailoverReasonServerError
	FailoverReasonQuotaExceeded
	FailoverReasonInvalidRequest
	FailoverReasonRetriableError
)

type ErrorClassifier struct{}

func (e *ErrorClassifier) Classify(err error) FailoverReason {
	if err == nil {
		return FailoverReasonNone
	}

	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		return FailoverReasonRateLimit
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "timeout") {
		return FailoverReasonTimeout
	}
	if strings.Contains(errStr, "500") || strings.Contains(errStr, "502") || strings.Contains(errStr, "503") || strings.Contains(errStr, "504") {
		return FailoverReasonServerError
	}
	if strings.Contains(errStr, "quota") || strings.Contains(errStr, "429") || strings.Contains(errStr, "insufficient") {
		return FailoverReasonQuotaExceeded
	}
	if strings.Contains(errStr, "400") || strings.Contains(errStr, "invalid") || strings.Contains(errStr, "malformed") {
		return FailoverReasonInvalidRequest
	}

	return FailoverReasonRetriableError
}

func (e *ErrorClassifier) IsRetriable(reason FailoverReason) bool {
	switch reason {
	case FailoverReasonRateLimit, FailoverReasonTimeout, FailoverReasonServerError, FailoverReasonRetriableError:
		return true
	default:
		return false
	}
}

type CooldownTracker struct {
	mu         sync.RWMutex
	cooldowns  map[string]time.Time
	failCounts map[string]int
	windows    map[string][]time.Time
}

func NewCooldownTracker() *CooldownTracker {
	return &CooldownTracker{
		cooldowns:  make(map[string]time.Time),
		failCounts: make(map[string]int),
		windows:    make(map[string][]time.Time),
	}
}

func (c *CooldownTracker) StartCooldown(provider string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cooldowns[provider] = time.Now().Add(duration)
}

func (c *CooldownTracker) IsInCooldown(provider string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if cooldown, ok := c.cooldowns[provider]; ok {
		return time.Now().Before(cooldown)
	}
	return false
}

func (c *CooldownTracker) RecordFailure(provider string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failCounts[provider]++

	now := time.Now()
	if c.windows[provider] == nil {
		c.windows[provider] = []time.Time{}
	}
	c.windows[provider] = append(c.windows[provider], now)

	windowStart := now.Add(-5 * time.Minute)
	var recent []time.Time
	for _, t := range c.windows[provider] {
		if t.After(windowStart) {
			recent = append(recent, t)
		}
	}
	c.windows[provider] = recent

	if len(recent) > 5 {
		c.cooldowns[provider] = now.Add(2 * time.Minute)
	}
}

func (c *CooldownTracker) RecordSuccess(provider string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failCounts[provider] = 0
	delete(c.cooldowns, provider)
}

func (c *CooldownTracker) GetFailCount(provider string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.failCounts[provider]
}

type FallbackAttempt struct {
	Provider string
	Model    string
	Error    error
	Reason   FailoverReason
	Duration time.Duration
	Skipped  bool
}

type FallbackResult struct {
	Response *CompletionResponse
	Provider string
	Model    string
	Attempts []FallbackAttempt
}

func (r *Router) EnhancedFallback(ctx context.Context, req *CompletionRequest) (*FallbackResult, error) {
	cooldown := NewCooldownTracker()
	classifier := &ErrorClassifier{}

	attempts := []FallbackAttempt{}

	models := []string{req.Model}
	if req.Model == "" {
		models = []string{""}
	}
	if s, ok := r.stats[r.activeName]; ok {
		models = append(models, s.FallbackModels...)
	}

	for _, m := range models {
		if cooldown.IsInCooldown(r.activeName) {
			attempts = append(attempts, FallbackAttempt{
				Provider: r.activeName,
				Model:    m,
				Reason:   FailoverReasonNone,
				Skipped:  true,
			})
			continue
		}

		start := time.Now()
		mReq := *req
		mReq.Model = m

		resp, err := r.activeProvider.Complete(ctx, &mReq)
		duration := time.Since(start)

		reason := classifier.Classify(err)

		attempt := FallbackAttempt{
			Provider: r.activeName,
			Model:    m,
			Error:    err,
			Reason:   reason,
			Duration: duration,
		}
		attempts = append(attempts, attempt)

		if err == nil {
			r.recordSuccess(r.activeName, duration)
			cooldown.RecordSuccess(r.activeName)
			logging.ProviderLatency.WithLabelValues(r.activeName, m).Observe(duration.Seconds())
			return &FallbackResult{
				Response: resp,
				Provider: r.activeName,
				Model:    m,
				Attempts: attempts,
			}, nil
		}

		r.recordFailure(r.activeName)
		cooldown.RecordFailure(r.activeName)
		logging.L(ctx).Warn("model failed", "provider", r.activeName, "model", m, "reason", reason, "error", err)

		if !classifier.IsRetriable(reason) {
			break
		}
	}

	for pname, p := range r.providers {
		if pname == r.activeName {
			continue
		}

		if !r.isHealthy(pname) || cooldown.IsInCooldown(pname) {
			attempts = append(attempts, FallbackAttempt{
				Provider: pname,
				Model:    req.Model,
				Reason:   FailoverReasonNone,
				Skipped:  true,
			})
			continue
		}

		start := time.Now()
		resp, err := p.Complete(ctx, req)
		duration := time.Since(start)

		reason := classifier.Classify(err)

		attempt := FallbackAttempt{
			Provider: pname,
			Model:    req.Model,
			Error:    err,
			Reason:   reason,
			Duration: duration,
		}
		attempts = append(attempts, attempt)

		if err == nil {
			r.recordSuccess(pname, duration)
			cooldown.RecordSuccess(pname)
			logging.ProviderLatency.WithLabelValues(pname, req.Model).Observe(duration.Seconds())
			logging.L(ctx).Info("fallback successful", "provider", pname)
			return &FallbackResult{
				Response: resp,
				Provider: pname,
				Model:    req.Model,
				Attempts: attempts,
			}, nil
		}

		r.recordFailure(pname)
		cooldown.RecordFailure(pname)
		logging.L(ctx).Warn("fallback provider failed", "provider", pname, "error", err)
	}

	return nil, fmt.Errorf("all providers failed")
}
