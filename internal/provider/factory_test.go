package provider

import (
	"testing"

	"github.com/wunderpus/wunderpus/internal/config"
)

// TestNewFromModelEntry_OpenAI tests creating an OpenAI-compatible provider
func TestNewFromModelEntry_OpenAI(t *testing.T) {
	entry := config.ModelEntry{
		Model:  "openai/gpt-4o",
		APIKey: "test-key",
	}

	prov, err := NewFromModelEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestNewFromModelEntry_Anthropic tests creating an Anthropic-compatible provider
func TestNewFromModelEntry_Anthropic(t *testing.T) {
	entry := config.ModelEntry{
		Model:  "anthropic/claude-3-5-sonnet",
		APIKey: "test-key",
	}

	prov, err := NewFromModelEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestNewFromModelEntry_Ollama tests creating an Ollama-compatible provider
func TestNewFromModelEntry_Ollama(t *testing.T) {
	entry := config.ModelEntry{
		Model: "ollama/llama2",
	}

	prov, err := NewFromModelEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestNewFromModelEntry_Gemini tests creating a Gemini-compatible provider
func TestNewFromModelEntry_Gemini(t *testing.T) {
	entry := config.ModelEntry{
		Model:  "gemini/gemini-1.5-pro",
		APIKey: "test-key",
	}

	prov, err := NewFromModelEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestNewFromModelEntry_MissingAPIKey tests error when API key is missing for OpenAI
func TestNewFromModelEntry_MissingAPIKey(t *testing.T) {
	entry := config.ModelEntry{
		Model: "openai/gpt-4o",
	}

	_, err := NewFromModelEntry(entry)
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

// TestNewFromModelEntry_DefaultMaxTokens tests default max tokens when not specified
func TestNewFromModelEntry_DefaultMaxTokens(t *testing.T) {
	entry := config.ModelEntry{
		Model:  "openai/gpt-4o",
		APIKey: "test-key",
	}

	prov, err := NewFromModelEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestNewFromModelEntry_CustomMaxTokens tests custom max tokens
func TestNewFromModelEntry_CustomMaxTokens(t *testing.T) {
	entry := config.ModelEntry{
		Model:     "openai/gpt-4o",
		APIKey:    "test-key",
		MaxTokens: 8192,
	}

	prov, err := NewFromModelEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestNewFromModelEntry_CustomBaseURL tests custom API base URL
func TestNewFromModelEntry_CustomBaseURL(t *testing.T) {
	entry := config.ModelEntry{
		Model:   "openai/gpt-4o",
		APIKey:  "test-key",
		APIBase: "https://custom.api.example.com/v1",
	}

	prov, err := NewFromModelEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestNewFromModelEntry_UnsupportedProtocol tests error for unsupported protocol
func TestNewFromModelEntry_UnsupportedProtocol(t *testing.T) {
	entry := config.ModelEntry{
		Model: "unknown/model",
	}

	_, err := NewFromModelEntry(entry)
	if err == nil {
		t.Error("expected error for unsupported protocol")
	}
}

// TestDetectProtocolFromModel tests protocol detection from model string
func TestDetectProtocolFromModel(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		{"openai/gpt-4", "openai"},
		{"anthropic/claude-3", "anthropic"},
		{"ollama/llama2", "ollama"},
		{"gemini/pro", "gemini"},
		{"google/gemini", "gemini"},
		{"groq/llama", "openai"},
		{"openrouter/llama", "openai"},
		{"deepseek/coder", "openai"},
		{"claude-3", "openai"},    // No prefix defaults to openai
		{"gpt-4", "openai"},       // No prefix defaults to openai
		{"randommodel", "openai"}, // Unknown defaults to openai
	}

	for _, tt := range tests {
		result := DetectProtocolFromModel(tt.model)
		if result != tt.expected {
			t.Errorf("DetectProtocolFromModel(%q) = %q, want %q", tt.model, result, tt.expected)
		}
	}
}

// TestProviderAPIBase tests default API base URLs
func TestProviderAPIBase(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"openai", "https://api.openai.com/v1"},
		{"openrouter", "https://openrouter.ai/api/v1"},
		{"groq", "https://api.groq.com/openai/v1"},
		{"deepseek", "https://api.deepseek.com/v1"},
		{"ollama", "http://localhost:11434"},
		{"anthropic", "https://api.anthropic.com"},
		{"gemini", "https://generativelanguage.googleapis.com/v1beta"},
		{"google", "https://generativelanguage.googleapis.com/v1beta"},
		{"qwen", "https://dashscope.aliyuncs.com/api/v1"},
		{"mistral", "https://api.mistral.ai/v1"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := ProviderAPIBase(tt.provider)
		if result != tt.expected {
			t.Errorf("ProviderAPIBase(%q) = %q, want %q", tt.provider, result, tt.expected)
		}
	}
}

// TestExtractProviderPrefix tests extracting provider prefix
func TestExtractProviderPrefix(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		{"openai/gpt-4", "openai"},
		{"anthropic/claude", "anthropic"},
		{"ollama/llama", "ollama"},
		{"gemini/pro", "gemini"},
		{"gpt-4", ""}, // No prefix
		{"", ""},
	}

	for _, tt := range tests {
		result := extractProviderPrefix(tt.model)
		if result != tt.expected {
			t.Errorf("extractProviderPrefix(%q) = %q, want %q", tt.model, result, tt.expected)
		}
	}
}
