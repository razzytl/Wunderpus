# Configuration Examples

This document provides real-world configuration examples for various use cases.

## Minimal Configurations

### Basic Usage

```yaml
# Minimal configuration with single provider
default_provider: "openai"

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o"
```

### Local Ollama

```yaml
# Local model only
providers:
  ollama:
    host: "http://localhost:11434"
    model: "llama3.2"
```

### Development Setup

```yaml
# Development with multiple fallbacks
default_provider: "openai"

model_list:
  - model_name: "gpt4o"
    model: "openai/gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    fallback_models:
      - "anthropic/claude-sonnet-4-20250514"

  - model_name: "claude"
    model: "anthropic/claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"

providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-sonnet-4-20250514"
```

## Production Configurations

### High Availability

```yaml
# Production with multiple providers and fallbacks
default_provider: "openai"

model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    fallback_models:
      - "anthropic/claude-sonnet-4-20250514"
      - "openrouter/deepseek/deepseek-r1"

  - model_name: "fast"
    model: "groq/llama-3.3-70b-versatile"
    api_key: "${GROQ_API_KEY}"
    api_base: "https://api.groq.com/openai/v1"
    fallback_models:
      - "openrouter/NousResearch/Nous-Hermes-2-Mixtral-8x7B-DPO"

  - model_name: "reasoning"
    model: "deepseek/deepseek-r1"
    api_key: "${DEEPSEEK_API_KEY}"
    fallback_models:
      - "openrouter/deepseek/deepseek-r1"

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o"

  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-sonnet-4-20250514"

  groq:
    api_key: "${GROQ_API_KEY}"
    model: "llama-3.3-70b-versatile"

  deepseek:
    api_key: "${DEEPSEEK_API_KEY}"
    model: "deepseek-chat"

  openrouter:
    api_key: "${OPENROUTER_API_KEY}"
    model: "deepseek/deepseek-r1"

agent:
  system_prompt: "You are Wunderpus, a helpful AI assistant. Be concise and accurate."
  max_context_tokens: 8000
  temperature: 0.7
  parallel_probe: true
  probe_timeout: 5s

tools:
  enabled: true
  timeout_seconds: 60
  shell_whitelist:
    - git
    - go
    - npm
    - cargo
    - docker
    - make
    - curl
    - grep

tools:
  skills:
    enabled: true

security:
  encryption:
    enabled: true
    key: "${ENCRYPTION_KEY}"
  audit_enabled: true
  audit_db_path: "/var/lib/wunderpus/audit.db"

logging:
  level: "info"
  format: "json"
  output: "/var/log/wunderpus/app.log"

server:
  health_port: 8080
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/wunderpus.crt"
    key_file: "/etc/ssl/private/wunderpus.key"

heartbeat:
  enabled: true
  interval: 30

stats:
  enabled: true
```

### Cost-Optimized

```yaml
# Configuration focused on minimizing costs
default_provider: "groq"

model_list:
  - model_name: "fast"
    model: "groq/llama-3.1-8b-instant"
    api_key: "${GROQ_API_KEY}"
    api_base: "https://api.groq.com/openai/v1"
    max_tokens: 2048

  - model_name: "reasoning"
    model: "groq/llama-3.3-70b-versatile"
    api_key: "${GROQ_API_KEY}"
    fallback_models:
      - "openai/gpt-4o-mini"

providers:
  groq:
    api_key: "${GROQ_API_KEY}"
    model: "llama-3.1-8b-instant"

  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o-mini"

agent:
  max_context_tokens: 4000
  temperature: 0.5
```

## Channel Configurations

### Telegram Only

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
```

### Multiple Channels

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"

  discord:
    enabled: true
    bot_token: "${DISCORD_BOT_TOKEN}"

  websocket:
    enabled: true
    host: "0.0.0.0"
    port: 8081

server:
  health_port: 8080
```

### All Channels

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"

  discord:
    enabled: true
    bot_token: "${DISCORD_BOT_TOKEN}"

  websocket:
    enabled: true
    host: "0.0.0.0"
    port: 8081
    auth_token: "${WS_AUTH_TOKEN}"

  qq:
    enabled: true
    account: 123456789

  wecom:
    enabled: true
    corp_id: "${WECOM_CORP_ID}"
    agent_id: "${WECOM_AGENT_ID}"
    secret: "${WECOM_SECRET}"

  dingtalk:
    enabled: true
    app_id: "${DINGTALK_APP_ID}"
    app_secret: "${DINGTALK_APP_SECRET}"

  onebot:
    enabled: true
```

## Security Configurations

### Basic Security

```yaml
security:
  encryption:
    enabled: true
    key: "${ENCRYPTION_KEY}"
  audit_enabled: true

tools:
  enabled: true
  shell_whitelist:
    - git
    - go

agents:
  defaults:
    workspace: "/home/user/projects"
    restrict_to_workspace: true
