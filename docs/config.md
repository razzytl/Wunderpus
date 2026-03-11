# Configuration Reference

Wunderpus uses a YAML-based configuration system. This document provides a comprehensive reference for all configuration options.

## Configuration File Location

Wunderpus looks for configuration files in the following order:

1. Path specified by `--config` flag
2. Path specified by `WUNDERPUS_CONFIG` environment variable
3. `./config.yaml` in current directory
4. `~/.wunderpus/config.yaml`

## Configuration File Structure

```yaml
# Model List (recommended format)
model_list:
  - model_name: "identifier"
    model: "provider/model-name"
    api_key: "${ENV_VAR}"
    # ... provider settings

# Legacy providers (for backward compatibility)
default_provider: "openai"
providers:
  openai:
    api_key: "..."
    model: "..."

# Agent settings
agent:
  system_prompt: "..."
  max_context_tokens: 8000
  temperature: 0.7

# Channel configuration
channels:
  telegram:
    enabled: true
    # ...

# Tools configuration
tools:
  enabled: true
  # ...

# Security settings
security:
  encryption:
    enabled: false
  # ...

# Logging
logging:
  level: "info"
  format: "json"

# Server
server:
  health_port: 8080

# Heartbeat
heartbeat:
  enabled: true
  interval: 30
```

## Model List Configuration

The `model_list` is the recommended configuration format. It provides a unified way to configure multiple providers.

### Basic Structure

```yaml
model_list:
  - model_name: "unique-name"
    model: "protocol/model-id"
    api_key: "${ENV_VAR}"
    api_base: "https://api.example.com"
    max_tokens: 4096
    temperature: 0.7
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model_name` | string | Yes | Unique identifier for this model configuration |
| `model` | string | Yes | Provider/model identifier in `provider/model` format |
| `api_key` | string | No | API key (supports environment variables) |
| `api_base` | string | No | Custom API endpoint |
| `max_tokens` | int | No | Maximum tokens in response (default: 4096) |
| `temperature` | float | No | Sampling temperature 0-2 (default: 0.7) |
| `top_p` | float | No | Nucleus sampling (default: 1.0) |
| `timeout` | duration | No | Request timeout (default: 120s) |
| `fallback_models` | []string | No | Fallback model names on failure |
| `headers` | map | No | Custom HTTP headers |

### Provider Protocols

| Protocol | Provider | Example |
|----------|----------|---------|
| `openai` | OpenAI-compatible APIs | openai/gpt-4o |
| `anthropic` | Anthropic Claude | anthropic/claude-sonnet-4 |
| `gemini` | Google Gemini | gemini/gemini-2.0-flash |
| `ollama` | Ollama | ollama/llama3.2 |

### Fallback Configuration

```yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    fallback_models:
      - "claude-sonnet"
      - "deepseek-r1"
```

## Legacy Provider Configuration

For backward compatibility, individual provider configurations are supported.

### OpenAI

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o"
    max_tokens: 4096
    temperature: 0.7
    organization: ""  # Optional: Organization ID
```

### Anthropic

```yaml
providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-sonnet-4-20250514"
    max_tokens: 4096
    temperature: 0.7
```

### Ollama

```yaml
providers:
  ollama:
    host: "http://localhost:11434"
    model: "llama3.2"
    max_tokens: 4096
```

### Gemini

```yaml
providers:
  gemini:
    api_key: "${GEMINI_API_KEY}"
    model: "gemini-2.0-flash"
    max_tokens: 4096
```

### OpenRouter

```yaml
providers:
  openrouter:
    api_key: "${OPENROUTER_API_KEY}"
    model: "deepseek/deepseek-r1"
    max_tokens: 4096
```

### Groq

```yaml
providers:
  groq:
    api_key: "${GROQ_API_KEY}"
    model: "llama-3.3-70b-versatile"
    max_tokens: 4096
```

### DeepSeek

```yaml
providers:
  deepseek:
    api_key: "${DEEPSEEK_API_KEY}"
    model: "deepseek-chat"
    max_tokens: 4096
