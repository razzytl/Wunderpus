# Configuration Reference

Complete reference for `config.yaml`.

## Root Structure

```yaml
model_list: []          # Recommended: vendor-agnostic model list
default_provider: ""    # Legacy: default provider name
providers: {}           # Legacy: provider-specific config
stats: {}               # Cost and usage tracking
agents: {}              # Agent defaults and workspace
agent: {}               # Agent behavior settings
tools: {}               # Tool system configuration
security: {}            # Security settings
logging: {}             # Logging configuration
server: {}              # Health server settings
channels: {}            # Channel integrations
heartbeat: {}           # Periodic task scheduler
genesis: {}             # Autonomous system flags
```

## Model List (Recommended)

```yaml
model_list:
  - model_name: "primary"           # Friendly name
    model: "openai/gpt-4o"          # protocol/model-id
    api_key: "sk-..."               # API key
    api_base: "https://..."         # Custom API base (optional)
    max_tokens: 4096                # Max response tokens
    temperature: 0.7                # Sampling temperature (0-2)
    top_p: 0.9                      # Nucleus sampling
    timeout: 120s                   # Request timeout
    fallback_models:                # Fallback chain
      - "anthropic/claude-sonnet-4-20250514"
      - "openrouter/deepseek/deepseek-r1"
    weight: 1                       # Load balancing weight
```

### Protocol Prefixes

| Prefix | Protocol | Example |
|---|---|---|
| `openai/` | OpenAI-compatible | `openai/gpt-4o` |
| `anthropic/` | Anthropic | `anthropic/claude-sonnet-4-20250514` |
| `ollama/` | Ollama | `ollama/llama3.2` |
| `gemini/` | Google Gemini | `gemini/gemini-2.0-flash` |

## Legacy Providers

```yaml
default_provider: "openai"

providers:
  openai:
    api_key: "sk-..."
    model: "gpt-4o"
    max_tokens: 4096

  anthropic:
    api_key: "sk-ant-..."
    model: "claude-sonnet-4-20250514"
    max_tokens: 4096

  ollama:
    host: "http://localhost:11434"
    model: "llama3.2"
    max_tokens: 4096

  gemini:
    api_key: "AIza..."
    model: "gemini-2.0-flash"
    max_tokens: 4096
```

## Stats

```yaml
stats:
  enabled: true  # Enable cost and usage tracking
```

## Agents (Global Defaults)

```yaml
agents:
  defaults:
    workspace: "."                    # Working directory
    restrict_to_workspace: true       # Sandbox file operations
```

## Agent (Behavior)

```yaml
agent:
  system_prompt: "You are Wunderpus..."  # System prompt
  max_context_tokens: 8000               # Max context window
  temperature: 0.7                       # Default temperature
```

## Tools

```yaml
tools:
  enabled: true
  timeout_seconds: 30

  allowed_paths:
    - "."
    - "/tmp"

  shell_whitelist:
    - ls
    - cat
    - echo
    - grep
    - find
    - git

  ssrf_blocklist:
    - "internal.company.com"

  skills:
    enabled: true
    global_skills_path: "~/.wunderpus/skills"
    builtin_skills_path: "./skills"
    registries:
      clawhub:
        enabled: false
        base_url: "https://clawhub.ai"
        auth_token: ""

  search:
    brave_api_key: ""
    tavily_api_key: ""
    perplexity_key: ""
```

## Security

```yaml
security:
  sanitization_enabled: true
  audit_db_path: "wunderpus_audit.db"
```

## Logging

```yaml
logging:
  level: "info"        # debug, info, warn, error
  format: "json"       # json, text
  output: "stderr"     # stderr, stdout, or file path
```

## Server

```yaml
server:
  health_port: 8080
```

## Channels

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"

  discord:
    enabled: true
    bot_token: "${DISCORD_BOT_TOKEN}"

  slack:
    enabled: false
    app_token: ""
    bot_token: ""

  whatsapp:
    enabled: false

  websocket:
    enabled: true
    host: "0.0.0.0"
    port: 9090
```

## Heartbeat

```yaml
heartbeat:
  enabled: true
  interval: 30  # Minutes (minimum: 5)
```

## Genesis

```yaml
genesis:
  # Infrastructure
  toolsynth_enabled: false
  worldmodel_enabled: false
  perception_enabled: false
  swarm_enabled: false

  # Core Autonomy
  rsi_enabled: false
  rsi_firewall_enabled: true
  rsi_self_referential_enabled: false
  rsi_fitness_threshold: 0.05

  ags_enabled: false
  uaa_enabled: false
  ra_enabled: false

  # Trust Budget
  trust_budget_max: 1000
  trust_regen_per_hour: 10

  # Spend Cap
  max_daily_spend_usd: 10.0

  # JWT Reset
  jwt_secret_env: "WUNDERPUS_JWT_SECRET"

  # Database Paths
  audit_log_db_path: ""
  profiler_db_path: ""
  trust_budget_db_path: ""
```

## Environment Variable Overrides

| Variable | Config Path |
|---|---|
| `WUNDERPUS_CONFIG` | Config file path |
| `WUNDERPUS_OPENAI_API_KEY` | `providers.openai.api_key` |
| `WUNDERPUS_ANTHROPIC_API_KEY` | `providers.anthropic.api_key` |
| `WUNDERPUS_GEMINI_API_KEY` | `providers.gemini.api_key` |
| `WUNDERPUS_DEFAULT_PROVIDER` | `default_provider` |
| `WUNDERPUS_OLLAMA_HOST` | `providers.ollama.host` |
| `WUNDERPUS_ENCRYPTION_KEY` | Security encryption key |
| `WUNDERPUS_LOG_LEVEL` | `logging.level` |
| `WUNDERPUS_TELEGRAM_TOKEN` | `channels.telegram.bot_token` |
| `WUNDERPUS_DISCORD_TOKEN` | `channels.discord.bot_token` |
| `WUNDERPUS_HEARTBEAT_ENABLED` | `heartbeat.enabled` |
| `WUNDERPUS_HEARTBEAT_INTERVAL` | `heartbeat.interval` |

## Config Version Migration

Wunderpus auto-migrates between config versions:

| Version | Changes |
|---|---|
| v0 | Legacy provider-only format |
| v1 | Added encryption salt |
| v2 | Added `model_list` support |
| v3 | Current — full Genesis configuration |
