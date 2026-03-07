# Heartbeat — Periodic Tasks

Wunderpus can periodically execute tasks defined in this file. Place HEARTBEAT.md in your workspace directory.

## Quick Tasks (respond directly)

These tasks are executed immediately and their results are returned directly.

- Report the current date and time
- Summarize the current conversation context

## Long Tasks (spawn subagents)

These tasks are executed asynchronously using spawned subagents, allowing multiple tasks to run in parallel.

- Search for recent technology news and provide a brief summary
- Check if there are any important updates or announcements

---

**Configuration:**

- `heartbeat.enabled`: Enable/disable heartbeat (default: true)
- `heartbeat.interval`: Interval in minutes (default: 30, minimum: 5)

Or use environment variables:
- `WUNDERPUS_HEARTBEAT_ENABLED=false`
- `WUNDERPUS_HEARTBEAT_INTERVAL=60`
