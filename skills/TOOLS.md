---
name: tools
description: "Guidelines on tool usage, safety, and constraints for the agent environment."
---

# Tool Usage Guidelines

You have access to a suite of powerful tools to interact with the system. Using them effectively is critical for success.

## Core Directives
1. **Prefer Specific Tools**: If a specific tool exists (e.g., `file_read`, `file_search`), use it over generalized tools (e.g., `shell_exec("cat ...")`).
2. **Iterative Usage**: Don't try to solve the entire problem in one tool call. Read a file, understand it, then use a write tool, then a shell tool to test it.
3. **Security Constraints**: The system protects against dangerous commands (like `rm -rf`) and SSRF. Understand that if a command is blocked, you must find a safer alternative.

## Shell Execution Limits
1. Avoid interactive commands that hang or wait for input (like `vi`, `nano`, `python` without `-c`).
2. Commands are subject to timeouts. Do not run infinite loops.
3. Never attempt to bypass the sandbox.

## Sub-Agent Tools

### spawn
Spawns a new independent sub-agent to execute a task asynchronously. Use this for long-running tasks that should run in the background.

Parameters:
- `task` (required): The task description for the sub-agent to execute
- `system_prompt` (optional): Custom system prompt for the sub-agent

Returns the sub-agent ID immediately. Use the `message` tool to communicate with or get results from the sub-agent.

### message
Send a message to a running sub-agent or get its status/result.

Parameters:
- `subagent_id` (required): The ID of the sub-agent (first 8 characters from spawn response)
- `message` (optional): Message to send to the sub-agent (omit to just get status)
- `wait` (optional): Wait for sub-agent to complete and return result (default: false)
