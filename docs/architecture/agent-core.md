# Agent Core

The Agent Core is the central processing unit of Wunderpus — it receives messages, manages context, executes tools, and orchestrates multi-agent workflows.

## Components

| Component | File | Purpose |
|---|---|---|
| `Agent` | `internal/agent/agent.go` | Core agent loop — message processing, tool execution, streaming |
| `ContextManager` | `internal/agent/context.go` | Conversation history with tiktoken counting and SQLite persistence |
| `Manager` | `internal/agent/manager.go` | Multi-session agent factory, budget checking, RAG integration |
| `StructuredOutputEnforcer` | `internal/agent/structured_output.go` | JSON validation with retry logic |
| `CheckpointStore` | `internal/agent/checkpoint.go` | Crash-resilient task checkpoints |
| `TaskPlanner` | `internal/agent/planner.go` | Decomposes complex tasks into dependency graphs |
| `Orchestrator` | `internal/agent/orchestrator.go` | Executes task graphs with scoped worker arms |
| `WorkerArm` | `internal/agent/worker.go` | Individual sub-agent with tool scoping |

## Agent Loop

```
HandleMessage(input)
    │
    ├── Rate limit check
    ├── Sanitize input (prompt injection detection)
    ├── Audit log input
    ├── Add user message to context
    │
    └── Loop (max MaxIterations):
        │
        ├── Build messages (system prompt + context + SOPs)
        ├── Select active provider
        ├── Attach tool schemas
        ├── Check response cache
        ├── provider.CompleteWithFallback()
        │
        ├── If no tool calls → return response
        │
        └── If tool calls:
            │
            ├── Add tool-call request to context
            ├── Execute tools in parallel (goroutines)
            │   ├── Policy-based approval check
            │   ├── Sandbox validation
            │   ├── Tool.Execute()
            │   └── Audit log result
            ├── Add tool results to context
            ├── Check summarization threshold
            └── Continue loop
```

## Context Management

The `ContextManager` handles:

- **Token-based truncation** using tiktoken (cl100k_base encoding)
- **SQLite persistence** — messages saved on every addition
- **AES-256-GCM encryption** at rest (optional)
- **Conversation branching** — create, switch, and navigate branches
- **Automatic summarization** when context exceeds 80% capacity

### Conversation Branching

Messages support `parent_message_id` and `branch_id` fields:

```
Session: abc123
├── Branch: main
│   ├── Message 1: "Help me write code"
│   ├── Message 2: "Here's the code..."
│   └── Message 3: "Can you optimize it?"
│
└── Branch: branch-abc123-2  (branched from message 2)
    ├── Message 2: "Here's the code..."
    ├── Message 4: "Try a different approach..."
    └── Message 5: "This version is faster"
```

API:
- `POST /api/branches` — create branch from message X
- `GET /api/branches/messages` — retrieve branch messages
- WebSocket: `branch_switch`, `list_branches` message types

### Checkpoint & Resume

After every tool execution step, a checkpoint can be saved:

```go
type CheckpointSnapshot struct {
    Messages  []provider.Message
    BranchID  string
    SessionID string
    StepDesc  string
}
```

On startup, `ScanRunningCheckpoints()` finds tasks with `status = 'running'` and `ResumeTask()` rehydrates the agent context.

## Structured Output Enforcement

When a tool or agent expects JSON output:

1. LLM generates response
2. `json.Valid()` validates the response
3. If invalid → append correction prompt, retry (configurable max retries)
4. Each retry counts against the iteration budget

```go
enforcer := NewStructuredOutputEnforcer(2) // max 2 retries
resp, retries, err := enforcer.ExecuteWithValidation(ctx, completeFn, messages, OutputFormat{
    Type:       "json",
    JSONSchema: `{"type": "object", "properties": {...}}`,
})
```

## Multi-Agent Orchestration

For complex tasks, the agent loop is bypassed in favor of a task graph:

1. **TaskPlanner** decomposes the input into a `TaskGraph` with `Subtask` nodes and dependencies
2. **Orchestrator** resolves the graph topologically and executes independent subtasks concurrently
3. **WorkerArm** instances run with scoped tool access (e.g., `io-scoped`, `compute-scoped`)
4. Results are merged by the synthesizer into a final response

## Manager

The `Manager` handles multiple agent instances (one per session):

- **Lazy initialization** — agents created on first message
- **Budget checking** — `cost.Tracker.IsOverBudget()` before processing
- **Rate limiting** — per-session rate limiting via `security.RateLimiter`
- **RAG integration** — SOP retrieval via `EnhancedStore.GetRelevantSOPs()`
