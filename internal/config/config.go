package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wonderpus/wonderpus/internal/errors"
	"github.com/wonderpus/wonderpus/internal/security"
	"github.com/wonderpus/wonderpus/internal/skills"
)

// Config is the root configuration for Wonderpus.
type Config struct {
	Version         int             `yaml:"version,omitempty"` // Config version (auto-managed)
	Home            string          `yaml:"home"`              // Custom data directory (default: ~/.wunderpus)
	Providers       ProvidersConfig `yaml:"providers"`
	DefaultProvider string          `yaml:"default_provider"`
	ModelList       []ModelEntry    `yaml:"model_list"`
	Agents          AgentsConfig    `yaml:"agents"`
	Agent           AgentConfig     `yaml:"agent"`
	Tools           ToolsConfig     `yaml:"tools"`
	Security        SecurityConfig  `yaml:"security"`
	Logging         LoggingConfig   `yaml:"logging"`
	Server          ServerConfig    `yaml:"server"`
	Channels        ChannelsConfig  `yaml:"channels"`
	Heartbeat       HeartbeatConfig `yaml:"heartbeat"`
}

const CurrentConfigVersion = 3

// AgentsConfig holds agent-level defaults including workspace restriction.
type AgentsConfig struct {
	Defaults AgentDefaults `yaml:"defaults"`
}

// AgentDefaults holds default settings for all agents.
type AgentDefaults struct {
	Workspace           string `yaml:"workspace"`             // Workspace directory (default: current dir)
	RestrictToWorkspace bool   `yaml:"restrict_to_workspace"` // Restrict file/shell ops to workspace (default: true)
}

// ModelEntry is a vendor-agnostic model configuration entry.
// It supports any provider via protocol-based routing.
type ModelEntry struct {
	ModelName      string   `yaml:"model_name"` // User-facing alias (e.g. "gpt-5.2")
	Model          string   `yaml:"model"`      // provider/model (e.g. "openai/gpt-5.2")
	APIKey         string   `yaml:"api_key"`
	APIBase        string   `yaml:"api_base"` // Custom endpoint URL
	Protocol       string   `yaml:"protocol"` // "openai", "anthropic", "ollama" (auto-detected if omitted)
	MaxTokens      int      `yaml:"max_tokens"`
	RequestTimeout int      `yaml:"request_timeout"` // seconds (default: 300)
	FallbackModels []string `yaml:"fallback_models"`
	Weight         int      `yaml:"weight"` // For load-balancing (default: 1)
}

// ProvidersConfig holds all LLM provider configurations (legacy format).
type ProvidersConfig struct {
	OpenAI    ProviderEntry `yaml:"openai"`
	Anthropic ProviderEntry `yaml:"anthropic"`
	Ollama    OllamaEntry   `yaml:"ollama"`
	Gemini    ProviderEntry `yaml:"gemini"`
}

// ProviderEntry is a generic API-key-based provider config.
type ProviderEntry struct {
	APIKey         string   `yaml:"api_key"`
	Model          string   `yaml:"model"`
	MaxTokens      int      `yaml:"max_tokens"`
	Endpoint       string   `yaml:"endpoint,omitempty"`
	FallbackModels []string `yaml:"fallback_models,omitempty"`
}

// OllamaEntry is config for the local Ollama provider.
type OllamaEntry struct {
	Host           string   `yaml:"host"`
	Model          string   `yaml:"model"`
	MaxTokens      int      `yaml:"max_tokens"`
	FallbackModels []string `yaml:"fallback_models,omitempty"`
}

// AgentConfig holds agent behavior settings.
type AgentConfig struct {
	SystemPrompt     string  `yaml:"system_prompt"`
	MaxContextTokens int     `yaml:"max_context_tokens"`
	Temperature      float64 `yaml:"temperature"`
	MemoryDBPath     string  `yaml:"memory_db_path"`
	Budget           float64 `yaml:"budget"` // max cost in dollars
	CostDBPath       string  `yaml:"cost_db_path"`
}

// ToolsConfig holds tool execution, sandbox, and allowlist settings.
type ToolsConfig struct {
	Enabled        bool         `yaml:"enabled"`
	TimeoutSeconds int          `yaml:"timeout_seconds"`
	AllowedPaths   []string     `yaml:"allowed_paths"`
	ShellWhitelist []string     `yaml:"shell_whitelist"`
	SSRFBlocklist  []string     `yaml:"ssrf_blocklist"`
	Skills         SkillsConfig `yaml:"skills"`
}

