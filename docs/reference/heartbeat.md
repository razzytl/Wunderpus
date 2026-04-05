# Heartbeat Reference

Wunderpus can periodically execute tasks defined in `HEARTBEAT.md` in your workspace directory.

## Overview

The heartbeat scheduler:
1. Parses `HEARTBEAT.md` at configured intervals
2. Executes quick tasks directly via the agent
3. Spawns sub-agents for long tasks (parallel execution)
4. Reports results

## Configuration

```yaml
heartbeat:
  enabled: true
  interval: 30  # Minutes (minimum: 5)
```

### Environment Variables

```bash
WUNDERPUS_HEARTBEAT_ENABLED=true
WUNDERPUS_HEARTBEAT_INTERVAL=30
```

## HEARTBEAT.md Format

```markdown
# Heartbeat — Periodic Tasks

## Quick Tasks (respond directly)

These tasks are executed immediately and results returned directly.

- Report the current date and time
- Summarize the current conversation context

## Long Tasks (spawn subagents)

These tasks are executed asynchronously using spawned subagents.

- Search for recent technology news and provide a brief summary
- Check if there are any important updates or announcements
```

### Task Types

| Type | Execution | Use Case |
|---|---|---|
| Quick Tasks | Immediate, synchronous | Simple queries, status checks |
| Long Tasks | Async via sub-agent | Research, analysis, multi-step tasks |

## CLI Commands

### List Tasks

```bash
wunderpus cron list
```

### Add Task

```bash
wunderpus cron add "Review open GitHub issues"
```

## Execution Flow

```
Heartbeat Interval Triggered
    │
    ▼
Parse HEARTBEAT.md
    │
    ├── Quick Tasks
    │     │
    │     ▼
    │   Agent.Execute(task)
    │     │
    │     ▼
    │   Return result directly
    │
    └── Long Tasks
          │
          ▼
        SubAgent.Spawn(task)
          │
          ▼
        Execute asynchronously
          │
          ▼
        Store result
```

## Best Practices

1. **Keep quick tasks simple** — They block the main agent loop
2. **Use long tasks for heavy work** — They run in parallel via sub-agents
3. **Set appropriate intervals** — Don't poll too frequently (minimum 5 minutes)
4. **Monitor task results** — Check logs for execution status
