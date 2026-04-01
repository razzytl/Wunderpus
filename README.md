# Wunderpus

<p align="center">
  <img src="resources/banner.jpg" alt="Wunderpus" width="400"/>
</p>

<p align="center">
  <strong>Universal Autonomous AI Agent Framework</strong>
</p>

<p align="center">
  <a href="https://github.com/wunderpus/wunderpus/actions/workflows/ci.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/wunderpus/wunderpus/ci.yml?branch=main&style=flat-square" alt="CI"/>
  </a>
  <a href="https://golang.org/doc/devel/release.html#policy">
    <img src="https://img.shields.io/github/go-mod/go-version/wunderpus/wunderpus?style=flat-square" alt="Go Version"/>
  </a>
  <a href="https://github.com/wunderpus/wunderpus/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/wunderpus/wunderpus?style=flat-square" alt="License"/>
  </a>
</p>

---

## What Is Wunderpus?

Wunderpus is a **production-grade, vendor-agnostic autonomous AI agent framework** written in Go. It provides a complete runtime for building, deploying, and managing AI agents that can interact with users across multiple channels, execute tools, synthesize new capabilities, and — with the Genesis system — operate autonomously with safety guardrails.

### Core Capabilities

| Capability | Description |
|---|---|
| **Multi-Provider LLM** | 15+ providers via protocol-based routing (OpenAI, Anthropic, Gemini, Ollama, Groq, DeepSeek, and more) with automatic fallback and parallel probing |
| **Multi-Channel** | Connect to Telegram, Discord, Slack, WhatsApp, Feishu, LINE, QQ, WeCom, DingTalk, OneBot, and WebSocket — simultaneously |
| **Tool System** | 15+ built-in tools (file I/O, shell, HTTP, browser, calculator) with sandboxing, approval gates, and MCP support |
| **Skills** | Markdown-based extensibility system with local, global, and remote (ClawHub) registries |
| **Memory & RAG** | SQLite-persisted sessions with AES-256-GCM encryption, vector search, and SOP (Standard Operating Procedure) retrieval |
| **World Model** | Persistent knowledge graph with entity/relation tracking, confidence scoring, and Cypher-like queries |
| **Perception** | Computer use via Playwright — navigate websites, fill forms, interact with any GUI |
| **Swarm** | Multi-agent orchestration with 7 specialist profiles (researcher, coder, writer, trader, operator, creator, security) |
| **Self-Improvement** | Tool synthesis engine that detects capability gaps and generates new tools autonomously |
| **Genesis Autonomy** | Four-pillar autonomous system: RSI (Recursive Self-Improvement), AGS (Autonomous Goal Synthesis), UAA (Unbounded Autonomous Action), RA (Resource Acquisition) |

---

## Quick Start

### Prerequisites

- Go 1.25+
- Make (optional, for Makefile targets)

### Install & Run

```bash
# Clone and build
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus
make build

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
│         Config → Logging → Security → Providers → Tools        │
│         → Memory → Skills → ToolSynth → WorldModel             │
│         → Perception → Agent → SubAgent → Swarm → Heartbeat    │
└─────────┬────────────────────┬────────────────────┬─────────────┘
          │                    │                    │
┌─────────▼──────┐  ┌─────────▼──────┐  ┌─────────▼──────────────┐
│  Agent Core    │  │   Channels     │  │   Genesis Systems      │
│  • Context Mgt │  │  • Telegram    │  │  • RSI (Self-Improve)  │
│  • Tool Exec   │  │  • Discord     │  │  • AGS (Goal Synth)    │
│  • Skills      │  │  • Slack       │  │  • UAA (Autonomy)      │
│  • RAG/SOP     │  │  • WhatsApp    │  │  • RA (Resources)      │
│  • Streaming   │  │  • WebSocket   │  │                        │
└────────────────┘  └────────────────┘  └────────────────────────┘
          │                    │                    │
┌─────────▼────────────────────▼────────────────────▼─────────────┐
│                    Provider Router                               │
│    OpenAI │ Anthropic │ Gemini │ Ollama │ Groq │ DeepSeek │ ... │
└─────────────────────────────────────────────────────────────────┘
```

