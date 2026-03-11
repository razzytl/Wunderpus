package constants

import (
	"strings"
	"testing"
	"time"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestDefaultTimeouts(t *testing.T) {
	tests := []struct {
		name     string
		got      time.Duration
		minValue time.Duration
		maxValue time.Duration
	}{
		{"DefaultTimeout", DefaultTimeout, 1 * time.Second, 10 * time.Minute},
		{"DefaultShortTimeout", DefaultShortTimeout, 1 * time.Second, 5 * time.Minute},
		{"DefaultLongTimeout", DefaultLongTimeout, 1 * time.Minute, 30 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got < tt.minValue || tt.got > tt.maxValue {
				t.Errorf("%s = %v, expected between %v and %v", tt.name, tt.got, tt.minValue, tt.maxValue)
			}
		})
	}
}

func TestDefaultTimeoutRelation(t *testing.T) {
	if DefaultShortTimeout >= DefaultTimeout {
		t.Errorf("DefaultShortTimeout (%v) should be less than DefaultTimeout (%v)", DefaultShortTimeout, DefaultTimeout)
	}
	if DefaultTimeout >= DefaultLongTimeout {
		t.Errorf("DefaultTimeout (%v) should be less than DefaultLongTimeout (%v)", DefaultTimeout, DefaultLongTimeout)
	}
}

func TestContextLimits(t *testing.T) {
	tests := []struct {
		name string
		got  int
		min  int
		max  int
	}{
		{"MaxInputLength", MaxInputLength, 1000, 100000},
		{"MaxResponseLength", MaxResponseLength, 1000, 1000000},
		{"DefaultMaxHistory", DefaultMaxHistory, 1, 1000},
		{"MaxIterations", MaxIterations, 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got < tt.min || tt.got > tt.max {
				t.Errorf("%s = %d, expected between %d and %d", tt.name, tt.got, tt.min, tt.max)
			}
		})
	}
}

func TestModelNames(t *testing.T) {
	models := []string{
		ModelGPT4o,
		ModelGPT4v,
		ModelClaude3Opus,
		ModelClaude3Sonnet,
		ModelGeminiPro,
		ModelLlama3,
	}

	for _, m := range models {
		if m == "" {
			t.Error("Model name should not be empty")
		}
	}
}

func TestModelNamesFormat(t *testing.T) {
	// GPT models should contain "gpt"
	if len(ModelGPT4o) > 0 && !strings.HasPrefix(ModelGPT4o, "gpt") {
		t.Errorf("ModelGPT4o should start with 'gpt', got %s", ModelGPT4o)
	}
	if len(ModelGPT4v) > 0 && !strings.HasPrefix(ModelGPT4v, "gpt") {
		t.Errorf("ModelGPT4v should start with 'gpt', got %s", ModelGPT4v)
	}

	// Claude models should contain "claude"
	if len(ModelClaude3Opus) > 0 && !strings.HasPrefix(ModelClaude3Opus, "claude") {
		t.Errorf("ModelClaude3Opus should start with 'claude', got %s", ModelClaude3Opus)
	}

	// Gemini should contain "gemini"
	if len(ModelGeminiPro) > 0 && !strings.Contains(ModelGeminiPro, "gemini") {
		t.Errorf("ModelGeminiPro should contain 'gemini', got %s", ModelGeminiPro)
	}

	// Llama should contain "llama"
	if len(ModelLlama3) > 0 && !strings.Contains(ModelLlama3, "llama") {
		t.Errorf("ModelLlama3 should contain 'llama', got %s", ModelLlama3)
	}
}