// SkillsConfig holds configuration for the skills system.
type SkillsConfig struct {
	Enabled           bool                   `yaml:"enabled"`
	GlobalSkillsPath  string                 `yaml:"global_skills_path"`
	BuiltinSkillsPath string                 `yaml:"builtin_skills_path"`
	Registries        SkillsRegistriesConfig `yaml:"registries"`
}

// SkillsRegistriesConfig holds configuration for skill registries.
type SkillsRegistriesConfig struct {
	ClawHub ClawHubRegistryConfig `yaml:"clawhub"`
}

// ClawHubRegistryConfig configures the ClawHub registry.
type ClawHubRegistryConfig struct {
	Enabled   bool   `yaml:"enabled"`
	BaseURL   string `yaml:"base_url"`
	AuthToken string `yaml:"auth_token"`
	Timeout   int    `yaml:"timeout"` // seconds
}

// SecurityConfig holds security settings.
type SecurityConfig struct {
	SanitizationEnabled bool             `yaml:"sanitization_enabled"`
	AuditDBPath         string           `yaml:"audit_db_path"`
	RateLimit           RateLimitConfig  `yaml:"rate_limit"`
	Encryption          EncryptionConfig `yaml:"encryption"`
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	Enabled     bool `yaml:"enabled"`
	MaxRequests int  `yaml:"max_requests"`
	WindowSec   int  `yaml:"window_sec"`
}

// EncryptionConfig holds encryption settings.
type EncryptionConfig struct {
	Enabled bool   `yaml:"enabled"`
	Key     string `yaml:"key"`  // Base64 encoded 32-byte key
	Salt    string `yaml:"salt"` // Base64 encoded 16+ byte salt for key derivation
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

// ChannelsConfig holds configuration for various communication channels.
type ChannelsConfig struct {
	Telegram  TelegramConfig  `yaml:"telegram"`
	Discord   DiscordConfig   `yaml:"discord"`
	WebSocket WebSocketConfig `yaml:"websocket"`
	Slack     SlackConfig     `yaml:"slack"`
	WhatsApp  WhatsAppConfig  `yaml:"whatsapp"`
	Feishu    FeishuConfig    `yaml:"feishu"`
	Line      LineConfig      `yaml:"line"`
}

// HeartbeatConfig holds configuration for the heartbeat (periodic task) system.
type HeartbeatConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"` // in minutes (default: 30, min: 5)
}

// TelegramConfig holds Telegram bot settings.
type TelegramConfig struct {
	Enabled bool   `yaml:"enabled"`
	Token   string `yaml:"token"`
}

// DiscordConfig holds Discord bot settings.
type DiscordConfig struct {
	Enabled bool   `yaml:"enabled"`
	Token   string `yaml:"token"`
	GuildID string `yaml:"guild_id"`
}

// WebSocketConfig holds WebSocket server settings.
type WebSocketConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

// SlackConfig holds Slack bot settings.
type SlackConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Token         string `yaml:"token"`
	AppToken      string `yaml:"app_token"`
	SigningSecret string `yaml:"signing_secret"`
	SocketMode    bool   `yaml:"socket_mode"`
}

// WhatsAppConfig holds WhatsApp integration settings.
type WhatsAppConfig struct {
	Enabled     bool   `yaml:"enabled"`
	SessionPath string `yaml:"session_path"`
}

// FeishuConfig holds Feishu (Lark) bot settings.
type FeishuConfig struct {
	Enabled           bool   `yaml:"enabled"`
	AppID             string `yaml:"app_id"`
	AppSecret         string `yaml:"app_secret"`
	EncryptKey        string `yaml:"encrypt_key"`
	VerificationToken string `yaml:"verification_token"`
}

// LineConfig holds LINE bot settings.
type LineConfig struct {
	Enabled       bool   `yaml:"enabled"`
	ChannelSecret string `yaml:"channel_secret"`
	AccessToken   string `yaml:"access_token"`
}

