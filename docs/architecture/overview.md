# System Architecture Overview

Wunderpus follows a layered, dependency-injected architecture with a central bootstrap pattern.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                               │
│    wunderpus │ agent │ gateway │ onboard │ status │ skills      │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    Application Bootstrap                         │
│         (internal/app/) — Composition Root                       │
│                                                                  │
│  Config → Logging → DB → Security → Providers → Tools           │
│  → Memory → Skills → ToolSynth → WorldModel → Perception        │
│  → Agent → SubAgent → Heartbeat → Health → Channels             │
└──────┬───────────────────┬───────────────────┬──────────────────┘
       │                   │                   │
┌──────▼──────┐   ┌───────▼───────┐   ┌───────▼──────────────────┐
│ Agent Core  │   │   Channels    │   │   Core Systems           │
│             │   │               │   │                          │
│ • Context   │   │ • Telegram    │   │ • Tool Synthesis         │
│ • Tools     │   │ • Discord     │   │ • World Model            │
│ • Skills    │   │ • Slack       │   │ • Perception             │
│ • RAG       │   │ • WhatsApp    │   │ • AGS (Goals)            │
│ • Streaming │   │ • WebSocket   │   │ • Sub-Agents             │
│ • Branching │   │               │   │ • Heartbeat              │
│ • Checkpoint│   │               │   │ • Webhooks               │
└──────┬──────┘   └───────┬───────┘   └──────────────────────────┘
       │                  │
┌──────▼──────────────────▼──────────────────────────────────────┐
│                    Provider Router                              │
│  OpenAI │ Anthropic │ Gemini │ Ollama │ Groq │ DeepSeek │ ...  │
│                                                                  │
│  • Auto-detect protocol from model prefix                       │
│  • Fallback chain on failure                                    │
│  • 5-minute response cache                                      │
└─────────────────────────────────────────────────────────────────┘
```

## Design Patterns

### Dependency Injection via Bootstrap

The `app.Bootstrap()` function is the single composition root. Every component is constructed in a defined order and passed explicitly to dependents.

### Interface-Based Abstraction

Every major subsystem uses Go interfaces:

| Interface | Purpose | Implementations |
|---|---|---|
| `provider.Provider` | LLM completion | OpenAI, Anthropic, Gemini, Ollama |
| `provider.Embedder` | Vector embeddings | OpenAI, Gemini, Ollama |
| `channel.Channel` | Messaging platform | Telegram, Discord, Slack, WhatsApp, WebSocket |
| `tool.Tool` | Agent actions | file_read, shell_exec, http_request, etc. |
| `skills.SkillRegistry` | Skill distribution | Memory (local) |

### Strategy Pattern (Provider Factory)

Adding a new LLM provider requires **zero code changes** — just a config entry:

```yaml
model_list:
  - model_name: "new-provider"
    model: "protocol/model-id"  # Protocol auto-detected from prefix
    api_key: "key"
```

Supported protocols: `openai`, `anthropic`, `ollama`, `gemini`

### Pipeline Pattern (Tool Synthesis)

The tool synthesis engine uses a 5-stage pipeline:

```
Detect → Design → Code → Test → Register
  │        │       │       │       │
  ▼        ▼       ▼       ▼       ▼
Gap      LLM     LLM     Run     Write
Analysis Spec    Source  Tests   to Disk
```

### Pub/Sub Event Bus

The `events.Bus` provides typed publish/subscribe with:
- Non-blocking handler execution (separate goroutines)
- Priority-based routing (`PriorityHigh` sync, `PriorityNormal` async)
- Panic recovery per handler
- Dead-letter queue (max 1000 entries)

## Data Flow

### Message Processing

```
User Input (any channel)
    │
    ▼
Channel Adapter ──────────────────────────────────┐
    │                                              │
    ▼                                              │
agent.Manager.ProcessMessage(sessionID, input)     │
    │                                              │
    ├── Budget check (cost.Tracker)                │
    ├── Rate limit check                           │
    │                                              │
    ▼                                              │
agent.Agent.HandleMessage(ctx, input)              │
    │                                              │
    ├── 1. Sanitize input                          │
    ├── 2. Add to context (tiktoken counting)      │
    ├── 3. Loop (max 5 iterations):                │
    │     ├── Build messages (system + context)    │
    │     ├── Attach tool schemas                  │
    │     ├── Check response cache                 │
    │     ├── provider.Router.CompleteWithFallback │
    │     ├── If tool calls:                       │
    │     │     ├── Execute in parallel            │
    │     │     ├── Audit log each execution       │
    │     │     └── Add results to context         │
    │     └── If no tool calls: return response    │
    │                                              │
    ▼                                              │
Response ──────────────────────────────────────────┘
    │
    ▼
Channel (format & send back to user)
```

### Complex Task Flow (Multi-Agent)

```
Input
    │
    ▼
TaskPlanner.Decompose()
    │  (LLM generates dependency graph)
    ▼
Orchestrator.Execute()
    │  (Concurrent worker arms with scoped tools)
    ├── Worker Arm 1 (io-scoped tools)
    ├── Worker Arm 2 (compute-scoped tools)
    └── Worker Arm 3 (general-scoped tools)
    │
    ▼
Synthesizer.Merge()
    │  (LLM combines sub-agent results)
    ▼
Final Response
```

## Persistence Model

Wunderpus uses **exactly 2 SQLite databases** with namespaced tables:

| Database | Purpose | Table Prefixes |
|---|---|---|
| `wunderpus.db` | Core data | `mem_` (memory), `wm_` (world model), `cost_` (cost tracking), `task_checkpoints` |
| `wunderpus-audit.db` | Tamper-evident audit log | `audit_log` (hash-chained entries) |

All databases support WAL mode for concurrent reads.
