# Wunderpus

<p align="center">
  <strong>Universal Autonomous AI Agent Framework — Written in Go</strong>
</p>

<p align="center">
  <a href="https://golang.org/doc/devel/release.html#policy">
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8?style=flat-square" alt="Go Version"/>
  </a>
  <a href="https://github.com/wunderpus/wunderpus/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"/>
  </a>
</p>

---

## What Is Wunderpus?

Wunderpus is a **production-grade, vendor-agnostic autonomous AI agent framework** written in Go. It provides a complete runtime for building, deploying, and managing AI agents that can interact with users across multiple channels, execute tools with policy-based approval gates, maintain persistent memory with RAG, and operate with structured autonomy.

### Core Capabilities

| Capability | Description |
|---|---|
| **Multi-Provider LLM** | 15+ providers via protocol-based routing (OpenAI, Anthropic, Gemini, Ollama, Groq, DeepSeek, and more) with automatic fallback |
| **Multi-Channel** | Connect to Telegram, Discord, Slack, WhatsApp, and WebSocket — simultaneously |
| **Tool System** | 15+ built-in tools (file I/O, shell, HTTP, browser, calculator) with workspace sandboxing and policy-based approval gates |
| **Skills** | Markdown-based extensibility system with local and global registries |
| **Memory & RAG** | SQLite-persisted sessions with AES-256-GCM encryption, vector search, and SOP (Standard Operating Procedure) retrieval |
| **World Model** | Persistent knowledge graph with entity/relation tracking, confidence scoring, and Cypher-like queries |
| **Perception** | Computer use via Playwright — navigate websites, fill forms, interact with any GUI |
| **Self-Improvement** | Tool synthesis engine that detects capability gaps and generates new tools |
| **Conversation Branching** | Create, switch, and navigate conversation branches for exploring alternative paths |
| **Multi-Modal Input** | Detect and route text, image, audio, PDF, and DOCX inputs through a unified pipeline |
| **Structured Output** | Enforce JSON output format with automatic validation and retry on invalid responses |
| **Cost Prediction** | Pre-execution cost estimation using token counting and model pricing matrices |
| **Checkpoint & Resume** | Crash-resilient task execution with persistent checkpoints and resume capability |
| **Observability** | OpenTelemetry tracing spans across providers, tools, and agent loops |
| **Health Dashboard** | Component-level health aggregation with `/health`, `/live`, and `/ready` endpoints |
| **Webhooks** | Event-driven webhook delivery with Go template rendering and retry/backoff |

---

## Quick Start

### Prerequisites

- Go 1.25+

### Install & Run

```bash
# Clone and build
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus
go build -o build/wunderpus ./cmd/wunderpus

# Configure
cp config.example.yaml config.yaml
# Edit config.yaml with your API keys

# Run interactive TUI
./build/wunderpus

# Or one-shot mode
./build/wunderpus agent -m "What can you do?"

# Or gateway mode (background services)
./build/wunderpus gateway
```

### Docker

```bash
docker build -t wunderpus:latest .
docker run -d -p 8080:8080 -p 9090:9090 -v $(pwd)/config.yaml:/app/config.yaml wunderpus:latest
```

---

## Architecture at a Glance

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                               │
│    wunderpus │ agent │ gateway │ onboard │ status │ skills      │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    Application Bootstrap                         │
│  Config → Logging → DB → Security → Providers → Tools          │
│  → Memory → Skills → ToolSynth → WorldModel → Perception       │
│  → Agent → SubAgent → Heartbeat → Health → Channels            │
└─────────┬────────────────────┬────────────────────┬─────────────┘
          │                    │                    │
┌─────────▼──────┐  ┌─────────▼──────┐  ┌─────────▼──────────────┐
│  Agent Core    │  │   Channels     │  │   Core Systems         │
│  • Context Mgt │  │  • Telegram    │  │  • Tool Synthesis      │
│  • Tool Exec   │  │  • Discord     │  │  • World Model         │
│  • Skills      │  │  • Slack       │  │  • Perception          │
│  • RAG/SOP     │  │  • WhatsApp    │  │  • AGS (Goals)         │
│  • Streaming   │  │  • WebSocket   │  │  • Sub-Agents          │
│  • Branching   │  │                │  │  • Heartbeat           │
└────────────────┘  └────────────────┘  └────────────────────────┘
          │                    │                    │
┌─────────▼────────────────────▼────────────────────▼─────────────┐
│                    Provider Router                               │
│    OpenAI │ Anthropic │ Gemini │ Ollama │ Groq │ DeepSeek │ ... │
└─────────────────────────────────────────────────────────────────┘
          │
