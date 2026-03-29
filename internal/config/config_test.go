package config

import (
	"path/filepath"
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
	t.Setenv("WUNDERPUS_OPENAI_API_KEY", "test-key")

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

func TestValidate_WithModelList(t *testing.T) {
	cfg := &Config{
		ModelList: []ModelEntry{
			{ModelName: "gpt-4", Model: "openai/gpt-4"},
		},
		Logging:   LoggingConfig{Level: "info"},
		Server:    ServerConfig{HealthPort: 8080},
		Heartbeat: HeartbeatConfig{Interval: 30},
	}

	err := validate(cfg)
	if err != nil {
		t.Errorf("validation should pass with model_list, got %v", err)
	}
}

func TestValidate_WithOpenAIKey(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: ProviderEntry{APIKey: "test-key"},
		},
		Logging:   LoggingConfig{Level: "info"},
		Server:    ServerConfig{HealthPort: 8080},
		Heartbeat: HeartbeatConfig{Interval: 30},
	}

	err := validate(cfg)
	if err != nil {
		t.Errorf("validation should pass with OpenAI key, got %v", err)
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: ProviderEntry{APIKey: "test-key"},
		},
		Logging: LoggingConfig{Level: "invalid"},
	}

	err := validate(cfg)
	if err == nil {
		t.Error("expected validation error for invalid log level")
	}
}

func TestValidate_InvalidHealthPort(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: ProviderEntry{APIKey: "test-key"},
		},
		Server: ServerConfig{HealthPort: 0},
	}

	err := validate(cfg)
	if err == nil {
		t.Error("expected validation error for invalid health port")
	}
}

func TestValidate_HeartbeatInterval(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: ProviderEntry{APIKey: "test-key"},
		},
		Heartbeat: HeartbeatConfig{Interval: 3}, // Less than 5
	}

	err := validate(cfg)
	if err == nil {
		t.Error("expected validation error for heartbeat interval < 5")
	}
}

func TestModelEntryDetectProtocol(t *testing.T) {
	tests := []struct {
		model    string
		explicit string
		want     string
	}{
		{"openai/gpt-4", "", "openai"},
		{"anthropic/claude-3", "", "anthropic"},
		{"ollama/llama2", "", "ollama"},
		{"gemini/pro", "", "gemini"},
		{"custom/model", "", "openai"}, // default
		{"custom/model", "anthropic", "anthropic"},
	}

	for _, tt := range tests {
		entry := ModelEntry{Model: tt.model, Protocol: tt.explicit}
		got := entry.DetectProtocol()
		if got != tt.want {
			t.Errorf("DetectProtocol(%s) = %s, want %s", tt.model, got, tt.want)
		}
	}
}

func TestModelEntryModelID(t *testing.T) {
	entry := ModelEntry{Model: "openai/gpt-4"}
	if entry.ModelID() != "gpt-4" {
		t.Errorf("ModelID() = %s, want gpt-4", entry.ModelID())
	}

	entry2 := ModelEntry{Model: "llama2"}
	if entry2.ModelID() != "llama2" {
		t.Errorf("ModelID() = %s, want llama2", entry2.ModelID())
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	// Check some default values
	if cfg.DefaultProvider != "openai" {
		t.Errorf("DefaultProvider = %s, want openai", cfg.DefaultProvider)
	}
	if cfg.Agent.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", cfg.Agent.Temperature)
	}
	if cfg.Agent.MaxContextTokens != 8000 {
		t.Errorf("MaxContextTokens = %d, want 8000", cfg.Agent.MaxContextTokens)
	}
	if !cfg.Tools.Enabled {
		t.Error("Tools.Enabled should be true by default")
	}
	if cfg.Tools.TimeoutSeconds != 30 {
		t.Errorf("TimeoutSeconds = %d, want 30", cfg.Tools.TimeoutSeconds)
	}
	if !cfg.Security.SanitizationEnabled {
		t.Error("SanitizationEnabled should be true by default")
	}
	if !cfg.Security.RateLimit.Enabled {
		t.Error("RateLimit.Enabled should be true by default")
	}
}

