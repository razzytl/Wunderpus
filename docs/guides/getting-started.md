# Getting Started

Get Wunderpus running in 5 minutes.

## Prerequisites

- **Go 1.25+** — [Install Go](https://go.dev/doc/install)
- **Git** — For cloning the repository
- **Make** (optional) — For using Makefile targets

## Installation

### Option 1: From Source (Recommended)

```bash
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus
make build
```

The binary will be at `build/wunderpus`.

### Option 2: Direct Build

```bash
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus
go build -o wunderpus ./cmd/wunderpus
```

### Option 3: Docker

```bash
docker build -t wunderpus:latest .
```

## Configuration

### Step 1: Create Config File

```bash
cp config.example.yaml config.yaml
```

### Step 2: Configure at Least One Provider

Edit `config.yaml` and add your API key:

```yaml
# Quick setup — just one provider to start
default_provider: "openai"

providers:
  openai:
    api_key: "sk-your-openai-key"
    model: "gpt-4o"
    max_tokens: 4096
```

Or use the recommended `model_list` format for multiple providers:

```yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    api_key: "sk-your-key"
    max_tokens: 4096

  - model_name: "fallback"
    model: "anthropic/claude-sonnet-4-20250514"
    api_key: "sk-ant-your-key"
    max_tokens: 4096
```

### Step 3: (Optional) Use Environment Variables

For production, store keys in environment variables:

```bash
export OPENAI_API_KEY="sk-your-key"
export ANTHROPIC_API_KEY="sk-ant-your-key"
```

Then reference them in config:

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
```

## Running Wunderpus

### Interactive Terminal UI (Default)

```bash
./build/wunderpus
```

This launches a full-featured terminal interface with:
- Streaming responses
- Tool execution output
- Provider switching (Tab)
- Command palette (Ctrl+P)
- Markdown rendering

### One-Shot Mode

```bash
./build/wunderpus agent -m "Summarize the latest Go release notes"
```

Perfect for scripting and CI pipelines.

### Gateway Mode (Background Services)

```bash
./build/wunderpus gateway
```

Starts all enabled channels (Telegram, Discord, etc.) and the heartbeat scheduler. Runs until interrupted.

### Web UI

```bash
./build/wunderpus --ui
```

Serves the embedded React frontend at `http://localhost:8080`.

### Check Status

```bash
./build/wunderpus status
```

Shows workspace, configured providers, and uptime.

## Interactive Onboarding

For first-time setup, run the interactive wizard:

```bash
./build/wunderpus onboard
```

This guides you through:
1. Provider selection and API key entry
2. Workspace configuration
3. Channel setup
4. Security settings

## Verify Your Setup

```bash
# Check provider authentication
./build/wunderpus auth status

# List installed skills
./build/wunderpus skills list

# Check heartbeat tasks
./build/wunderpus cron list
```

## Next Steps

| Topic | Document |
|---|---|
| Configure multiple providers | [Provider Reference](../reference/providers.md) |
| Connect messaging channels | [Channel Reference](../reference/channels.md) |
| Understand the architecture | [System Overview](../architecture/overview.md) |
| Deploy to production | [Deployment Guide](../operations/deployment.md) |
| Secure your installation | [Security Guide](../guides/security.md) |