┌─────────▼───────────────────────────────────────────────────────┐
│                    Persistence Layer                             │
│    wunderpus.db (core)  │  wunderpus-audit.db (audit log)       │
└─────────────────────────────────────────────────────────────────┘
```

---

## Documentation

| Section | Description |
|---|---|
| **[Getting Started](docs/guides/getting-started.md)** | Installation, configuration, first run |
| **[Architecture](docs/architecture/overview.md)** | System design, data flow, extension points |
| **[Providers](docs/reference/providers.md)** | LLM provider configuration (15+ supported) |
| **[Channels](docs/reference/channels.md)** | Messaging platform integrations |
| **[Tools](docs/reference/tools.md)** | Built-in tools and tool system |
| **[Skills](docs/reference/skills.md)** | Creating, installing, and managing skills |
| **[CLI Reference](docs/reference/cli.md)** | Complete command-line documentation |
| **[Configuration](docs/reference/configuration.md)** | Full config.yaml reference |
| **[Security](docs/guides/security.md)** | Security model, hardening, best practices |
| **[Deployment](docs/operations/deployment.md)** | Production deployment strategies |
| **[Monitoring](docs/operations/monitoring.md)** | Health dashboard, OpenTelemetry, metrics |
| **[Heartbeat](docs/reference/heartbeat.md)** | Periodic task scheduling |
| **[Troubleshooting](docs/operations/troubleshooting.md)** | Common issues and solutions |

---

## Project Structure

```
wunderpus/
├── cmd/wunderpus/          # CLI entry point (Cobra commands)
├── internal/               # All application code
│   ├── app/                # Bootstrap & wiring (composition root)
│   ├── agent/              # Core agent loop, context, branching, checkpoints
│   ├── agents/             # Sub-agent lifecycle management
│   ├── provider/           # LLM provider adapters & router
│   ├── channel/            # Messaging channel implementations
│   ├── tool/               # Tool system & built-in tools
│   ├── toolsynth/          # Tool synthesis engine
│   ├── skills/             # Skills loading & registry
│   ├── memory/             # Session storage, RAG, vector search
│   ├── worldmodel/         # Knowledge graph
│   ├── perception/         # Computer use (Playwright) + multi-modal input
│   ├── subagent/           # Sub-agent management
│   ├── security/           # Sanitization, sandbox, encryption
│   ├── audit/              # Tamper-evident audit log
│   ├── config/             # Configuration loading & validation
│   ├── health/             # Health check server with aggregator
│   ├── heartbeat/          # Periodic task scheduler
│   ├── logging/            # Structured logging & observability
│   ├── cost/               # Cost tracking, budgeting & prediction
│   ├── events/             # Pub/sub event bus with priority routing
│   ├── telemetry/          # OpenTelemetry tracer initialization
│   ├── prompts/            # Prompt versioning manager
│   ├── webhook/            # Webhook delivery system
│   ├── types/              # Shared type definitions
│   ├── errors/             # Typed error system
│   ├── constants/          # Global constants
│   ├── tui/                # Terminal UI (Bubbletea)
│   ├── ags/                # Autonomous Goal Synthesis (manual trigger)
│   ├── money/              # Income generation capabilities
│   ├── planning/           # Planning & self-mapping
│   └── db/                 # Shared database manager (2 DBs)
├── contrib/channels/       # Optional channel plugins (Feishu, QQ, WeCom, DingTalk)
├── web/                    # Web server (HTTP + WebSocket)
├── skills/                 # Built-in skills
├── docs/                   # Documentation
├── config.example.yaml     # Example configuration
├── free_tiers.yaml         # Free-tier provider definitions
├── HEARTBEAT.md            # Periodic task definitions
├── Dockerfile              # Production Docker image
├── docker-compose.yml      # Multi-service Docker Compose
├── Makefile                # Build automation
└── go.mod                  # Go module definition
```

---

## Supported LLM Providers

| Provider | Protocol | Models | Free Tier |
|---|---|---|---|
| OpenAI | `openai` | gpt-4o, gpt-4o-mini | No |
| Anthropic | `anthropic` | claude-sonnet-4, claude-opus-4 | No |
| Google Gemini | `gemini` | gemini-2.0-flash, gemini-1.5-pro | Yes |
| Ollama | `ollama` | Any local model | Yes |
| OpenRouter | `openai` | 100+ models | Yes |
| Groq | `openai` | llama-3.3-70b, mixtral-8x7b | Yes |
| DeepSeek | `openai` | deepseek-chat, deepseek-r1 | Yes |
| Cerebras | `openai` | llama-3.3-70b | No |
| NVIDIA NIM | `openai` | nemotron-70b | No |
| Mistral | `openai` | mistral-large | No |
| Zhipu (GLM) | `openai` | glm-4 | No |
| Moonshot (Kimi) | `openai` | kimi-latest | No |
| Qwen | `openai` | qwen-turbo | No |
| vLLM | `openai` | Any self-hosted | Yes |
| LiteLLM | `openai` | Any proxied | Yes |

---

## Security Model

Wunderpus implements defense-in-depth across five layers:

| Layer | Mechanism | Purpose |
|---|---|---|
| **Input** | Unicode normalization + 9-pattern injection detection | Block prompt injection attacks |
| **Execution** | Workspace sandbox + command chaining prevention | Restrict file/shell operations |
| **Network** | SSRF blocklist (localhost, private IPs, cloud metadata) | Prevent internal network access |
| **Approval** | Policy-based tool classification (AutoExecute, NotifyOnly, RequiresApproval, Blocked) | Control tool execution |
| **Storage** | AES-256-GCM encryption + SHA-256 hash-chained audit log | Protect data at rest |

---

## License

MIT — See [LICENSE](LICENSE) for details.
