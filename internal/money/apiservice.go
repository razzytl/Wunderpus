package money

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"time"
)

// APIEndpoint represents a paid API endpoint.
type APIEndpoint struct {
	Path        string  `json:"path"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	PricePer1K  float64 `json:"price_per_1k_tokens"`
}

// APIKey represents an API key for access control.
type APIKey struct {
	Key         string    `json:"key"`
	Owner       string    `json:"owner"`
	CreatedAt   time.Time `json:"created_at"`
	UsageTokens int       `json:"usage_tokens"`
	RateLimit   int       `json:"rate_limit"`
	Enabled     bool      `json:"enabled"`
}

// APIServiceEngine exposes Wunderpus capabilities as paid APIs.
type APIServiceEngine struct {
	endpoints  map[string]*APIEndpoint
	keys       map[string]*APIKey
	keyManager KeyManager
	rateLimit  RateLimiter
}

// KeyManager manages API keys.
type KeyManager interface {
	CreateKey(ctx context.Context, owner string) (*APIKey, error)
	ValidateKey(ctx context.Context, key string) (*APIKey, error)
}

// RateLimiter implements rate limiting.
type RateLimiter interface {
	Allow(key string) bool
}

// APIConfig holds API service configuration.
type APIConfig struct {
	Enabled   bool
	StripeKey string
	BaseRate  float64 // base price per 1K tokens
}

// NewAPIServiceEngine creates a new API service engine.
func NewAPIServiceEngine(cfg APIConfig) *APIServiceEngine {
	engine := &APIServiceEngine{
		endpoints: make(map[string]*APIEndpoint),
		keys:      make(map[string]*APIKey),
	}

	// Register built-in endpoints
	engine.registerEndpoints()

	return engine
}

func (e *APIServiceEngine) registerEndpoints() {
	e.endpoints["/v1/research"] = &APIEndpoint{
		Path:        "/v1/research",
		Name:        "research",
		Description: "Deep research with world model context",
		PricePer1K:  0.50,
	}
	e.endpoints["/v1/code"] = &APIEndpoint{
		Path:        "/v1/code",
		Name:        "code",
		Description: "Code generation with RSI-improved quality",
		PricePer1K:  0.80,
	}
	e.endpoints["/v1/analyze"] = &APIEndpoint{
		Path:        "/v1/analyze",
		Name:        "analyze",
		Description: "Data analysis and reporting",
		PricePer1K:  0.60,
	}
	e.endpoints["/v1/automate"] = &APIEndpoint{
		Path:        "/v1/automate",
		Name:        "automate",
		Description: "Browser automation as a service",
		PricePer1K:  1.00,
	}
}

// HandleRequest processes an API request.
func (e *APIServiceEngine) HandleRequest(ctx context.Context, path, apiKey string, input interface{}) (interface{}, error) {
	// Validate API key
	key := e.keys[apiKey]
	if key == nil || !key.Enabled {
		return nil, ErrUnauthorized
	}

	// Check rate limit
	if e.rateLimit != nil && !e.rateLimit.Allow(apiKey) {
		return nil, ErrRateLimited
	}

	// Route to appropriate handler
	switch path {
	case "/v1/research":
		return e.handleResearch(ctx, input)
	case "/v1/code":
		return e.handleCode(ctx, input)
	case "/v1/analyze":
		return e.handleAnalyze(ctx, input)
	case "/v1/automate":
		return e.handleAutomate(ctx, input)
	default:
		return nil, ErrNotFound
	}
}

func (e *APIServiceEngine) handleResearch(ctx context.Context, input interface{}) (interface{}, error) {
	// Implementation would call the research engine
	return map[string]string{"result": "research completed"}, nil
}

func (e *APIServiceEngine) handleCode(ctx context.Context, input interface{}) (interface{}, error) {
	// Implementation would call RSI coder
	return map[string]string{"result": "code generated"}, nil
}

func (e *APIServiceEngine) handleAnalyze(ctx context.Context, input interface{}) (interface{}, error) {
	// Implementation would analyze data
	return map[string]string{"result": "analysis completed"}, nil
}

func (e *APIServiceEngine) handleAutomate(ctx context.Context, input interface{}) (interface{}, error) {
	// Implementation would run browser automation
	return map[string]string{"result": "automation completed"}, nil
}

// CreateKey generates a new API key.
func (e *APIServiceEngine) CreateKey(ctx context.Context, owner string) (*APIKey, error) {
	key := generateAPIKey()
	apiKey := &APIKey{
		Key:       key,
		Owner:     owner,
		CreatedAt: time.Now(),
		Enabled:   true,
	}
	e.keys[key] = apiKey
	slog.Info("apikey: created", "owner", owner)
	return apiKey, nil
}

// ValidateKey checks if an API key is valid.
func (e *APIServiceEngine) ValidateKey(ctx context.Context, key string) (*APIKey, error) {
	apiKey, ok := e.keys[key]
	if !ok || !apiKey.Enabled {
		return nil, ErrUnauthorized
	}
	return apiKey, nil
}

// RecordUsage records token usage for billing.
func (e *APIServiceEngine) RecordUsage(apiKey string, tokens int) {
	if key, ok := e.keys[apiKey]; ok {
		key.UsageTokens += tokens
	}
}

func generateAPIKey() string {
	h := hmac.New(sha256.New, []byte(time.Now().String()))
	return "wpsk_" + hex.EncodeToString(h.Sum(nil))[:32]
}

// Common API errors
var (
	ErrUnauthorized = &APIError{Code: 401, Message: "Invalid or disabled API key"}
	ErrRateLimited  = &APIError{Code: 429, Message: "Rate limit exceeded"}
	ErrNotFound     = &APIError{Code: 404, Message: "Endpoint not found"}
)

// APIError represents an API error response.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return e.Message
}
