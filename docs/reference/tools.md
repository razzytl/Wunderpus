# Tool Reference

Wunderpus provides a comprehensive tool system that enables the agent to interact with the world.

## Built-in Tools

| Tool | Description | Approval Level |
|---|---|---|
| `file_read` | Read file contents (sandboxed to workspace) | AutoExecute |
| `file_write` | Write content to file (sandboxed) | RequiresApproval |
| `file_list` | List directory contents (sandboxed) | NotifyOnly |
| `file_glob` | Find files by pattern (sandboxed) | NotifyOnly |
| `search_files` | Search file contents (sandboxed) | AutoExecute |
| `shell_exec` | Execute shell commands (whitelisted + sandboxed) | RequiresApproval |
| `http_request` | Make HTTP requests (SSRF-blocked) | RequiresApproval |
| `calculator` | Mathematical calculations | AutoExecute |
| `system_info` | System information | AutoExecute |
| `browser` | Browser automation via Playwright | RequiresApproval |
| `spawn` | Create sub-agent | RequiresApproval |
| `message` | Send message to sub-agent | AutoExecute |
| `ask_human` | Request human input | AutoExecute |
| `send_file` | Send file to user via channel | AutoExecute |

## Approval Levels

Tools are classified by their `ApprovalLevel()`:

| Level | Behavior | When to Use |
|---|---|---|
| `AutoExecute` | Run immediately | Read-only, safe operations |
| `NotifyOnly` | Run with log entry | Low-risk operations worth auditing |
| `RequiresApproval` | Pause for human approval | Destructive or external operations |
| `Blocked` | Reject immediately | Dangerous or disabled tools |

## Tool Execution Pipeline

```
Agent requests tool
    │
    ▼
1. Lookup tool in registry
    │
    ▼
2. Policy-based approval check
    │  ├── AutoExecute → run immediately
    │  ├── RequiresApproval → pause for human
    │  ├── Blocked → reject
    │  └── NotifyOnly → log and run
    ▼
3. Timeout enforcement
    │
    ▼
4. Execute tool
    │
    ▼
5. Record analytics
    │
    ▼
6. Audit log execution
    │
    ▼
7. Return result to agent
```

## Tool Interface

```go
type Tool interface {
    Name() string                              // Unique identifier
    Description() string                       // What the tool does
    Parameters() []ParameterDef                // Input parameters
    Execute(ctx context.Context, args) (*Result, error)  // Run the tool
    ApprovalLevel() ApprovalLevel              // Policy-based classification
    Version() string                           // Semantic version
    Dependencies() []string                    // Required dependencies
}
```

## Tool Analytics

Per-tool tracking includes:
- Call count
- Error count
- Total latency

Access via `executor.GetStats()`.

## MCP Support

Wunderpus supports the Model Context Protocol (MCP) for external tool integration:

```yaml
tools:
  mcp:
    enabled: true
    servers:
      - name: "filesystem"
        command: "npx"
        args: ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/workspace"]
```

## Configuration

```yaml
tools:
  enabled: true
  timeout_seconds: 30
  shell_whitelist:
    - ls
    - git
    - go
    - cat
    - echo
    - grep
    - find
  ssrf_blocklist:
    - "internal.company.com"
```

## Tool Synthesis

The tool synthesis engine (`internal/toolsynth/`) can automatically generate new tools:

```
Detect Gap → Design Spec → Generate Code → Test → Register
```

When enabled, the agent:
1. Scans memory for failed tasks or workarounds
2. Designs tool specifications via LLM
3. Generates Go source code at 3 temperatures
4. Tests compiled code
5. Registers valid tools automatically

```yaml
genesis:
  tool_synth_enabled: true
```
