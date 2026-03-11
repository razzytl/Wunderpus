package provider

import (
	"fmt"
	"strings"

	"github.com/wunderpus/wunderpus/internal/config"
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

	provider := extractProviderPrefix(entry.Model)

	switch protocol {
	case "openai":
		if entry.APIKey == "" {
			return nil, fmt.Errorf("model %q (openai protocol): api_key is required", entry.ModelName)
		}
		baseURL := entry.APIBase
		if baseURL == "" {
			baseURL = ProviderAPIBase(provider)
			if baseURL == "" {
				baseURL = "https://api.openai.com/v1"
			}
		}
		return NewOpenAICompatible(entry.APIKey, modelID, maxTokens, baseURL, entry.ModelName), nil

	case "anthropic":
		if entry.APIKey == "" {
			return nil, fmt.Errorf("model %q (anthropic protocol): api_key is required", entry.ModelName)
		}
		baseURL := entry.APIBase
		if baseURL == "" {
			baseURL = ProviderAPIBase(provider)
			if baseURL == "" {
				baseURL = "https://api.anthropic.com"
			}
		}
		return NewAnthropicCompatible(entry.APIKey, modelID, maxTokens, baseURL, entry.ModelName), nil

	case "ollama":
		host := entry.APIBase
		if host == "" {
			host = ProviderAPIBase(provider)
			if host == "" {
				host = "http://localhost:11434"
			}
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

// extractProviderPrefix extracts the provider prefix from a model string (e.g., "openai/gpt-4" -> "openai")
func extractProviderPrefix(model string) string {
	if idx := strings.Index(model, "/"); idx > 0 {
		return strings.ToLower(model[:idx])
	}
	return ""
}

// DetectProtocolFromModel determines the protocol from a model string prefix.
// This is a convenience function for external use.
func DetectProtocolFromModel(model string) string {
	if idx := strings.Index(model, "/"); idx > 0 {
		prefix := strings.ToLower(model[:idx])
		switch prefix {
		case "openai", "groq", "openrouter", "zhipu", "vllm", "together", "mistral", "nvidia", "deepseek",
			"moonshot", "cerebras", "litellm", "qwen", "volcanic":
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

// ProviderAPIBase returns the default API base URL for a provider.
func ProviderAPIBase(provider string) string {
	switch strings.ToLower(provider) {
	case "openai":
		return "https://api.openai.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "zhipu", "glm":
		return "https://open.bigmodel.cn/api/paas/v4"
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "moonshot":
		return "https://api.moonshot.cn/v1"
	case "cerebras":
		return "https://api.cerebras.ai/v1"
	case "nvidia":
		return "https://integrate.api.nvidia.com/v1"
	case "litellm":
		return "http://localhost:4000/v1"
	case "vllm":
		return "http://localhost:8000/v1"
	case "mistral":
		return "https://api.mistral.ai/v1"
	case "volcanic":
		return "https://ark.cn-beijing.volces.com/api/v3"
	case "qwen":
		return "https://dashscope.aliyuncs.com/api/v1"
	case "ollama":
		return "http://localhost:11434"
	case "anthropic":
		return "https://api.anthropic.com"
	case "gemini", "google":
		return "https://generativelanguage.googleapis.com/v1beta"
	default:
		return ""
	}
}
