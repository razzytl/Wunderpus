package provider

import (
	"fmt"
	"log/slog"

	"github.com/wonderpus/wonderpus/internal/config"
)

// Router manages available providers and routes to the right one.
type Router struct {
	providers      map[string]Provider
	defaultName    string
	activeName     string
	activeProvider Provider
}

// NewRouter creates a router from the app config.
func NewRouter(cfg *config.Config) (*Router, error) {
	r := &Router{
		providers:   make(map[string]Provider),
		defaultName: cfg.DefaultProvider,
	}

	// Register configured providers
	if cfg.Providers.OpenAI.APIKey != "" {
		p := NewOpenAI(cfg.Providers.OpenAI.APIKey, cfg.Providers.OpenAI.Model, cfg.Providers.OpenAI.MaxTokens)
		r.providers["openai"] = p
		slog.Info("provider registered", "name", "openai", "model", cfg.Providers.OpenAI.Model)
	}
	if cfg.Providers.Anthropic.APIKey != "" {
		p := NewAnthropic(cfg.Providers.Anthropic.APIKey, cfg.Providers.Anthropic.Model, cfg.Providers.Anthropic.MaxTokens)
		r.providers["anthropic"] = p
		slog.Info("provider registered", "name", "anthropic", "model", cfg.Providers.Anthropic.Model)
	}
	if cfg.Providers.Ollama.Host != "" {
		p := NewOllama(cfg.Providers.Ollama.Host, cfg.Providers.Ollama.Model, cfg.Providers.Ollama.MaxTokens)
		r.providers["ollama"] = p
		slog.Info("provider registered", "name", "ollama", "model", cfg.Providers.Ollama.Model)
	}
	if cfg.Providers.Gemini.APIKey != "" {
		p := NewGemini(cfg.Providers.Gemini.APIKey, cfg.Providers.Gemini.Model, cfg.Providers.Gemini.MaxTokens)
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