```

### Cerebras

```yaml
providers:
  cerebras:
    api_key: "${CEREBRAS_API_KEY}"
    model: "llama-3.3-70b"
```

### NVIDIA NIM

```yaml
providers:
  nvidia:
    api_key: "${NVIDIA_API_KEY}"
    model: "nvidia/llama-3.1-nemotron-70b-instruct"
```

### LiteLLM

```yaml
providers:
  litellm:
    endpoint: "http://localhost:4000/v1"
    api_key: "${LITELLM_KEY}"
    model: "gpt-4"
```

### vLLM

```yaml
providers:
  vllm:
    endpoint: "http://localhost:8000/v1"
    api_key: "${VLLM_KEY}"
    model: "llama-3.2"
```

## Agent Configuration

```yaml
agent:
  # System prompt that guides agent behavior
  system_prompt: "You are Wunderpus, a helpful AI assistant."
  
  # Maximum tokens in conversation context
  max_context_tokens: 8000
  
  # Maximum tokens in response
  max_response_tokens: 4096
  
  # Sampling temperature (0.0 - 2.0)
  temperature: 0.7
  
  # Enable parallel provider probing
  parallel_probe: true
  
  # Timeout for probing
  probe_timeout: 5s
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `system_prompt` | string | - | Instructions for the agent |
| `max_context_tokens` | int | 8000 | Token limit for conversation history |
| `max_response_tokens` | int | 4096 | Maximum response length |
| `temperature` | float | 0.7 | Response randomness |
| `parallel_probe` | bool | false | Send to multiple providers |
| `probe_timeout` | duration | 5s | Timeout for probing |

## Channel Configuration

### Telegram

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
    parse_mode: "MarkdownV2"  # Markdown, MarkdownV2, HTML
    allowed_users: []  # User IDs to restrict
    allowed_chats: []  # Chat IDs to restrict
    use_polling: false  # Use long polling instead of webhook
```

### Discord

```yaml
channels:
  discord:
    enabled: true
    bot_token: "${DISCORD_BOT_TOKEN}"
    guild_id: ""  # For guild-specific commands
    channel_id: ""  # Default channel
    prefix: "!"  # Command prefix
```

### WebSocket

```yaml
channels:
  websocket:
    enabled: true
    host: "0.0.0.0"
    port: 8081
    path: "/ws"
    auth_token: "${WS_AUTH_TOKEN}"
    max_connections: 100
```

### QQ

```yaml
channels:
  qq:
    enabled: true
    account: 123456789
```

### WeCom

```yaml
channels:
  wecom:
    enabled: true
    corp_id: "${WECOM_CORP_ID}"
    agent_id: "${WECOM_AGENT_ID}"
    secret: "${WECOM_SECRET}"
    token: ""  # For webhook verification
    encoding_aes_key: ""  # For encrypted messages
```

### DingTalk

```yaml
channels:
  dingtalk:
    enabled: true
    app_id: "${DINGTALK_APP_ID}"
    app_secret: "${DINGTALK_APP_SECRET}"
    token: ""
    encoding_aes_key: ""
```

### OneBot

```yaml
channels:
  onebot:
    enabled: true
    protocol: "http"  # http or websocket
    host: "0.0.0.0"
    port: 8082
```

## Tools Configuration

```yaml
tools:
  enabled: true
  timeout_seconds: 30
  
  # Shell command allowlist
  shell_whitelist:
    - git
    - go
    - npm
    - cargo
    - docker
    - make
  
  # Additional blocked shell patterns
  shell_blocklist:
    - "^rm -rf"
    - "^dd "
  
  # Allowed paths for file operations
  allowed_paths:
    - "."
    - "/tmp"
  
  # Restrict file operations to workspace
  restrict_to_workspace: true
  
  # HTTP SSRF protection
  ssrf_protection_enabled: true
  ssrf_blocklist:
    - "internal.company.com"
    - "admin.local"
