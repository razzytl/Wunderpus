# Tool Reference

Wunderpus provides a comprehensive tool system that enables the agent to interact with the world.

## Built-in Tools

| Tool | Description | Sensitive |
|---|---|---|
| `file_read` | Read file contents | No |
| `file_write` | Write content to file | Yes |
| `file_list` | List directory contents | No |
| `file_glob` | Find files by pattern | No |
| `search_files` | Search file contents | No |
| `shell_exec` | Execute shell commands | Yes |
| `http_request` | Make HTTP requests | Yes |
| `calculator` | Mathematical calculations | No |
| `system_info` | System information | No |
| `browser` | Browser automation | Yes |
| `spawn` | Create sub-agent | Yes |
| `message` | Send message to sub-agent | No |
| `ask_human` | Request human input | No |
| `send_file` | Send file to user | No |

## Tool Execution Pipeline

```
Agent requests tool
    │
    ▼
1. Lookup tool in registry
    │
    ▼
2. Approval check (sensitive tools)
    │  (pause for human approval if needed)
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
    Name() string                          // Unique identifier
    Description() string                   // What the tool does
    Parameters() []ParameterDef            // Input parameters
    Execute(ctx context.Context, args) (*Result, error)  // Run the tool
    Sensitive() bool                       // Requires human approval
    Version() string                       // Semantic version
    Dependencies() []string                // Required dependencies
}
```

## Tool Analytics

Per-tool tracking includes:
- Call count
- Error count
- Total latency

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

  # Sensitive tools require approval
  sensitive_tools:
    - shell_exec
    - http_request
    - file_write
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
  toolsynth_enabled: true
```
