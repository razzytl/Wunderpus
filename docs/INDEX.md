# Wunderpus Documentation

> **Version 2.0** — Universal Autonomous AI Agent Framework

## Documentation Map

### 🚀 Getting Started

| Document | Purpose |
|---|---|
| [Getting Started](guides/getting-started.md) | Install, configure, and run Wunderpus in 5 minutes |

### 🏗️ Architecture

| Document | Purpose |
|---|---|
| [System Overview](architecture/overview.md) | High-level architecture, design patterns, and data flow |
| [Agent Core](architecture/agent-core.md) | Agent loop, context management, branching, checkpoints, tool execution |
| [Provider System](architecture/provider-system.md) | LLM provider routing, fallback, and caching |
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
| [Deployment Guide](operations/deployment.md) | Production deployment: Docker, systemd |
| [Monitoring & Observability](operations/monitoring.md) | Health dashboard, OpenTelemetry, Prometheus metrics, logging |
| [Security Guide](guides/security.md) | Security model, hardening, best practices |
| [Troubleshooting](operations/troubleshooting.md) | Common issues and solutions |

---

## Architecture Quick Reference

```
CLI Layer → Bootstrap → Agent Core → Provider Router → LLM APIs
                      ↕
              Channels │ Tools │ Skills │ Memory │ World Model
```

## Key Concepts

| Concept | Description |
|---|---|
| **Provider** | An LLM backend (OpenAI, Anthropic, etc.) — Wunderpus supports 15+ via protocol-based routing |
| **Channel** | A communication platform (Telegram, Discord, etc.) — connect multiple simultaneously |
| **Tool** | An action the agent can perform (read file, run command, make HTTP request) |
| **Skill** | A markdown-based capability extension that guides agent behavior |
| **Session** | An isolated conversation context with its own history and preferences |
| **Branch** | A diverged conversation path from any message, enabling exploration of alternatives |
| **Checkpoint** | A persisted snapshot of agent context for crash-resilient resume |
| **Approval Level** | Policy-based tool classification: AutoExecute, NotifyOnly, RequiresApproval, Blocked |
| **World Model** | A persistent knowledge graph that learns from every interaction |
| **AGS** | Autonomous Goal Synthesis — goal creation and execution triggered via API/CLI/Cron only |

---

## Version Information

This documentation corresponds to **Wunderpus v2.0** with the following systems implemented:

| System | Status |
|---|---|
| Core Agent | ✅ Stable |
| Provider Router (15+ providers) | ✅ Stable |
| Channel System (5 channels) | ✅ Stable |
| Tool System (15+ tools) | ✅ Stable |
| Skills System | ✅ Stable |
| Memory & RAG | ✅ Stable |
| World Model | ✅ Stable |
| Perception (Computer Use) | ✅ Stable |
| Tool Synthesis | ✅ Stable |
| AGS (Goals — manual trigger) | ✅ Stable |
| Conversation Branching | ✅ Stable |
| Multi-Modal Input Detection | ✅ Stable |
| Structured Output Enforcement | ✅ Stable |
| Cost Prediction | ✅ Stable |
| Checkpoint & Resume | ✅ Stable |
| OpenTelemetry Tracing | ✅ Stable |
| Health Dashboard | ✅ Stable |
| Prompt Versioning | ✅ Stable |
| Webhook System | ✅ Stable |
| Sub-Agent Orchestration | ✅ Stable |
| Heartbeat Scheduler | ✅ Stable |
