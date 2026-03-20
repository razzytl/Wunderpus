package provider

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/config"
	"github.com/wunderpus/wunderpus/internal/logging"
)

// Router manages available providers and routes to the right one.
type Router struct {
	providers       map[string]Provider
	defaultName     string
	activeName      string
	activeProvider  Provider
	Cache           *ResponseCache
	stats           map[string]*ProviderStats
	cooldown        *CooldownTracker
	errorClassifier *ErrorClassifier
	mu              sync.RWMutex
}

// ParallelConfig holds configuration for parallel provider probing
type ParallelConfig struct {
	Enabled      bool
	Timeout      time.Duration // Max time to wait for any provider
	MaxProviders int           // Max number of providers to probe in parallel (0 = all)
}

type ProviderStats struct {
	FailCount      int
	LastFailure    time.Time
	Latencies      []time.Duration
	FallbackModels []string
}

// NewRouter creates a router from the app config.
// It supports both the new model_list format and the legacy providers format.
func NewRouter(cfg *config.Config) (*Router, error) {
	r := &Router{
		providers:       make(map[string]Provider),
		defaultName:     cfg.DefaultProvider,
		Cache:           NewResponseCache(5 * time.Minute),
		stats:           make(map[string]*ProviderStats),
		cooldown:        NewCooldownTracker(),
		errorClassifier: &ErrorClassifier{},
	}

	// Determine which model entries to use
	var modelEntries []config.ModelEntry
	if len(cfg.ModelList) > 0 {
		// Use the new model_list format
		modelEntries = cfg.ModelList
	} else {
		// Convert legacy providers to model entries
		modelEntries = config.ConvertLegacyToModelList(cfg)
	}

	if len(modelEntries) > 0 {
		// Build providers from model entries via the factory
		for _, entry := range modelEntries {
			// Skip entries already registered (support multiple entries with same name for load balancing later)
			if _, exists := r.providers[entry.ModelName]; exists {
				continue
			}

			p, err := NewFromModelEntry(entry)
			if err != nil {
				slog.Warn("failed to create provider from model_list entry", "model_name", entry.ModelName, "error", err)
				continue
			}
			r.providers[entry.ModelName] = p
			r.stats[entry.ModelName] = &ProviderStats{FallbackModels: entry.FallbackModels}
			slog.Info("provider registered", "name", entry.ModelName, "model", entry.ModelID(), "protocol", entry.DetectProtocol())
		}

		// Also register by protocol name for backward compatibility with SetActive("openai")
		for _, entry := range modelEntries {
			protocol := entry.DetectProtocol()
			if _, exists := r.providers[protocol]; !exists {
				if p, exists2 := r.providers[entry.ModelName]; exists2 {
					r.providers[protocol] = p
					if _, exists3 := r.stats[protocol]; !exists3 {
						r.stats[protocol] = r.stats[entry.ModelName]
					}
				}
			}
		}
	} else {
		// Fallback: legacy hardcoded path (should not happen if config has any providers)
		return nil, fmt.Errorf("no providers configured")
	}

	if len(r.providers) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	// Set active provider
	if err := r.SetActive(r.defaultName); err != nil {
		// Fallback to first available
		for name := range r.providers {
			_ = r.SetActive(name)
			break
		}
	}

	return r, nil
}

// Active returns the currently active provider.
func (r *Router) Active() Provider {
	return r.activeProvider
}

// GetEmbedder returns an Embedder if the active provider supports embeddings.
func (r *Router) GetEmbedder() Embedder {
	if r.activeProvider == nil {
		return nil
	}
	// Try type assertion on common providers
	if emb, ok := r.activeProvider.(Embedder); ok {
		return emb
	}
	// Check if it's an OpenAI-compatible provider that has embedding support
	// by looking for common embedding patterns
	return nil
}

// ActiveName returns the name of the active provider.
func (r *Router) ActiveName() string {
	return r.activeName
}

// SetActive switches to a different provider by name.
func (r *Router) SetActive(name string) error {
	p, ok := r.providers[name]
	if !ok {
		return fmt.Errorf("provider %q not found (available: %v)", name, r.List())
	}
	r.activeName = name
	r.activeProvider = p
	slog.Info("active provider set", "name", name)
	return nil
}

// List returns the names of all available providers.
func (r *Router) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// Get returns a provider by name, or nil if not found.
func (r *Router) Get(name string) Provider {
	return r.providers[name]
}

