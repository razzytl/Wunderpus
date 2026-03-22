# Architecture Overview

This document provides a detailed overview of the Wunderpus system architecture, explaining the design decisions, component interactions, and extension points.

## High-Level Architecture

Wunderpus follows a layered architecture pattern with clear separation of concerns:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              CLI Layer                                        │
│                    (Cobra Commands: agent, gateway, skills,                  │
│                        cron, auth, onboard, status)                           │
└──────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                           Application Layer                                   │
│                                                                               │
│    ┌─────────────────────────────────────────────────────────────────────┐   │
│    │                      Application Bootstrap                           │   │
│    │         (Configuration loading, dependency injection,              │   │
│    │          logging initialization, health server startup)             │   │
│    └─────────────────────────────────────────────────────────────────────┘   │
│                                       │                                       │
│          ┌────────────────────────────┼────────────────────────────┐        │
│          │                            │                            │        │
│          ▼                            ▼                            ▼        │
│    ┌─────────────┐            ┌─────────────┐              ┌─────────────┐  │
│    │   Agent     │            │  Channel    │              │   Skills    │  │
│    │  Manager    │            │  Manager    │              │   Loader    │  │
│    └─────────────┘            └─────────────┘              └─────────────┘  │
└──────────────────────────────────────────────────────────────────────────────┘
                     │                    │                    │
                     ▼                    ▼                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                           Provider Layer                                      │
│                                                                               │
│   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐         ┌─────────┐     │
│   │  OpenAI │  │Anthropic│  │  Gemini │  │ Ollama  │  ...    │ [15+]   │     │
│   └─────────┘  └─────────┘  └─────────┘  └─────────┘         └─────────┘     │
└──────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Infrastructure Layer                                │
│                                                                               │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│   │  SQLite  │  │  Memory  │  │  Health  │  │   Log    │  │  Config  │      │
│   │  (Audit) │  │  Store   │  │  Server  │  │  Output  │  │  Loader  │      │
│   └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘      │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Core Components

### Application Bootstrap

The bootstrap process in `internal/app/bootstrap.go` initializes all components:

1. **Configuration Loading**: Parses YAML configuration, validates schema, loads environment variables
2. **Logging Initialization**: Sets up structured logging with zerolog
3. **Security Initialization**: Configures encryption, audit logging, rate limiting
4. **Provider Initialization**: Creates provider adapters based on configuration
5. **Memory Store**: Initializes SQLite session storage
6. **Agent Manager**: Sets up session management and tool execution
7. **Channel Manager**: Initializes communication channels
8. **Skills Loader**: Discovers and loads skill packages
9. **Health Server**: Starts HTTP health check endpoint

### Agent Manager

The Agent Manager (`internal/agent/manager.go`) is the core orchestration component:

**Responsibilities:**
- Session lifecycle management
- Provider selection and fallback logic
- Tool execution coordination
- Context window management
- Cost tracking

**Session Management:**
Each conversation is represented by a Session with:
- Unique identifier
- Message history
- Provider preferences
- Tool execution context
- Token usage tracking

**Provider Selection:**
The manager selects providers based on:
1. Explicit provider specification in request
2. Default provider from configuration
3. Fallback chain on provider failure
4. Parallel probing for fastest response (when enabled)

### Provider System

The provider abstraction (`internal/provider/`) enables vendor-agnostic LLM interactions:

**Provider Interface:**
```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req *Request) (*Response, error)
    Embed(ctx context.Context, text string) ([]float64, error)
    SupportsStreaming() bool
}
```

**Provider Implementations:**
- `openai.go`: OpenAI API (GPT-4, GPT-4o)
- `anthropic.go`: Anthropic Claude API
- `gemini.go`: Google Gemini API
- `ollama.go`: Local Ollama instances
- `openrouter.go`: OpenRouter aggregation
- `groq.go`: Groq inference
- `deepseek.go`: DeepSeek models
- `cerebras.go`: Cerebras inference
- And more...

**Fallback System:**
Providers can be configured with fallback chains:
```yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    fallback_models:
      - "anthropic/claude-sonnet-4"
      - "openrouter/deepseek/deepseek-r1"
```

When the primary provider fails, the system automatically tries each fallback in sequence with configurable cooldown periods.

### Channel System

Channels (`internal/channel/`) bridge communication platforms with the agent:

**Channel Interface:**
```go
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop() error
    Send(msg *Message) error
    Receive() (<-chan *Message, error)
}
```

**Implemented Channels:**
- `tui/`: Terminal user interface (charmbracelet/bubbletea)
- `telegram/`: Bot API integration
- `discord/`: Discord bot with slash commands
- `websocket/`: WebSocket server for custom clients
- `qq/`: QQ bot protocol
- `wecom/`: WeChat Work integration
- `dingtalk/`: DingTalk webhook
- `onebot/`: OneBot v11 protocol

