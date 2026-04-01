# Wunderpus Documentation

> **Version 2.0** — Universal Autonomous AI Agent Framework

## Documentation Map

### 🚀 Getting Started

| Document | Purpose |
|---|---|
| [Getting Started](guides/getting-started.md) | Install, configure, and run Wunderpus in 5 minutes |
| [Quick Reference](reference/quick-reference.md) | One-page cheat sheet for common operations |

### 🏗️ Architecture

| Document | Purpose |
|---|---|
| [System Overview](architecture/overview.md) | High-level architecture, design patterns, and data flow |
| [Agent Core](architecture/agent-core.md) | Agent loop, context management, tool execution, orchestration |
| [Provider System](architecture/provider-system.md) | LLM provider routing, fallback, caching, and parallel probing |
| [Genesis Autonomy](architecture/genesis-autonomy.md) | RSI, AGS, UAA, and RA — the four pillars of autonomous operation |
| [Memory & Knowledge](architecture/memory-knowledge.md) | Session storage, RAG, world model, and SOP retrieval |

### 📖 Reference

| Document | Purpose |
|---|---|
| [Configuration Reference](reference/configuration.md) | Complete config.yaml reference with all options |
| [CLI Reference](reference/cli.md) | All CLI commands, flags, and subcommands |
| [Provider Reference](reference/providers.md) | Supported LLM providers and configuration |
| [Channel Reference](reference/channels.md) | Messaging platform integrations |
| [Skills Reference](reference/skills.md) | Skills system — creating, installing, managing |
| [Tool Reference](reference/tools.md) | Built-in tools and tool system |
| [Heartbeat Reference](reference/heartbeat.md) | Periodic task scheduling |

### 🔧 Operations

| Document | Purpose |
|---|---|
| [Deployment Guide](operations/deployment.md) | Production deployment: Docker, systemd, Kubernetes |
| [Monitoring & Observability](operations/monitoring.md) | Prometheus metrics, health checks, logging |
| [Security Guide](guides/security.md) | Security model, hardening, best practices |
| [Troubleshooting](operations/troubleshooting.md) | Common issues and solutions |

---

## Architecture Quick Reference

```
CLI Layer → Bootstrap → Agent Core → Provider Router → LLM APIs
                      ↕
              Channels │ Tools │ Skills │ Memory
                      ↕
              Genesis Systems (RSI · AGS · UAA · RA)
```

## Key Concepts

| Concept | Description |
|---|---|
| **Provider** | An LLM backend (OpenAI, Anthropic, etc.) — Wunderpus supports 20+ via protocol-based routing |
| **Channel** | A communication platform (Telegram, Discord, etc.) — connect multiple simultaneously |
| **Tool** | An action the agent can perform (read file, run command, make HTTP request) |
| **Skill** | A markdown-based capability extension that guides agent behavior |
| **Session** | An isolated conversation context with its own history and preferences |
| **Genesis** | The autonomous operation system with four pillars: RSI, AGS, UAA, RA |
| **Trust Budget** | A points-based system that limits what autonomous actions the agent can take |
| **World Model** | A persistent knowledge graph that learns from every interaction |

---

## Version Information

This documentation corresponds to **Wunderpus v2.0** with all core systems implemented:

| System | Status |
|---|---|
| Core Agent | ✅ Stable |
| Provider Router (20+ providers) | ✅ Stable |
| Channel System (11 channels) | ✅ Stable |
| Tool System (15+ tools) | ✅ Stable |
| Skills System | ✅ Stable |
| Memory & RAG | ✅ Stable |
| World Model | ✅ Stable |
| Perception (Computer Use) | ✅ Stable |
| Swarm (Multi-Agent) | ✅ Stable |
| Tool Synthesis | ✅ Stable |
| Genesis — RSI | ✅ Implemented |
| Genesis — AGS | ✅ Implemented |
| Genesis — UAA | ✅ Implemented |
| Genesis — RA | ✅ Implemented |