// CompleteWithFallback tries the active provider (with model fallbacks), then others.
func (r *Router) CompleteWithFallback(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()
	prov := r.activeProvider
	name := r.activeName

	// 1. Try active provider (including tiered model fallbacks)
	models := []string{req.Model}
	if req.Model == "" {
		models = []string{""} // Use provider default
	}
	if s, ok := r.stats[name]; ok {
		models = append(models, s.FallbackModels...)
	}

	var lastErr error
	for _, m := range models {
		mReq := *req
		mReq.Model = m

		resp, err := prov.Complete(ctx, &mReq)
		if err == nil {
			r.recordSuccess(name, time.Since(start))
			logging.ProviderLatency.WithLabelValues(prov.Name(), m).Observe(time.Since(start).Seconds())
			return resp, nil
		}
		lastErr = err
		r.recordFailure(name)
		logging.L(ctx).Warn("model failed, trying internal fallback", "provider", name, "model", m, "error", err)
	}

	logging.L(ctx).Warn("primary provider exhausted all models, attempting alternative provider", "provider", name, "error", lastErr)

	// 2. Try all other healthy providers
	for pname, p := range r.providers {
		if pname == r.activeName || !r.isHealthy(pname) {
			continue
		}

		logging.L(ctx).Info("trying fallback provider", "provider", pname)
		fStart := time.Now()
		resp, err := p.Complete(ctx, req)
		if err == nil {
			r.recordSuccess(pname, time.Since(fStart))
			logging.ProviderLatency.WithLabelValues(pname, req.Model).Observe(time.Since(fStart).Seconds())
			logging.L(ctx).Info("fallback successful", "provider", pname)
			return resp, nil
		}
		r.recordFailure(pname)
		logging.L(ctx).Warn("fallback provider failed", "provider", pname, "error", err)
	}

	return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
}

func (r *Router) recordSuccess(name string, d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.stats[name]
	if s == nil {
		return
	}
	s.FailCount = 0 // Reset on success
	s.Latencies = append(s.Latencies, d)
	if len(s.Latencies) > 20 {
		s.Latencies = s.Latencies[1:]
	}
}

func (r *Router) recordFailure(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.stats[name]
	if s == nil {
		return
	}
	s.FailCount++
	s.LastFailure = time.Now()
}

func (r *Router) isHealthy(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s := r.stats[name]
	if s == nil {
		return true
	}
	if r.cooldown != nil && r.cooldown.IsInCooldown(name) {
		return false
	}
	if s.FailCount > 5 && time.Since(s.LastFailure) < 5*time.Minute {
		return false
	}
	return true
}

// GetProviderHealth returns health status for all providers
func (r *Router) GetProviderHealth() map[string]map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health := make(map[string]map[string]any)
	for name, stats := range r.stats {
		health[name] = map[string]any{
			"fail_count":   stats.FailCount,
			"last_failure": stats.LastFailure,
			"in_cooldown":  r.cooldown.IsInCooldown(name),
			"healthy":      r.isHealthy(name),
		}
	}
	return health
}

// CompleteParallel attempts multiple providers in parallel and returns the first successful response.
// This is useful when you want the fastest response regardless of which provider provides it.
// The config controls how many providers to probe and the timeout.
func (r *Router) CompleteParallel(ctx context.Context, req *CompletionRequest, cfg ParallelConfig) (*CompletionResponse, string, error) {
	if !cfg.Enabled {
		return nil, "", fmt.Errorf("parallel mode is not enabled")
	}

	// Get list of healthy providers
	r.mu.RLock()
	var healthyProviders []struct {
		name     string
		provider Provider
	}
	for name, p := range r.providers {
		if r.isHealthy(name) {
			healthyProviders = append(healthyProviders, struct {
				name     string
				provider Provider
			}{name, p})
		}
	}
	r.mu.RUnlock()

	if len(healthyProviders) == 0 {
		return nil, "", fmt.Errorf("no healthy providers available")
	}

	// Limit number of providers if configured
	if cfg.MaxProviders > 0 && len(healthyProviders) > cfg.MaxProviders {
		healthyProviders = healthyProviders[:cfg.MaxProviders]
	}

	// Set up timeout
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second // default timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Result channel
	type result struct {
		resp     *CompletionResponse
		provider string
		err      error
	}
	resultCh := make(chan result, len(healthyProviders))

	// Launch all providers in parallel
	var wg sync.WaitGroup
	for _, hp := range healthyProviders {
		wg.Add(1)
		go func(name string, p Provider) {
			defer wg.Done()
			resp, err := p.Complete(ctx, req)
			select {
			case resultCh <- result{resp, name, err}:
			default:
				// Another result already came through, this one is irrelevant
			}
		}(hp.name, hp.provider)
	}

	// Wait for first result or timeout
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	var firstErr error
	for res := range resultCh {
		if res.err == nil {
			r.recordSuccess(res.provider, 0) // Duration not tracked in parallel mode
			return res.resp, res.provider, nil
		}
		if firstErr == nil {
			firstErr = res.err
		}
	}

	if firstErr != nil {
		return nil, "", fmt.Errorf("all parallel providers failed: %w", firstErr)
	}
	return nil, "", fmt.Errorf("no providers completed in time")
}
