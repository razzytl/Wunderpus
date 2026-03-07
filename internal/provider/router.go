package provider

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wonderpus/wonderpus/internal/config"
	"github.com/wonderpus/wonderpus/internal/logging"
)

// Router manages available providers and routes to the right one.
type Router struct {
	providers      map[string]Provider
	defaultName    string
	activeName     string
	activeProvider Provider
	Cache          *ResponseCache
	stats          map[string]*ProviderStats
	mu             sync.RWMutex
}

type ProviderStats struct {
	FailCount    int
	LastFailure  time.Time
	Latencies    []time.Duration
	FallbackModels []string
}

// NewRouter creates a router from the app config.
// It supports both the new model_list format and the legacy providers format.
func NewRouter(cfg *config.Config) (*Router, error) {
	r := &Router{
		providers:   make(map[string]Provider),
		defaultName: cfg.DefaultProvider,
		Cache:       NewResponseCache(5 * time.Minute),
		stats:       make(map[string]*ProviderStats),
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
	// Circuit breaker: trip if > 5 fails in last 5 mins
	if s.FailCount > 5 && time.Since(s.LastFailure) < 5*time.Minute {
		return false
	}
	return true
}
