package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/wonderpus/wonderpus/internal/config"
	"github.com/wonderpus/wonderpus/internal/logging"
	"github.com/wonderpus/wonderpus/internal/security"
	"sync"
	"time"
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
func NewRouter(cfg *config.Config) (*Router, error) {
	r := &Router{
		providers:   make(map[string]Provider),
		defaultName: cfg.DefaultProvider,
		Cache:       NewResponseCache(5 * time.Minute), // Default 5 min TTL
	}

	// encryption key
	var decodedKey []byte
	if cfg.Security.Encryption.Enabled && cfg.Security.Encryption.Key != "" {
		var err error
		decodedKey, err = base64.StdEncoding.DecodeString(cfg.Security.Encryption.Key)
		if err != nil {
			return nil, fmt.Errorf("invalid encryption key: %w", err)
		}
	}

	// Helper to get key (decrypt if needed)
	getKey := func(k string) (string, error) {
		if k == "" {
			return "", nil
		}
		if len(decodedKey) > 0 {
			return security.Decrypt(k, decodedKey)
		}
		return k, nil
	}

	// OpenAI
	openAIKey, err := getKey(cfg.Providers.OpenAI.APIKey)
	if err != nil {
		return nil, fmt.Errorf("openai key decryption: %w", err)
	}
	if openAIKey != "" {
		p := NewOpenAI(openAIKey, cfg.Providers.OpenAI.Model, cfg.Providers.OpenAI.MaxTokens)
		r.providers["openai"] = p
		slog.Info("provider registered", "name", "openai", "model", cfg.Providers.OpenAI.Model)
	}

	// Anthropic
	anthropicKey, err := getKey(cfg.Providers.Anthropic.APIKey)
	if err != nil {
		return nil, fmt.Errorf("anthropic key decryption: %w", err)
	}
	if anthropicKey != "" {
		p := NewAnthropic(anthropicKey, cfg.Providers.Anthropic.Model, cfg.Providers.Anthropic.MaxTokens)
		r.providers["anthropic"] = p
		slog.Info("provider registered", "name", "anthropic", "model", cfg.Providers.Anthropic.Model)
	}

	// Ollama (no key needed)
	if cfg.Providers.Ollama.Host != "" {
		p := NewOllama(cfg.Providers.Ollama.Host, cfg.Providers.Ollama.Model, cfg.Providers.Ollama.MaxTokens)
		r.providers["ollama"] = p
		slog.Info("provider registered", "name", "ollama", "model", cfg.Providers.Ollama.Model)
	}

	// Gemini
	geminiKey, err := getKey(cfg.Providers.Gemini.APIKey)
	if err != nil {
		return nil, fmt.Errorf("gemini key decryption: %w", err)
	}
	if geminiKey != "" {
		p := NewGemini(geminiKey, cfg.Providers.Gemini.Model, cfg.Providers.Gemini.MaxTokens)
		r.providers["gemini"] = p
		slog.Info("provider registered", "name", "gemini", "model", cfg.Providers.Gemini.Model)
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

	// Init stats
	r.stats = make(map[string]*ProviderStats)
	r.stats["openai"] = &ProviderStats{FallbackModels: cfg.Providers.OpenAI.FallbackModels}
	r.stats["anthropic"] = &ProviderStats{FallbackModels: cfg.Providers.Anthropic.FallbackModels}
	r.stats["gemini"] = &ProviderStats{FallbackModels: cfg.Providers.Gemini.FallbackModels}
	r.stats["ollama"] = &ProviderStats{FallbackModels: cfg.Providers.Ollama.FallbackModels}

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