// Load reads config from a YAML file, then overlays environment variables.
// If no config file exists, it auto-generates one based on environment variables.
func Load(path string) (*Config, error) {
	cfg := &Config{}
	applyDefaults(cfg)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			generated, genErr := tryGenerateConfig(path)
			if genErr == nil && generated {
				if genData, readErr := os.ReadFile(path); readErr == nil {
					data = genData
				}
			}

			if data == nil {
				env := os.Getenv("WONDERPUS_ENV")
				if env != "" {
					altPath := fmt.Sprintf("config.%s.yaml", env)
					if altData, err := os.ReadFile(altPath); err == nil {
						data = altData
						path = altPath
					}
				}

				if data == nil {
					applyEnv(cfg)
					if verr := validate(cfg); verr != nil {
						return nil, verr
					}
					return cfg, nil
				}
			}
		} else {
			return nil, errors.Wrap(errors.ConfigError, "reading config file", err)
		}
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, errors.Wrap(errors.ConfigError, "parsing config YAML", err)
	}

	applyEnv(cfg)

	if err := cfg.decryptSecrets(); err != nil {
		return nil, err
	}

	if cfg.migrate() {
		slog.Info("config migrated", "new_version", CurrentConfigVersion)
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	cfg.resolvePaths()

	return cfg, nil
}

func (c *Config) resolvePaths() {
	if c.Home == "" {
		homeDir, _ := os.UserHomeDir()
		c.Home = filepath.Join(homeDir, ".wunderpus")
	}

	c.Agent.MemoryDBPath = resolveDBPath(c.Agent.MemoryDBPath, c.Home, "wonderpus_memory.db")
	c.Agent.CostDBPath = resolveDBPath(c.Agent.CostDBPath, c.Home, "wonderpus_cost.db")
	c.Security.AuditDBPath = resolveDBPath(c.Security.AuditDBPath, c.Home, "wunderpus_audit.db")
	c.Tools.Skills.GlobalSkillsPath = resolveDBPath(c.Tools.Skills.GlobalSkillsPath, c.Home, "skills")
}

func resolveDBPath(path, home, defaultFile string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if path == "" || path == defaultFile {
		return filepath.Join(home, defaultFile)
	}
	return path
}

func (c *Config) migrate() bool {
	migrated := false

	if c.Version == 0 {
		migrated = c.migrateV0toV1() || migrated
	}
	if c.Version == 1 {
		migrated = c.migrateV1toV2() || migrated
	}
	if c.Version == 2 {
		migrated = c.migrateV2toV3() || migrated
	}

	if migrated {
		c.Version = CurrentConfigVersion
	}

	return migrated
}

func (c *Config) migrateV0toV1() bool {
	if c.Providers.OpenAI.APIKey != "" && len(c.ModelList) == 0 {
		c.ModelList = append(c.ModelList, ModelEntry{
			ModelName: c.Providers.OpenAI.Model,
			Model:     "openai/" + c.Providers.OpenAI.Model,
			APIKey:    c.Providers.OpenAI.APIKey,
			APIBase:   "https://api.openai.com/v1",
			Protocol:  "openai",
			MaxTokens: c.Providers.OpenAI.MaxTokens,
		})
		return true
	}
	return false
}

func (c *Config) migrateV1toV2() bool {
	migrated := false

	if c.Agents.Defaults.Workspace != "" && c.Agents.Defaults.RestrictToWorkspace && c.Security.SanitizationEnabled {
		migrated = true
	}

	return migrated
}

func (c *Config) migrateV2toV3() bool {
	migrated := false

	// Migration for encryption salt: generate a new salt if encryption is enabled but no salt exists
	if c.Security.Encryption.Enabled && c.Security.Encryption.Key != "" && c.Security.Encryption.Salt == "" {
		// Generate a new random salt for key derivation
		newSalt, err := security.GenerateSaltString()
		if err == nil {
			c.Security.Encryption.Salt = newSalt
			migrated = true
		}
	}

	return migrated
}

