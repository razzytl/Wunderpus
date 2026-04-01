# System Architecture Overview

Wunderpus follows a layered, dependency-injected architecture with a central bootstrap pattern.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                               │
│    wunderpus │ agent │ gateway │ onboard │ status │ skills │ cron│
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    Application Bootstrap                         │
│         (internal/app/) — Composition Root                       │
│                                                                  │
│  12-step initialization:                                         │
│  Config → Logging → Security → Providers → Tools → Memory       │
│  → Skills → ToolSynth → WorldModel → Perception → Agent         │
│  → SubAgent → Swarm → Heartbeat → Health → Channels             │
└──────┬───────────────────┬───────────────────┬──────────────────┘
       │                   │                   │
┌──────▼──────┐   ┌───────▼───────┐   ┌───────▼──────────────────┐
│ Agent Core  │   │   Channels    │   │   Genesis Systems        │
│             │   │               │   │                          │
│ • Context   │   │ • Telegram    │   │ • RSI (Self-Improve)     │
│ • Tools     │   │ • Discord     │   │ • AGS (Goal Synthesis)   │
│ • Skills    │   │ • Slack       │   │ • UAA (Autonomous Action)│
│ • RAG       │   │ • WhatsApp    │   │ • RA (Resource Acq.)     │
│ • Streaming │   │ • WebSocket   │   │                          │
└──────┬──────┘   └───────┬───────┘   └──────────────────────────┘
       │                  │
┌──────▼──────────────────▼──────────────────────────────────────┐
│                    Provider Router                              │
│  OpenAI │ Anthropic │ Gemini │ Ollama │ Groq │ DeepSeek │ ...  │
│                                                                  │
│  • Auto-detect protocol from model prefix                       │
│  • Fallback chain on failure                                    │
│  • Parallel probing for fastest response                        │
│  • 5-minute response cache                                      │
└─────────────────────────────────────────────────────────────────┘
```

## Design Patterns

### Dependency Injection via Bootstrap

The `app.Bootstrap()` function is the single composition root. Every component is constructed in a defined order and passed explicitly to dependents:

```go
// Simplified bootstrap flow
config := config.Load(path)
logger := logging.Init(config.Logging)
security := security.Init(config.Security)
providers := provider.NewRouter(config)
tools := tool.NewRegistry()
memory := memory.NewStore(config)
agent := agent.NewAgent(providers, tools, memory, security)
```

### Interface-Based Abstraction

Every major subsystem uses Go interfaces:

| Interface | Purpose | Implementations |
|---|---|---|
| `provider.Provider` | LLM completion | OpenAI, Anthropic, Gemini, Ollama |
| `provider.Embedder` | Vector embeddings | OpenAI, Gemini, Ollama |
| `channel.Channel` | Messaging platform | Telegram, Discord, Slack, WhatsApp, etc. |
| `tool.Tool` | Agent actions | file_read, shell_exec, http_request, etc. |
| `skills.SkillRegistry` | Skill distribution | Memory, ClawHub |

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

## Component Relationships

```
                    ┌─────────────┐
                    │   Config    │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
        ┌─────▼─────┐ ┌───▼────┐ ┌────▼─────┐
        │ Providers │ │ Tools  │ │ Channels │
        └─────┬─────┘ └───┬────┘ └────┬─────┘
              │           │           │
              └───────────┼───────────┘
                          │
                    ┌─────▼─────┐
                    │   Agent   │
                    └─────┬─────┘
                          │
              ┌───────────┼───────────┐
              │           │           │
        ┌─────▼─────┐ ┌───▼────┐ ┌────▼─────┐
        │  Memory   │ │ Skills │ │  Cost    │
        └───────────┘ └────────┘ └──────────┘
```

## Extension Points

### Add a New Provider

1. Implement `provider.Provider` interface
2. Register in `provider.NewFromModelEntry()` factory
3. Add to `config.example.yaml`

### Add a New Channel

1. Create directory in `internal/channel/`
2. Implement `channel.Channel` interface (`Start`, `Stop`, `Name`)
3. Register in `app.Bootstrap()`

### Add a New Tool

1. Implement `tool.Tool` interface (`Name`, `Description`, `Parameters`, `Execute`, `Sensitive`, `Version`)
2. Register in tool registry during bootstrap
3. Tool is automatically available to the agent

### Add a New Skill

1. Create directory: `skills/my-skill/`
2. Add `SKILL.md` with frontmatter:
   ```yaml
   ---
   name: my-skill
   description: "What this skill does"
   ---
   ```
3. Document usage in the markdown body

## Persistence Model

All persistence uses SQLite — no external database required:

| Database | Purpose | Package |
|---|---|---|
| `wonderpus_memory.db` | Sessions, messages, preferences | `memory/` |
| `wonderpus_audit.db` | Tamper-evident audit log | `audit/` |
| `wunderpus_cost.db` | Cost tracking, budgets | `cost/` |
| `wunderpus_worldmodel.db` | Knowledge graph | `worldmodel/` |
| `wunderpus_profiler.db` | RSI performance metrics | `rsi/` |
| `wunderpus_trust.db` | Trust budget | `uaa/` |
| `wunderpus_resources.db` | Provisioned resources | `ra/` |

All databases support WAL mode for concurrent reads.