func TestApplyEnvMultiple(t *testing.T) {
	// Set multiple env vars
	t.Setenv("WUNDERPUS_OPENAI_API_KEY", "key1")
	t.Setenv("WUNDERPUS_ANTHROPIC_API_KEY", "key2")
	t.Setenv("WUNDERPUS_LOG_LEVEL", "debug")
	t.Setenv("WUNDERPUS_DEFAULT_PROVIDER", "anthropic")

	cfg := &Config{}
	applyEnv(cfg)

	if cfg.Providers.OpenAI.APIKey != "key1" {
		t.Errorf("OpenAI.APIKey = %s, want key1", cfg.Providers.OpenAI.APIKey)
	}
	if cfg.Providers.Anthropic.APIKey != "key2" {
		t.Errorf("Anthropic.APIKey = %s, want key2", cfg.Providers.Anthropic.APIKey)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %s, want debug", cfg.Logging.Level)
	}
	if cfg.DefaultProvider != "anthropic" {
		t.Errorf("DefaultProvider = %s, want anthropic", cfg.DefaultProvider)
	}
}

func TestResolvePaths(t *testing.T) {
	cfg := &Config{
		Home: "/custom/home",
	}
	cfg.resolvePaths()

	if cfg.Home != "/custom/home" {
		t.Errorf("Home = %s, want /custom/home", cfg.Home)
	}
}

func TestResolveDBPath(t *testing.T) {
	// Absolute path should remain unchanged
	path := resolveDBPath("/absolute/path.db", "/home", "default.db")
	if path != "/absolute/path.db" {
		t.Errorf("resolveDBPath absolute = %s, want /absolute/path.db", path)
	}

	// Empty path should use default
	path = resolveDBPath("", "/home", "default.db")
	expected := filepath.Join("/home", "default.db")
	if path != expected {
		t.Errorf("resolveDBPath empty = %s, want %s", path, expected)
	}

	// Default value should use home directory
	path = resolveDBPath("default.db", "/home", "default.db")
	expected = filepath.Join("/home", "default.db")
	if path != expected {
		t.Errorf("resolveDBPath default = %s, want %s", path, expected)
	}
}

func TestAvailableProviders(t *testing.T) {
	cfg := &Config{
		ModelList: []ModelEntry{
			{ModelName: "gpt-4", Model: "openai/gpt-4", APIKey: "key1"},
			{ModelName: "claude-3", Model: "anthropic/claude-3", APIKey: "key2"},
		},
		Providers: ProvidersConfig{
			Ollama: OllamaEntry{Host: "http://localhost:11434"},
		},
	}

	providers := cfg.AvailableProviders()

	// Should contain unique providers
	if len(providers) < 2 {
		t.Errorf("Expected at least 2 providers, got %d", len(providers))
	}
}

// TestGenesisConfig_NewFields verifies the two checklist-required genesis fields
// are present in the struct and receive correct default values.
func TestGenesisConfig_NewFields(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	// RSIFitnessThreshold: default 0.05 per checklist
	if cfg.Genesis.RSIFitnessThreshold != 0.05 {
		t.Errorf("RSIFitnessThreshold default = %v, want 0.05", cfg.Genesis.RSIFitnessThreshold)
	}

	// RSISelfReferentialEnabled: default false per checklist
	if cfg.Genesis.RSISelfReferentialEnabled != false {
		t.Errorf("RSISelfReferentialEnabled default = %v, want false", cfg.Genesis.RSISelfReferentialEnabled)
	}

	// Sanity: trust budget defaults still correct
	if cfg.Genesis.TrustBudgetMax != 1000 {
		t.Errorf("TrustBudgetMax default = %v, want 1000", cfg.Genesis.TrustBudgetMax)
	}
	if cfg.Genesis.TrustRegenPerHour != 10 {
		t.Errorf("TrustRegenPerHour default = %v, want 10", cfg.Genesis.TrustRegenPerHour)
	}
	if cfg.Genesis.MaxDailySpendUSD != 10.0 {
		t.Errorf("MaxDailySpendUSD default = %v, want 10.0", cfg.Genesis.MaxDailySpendUSD)
	}
}