---

## Documentation

| Section | Description |
|---|---|
| **[Getting Started](docs/guides/getting-started.md)** | Installation, configuration, first run |
| **[Architecture](docs/architecture/overview.md)** | System design, data flow, extension points |
| **[Providers](docs/reference/providers.md)** | LLM provider configuration (20+ supported) |
| **[Channels](docs/reference/channels.md)** | Messaging platform integrations |
| **[Skills](docs/reference/skills.md)** | Creating, installing, and managing skills |
| **[CLI Reference](docs/reference/cli.md)** | Complete command-line documentation |
| **[Configuration](docs/reference/configuration.md)** | Full config.yaml reference |
| **[Security](docs/guides/security.md)** | Security model, hardening, best practices |
| **[Deployment](docs/operations/deployment.md)** | Production deployment strategies |
| **[Monitoring](docs/operations/monitoring.md)** | Metrics, logging, alerting |
| **[Heartbeat](docs/reference/heartbeat.md)** | Periodic task scheduling |
| **[Troubleshooting](docs/operations/troubleshooting.md)** | Common issues and solutions |

---

## Project Structure

```
wunderpus/
├── cmd/wunderpus/          # CLI entry point (Cobra commands)
├── internal/               # All application code
│   ├── app/                # Bootstrap & wiring (composition root)
│   ├── agent/              # Core agent loop, context, orchestration
│   ├── agents/             # Sub-agent lifecycle management
│   ├── provider/           # LLM provider adapters & router
│   ├── channel/            # Messaging channel implementations
│   ├── tool/               # Tool system & built-in tools
│   ├── toolsynth/          # Tool synthesis engine
│   ├── skills/             # Skills loading & registry
│   ├── memory/             # Session storage, RAG, vector search
│   ├── worldmodel/         # Knowledge graph
│   ├── perception/         # Computer use (Playwright)
│   ├── swarm/              # Multi-agent orchestration
│   ├── a2a/                # Agent-to-Agent protocol
│   ├── subagent/           # Sub-agent management
│   ├── security/           # Sanitization, sandbox, encryption
│   ├── audit/              # Tamper-evident audit log
│   ├── config/             # Configuration loading & validation
│   ├── health/             # Health check server
│   ├── heartbeat/          # Periodic task scheduler
│   ├── logging/            # Structured logging & observability
│   ├── cost/               # Cost tracking & budgeting
│   ├── events/             # Pub/sub event bus
│   ├── types/              # Shared type definitions
│   ├── errors/             # Typed error system
│   ├── constants/          # Global constants
│   ├── tui/                # Terminal UI (Bubbletea)
│   ├── rsi/                # Recursive Self-Improvement
│   ├── ags/                # Autonomous Goal Synthesis
│   ├── uaa/                # Unbounded Autonomous Action
│   ├── ra/                 # Resource Acquisition
│   ├── money/              # Income generation capabilities
│   ├── engineering/        # Software engineering domain
│   ├── creative/           # Creative capabilities
│   ├── research/           # Agentic RAG
│   ├── social/             # Social media & outreach
│   ├── business/           # Business logic
│   ├── planning/           # Planning & self-improvement
│   ├── edge/               # Edge computing & local LLM
│   └── bootstrap/          # Bootstrap utilities
├── web/                    # Web server (HTTP + WebSocket)
├── ui/                     # React frontend (Vite + TypeScript)
├── skills/                 # Built-in skills
├── docs/                   # Documentation
├── grafana/                # Grafana dashboard configs
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
| Volcanic Engine | `openai` | doubao-1-5-pro | No |
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
| **Autonomy** | Trust budget (4-tier action classification) | Limit autonomous actions |
| **Storage** | AES-256-GCM encryption + SHA-256 hash-chained audit log | Protect data at rest |

---

## License

MIT — See [LICENSE](LICENSE) for details.
