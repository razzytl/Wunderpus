# Quick Reference

One-page cheat sheet for common Wunderpus operations.

## Install & Build

```bash
git clone https://github.com/wunderpus/wunderpus.git && cd wunderpus
make build              # Build for current platform
make build-all          # Build for all platforms
make install            # Install to ~/.local/bin
make test               # Run tests
make lint               # Run linters
make docker-build       # Build Docker image
```

## Run

```bash
./build/wunderpus                    # Interactive TUI
./build/wunderpus --ui               # Web UI (port 8080)
./build/wunderpus agent -m "msg"     # One-shot
./build/wunderpus gateway            # Background services
./build/wunderpus status             # Check status
./build/wunderpus onboard            # Setup wizard
```

## Configure

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with API keys

# Or use env vars
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
```

## Skills

```bash
wunderpus skills list                          # List installed
wunderpus skills install user/repo             # Install from GitHub
wunderpus skills install ./path/to/skill       # Install from local
```

## Heartbeat

```bash
wunderpus cron list                            # List tasks
wunderpus cron add "Check GitHub issues"       # Add task
```

## Auth

```bash
wunderpus auth status                          # Check auth
wunderpus auth login openai                    # Login to provider
```

## Docker

```bash
docker build -t wunderpus:latest .
docker run -d -p 8080:8080 -p 9090:9090 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  wunderpus:latest

docker compose --profile gateway up -d
```

## Key Directories

| Path | Purpose |
|---|---|
| `cmd/wunderpus/` | CLI entry point |
| `internal/` | All application code (39 packages) |
| `skills/` | Built-in skills |
| `web/` | HTTP + WebSocket server |
| `ui/` | React frontend |
| `docs/` | Documentation |
| `grafana/` | Grafana dashboards |

## Key Config Options

| Option | Default | Description |
|---|---|---|
| `default_provider` | `"openai"` | Default LLM provider |
| `agent.max_context_tokens` | `8000` | Max conversation tokens |
| `agent.temperature` | `0.7` | Sampling temperature |
| `tools.timeout_seconds` | `30` | Tool execution timeout |
| `heartbeat.interval` | `30` | Heartbeat interval (min) |
| `logging.level` | `"info"` | Log level |
| `logging.format` | `"json"` | Log format |

## Architecture at a Glance

```
CLI → Bootstrap → Agent → Provider Router → LLM APIs
            ↕
    Channels │ Tools │ Skills │ Memory │ Genesis
```

## Provider Protocols

| Protocol | Providers |
|---|---|
| `openai/` | OpenAI, Groq, OpenRouter, DeepSeek, Cerebras, NVIDIA, Mistral, +7 more |
| `anthropic/` | Claude |
| `ollama/` | Local models |
| `gemini/` | Google Gemini |

## TUI Shortcuts

| Key | Action |
|---|---|
| `Enter` | Send |
| `Ctrl+C` | Exit |
| `Tab` | Switch provider |
| `Ctrl+P` | Command palette |
| `Up/Down` | History |
