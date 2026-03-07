package provider

import (
	"fmt"
	"strings"

	"github.com/wonderpus/wonderpus/internal/config"
)

// NewFromModelEntry creates a Provider instance from a ModelEntry using protocol-based routing.
// This enables adding new providers with zero code changes — just config.
func NewFromModelEntry(entry config.ModelEntry) (Provider, error) {
	protocol := entry.DetectProtocol()
	modelID := entry.ModelID()
	maxTokens := entry.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	switch protocol {
	case "openai":
		if entry.APIKey == "" {
			return nil, fmt.Errorf("model %q (openai protocol): api_key is required", entry.ModelName)
		}
		baseURL := entry.APIBase
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return NewOpenAICompatible(entry.APIKey, modelID, maxTokens, baseURL, entry.ModelName), nil

	case "anthropic":
		if entry.APIKey == "" {
			return nil, fmt.Errorf("model %q (anthropic protocol): api_key is required", entry.ModelName)
		}
		baseURL := entry.APIBase
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		return NewAnthropicCompatible(entry.APIKey, modelID, maxTokens, baseURL, entry.ModelName), nil

	case "ollama":
		host := entry.APIBase
		if host == "" {
			host = "http://localhost:11434"
		}
		return NewOllamaCompatible(host, modelID, maxTokens, entry.ModelName), nil

	case "gemini":
		if entry.APIKey == "" {
			return nil, fmt.Errorf("model %q (gemini protocol): api_key is required", entry.ModelName)
		}
		return NewGeminiCompatible(entry.APIKey, modelID, maxTokens, entry.ModelName), nil

	default:
		return nil, fmt.Errorf("unsupported protocol %q for model %q (supported: openai, anthropic, ollama, gemini)",
			protocol, entry.ModelName)
	}
}

// DetectProtocolFromModel determines the protocol from a model string prefix.
// This is a convenience function for external use.
func DetectProtocolFromModel(model string) string {
	if idx := strings.Index(model, "/"); idx > 0 {
		prefix := strings.ToLower(model[:idx])
		switch prefix {
		case "openai", "groq", "openrouter", "zhipu", "vllm", "together", "mistral", "nvidia", "deepseek":
			return "openai"
		case "anthropic", "claude":
			return "anthropic"
		case "ollama":
			return "ollama"
		case "gemini", "google":
			return "gemini"
		}
	}
	return "openai"
}
