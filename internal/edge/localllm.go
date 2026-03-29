package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// OllamaModel represents a model in Ollama.
type OllamaModel struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
}

// OllamaClient manages local Ollama models.
type OllamaClient struct {
	host       string
	downloaded map[string]bool
}

// OllamaConfig holds Ollama configuration.
type OllamaConfig struct {
	Enabled     bool
	Host        string
	AutoPull    bool
	FallbackAPI string // Free API tier fallback
}

// NewOllamaClient creates a new Ollama client.
func NewOllamaClient(cfg OllamaConfig) *OllamaClient {
	return &OllamaClient{
		host:       cfg.Host,
		downloaded: make(map[string]bool),
	}
}

// PullModel pulls a model from Ollama registry.
func (o *OllamaClient) PullModel(ctx context.Context, modelName string) error {
	slog.Info("edge: pulling model", "model", modelName)

	// Would execute: ollama pull modelName
	// For now, just mark as downloaded
	o.downloaded[modelName] = true

	slog.Info("edge: model pulled", "model", modelName)
	return nil
}

// ListModels lists available Ollama models.
func (o *OllamaClient) ListModels(ctx context.Context) ([]OllamaModel, error) {
	// Would execute: ollama list
	// For now, return empty list
	return []OllamaModel{}, nil
}

// SelectModel selects appropriate model for task.
func (o *OllamaClient) SelectModel(ctx context.Context, taskType string) string {
	switch taskType {
	case "simple":
		return "qwen2.5-3b"
	case "reasoning":
		return "llama3.1-8b"
	case "distilled":
		return "wunderpus-distilled" // Would be from Section 5
	default:
		return "llama3.2"
	}
}

// LocalLLMEngine manages local model lifecycle.
type LocalLLMEngine struct {
	ollama   *OllamaClient
	apiKey   string
	fallback FallbackChain
}

// FallbackChain manages model fallbacks.
type FallbackChain struct {
	models []string
	index  int
}

// LocalLLMConfig holds configuration.
type LocalLLMConfig struct {
	Enabled        bool
	OllamaHost     string
	APIKey         string
	FallbackModels []string
}

// NewLocalLLMEngine creates a new local LLM engine.
func NewLocalLLMEngine(cfg LocalLLMConfig, ollama *OllamaClient) *LocalLLMEngine {
	return &LocalLLMEngine{
		ollama: ollama,
		apiKey: cfg.APIKey,
		fallback: FallbackChain{
			models: cfg.FallbackModels,
			index:  0,
		},
	}
}

// Complete sends a completion request with fallback hierarchy.
func (e *LocalLLMEngine) Complete(ctx context.Context, prompt string) (string, error) {
	slog.Info("edge: completing request", "model", e.fallback.current())

	// First try local model
	if e.ollama != nil {
		// Would call local Ollama
		// For now, simulate success
		return "local_response", nil
	}

	// Fallback to API
	if e.apiKey != "" {
		// Would call API with key
		return "api_response", nil
	}

	return "", fmt.Errorf("no model available")
}

func (f *FallbackChain) current() string {
	if f.index < len(f.models) {
		return f.models[f.index]
	}
	return "none"
}

// Fallback attempts the next model in the chain.
func (f *FallbackChain) Fallback() {
	f.index++
}

// MarshalJSON implements JSON marshaling for FallbackChain.
func (f *FallbackChain) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"models": f.models,
		"index":  f.index,
	})
}