func tryGenerateConfig(path string) (bool, error) {
	hasOpenAI := os.Getenv("WONDERPUS_OPENAI_API_KEY") != ""
	hasAnthropic := os.Getenv("WONDERPUS_ANTHROPIC_API_KEY") != ""
	hasGemini := os.Getenv("WONDERPUS_GEMINI_API_KEY") != ""
	hasOllama := os.Getenv("WONDERPUS_OLLAMA_HOST") != ""

	if !hasOpenAI && !hasAnthropic && !hasGemini && !hasOllama {
		return false, nil
	}

	cfg := &Config{}
	applyDefaults(cfg)
	applyEnv(cfg)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return false, err
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return false, err
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Config) decryptSecrets() error {
	if !c.Security.Encryption.Enabled || c.Security.Encryption.Key == "" {
		return nil
	}

	key, err := base64.StdEncoding.DecodeString(c.Security.Encryption.Key)
	if err != nil {
		return errors.Wrap(errors.ConfigError, "invalid encryption key (must be base64)", err)
	}

	decrypt := func(val *string, name string) {
		if *val == "" || !strings.HasPrefix(*val, "enc:") {
			return
		}
		cipherText := strings.TrimPrefix(*val, "enc:")
		plain, err := security.Decrypt(cipherText, key)
		if err != nil {
			return
		}
		*val = plain
	}

	// Legacy providers
	decrypt(&c.Providers.OpenAI.APIKey, "openai")
	decrypt(&c.Providers.Anthropic.APIKey, "anthropic")
	decrypt(&c.Providers.Gemini.APIKey, "gemini")
	decrypt(&c.Channels.Telegram.Token, "telegram")
	decrypt(&c.Channels.Discord.Token, "discord")

	// model_list entries
	for i := range c.ModelList {
		decrypt(&c.ModelList[i].APIKey, c.ModelList[i].ModelName)
	}

	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.Home == "" {
		homeDir, _ := os.UserHomeDir()
		cfg.Home = filepath.Join(homeDir, ".wunderpus")
	}

	cfg.DefaultProvider = "openai"
	cfg.Agent.SystemPrompt = "You are Wonderpus, a helpful AI assistant. Be concise and accurate. You have access to tools; use them when necessary to fulfill the user's request."
	cfg.Agent.MaxContextTokens = 8000
	cfg.Agent.Temperature = 0.7
	cfg.Agent.MemoryDBPath = filepath.Join(cfg.Home, "wonderpus_memory.db")
	cfg.Agent.Budget = 10.00 // Default $10 budget
	cfg.Agent.CostDBPath = filepath.Join(cfg.Home, "wonderpus_cost.db")
	cfg.Tools.Enabled = true
	cfg.Tools.TimeoutSeconds = 30
	cfg.Tools.AllowedPaths = []string{"."}
	cfg.Tools.ShellWhitelist = []string{
		"ls", "dir", "cat", "type", "echo", "head", "tail",
		"wc", "grep", "find", "pwd", "date", "whoami",
	}
	cfg.Security.SanitizationEnabled = true
	cfg.Security.AuditDBPath = filepath.Join(cfg.Home, "wunderpus_audit.db")
	cfg.Security.RateLimit.Enabled = true
	cfg.Security.RateLimit.MaxRequests = 10
	cfg.Security.RateLimit.WindowSec = 60
	cfg.Security.Encryption.Enabled = false

	// Skills system defaults
	cfg.Tools.Skills.Enabled = true
	cfg.Tools.Skills.GlobalSkillsPath = filepath.Join(cfg.Home, "skills")
	cfg.Tools.Skills.BuiltinSkillsPath = "./skills"
	cfg.Tools.Skills.Registries.ClawHub.Enabled = false
	cfg.Tools.Skills.Registries.ClawHub.BaseURL = "https://clawhub.ai"
	cfg.Tools.Skills.Registries.ClawHub.Timeout = 30
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"
	cfg.Logging.Output = "stderr"
	cfg.Server.HealthPort = 8080

	cfg.Channels.Telegram.Enabled = false
	cfg.Channels.Discord.Enabled = false
	cfg.Channels.WebSocket.Enabled = false
	cfg.Channels.WebSocket.Port = 9090

	cfg.Channels.Slack.Enabled = false
	cfg.Channels.Slack.SocketMode = true // Default to Socket Mode for easier dev

	cfg.Channels.WhatsApp.Enabled = false
	cfg.Channels.WhatsApp.SessionPath = "whatsapp_session.db"

	cfg.Channels.Feishu.Enabled = false
	cfg.Channels.Line.Enabled = false

	// Heartbeat defaults
	cfg.Heartbeat.Enabled = true
	cfg.Heartbeat.Interval = 30 // 30 minutes

	// Agent defaults — workspace restriction
	cfg.Agents.Defaults.Workspace = "."
	cfg.Agents.Defaults.RestrictToWorkspace = true

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
	if v := os.Getenv("WUNDERPUS_HOME"); v != "" {
		cfg.Home = v
	}
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
	if v := os.Getenv("WONDERPUS_ENCRYPTION_KEY"); v != "" {
		cfg.Security.Encryption.Key = v
		cfg.Security.Encryption.Enabled = true
		// Auto-generate salt if not set
		if cfg.Security.Encryption.Salt == "" {
			if newSalt, err := security.GenerateSaltString(); err == nil {
				cfg.Security.Encryption.Salt = newSalt
			}
		}
	}

	// Workspace restriction override
	if v := os.Getenv("WUNDERPUS_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"); v != "" {
		cfg.Agents.Defaults.RestrictToWorkspace = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("WUNDERPUS_AGENTS_DEFAULTS_WORKSPACE"); v != "" {
		cfg.Agents.Defaults.Workspace = v
	}

	// Heartbeat overrides
	if v := os.Getenv("WUNDERPUS_HEARTBEAT_ENABLED"); v != "" {
		cfg.Heartbeat.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("WUNDERPUS_HEARTBEAT_INTERVAL"); v != "" {
		if interval, err := strconv.Atoi(v); err == nil && interval >= 5 {
			cfg.Heartbeat.Interval = interval
		}
	}

	// Channel Overrides
	if v := os.Getenv("WONDERPUS_SLACK_TOKEN"); v != "" {
		cfg.Channels.Slack.Token = v
	}
	if v := os.Getenv("WONDERPUS_SLACK_APP_TOKEN"); v != "" {
		cfg.Channels.Slack.AppToken = v
	}
	if v := os.Getenv("WONDERPUS_WHATSAPP_ENABLED"); v != "" {
		cfg.Channels.WhatsApp.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
}

func validate(cfg *Config) error {
	// At least one provider must be configured (via model_list or legacy providers)
	hasLegacyProvider := cfg.Providers.OpenAI.APIKey != "" ||
		cfg.Providers.Anthropic.APIKey != "" ||
		cfg.Providers.Ollama.Host != "" ||
		cfg.Providers.Gemini.APIKey != ""

	hasModelList := len(cfg.ModelList) > 0

	if !hasLegacyProvider && !hasModelList {
		return errors.New(errors.ConfigError, "at least one LLM provider must be configured (set API key, Ollama host, or add model_list entries)")
	}

	// Validate model_list entries
	for i, entry := range cfg.ModelList {
		if entry.ModelName == "" {
			return errors.New(errors.ConfigError, fmt.Sprintf("model_list[%d]: model_name is required", i))
		}
		if entry.Model == "" {
			return errors.New(errors.ConfigError, fmt.Sprintf("model_list[%d] (%s): model is required", i, entry.ModelName))
		}
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(cfg.Logging.Level)] {
		return errors.New(errors.ConfigError, fmt.Sprintf("invalid log level %q (use debug, info, warn, error)", cfg.Logging.Level))
	}

	if cfg.Server.HealthPort < 1 || cfg.Server.HealthPort > 65535 {
		return errors.New(errors.ConfigError, fmt.Sprintf("health_port must be 1-65535, got %d", cfg.Server.HealthPort))
	}

	// Validate heartbeat interval
	if cfg.Heartbeat.Interval < 5 {
		return errors.New(errors.ConfigError, fmt.Sprintf("heartbeat interval must be at least 5 minutes, got %d", cfg.Heartbeat.Interval))
	}

	return nil
}

// AvailableProviders returns names of providers that have keys/hosts configured.
func (c *Config) AvailableProviders() []string {
	seen := make(map[string]bool)
	var out []string

	// model_list entries first
	for _, entry := range c.ModelList {
		protocol := entry.DetectProtocol()
		if !seen[protocol] {
			seen[protocol] = true
			out = append(out, protocol)
		}
	}

	// Legacy providers
	if c.Providers.OpenAI.APIKey != "" && !seen["openai"] {
		out = append(out, "openai")
	}
	if c.Providers.Anthropic.APIKey != "" && !seen["anthropic"] {
		out = append(out, "anthropic")
	}
	if c.Providers.Ollama.Host != "" && !seen["ollama"] {
		out = append(out, "ollama")
	}
	if c.Providers.Gemini.APIKey != "" && !seen["gemini"] {
		out = append(out, "gemini")
	}
	return out
}

// DetectProtocol determines the protocol from the Model field prefix or explicit Protocol.
func (e *ModelEntry) DetectProtocol() string {
	if e.Protocol != "" {
		return strings.ToLower(e.Protocol)
	}
	// Auto-detect from model prefix: "openai/gpt-4" → "openai"
	if idx := strings.Index(e.Model, "/"); idx > 0 {
		prefix := strings.ToLower(e.Model[:idx])
		switch prefix {
		case "openai", "groq", "openrouter", "zhipu", "vllm", "together", "mistral", "nvidia", "deepseek":
			return "openai" // All OpenAI-compatible
		case "anthropic", "claude":
			return "anthropic"
		case "ollama":
			return "ollama"
		case "gemini", "google":
			return "gemini"
		default:
			return "openai" // Default to OpenAI-compatible for unknown prefixes
		}
	}
	return "openai" // Default
}

// ModelID extracts the model identifier without the provider prefix.
func (e *ModelEntry) ModelID() string {
	if idx := strings.Index(e.Model, "/"); idx > 0 {
		return e.Model[idx+1:]
	}
	return e.Model
}

// ConvertLegacyToModelList converts legacy ProvidersConfig to ModelEntry list.
func ConvertLegacyToModelList(cfg *Config) []ModelEntry {
	var entries []ModelEntry

	if cfg.Providers.OpenAI.APIKey != "" {
		entry := ModelEntry{
			ModelName:      cfg.Providers.OpenAI.Model,
			Model:          "openai/" + cfg.Providers.OpenAI.Model,
			APIKey:         cfg.Providers.OpenAI.APIKey,
			APIBase:        cfg.Providers.OpenAI.Endpoint,
			Protocol:       "openai",
			MaxTokens:      cfg.Providers.OpenAI.MaxTokens,
			FallbackModels: cfg.Providers.OpenAI.FallbackModels,
			Weight:         1,
		}
		if entry.APIBase == "" {
			entry.APIBase = "https://api.openai.com/v1"
		}
		entries = append(entries, entry)
	}

	if cfg.Providers.Anthropic.APIKey != "" {
		entry := ModelEntry{
			ModelName:      cfg.Providers.Anthropic.Model,
			Model:          "anthropic/" + cfg.Providers.Anthropic.Model,
			APIKey:         cfg.Providers.Anthropic.APIKey,
			APIBase:        cfg.Providers.Anthropic.Endpoint,
			Protocol:       "anthropic",
			MaxTokens:      cfg.Providers.Anthropic.MaxTokens,
			FallbackModels: cfg.Providers.Anthropic.FallbackModels,
			Weight:         1,
		}
		if entry.APIBase == "" {
			entry.APIBase = "https://api.anthropic.com"
		}
		entries = append(entries, entry)
	}

	if cfg.Providers.Ollama.Host != "" {
		entries = append(entries, ModelEntry{
			ModelName:      cfg.Providers.Ollama.Model,
			Model:          "ollama/" + cfg.Providers.Ollama.Model,
			APIBase:        cfg.Providers.Ollama.Host,
			Protocol:       "ollama",
			MaxTokens:      cfg.Providers.Ollama.MaxTokens,
			FallbackModels: cfg.Providers.Ollama.FallbackModels,
			Weight:         1,
		})
	}

	if cfg.Providers.Gemini.APIKey != "" {
		entry := ModelEntry{
			ModelName:      cfg.Providers.Gemini.Model,
			Model:          "gemini/" + cfg.Providers.Gemini.Model,
			APIKey:         cfg.Providers.Gemini.APIKey,
			APIBase:        cfg.Providers.Gemini.Endpoint,
			Protocol:       "gemini",
			MaxTokens:      cfg.Providers.Gemini.MaxTokens,
			FallbackModels: cfg.Providers.Gemini.FallbackModels,
			Weight:         1,
		}
		entries = append(entries, entry)
	}

	return entries
}

// ToSkillsRegistryConfig converts the config to skills.RegistryConfig for the skills system.
func (c *Config) ToSkillsRegistryConfig() skills.RegistryConfig {
	return skills.RegistryConfig{
		ClawHub: skills.ClawHubConfig{
			Enabled:         c.Tools.Skills.Registries.ClawHub.Enabled,
			BaseURL:         c.Tools.Skills.Registries.ClawHub.BaseURL,
			AuthToken:       c.Tools.Skills.Registries.ClawHub.AuthToken,
			SearchPath:      "/api/v1/search",
			SkillsPath:      "/api/v1/skills",
			DownloadPath:    "/api/v1/download",
			Timeout:         c.Tools.Skills.Registries.ClawHub.Timeout,
			MaxZipSize:      50 * 1024 * 1024, // 50 MB
			MaxResponseSize: 2 * 1024 * 1024,  // 2 MB
		},
	}
}
