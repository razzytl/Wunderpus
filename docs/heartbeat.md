# Heartbeat - Periodic Task Scheduling

Wunderpus includes a heartbeat system that periodically executes tasks defined in a HEARTBEAT.md file. This feature enables automated recurring actions such as reports, monitoring, and maintenance tasks.

## Overview

The heartbeat scheduler runs in the background and executes tasks at configurable intervals. Tasks are defined in a markdown file called HEARTBEAT.md in the workspace directory.

## How It Works

The heartbeat system:
1. Reads HEARTBEAT.md from the workspace directory
2. Parses tasks from the file
3. Executes tasks at configured intervals
4. Supports both quick tasks (synchronous) and long tasks (asynchronous)
5. Marks completed tasks and continues to the next interval

## Configuration

### Basic Configuration

```yaml
heartbeat:
  enabled: true
  interval: 30  # minutes (minimum: 5)
```

### Environment Variables

You can also configure via environment variables:

```bash
# Disable heartbeat
export WUNDERPUS_HEARTBEAT_ENABLED=false

# Set interval (in minutes)
export WUNDERPUS_HEARTBEAT_INTERVAL=60
```

## HEARTBEAT.md Format

Create a HEARTBEAT.md file in your workspace directory:

```markdown
# Heartbeat — Periodic Tasks

## Quick Tasks (respond directly)

These tasks are executed immediately and their results are returned directly.

- Task description here
- Another task

## Long Tasks (spawn subagents)

These tasks are executed asynchronously using spawned subagents.

- Long-running task description
- Another background task
```

### Task Types

#### Quick Tasks

Quick tasks execute synchronously and return results directly. Use these for:
- Simple queries
- Status checks
- Lightweight operations

```markdown
## Quick Tasks (respond directly)

- Report the current date and time
- Check disk usage
- List active sessions
```

#### Long Tasks

Long tasks execute asynchronously using sub-agents. Use these for:
- Complex operations
- Multiple-step workflows
- Tasks that may take longer

```markdown
## Long Tasks (spawn subagents)

- Search for recent news and summarize
- Generate daily report
- Monitor system health
```

## Examples

### Daily Standup Reminder

```markdown
## Quick Tasks (respond directy)

- It's 9am, remind me about daily standup at 10am
```

### News Summary

```markdown
## Long Tasks (spawn subagents)

- Search for top technology news headlines and provide a 3-bullet summary
```

### System Monitoring

```markdown
## Quick Tasks (respond directly)

- Check if all channels are connected
- Report current memory usage
- Show active session count
```

### Report Generation

```markdown
## Long Tasks (spawn subagents)

- Generate a summary of this week's conversations
- Create a token usage report for the past 24 hours
```

## Execution

### Starting Heartbeat

Heartbeat starts automatically when running the gateway:

```bash
wunderpus gateway
```

You can verify heartbeat status:

```bash
wunderpus cron list
```

Output:
```
Periodic Tasks Status: running
Interval: 30 minutes
Quick Tasks: 3
Long Tasks: 2
```

### Adding Tasks Programmatically

You can add tasks using the CLI:

```bash
# Add a new periodic task
wunderpus cron add "Generate daily report"
```

This appends a new task to HEARTBEAT.md with the current timestamp.

## Natural Language Scheduling

Wunderpus supports natural language for scheduling. The system parses expressions like:

- "at 9am daily"
- "every hour"
- "every Monday at 9am"

However, these require parsing the HEARTBEAT.md file. For simple recurring tasks, adjust the heartbeat interval:

```yaml
heartbeat:
  enabled: true
  interval: 60  # Every hour
```

## Best Practices

### Task Design

1. **Keep quick tasks simple**: They should complete in seconds
2. **Use long tasks for complex work**: They run asynchronously
3. **Clear task descriptions**: Make intent obvious
4. **Avoid dependencies**: Each task should be independent

### Workspace Placement

Place HEARTBEAT.md in your workspace:

```yaml
agents:
  defaults:
    workspace: "/path/to/workspace"  # HEARTBEAT.md should be here
```

### Monitoring

Check heartbeat execution in logs:

```bash
# View logs
wunderpus gateway -v 2>&1 | grep heartbeat

# Or check logs directory
cat workspace/logs/wunderpus.log
```

## Troubleshooting

### Heartbeat Not Running

1. Check if enabled in configuration:
   ```yaml
   heartbeat:
     enabled: true
   ```

2. Verify HEARTBEAT.md exists in workspace
3. Check logs for errors

### Tasks Not Executing

1. Verify task format is correct
2. Check that tasks are under proper headers
3. Ensure minimum interval (5 minutes) is met

### HEARTBEAT.md Not Found

The default workspace is the current directory. Specify explicitly:

```yaml
agents:
  defaults:
    workspace: "/path/to/workspace"
```

## CLI Commands

### Check Status

```bash
wunderpus cron list
```

### Add Task

```bash
wunderpus cron add "My periodic task"
```

## Integration with Other Features

### Skills in Heartbeat

Skills can be invoked in heartbeat tasks:

```markdown
## Quick Tasks (respond directly)

- Use the github skill to check open issues
```

### Tool Access

Heartbeat tasks have access to all configured tools:

```markdown
## Long Tasks (spawn subagents)

- Run `git status` and report any uncommitted changes
```
