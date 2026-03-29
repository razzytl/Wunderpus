# Wunderpus v2 — The Omnipotence Plan

<p align="center">
  <img src="resources/banner.jpg" alt="Wunderpus" width="400"/>
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

Wunderpus is an autonomous AI agent written in Go that can:
- Improve itself through RSI (Recursive Self-Improvement)
- Create its own goals through AGS (Autonomous Goal Synthesis)
- Operate with a Trust Budget to limit autonomous actions
- Acquire cloud resources through RA (Resource Acquisition)

**Plus:** With the Omnipotence Plan (v2), it now has:
- Tool Synthesis Engine (writes its own tools)
- World Model / Knowledge Graph (learns and remembers)
- Computer Use / GUI Control (uses any website)
- Agent Swarm Architecture (teams of specialists)
- Money Making Engine (freelancing, content, trading)
- And much more...

---

## What's New in v2 — The Omnipotence Plan

### Layer 1 — Infrastructure (Completed ✅)

| Component | Files | Tests | Description |
|-----------|-------|-------|-------------|
| **Tool Synthesis** | 10 | 25 | Agent writes its own tools |
| **World Model** | 8 | 22 | Persistent knowledge graph |
| **Perception** | 7 | 18 | Computer use / GUI control |
| **A2A Protocol** | 3 | 10 | Agent-to-agent communication |
| **Swarm** | 4 | 14 | Multi-agent orchestration |

### Layer 2 — Capability Domains (20 sections implemented)

| Domain | Status | Description |
|--------|--------|-------------|
| **Money Engine** | ✅ | Freelancing, content, APIs, trading |
| **Engineering** | ✅ | Project builder, bug hunter, OSS |
| **Creative** | ✅ | Book publisher |
| **Research** | ✅ | Agentic RAG |
| **Security** | ✅ | Reconnaissance |
| **Social** | ✅ | Social media, outreach |
| **Business** | ✅ | Launch, support, compliance |
| **Planning** | ✅ | Project manager, self-improvement |
| **Edge** | ✅ | Minimal mode, local LLM |

---

## Core Architecture

### The RSI Loop (Self-Improvement)

```
Every 100 tasks or 1 hour:
    │
    ▼
Profiler → Weakness Reporter → Proposal Engine →
    │                                    │
    │     3 versions at [0.2, 0.5, 0.8] │
    │                                    │
    ▼                                    ▼
Sandbox Testing ←←←←←←←←←←←←←←←←←←←←←
    │                                    
    ▼                                   
Fitness Gate → Deploy if score > 0.05
```

### The Trust Budget

| Tier | Actions | Cost |
|------|---------|------|
| 1 | Read-only (search, read) | 0 |
| 2 | Ephemeral (build, test) | 1 |
| 3 | Persistent (git commit, write DB) | 5 |
| 4 | External (POST, deploy, spend) | 20 |

### The Autonomy Gate (UAA)

```
Action → Classify → Trust Check → Shadow Sim → Execute
```

Every Tier 3+ action is simulated by Shadow before execution.

---

## Quick Start

### Installation

```bash
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus
go build ./...
```

### Configuration

```bash
cp config.example.yaml config.yaml
```

Enable the capabilities you want:

```yaml
genesis:
  # Layer 1 Infrastructure
  toolsynth_enabled: true
  worldmodel_enabled: true
  perception_enabled: true
  swarm_enabled: true
  
  # Core Autonomy
  rsi_enabled: false        # Enable when ready
  ags_enabled: false        # Goal synthesis
  uaa_enabled: false        # Full autonomy
  ra_enabled: false         # Cloud resources

# Layer 2 - Enable individual domains
# Each has its own config section
```

### Running

```bash
# Interactive TUI
wunderpus

# One-shot
wunderpus agent -m "Your task here"

# Gateway mode
wunderpus gateway
```

---

## Project Structure

```
internal/
  # Layer 1 - Infrastructure
  toolsynth/       # Tool synthesis engine (10 files)
  worldmodel/      # Knowledge graph (8 files)
  perception/      # Computer use / GUI (7 files)
  a2a/             # Agent2Agent protocol
  swarm/           # Multi-agent orchestration
  
  # Layer 2 - Capability Domains
  money/           # Freelance, content, trading (6 files)
  engineering/     # Builder, bug hunter, OSS (4 files)
  creative/        # Book publisher (2 files)
  research/        # Agentic RAG (2 files)
  security/        # Reconnaissance (2 files)
  social/          # Social media, outreach (4 files)
  business/        # Launch, support, compliance (4 files)
  planning/        # Project manager, self-improv (4 files)
  edge/            # Minimal mode, local LLM (3 files)
  
  # Core Systems
  rsi/             # Recursive self-improvement
  ags/             # Autonomous goal synthesis
  uaa/             # Unbounded autonomous action
  ra/              # Resource acquisition
  audit/           # Tamper-evident log
  events/          # Event bus
```

---

## Supported Providers

| Provider | Protocol | Models |
|----------|----------|--------|
| OpenAI | openai | gpt-4o, gpt-4o-mini |
| Anthropic | anthropic | claude-sonnet-4 |
| Google Gemini | gemini | gemini-2.0-flash |
| Ollama | ollama | llama3.2, qwen2.5 (local) |
| OpenRouter | openai | 100+ models |
| Groq | openapi | llama-3.3-70b (fast) |
| DeepSeek | openai | deepseek-r1 |
| Cerebras | openai | llama-3.3-70b (fastest) |
| + 7 more | | See `config.example.yaml` |

---

## Security Features

- **Audit Log**: SHA-256 hash chain with verification
- **Trust Budget**: Actions cost points, lockdown at zero
- **RSI Firewall**: Can only modify `internal/`, never `cmd/`
- **Shadow Mode**: Tier 3+ actions simulated before execution
- **AES-256-GCM**: Credentials encrypted at rest

---

## Documentation

- [Installation](docs/installation.md)
- [Configuration](docs/config.md)
- [Architecture](docs/architecture.md)
- [Skills System](docs/skills.md)
- [CLI Reference](docs/cli.md)
- [Deployment](docs/deployment.md)
- [Troubleshooting](docs/troubleshooting.md)

---

## The Omnipotence Roadmap

The plan has two layers:

1. **Layer 1 — Infrastructure**: The nervous system that makes every capability possible
2. **Layer 2 — Capability Domains**: The actual things Wunderpus can DO

**Completed**: 20 capability domains  
**Pending**: 8 sections (require external APIs, training data, or are experimental)

See `WUNDERPUS_OMNIPOTENCE_PLAN.md` for the full roadmap.

---

## License

MIT. See [LICENSE](LICENSE).