**Message Flow:**
1. Channel receives message from platform
2. Message parsed into standardized format
3. Message queued for agent processing
4. Agent generates response
5. Response formatted for channel
6. Response sent back to platform

### Skills System

The skills system (`internal/skills/`) provides extensibility:

**Skill Structure:**
```
skills/
  github/
    SKILL.md          # Manifest and documentation
    handler.go        # Optional custom Go handler
    tools/            # Additional tools
  weather/
    SKILL.md
  ...
```

**Skill Manifest (SKILL.md):**
```markdown
---
name: github
description: "Interact with GitHub using gh CLI"
metadata:
  version: "1.0.0"
  author: "Wunderpus Team"
---

# GitHub Skill

Usage documentation...
```

**Skill Loading:**
1. Scan configured skill directories
2. Parse SKILL.md manifest
3. Register skill metadata
4. Load custom handlers if present
5. Make available to agent

**Skill Execution:**
- Skills provide structured prompts for agent behavior
- Can define tool dependencies
- Version checking for compatibility
- Registry support for remote skill distribution

### Tool System

The tool system (`internal/tool/`) enables agent actions:

**Tool Interface:**
```go
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
    Schema() *ToolSchema
}
```

**Built-in Tools:**
- `shell/`: Sandboxed shell command execution
- `file/`: File read, write, search operations
- `mcp/`: Model Context Protocol client
- `search/`: Web search (Brave, Tavily, DuckDuckGo)
- `message/`: Sub-agent communication
- `spawn/`: Sub-agent creation
- `http/`: HTTP request execution

**Tool Execution Pipeline:**
1. Agent requests tool execution
2. Tool system validates arguments against schema
3. Security checks (sandbox, SSRF protection)
4. Tool executes with timeout
5. Result returned to agent

**Shell Sandboxing:**
Commands are filtered against allowlist:
```yaml
tools:
  shell_whitelist:
    - git
    - go
    - npm
```

Regex patterns also block dangerous commands:
- Command injection patterns
- Destructive operations (rm -rf /)
- Network operations to blocked destinations

### Memory System

The memory system (`internal/memory/`) manages conversation state:

**Storage Backend:**
- SQLite for persistent session storage
- In-memory cache for active sessions
- Automatic token limit management

**Token Management:**
- Track token usage per session
- Automatic pruning when limit reached
- Configurable context window size

### Security System

The security system (`internal/security/`) provides enterprise features:

**Components:**
- `encryption/`: AES-256-GCM encryption for sensitive data
- `audit/`: SQLite-based audit logging
- `ratelimit/`: Token bucket rate limiting
- `sanitizer/`: Input validation and sanitization

### Heartbeat System

The heartbeat system (`internal/heartbeat/`) enables periodic tasks:

**Configuration:**
```yaml
heartbeat:
  enabled: true
  interval: 30  # minutes
```

**Task Definition (HEARTBEAT.md):**
```markdown
## 2024-01-15 09:00
- [ ] Daily report generation

## Weekly Review
- [ ] Review open PRs
```

**Execution:**
- Timer-based interval checking
- Natural language cron parsing ("at 9am daily")
- Async task execution
- Status reporting

## Data Flow

### User Message Flow

```
User (Telegram/Discord/TUI)
        │
        ▼
Channel (protocol handler)
        │
        ▼
Message Queue
        │
        ▼
Agent Manager
        │
        ├──────────────────┐
        ▼                  ▼
Provider (LLM)        Tools (execute)
        │                  │
        └──────────────────┘
                │
                ▼
        Response
                │
                ▼
Channel (format & send)
                │
                ▼
User
```

### Provider Request Flow

```
Agent Manager
        │
        ▼
Session (load context)
        │
        ▼
Provider Selector
        │
        ├──────────────────┐
        ▼                  ▼
Primary Provider    Fallback Providers
        │                  │
        └────────┬─────────┘
                 ▼
         Response + Cost Tracking
                 │
                 ▼
         Memory Store (save)
```

## Extension Points

### Custom Providers

Implement the `Provider` interface and register in the bootstrap:
```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req *Request) (*Response, error)
    Embed(ctx context.Context, text string) ([]float64, error)
}
```

### Custom Channels

Implement the `Channel` interface:
```go
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop() error
    Send(msg *Message) error
}
```

### Custom Tools

Implement the `Tool` interface:
```go
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
    Schema() *ToolSchema
}
```

### Custom Skills

Create a skill directory with SKILL.md manifest:
- Define skill behavior through prompts
- Optionally implement custom Go handlers
- Declare dependencies on tools or other skills

## Configuration Model

The configuration flows through the system:

```
config.yaml
    │
    ▼
Config Loader (internal/config/)
    │
    ├─────────────────────────────┐
    │                             │
    ▼                             ▼
Provider Config              Channel Config
    │                             │
    ▼                             ▼
Provider Adapter            Channel Adapter
    │                             │
    └────────────┬──────────────┘
                 ▼
         Application Context
```

## Deployment Architecture

### Development

```
Developer
    │
    ▼
Terminal (TUI)
    │
    ▼
wunderpus agent
```

### Production

```
Users ──┬──> Telegram Bot
        │
        ├──> Discord Bot
        │
        ├──> WebSocket Clients
        │
        └──> Custom Channels

              │
              ▼
        wunderpus gateway
              │
        ┌─────┼─────┬──────────┐
        │     │     │          │
        ▼     ▼     ▼          ▼
    Provider  DB   Skills   Channels
    APIs   (SQLite)
```

### Containerized

```
┌─────────────────────────┐
│    Docker Container     │
│                         │
│  ┌───────────────────┐  │
│  │   wunderpus       │  │
│  │   gateway         │  │
│  └───────────────────┘  │
│           │             │
│    ┌──────┼──────┐      │
│    ▼      ▼      ▼      │
│  DB    Config  Logs     │
└─────────────────────────┘
           │
           ▼
    External Services
    (LLM APIs, Telegram,
     Discord, etc.)
```

## Performance Considerations

### Latency Optimization

- **Parallel Provider Probing**: Send requests to multiple providers simultaneously, use fastest response
- **Connection Pooling**: Reuse HTTP connections to providers
- **Caching**: Cache provider capabilities and model lists

### Resource Management

- **Token Limits**: Automatic pruning of conversation history
- **Timeout Configuration**: Per-provider and global timeouts
- **Rate Limiting**: Prevent quota exhaustion

### Scalability

- **Stateless Gateway**: Multiple gateway instances can run behind a load balancer
- **Channel Isolation**: Channel failures don't affect each other
- **Provider Independence**: Provider issues don't cascade

## Monitoring and Observability

### Metrics

Prometheus metrics available:
- Request count by provider
- Token usage and cost
- Response latency
- Error rates
- Channel message counts
- Tool execution times

### Logging

Structured JSON logging with:
- Request/response correlation IDs
- Provider, channel, tool context
- Error stack traces
- Configurable log levels

### Health Checks

HTTP health endpoint at `/health`:
- Overall status
- Provider connectivity
- Channel status
- Database connectivity

---

## Genesis Plan — Autonomous Architecture

The Wunderpus Genesis Plan adds four autonomous capabilities to the existing agent framework:

### Overview

| Pillar | Package | Purpose |
|--------|---------|---------|
| **RSI** (Recursive Self-Improvement) | `internal/rsi/` | Agent rewrites and upgrades its own code via AST analysis, LLM proposals, sandboxed testing, and fitness-gated deployment |
| **AGS** (Autonomous Goal Synthesis) | `internal/ags/` | Agent sets its own goals from episodic memory patterns, prioritizes them, executes them, and adjusts its own goal-scoring weights |
| **UAA** (Unbounded Autonomous Action) | `internal/uaa/` | Agent acts autonomously through a trust budget system with 4-tier action classification and shadow mode simulation |
| **RA** (Resource Acquisition) | `internal/ra/` | Agent provisions its own compute, manages API keys, routes LLM requests, and forecasts resource needs |

### Shared Infrastructure

All pillars share:
- **Audit Log** (`internal/audit/`) — SHA-256 hash-chained, append-only, SQLite-backed
- **Event Bus** (`internal/events/`) — Typed pub/sub with dead-letter queue
- **Profiler** (`internal/rsi/profiler.go`) — Ring-buffer P99 latency tracking

### Key Files

| File | Description |
|------|-------------|
| `WUNDERPUS_GENESIS_PLAN.md` | Architectural blueprint — approach comparisons, design decisions, implementation roadmaps |
| `WUNDERPUS_CHECKLIST.md` | 187-item implementation checklist with code/test/infra gates per phase |
| `WUNDERPUS_IMPLEMENTATION_SUMMARY.md` | What was built — deliverables, design decisions, test results, quality audit |

### Phase Status

| Phase | Description | Status |
|-------|-------------|--------|
| 0 | Foundations (audit, events, profiler, trust, classifier) | ✅ Complete |
| 1 | RSI (code mapper, weakness report, proposal engine, sandbox, fitness, deployer) | ✅ Complete |
| 2 | AGS (goal model, scorer, synthesizer, executor, metacognition) | ✅ Complete |
| 3 | UAA+RA (shadow mode, executor, resource registry, key manager, cloud adapter, forecaster) | ✅ Complete |
| 4 | Sovereignty (WASM sandbox, self-referential RSI, multi-agent, financial, bootstrap) | ✅ Complete |

For detailed implementation notes, see `WUNDERPUS_IMPLEMENTATION_SUMMARY.md`.
