package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration for Wonderpus.
type Config struct {
	Providers       ProvidersConfig `yaml:"providers"`
	DefaultProvider string          `yaml:"default_provider"`
	Agent           AgentConfig     `yaml:"agent"`
	Security        SecurityConfig  `yaml:"security"`
	Logging         LoggingConfig   `yaml:"logging"`
	Server          ServerConfig    `yaml:"server"`
}

// ProvidersConfig holds all LLM provider configurations.
type ProvidersConfig struct {
	OpenAI    ProviderEntry `yaml:"openai"`
	Anthropic ProviderEntry `yaml:"anthropic"`
	Ollama    OllamaEntry   `yaml:"ollama"`
	Gemini    ProviderEntry `yaml:"gemini"`
}

// ProviderEntry is a generic API-key-based provider config.
type ProviderEntry struct {
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

// OllamaEntry is config for the local Ollama provider.
type OllamaEntry struct {
	Host      string `yaml:"host"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

// AgentConfig holds agent behavior settings.
type AgentConfig struct {
	SystemPrompt    string  `yaml:"system_prompt"`
	MaxContextTokens int    `yaml:"max_context_tokens"`
	Temperature     float64 `yaml:"temperature"`
}

// SecurityConfig holds security settings.
type SecurityConfig struct {
	SanitizationEnabled bool   `yaml:"sanitization_enabled"`
	AuditDBPath         string `yaml:"audit_db_path"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	HealthPort int `yaml:"health_port"`
}

// Load reads config from a YAML file, then overlays environment variables.
func Load(path string) (*Config, error) {
	cfg := &Config{}
	applyDefaults(cfg)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file — rely on ENV vars only
			applyEnv(cfg)
			if verr := validate(cfg); verr != nil {
				return nil, verr
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyEnv(cfg)

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	cfg.DefaultProvider = "openai"
	cfg.Agent.SystemPrompt = "You are Wonderpus, a helpful AI assistant. Be concise and accurate."
	cfg.Agent.MaxContextTokens = 8000
	cfg.Agent.Temperature = 0.7
	cfg.Security.SanitizationEnabled = true
	cfg.Security.AuditDBPath = "wonderpus_audit.db"
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"
	cfg.Logging.Output = "stderr"
	cfg.Server.HealthPort = 8080
	cfg.Providers.OpenAI.Model = "gpt-4o"
	cfg.Providers.OpenAI.MaxTokens = 4096
	cfg.Providers.Anthropic.Model = "claude-sonnet-4-20250514"
	cfg.Providers.Anthropic.MaxTokens = 4096
	cfg.Providers.Ollama.Host = "http://localhost:11434"
	cfg.Providers.Ollama.Model = "llama3.2"
	cfg.Providers.Ollama.MaxTokens = 4096
	cfg.Providers.Gemini.Model = "gemini-2.0-flash"
	cfg.Providers.Gemini.MaxTokens = 4096
}

// applyEnv overlays environment variables on top of file config.
func applyEnv(cfg *Config) {
	if v := os.Getenv("WONDERPUS_OPENAI_API_KEY"); v != "" {
		cfg.Providers.OpenAI.APIKey = v
	}
	if v := os.Getenv("WONDERPUS_ANTHROPIC_API_KEY"); v != "" {
		cfg.Providers.Anthropic.APIKey = v
	}
	if v := os.Getenv("WONDERPUS_GEMINI_API_KEY"); v != "" {
		cfg.Providers.Gemini.APIKey = v
	}
	if v := os.Getenv("WONDERPUS_OLLAMA_HOST"); v != "" {
		cfg.Providers.Ollama.Host = v
	}
	if v := os.Getenv("WONDERPUS_DEFAULT_PROVIDER"); v != "" {
		cfg.DefaultProvider = v
	}
	if v := os.Getenv("WONDERPUS_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
}

func validate(cfg *Config) error {
	// At least one provider must be configured
	hasProvider := cfg.Providers.OpenAI.APIKey != "" ||
		cfg.Providers.Anthropic.APIKey != "" ||
		cfg.Providers.Ollama.Host != "" ||
		cfg.Providers.Gemini.APIKey != ""

	if !hasProvider {
		return fmt.Errorf("config: at least one LLM provider must be configured (set API key or Ollama host)")
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(cfg.Logging.Level)] {
		return fmt.Errorf("config: invalid log level %q (use debug, info, warn, error)", cfg.Logging.Level)
	}

	if cfg.Server.HealthPort < 1 || cfg.Server.HealthPort > 65535 {
		return fmt.Errorf("config: health_port must be 1-65535, got %d", cfg.Server.HealthPort)
	}

	return nil
}

// AvailableProviders returns names of providers that have keys/hosts configured.
func (c *Config) AvailableProviders() []string {
	var out []string
	if c.Providers.OpenAI.APIKey != "" {
		out = append(out, "openai")
	}
	if c.Providers.Anthropic.APIKey != "" {
		out = append(out, "anthropic")
	}
	if c.Providers.Ollama.Host != "" {
		out = append(out, "ollama")
	}
	if c.Providers.Gemini.APIKey != "" {
		out = append(out, "gemini")
	}
	return out
}
