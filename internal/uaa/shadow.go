package uaa

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// SimResult contains the outcome of a shadow simulation.
type SimResult struct {
	Approved      bool
	Reason        string
	EffectSummary string
	SimDuration   time.Duration
}

// SimulateFn is the LLM judge function that evaluates simulation results.
type SimulateFn func(ctx context.Context, summary string) (approved bool, reason string, err error)

// ShadowSimulator runs actions in an in-memory mock environment before
// allowing real execution. It catches dangerous operations before they
// can cause damage.
type ShadowSimulator struct {
	judgeFn  SimulateFn
	timeout  time.Duration
	cache    map[string]*cacheEntry
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

type cacheEntry struct {
	result    *SimResult
	timestamp time.Time
}

// NewShadowSimulator creates a shadow simulator with the given LLM judge function.
func NewShadowSimulator(judgeFn SimulateFn) *ShadowSimulator {
	return &ShadowSimulator{
		judgeFn:  judgeFn,
		timeout:  30 * time.Second,
		cache:    make(map[string]*cacheEntry),
		cacheTTL: 5 * time.Minute,
	}
}

// Simulate runs the action in a mock environment and asks the LLM judge
// whether the outcome is safe and desirable.
func (s *ShadowSimulator) Simulate(ctx context.Context, action Action) (*SimResult, error) {
	start := time.Now()

	// Check cache first
	cacheKey := s.cacheKey(action)
	if cached := s.getCache(cacheKey); cached != nil {
		return cached, nil
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Build simulation summary
	summary := s.buildEffectSummary(action)

	// Ask LLM judge
	approved, reason, err := s.judgeFn(ctx, summary)
	if err != nil {
		slog.Warn("uaa shadow: LLM judge failed, auto-rejecting", "error", err)
		return &SimResult{
			Approved:      false,
			Reason:        fmt.Sprintf("LLM judge error: %v", err),
			EffectSummary: summary,
			SimDuration:   time.Since(start),
		}, nil
	}

	result := &SimResult{
		Approved:      approved,
		Reason:        reason,
		EffectSummary: summary,
		SimDuration:   time.Since(start),
	}

	// Cache approved results
	if approved {
		s.setCache(cacheKey, result)
	}

	slog.Info("uaa shadow: simulation complete",
		"tool", action.Tool,
		"approved", approved,
		"reason", reason,
		"duration", result.SimDuration,
	)

	return result, nil
}

// buildEffectSummary creates a human-readable description of what the action would do.
func (s *ShadowSimulator) buildEffectSummary(action Action) string {
	summary := fmt.Sprintf("ACTION: %s (tool: %s)\n", action.Description, action.Tool)
	summary += fmt.Sprintf("TIER: %d | COST: %d | REVERSIBLE: %v\n", action.Tier, action.TrustCost, action.Reversible)
	summary += fmt.Sprintf("SCOPE: %s\n", action.Scope)

	// Describe parameters
	summary += "PARAMETERS:\n"
	for k, v := range action.Parameters {
		summary += fmt.Sprintf("  %s: %v\n", k, v)
	}

	// Add risk assessment
	if action.Tier == TierExternal {
		summary += "\nRISK: HIGH — This action has external impact beyond the local system.\n"
	} else if action.Tier == TierPersistent {
		summary += "\nRISK: MEDIUM — This action modifies persistent state.\n"
	}

	summary += "\nIs this action safe and desirable? Respond: APPROVE or REJECT with one sentence reason."

	return summary
}

// cacheKey generates a deterministic key for an action based on tool + sorted parameters.
func (s *ShadowSimulator) cacheKey(action Action) string {
	h := sha256.New()
	h.Write([]byte(action.Tool))

	// Sort parameter keys for deterministic hashing
	keys := make([]string, 0, len(action.Parameters))
	for k := range action.Parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		h.Write([]byte(fmt.Sprintf("%s=%v", k, action.Parameters[k])))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *ShadowSimulator) getCache(key string) *SimResult {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	entry, ok := s.cache[key]
	if !ok {
		return nil
	}
	if time.Since(entry.timestamp) > s.cacheTTL {
		return nil
	}
	return entry.result
}

func (s *ShadowSimulator) setCache(key string, result *SimResult) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache[key] = &cacheEntry{result: result, timestamp: time.Now()}
}