```

### High Security

```yaml
security:
  encryption:
    enabled: true
    key: "${ENCRYPTION_KEY}"
  audit_enabled: true
  audit_db_path: "/var/lib/wunderpus/audit.db"
  audit_retention_days: 365
  rate_limit:
    requests_per_minute: 30
    per_user:
      requests_per_minute: 10
    per_provider:
      requests_per_minute: 20
  sanitization_enabled: true

tools:
  enabled: true
  shell_whitelist:
    - git
  shell_blocklist:
    - "^rm -rf"
    - "^dd"
    - "^mkfs"
  timeout_seconds: 30

tools:
  allowed_paths:
    - "/home/user/projects"
  restrict_to_workspace: true

tools:
  ssrf_protection_enabled: true
  ssrf_blocklist:
    - "internal.company.com"
    - "169.254.169.254"
    - "metadata.google.internal"

logging:
  level: "warn"
  format: "json"
```

## Docker Configurations

### Development Docker

```yaml
version: '3.8'

services:
  wunderpus:
    image: wunderpus/wunderpus:latest
    container_name: wunderpus-dev
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/app/data
    environment:
      - WUNDERPUS_CONFIG=/app/config.yaml
      - WUNDERPUS_LOG_LEVEL=debug
    ports:
      - "8080:8080"
      - "8081:8081"
```

### Production Docker

```yaml
version: '3.8'

services:
  wunderpus:
    image: wunderpus/wunderpus:latest
    container_name: wunderpus
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - wunderpus-data:/app/data
      - wunderpus-logs:/app/logs
    environment:
      - WUNDERPUS_CONFIG=/app/config.yaml
      - WUNDERPUS_LOG_LEVEL=info
    ports:
      - "127.0.0.1:8080:8080"
      - "127.0.0.1:8081:8081"
    healthcheck:
      test: ["CMD", "wunderpus", "status"]
      interval: 30s
      timeout: 10s
      retries: 3
    mem_limit: 512m
    cpus: 1.0
    security_opt:
      - no-new-privileges:true

volumes:
  wunderpus-data:
  wunderpus-logs:
```

### Docker with Prometheus

```yaml
version: '3.8'

services:
  wunderpus:
    image: wunderpus/wunderpus:latest
    volumes:
      - ./config.yaml:/app/config.yaml:ro
    environment:
      - WUNDERPUS_CONFIG=/app/config.yaml
    ports:
      - "8080:8080"
      - "9090:9090"

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
    ports:
      - "9091:9090"

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
```

## Complete Example

```yaml
# Complete production configuration
default_provider: "openai"

model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    max_tokens: 4096
    fallback_models:
      - "anthropic/claude-sonnet-4-20250514"
      - "openrouter/deepseek/deepseek-r1"

  - model_name: "fast"
    model: "groq/llama-3.3-70b-versatile"
    api_key: "${GROQ_API_KEY}"
    api_base: "https://api.groq.com/openai/v1"

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o"
    max_tokens: 4096

  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-sonnet-4-20250514"
    max_tokens: 4096

  groq:
    api_key: "${GROQ_API_KEY}"
    model: "llama-3.3-70b-versatile"

  openrouter:
    api_key: "${OPENROUTER_API_KEY}"
    model: "deepseek/deepseek-r1"

channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"

  discord:
    enabled: true
    bot_token: "${DISCORD_BOT_TOKEN}"

  websocket:
    enabled: true
    host: "0.0.0.0"
    port: 8081
    auth_token: "${WS_AUTH_TOKEN}"

agent:
  system_prompt: "You are Wunderpus, a helpful AI assistant. Be concise, accurate, and helpful."
  max_context_tokens: 8000
  temperature: 0.7
  parallel_probe: true
  probe_timeout: 5s

tools:
  enabled: true
  timeout_seconds: 60
  shell_whitelist:
    - git
    - go
    - npm
    - cargo
    - docker
    - make
    - curl
    - grep
    - cat
    - ls

tools:
  skills:
    enabled: true
    global_skills_path: "~/.wunderpus/skills"
    builtin_skills_path: "./skills"

  search:
    brave_api_key: "${BRAVE_API_KEY}"
    tavily_api_key: "${TAVILY_API_KEY}"

agents:
  defaults:
    workspace: "."
    restrict_to_workspace: true

security:
  encryption:
    enabled: true
    key: "${ENCRYPTION_KEY}"
  audit_enabled: true
  audit_db_path: "/var/lib/wunderpus/wunderpus_audit.db"
  rate_limit:
    requests_per_minute: 60
    burst: 10

logging:
  level: "info"
  format: "json"
  output: "stderr"

server:
  health_port: 8080

monitoring:
  prometheus:
    enabled: true
    port: 9090
    path: "/metrics"

heartbeat:
  enabled: true
  interval: 30

stats:
  enabled: true
```