```

### Skills Configuration

```yaml
tools:
  skills:
    enabled: true
    
    # Global skills directory (~/.wunderpus/skills)
    global_skills_path: "~/.wunderpus/skills"
    
    # Built-in skills directory
    builtin_skills_path: "./skills"
    
    # Registry configuration
    registries:
      clawhub:
        enabled: false
        base_url: "https://clawhub.ai"
        auth_token: ""
    
    # Version constraints
    constraints:
      github:
        min_version: "1.0.0"
        max_version: "2.0.0"
```

### Search Configuration

```yaml
tools:
  search:
    brave_api_key: "${BRAVE_API_KEY}"
    tavily_api_key: "${TAVILY_API_KEY}"
    tavily_base_url: ""  # Custom endpoint
    perplexity_key: "${PERPLEXITY_KEY}"
```

## Security Configuration

```yaml
security:
  # Encryption settings
  encryption:
    enabled: false
    key: "${ENCRYPTION_KEY}"  # 32-byte, base64 encoded
  
  # Audit logging
  audit_enabled: true
  audit_db_path: "wunderpus_audit.db"
  audit_retention_days: 90
  
  # Rate limiting
  rate_limit:
    requests_per_minute: 60
    burst: 10
    per_user:
      requests_per_minute: 20
      burst: 5
    per_provider:
      requests_per_minute: 30
      burst: 5
  
  # Input sanitization
  sanitization_enabled: true
```

## Logging Configuration

```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "stderr"  # stderr, stdout, or file path
  
  # File output (when output is a path)
  max_size: 100  # MB
  max_backups: 3
  max_age: 30  # days
  compress: true
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `level` | string | "info" | Log level |
| `format` | string | "json" | Output format |
| `output` | string | "stderr" | Output destination |
| `max_size` | int | 100 | Max log file size (MB) |
| `max_backups` | int | 3 | Number of backups |
| `max_age` | int | 30 | Retention days |
| `compress` | bool | true | Compress old logs |

## Server Configuration

```yaml
server:
  health_port: 8080
  
  # TLS settings
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
  
  # HTTP to HTTPS redirect
  https_redirect: false
```

## Heartbeat Configuration

```yaml
heartbeat:
  enabled: true
  interval: 30  # minutes (minimum: 5)
```

## Monitoring Configuration

```yaml
monitoring:
  prometheus:
    enabled: false
    port: 9090
    path: "/metrics"
```

## Stats Configuration

```yaml
stats:
  enabled: true
```

## Agent Defaults

```yaml
agents:
  defaults:
    # Workspace directory
    workspace: "."
    
    # Restrict file operations to workspace
    restrict_to_workspace: true
```

## Environment Variables

Configuration values support environment variable interpolation:

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"

security:
  encryption:
    key: "${ENCRYPTION_KEY}"
```

Supported formats:
- `${VAR}` - Value of VAR
- `${VAR:-default}` - Value of VAR or "default"
- `${VAR:-$OTHER}` - Value of VAR or OTHER

## Complete Example

```yaml
default_provider: "openai"

model_list:
  - model_name: "gpt4o"
    model: "openai/gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    max_tokens: 4096

  - model_name: "claude"
    model: "anthropic/claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o"

  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-sonnet-4-20250514"

channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"

  websocket:
    enabled: true
    host: "0.0.0.0"
    port: 8081

agent:
  system_prompt: "You are Wunderpus, a helpful AI assistant."
  max_context_tokens: 8000
  temperature: 0.7

tools:
  enabled: true
  timeout_seconds: 30
  shell_whitelist:
    - git
    - go
    - npm

agents:
  defaults:
    workspace: "."
    restrict_to_workspace: true

security:
  encryption:
    enabled: false
  audit_enabled: true
  rate_limit:
    requests_per_minute: 60

logging:
  level: "info"
  format: "json"
  output: "stderr"

server:
  health_port: 8080

heartbeat:
  enabled: true
  interval: 30

stats:
  enabled: true
```
