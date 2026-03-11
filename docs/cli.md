# CLI Reference

Complete command-line interface documentation for Wunderpus.

## Synopsis

```bash
wunderpus [command] [flags]
```

## Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | - | Path to configuration file | `config.yaml` |
| `--verbose` | `-v` | Enable debug logging | `false` |
| `--version` | - | Show version information | - |
| `--help` | `-h` | Show help message | - |

## Commands

### agent

Run the Wunderpus agent interactively or process a single message.

```bash
wunderpus agent [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--message` | `-m` | One-shot message to send to the agent |

**Examples:**

```bash
# Start interactive TUI
wunderpus agent

# Send a single message and exit
wunderpus agent -m "Hello, how are you?"

# With custom config
wunderpus agent --config /path/to/config.yaml -m "Your message"
```

---

### gateway

Start the Wunderpus gateway, enabling background services and communication channels.

```bash
wunderpus gateway [flags]
```

**Description:**

Starts all enabled channels (Telegram, Discord, WebSocket, etc.), the heartbeat scheduler, and the health check server. The gateway runs until interrupted.

**Examples:**

```bash
# Start gateway
wunderpus gateway

# Start with verbose logging
wunderpus gateway -v

# With custom config
wunderpus gateway --config /etc/wunderpus/config.yaml
```

---

### status

Display current Wunderpus system status.

```bash
wunderpus status
```

**Output:**

```
Wunderpus Status
Workspace: /home/user/projects
Providers: [openai anthropic]
DefaultProvider: openai
Uptime: 24h0m0s
```

---

### cron

Manage periodic (heartbeat) tasks.

```bash
wunderpus cron [command]
```

#### cron list

List all scheduled periodic tasks.

```bash
wunderpus cron list
```

**Output:**

```
Periodic Tasks Status: running
Interval: 30 minutes
Quick Tasks: 5
Long Tasks: 2
```

#### cron add

Add a new periodic task to HEARTBEAT.md.

```bash
wunderpus cron add [task description]
```

**Examples:**

```bash
# Add a daily task
wunderpus cron add "Generate daily report"

# Add a weekly task
wunderpus cron add "Review open PRs every Monday"
```

---

### skills

Manage and discover agent skills.

```bash
wunderpus skills [command]
```

#### skills list

List all installed skills.

```bash
wunderpus skills list
```

**Output:**

```
Installed Skills (5):
- github (builtin): GitHub CLI integration
- tmux (builtin): Terminal multiplexer automation
- weather (builtin): Weather information
- summarize (builtin): Content summarization
- my-skill (local): Custom skill description
```

#### skills install

Install a skill from a remote repository or local path.

```bash
wunderpus skills install [source]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `source` | GitHub URL or local path |

**Examples:**

```bash
# Install from GitHub
wunderpus skills install https://github.com/user/skill-name

# Install from local path
wunderpus skills install ./path/to/skill

# Install using shorthand
wunderpus skills install user/repo
```

---

### auth

Manage authentication and API keys for providers.

```bash
wunderpus auth [command]
```

#### auth status

Show authentication status for all configured providers.

```bash
wunderpus auth status
```

**Output:**

```
Authentication Status:
- openai: Authenticated
- anthropic: Not configured
- ollama: Connected
```

#### auth login

Initiate authentication flow for a provider.

```bash
wunderpus auth login [provider]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `provider` | Provider name (openai, anthropic, etc.) |

**Examples:**

```bash
wunderpus auth login openai
wunderpus auth login anthropic
```

**Note:** Most providers require API keys rather than OAuth. Use `wunderpus onboard` for interactive configuration.

---

### onboard

Run interactive onboarding to configure Wunderpus.

```bash
wunderpus onboard [flags]
```

**Description:**

Launches an interactive wizard to:
- Select and configure LLM providers
- Enter API keys
- Enable and configure channels
- Test configuration

**Examples:**

```bash
wunderpus onboard
wunderpus onboard --config /path/to/config.yaml
```

---

## Configuration File

Wunderpus uses a YAML configuration file. By default, it looks for:

1. `./config.yaml` (current directory)
2. `~/.wunderpus/config.yaml`
3. Path specified by `--config` flag
4. Path specified by `WUNDERPUS_CONFIG` environment variable

### Environment Variable Support

Configuration values can reference environment variables:

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
```

---

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Provider error |
| 4 | Channel error |
| 130 | Interrupted (Ctrl+C) |

---

## Shell Completion

Generate shell completion scripts:

```bash
# Bash
wunderpus completion bash > /etc/bash_completion.d/wunderpus

# Zsh
wunderpus completion zsh > "${fpath[1]}/_wunderpus"

# Fish
wunderpus completion fish > ~/.config/fish/completions/wunderpus.fish
```

---

## Configuration Examples

### Minimal Configuration

```yaml
# config.yaml
providers:
  openai:
    api_key: "sk-your-key"
    model: "gpt-4o"
```

### Multi-Provider Configuration

```yaml
# config.yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    api_key: "${OPENAI_API_KEY}"
  - model_name: "fast"
    model: "groq/llama-3.3-70b-versatile"
    api_key: "${GROQ_API_KEY}"
  - model_name: "local"
    model: "ollama/llama3.2"
    api_base: "http://localhost:11434"
```

### Full Configuration

```yaml
# Full configuration example
default_provider: "openai"

model_list:
  - model_name: "gpt4o"
    model: "openai/gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    max_tokens: 4096
    temperature: 0.7

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o"

  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-sonnet-4-20250514"

  ollama:
    host: "http://localhost:11434"
    model: "llama3.2"

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
    - cargo

tools:
  skills:
    enabled: true
    global_skills_path: "~/.wunderpus/skills"
    builtin_skills_path: "./skills"

security:
  encryption:
    enabled: false
  audit_db_path: "wunderpus_audit.db"

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

---

## Troubleshooting

### Command Not Found

Ensure Wunderpus is in your PATH:

```bash
# Add to PATH
export PATH=$PATH:/usr/local/bin

# Or use full path
./wunderpus status
```

### Configuration Not Found

Specify configuration explicitly:

```bash
wunderpus --config /path/to/config.yaml status
```

### Verbose Output

Use `-v` flag for debug information:

```bash
wunderpus -v agent -m "test"
```

---

## Getting Help

```bash
# General help
wunderpus --help

# Command-specific help
wunderpus agent --help
wunderpus gateway --help
wunderpus skills --help
```
