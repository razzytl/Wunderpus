# Getting Started

Get Wunderpus running in 5 minutes.

## Prerequisites

- **Go 1.25+** — [Install Go](https://go.dev/doc/install)
- **Git** — For cloning the repository

## Installation

### From Source

```bash
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus
go build -o build/wunderpus ./cmd/wunderpus
```

The binary will be at `build/wunderpus`.

### Docker

```bash
docker build -t wunderpus:latest .
docker run -d -p 8080:8080 -p 9090:9090 -v $(pwd)/config.yaml:/app/config.yaml wunderpus:latest
```

## Configuration

### Step 1: Create Config File

```bash
cp config.example.yaml config.yaml
```

### Step 2: Configure at Least One Provider

Edit `config.yaml` and add your API key:

```yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    api_key: "sk-your-key"
    max_tokens: 4096
```

Or use the legacy format:

```yaml
default_provider: "openai"

providers:
  openai:
    api_key: "sk-your-openai-key"
    model: "gpt-4o"
    max_tokens: 4096
```

### Step 3: (Optional) Use Environment Variables

```bash
export OPENAI_API_KEY="sk-your-key"
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

This launches a full-featured terminal interface with streaming responses, tool execution output, and provider switching.

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

This guides you through provider selection, API key entry, workspace configuration, channel setup, and security settings.

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
