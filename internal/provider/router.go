package provider

import (
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/wonderpus/wonderpus/internal/config"
	"github.com/wonderpus/wonderpus/internal/security"
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
