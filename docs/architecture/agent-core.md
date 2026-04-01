# Agent Core

The agent is the central processing unit of Wunderpus — it receives messages, manages context, executes tools, and generates responses.

## Agent Loop

```
User Message
    │
    ▼
┌─────────────────────────────────────────┐
│ 1. Sanitize Input                        │
│    - Unicode normalization               │
│    - Injection pattern detection         │
│    - Block high-severity threats         │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│ 2. Add to Context                        │
│    - tiktoken counting (cl100k_base)    │
│    - SQLite persistence                 │
│    - Optional AES encryption            │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│ 3. Loop (max 5 iterations)              │
│                                         │
│   a. Build messages                     │
│      - System prompt                    │
│      - Conversation context             │
│      - RAG SOPs (if relevant)           │
│      - Tool schemas                     │
│                                         │
│   b. Check response cache (5-min TTL)   │
│                                         │
│   c. provider.Router.CompleteWithFallback│
│      - Try active provider              │
│      - Fallback to model chain          │
│      - Fallback to other providers      │
│                                         │
│   d. If tool calls:                     │
│      - Execute in parallel              │
│      - Approval gate (sensitive tools)  │
│      - Timeout enforcement              │
│      - Audit log                        │
│      - Add results to context           │
│      - Check summarization (>80%)       │
│      - Continue loop                    │
│                                         │
│   e. If no tool calls:                  │
│      - Return response                  │
└─────────────────────────────────────────┘
```

## Context Management

### Token Counting

Uses tiktoken (cl100k_base encoding) for accurate token counting:

```go
// Automatic truncation when context exceeds limit
if contextManager.NeedsSummarization() {
    // Summarize oldest messages
    summary := llm.Summarize(oldestMessages)
    contextManager.ReplaceWithSummary(summary)
}
```

### Context Lifecycle

| Stage | Action |
|---|---|
| New message | Add to context, count tokens |
| > 80% capacity | Trigger summarization |
| > 100% capacity | Truncate oldest messages (keep min 2) |
| Session end | Persist to SQLite |

## Tool Execution

### Parallel Execution

Tools are executed concurrently using `sync.WaitGroup`:

```go
var wg sync.WaitGroup
for _, call := range toolCalls {
    wg.Add(1)
    go func(call ToolCall) {
        defer wg.Done()
        result := executor.Execute(ctx, call)
        results = append(results, result)
    }(call)
}
wg.Wait()
```

### Tool Scopes (Multi-Agent)

In multi-agent orchestration, worker arms receive scoped tool registries:

| Worker Type | Available Tools |
|---|---|
| I/O | http_request, file_read, file_write, file_list |
| Compute | calculator, system_info |
| General | All tools |

This prevents lateral drift and reduces attack surface.

### Approval Gates

Sensitive tools require human approval:

```yaml
tools:
  sensitive_tools:
    - shell_exec
    - http_request
```

When a sensitive tool is called:
1. Execution pauses
2. Human is notified (via TUI or channel)
3. Human approves or denies
4. Execution continues or aborts

## Session Management

The `agent.Manager` handles multiple concurrent sessions:

```go
// Get or create agent for session
ag := manager.GetAgent(sessionID)

// Process message
resp, err := manager.ProcessMessage(ctx, sessionID, input)
```

### Session Isolation

Each session has:
- Independent conversation context
- Separate token counter
- Individual rate limit tracking
- Isolated cost tracking

## Streaming

Providers that support streaming deliver tokens incrementally:

```go
// Streaming response
err := agent.StreamMessage(ctx, input, func(token string) {
    // Display token in real-time
    fmt.Print(token)
})
```

## Complex Task Orchestration

For complex goals, the agent decomposes and delegates:

```
Input: "Build a REST API with authentication"
    │
    ▼
TaskPlanner.Decompose()
    │
    │  LLM generates:
    │  ├── Task 1: Design API schema (compute)
    │  ├── Task 2: Implement endpoints (compute)
    │  ├── Task 3: Add auth middleware (compute)
    │  └── Task 4: Write tests (compute)
    │
    ▼
Orchestrator.Execute(dependencyGraph)
    │
    │  Workers execute in parallel where possible:
    │  ├── Task 1 ──┐
    │  ├── Task 2 ──┼── (depends on Task 1)
    │  ├── Task 3 ──┤
    │  └── Task 4 ──┘  (depends on Tasks 2, 3)
    │
    ▼
Synthesizer.Merge(results)
    │
    ▼
"API built with 4 endpoints, JWT auth, and 12 tests"
```

## Configuration

```yaml
agent:
  system_prompt: "You are Wunderpus, a helpful AI assistant..."
  max_context_tokens: 8000
  temperature: 0.7

tools:
  enabled: true
  timeout_seconds: 30
```
