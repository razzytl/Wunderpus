package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("non-existent.yaml")
	if err != nil {
		t.Fatalf("Load should not fail if file doesn't exist, got %v", err)
	}

	if cfg.DefaultProvider != "openai" {
		t.Errorf("expected default provider openai, got %s", cfg.DefaultProvider)
	}
}

func TestApplyEnv(t *testing.T) {
	os.Setenv("WONDERPUS_OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("WONDERPUS_OPENAI_API_KEY")

	cfg := &Config{}
	applyEnv(cfg)

	if cfg.Providers.OpenAI.APIKey != "test-key" {
		t.Errorf("expected API key test-key, got %s", cfg.Providers.OpenAI.APIKey)
	}
}

func TestValidate_Failure(t *testing.T) {
	cfg := &Config{}
	// No providers configured
	err := validate(cfg)
	if err == nil {
		t.Error("expected validation error for missing providers, got nil")
	}
}
