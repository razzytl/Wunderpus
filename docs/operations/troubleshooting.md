# Troubleshooting

Common issues and solutions.

## Installation Issues

### Go Version Too Old

```
error: package requires go1.25
```

**Solution:** Update Go to 1.25+:

```bash
go install golang.org/dl/go1.25.0@latest
go1.25.0 download
```

### Build Fails

```
cannot find package
```

**Solution:**

```bash
go mod download
go mod tidy
go build -o build/wunderpus ./cmd/wunderpus
```

### CGO Errors

```
exec: "gcc": executable file not found
```

**Solution:** Install a C compiler:

```bash
# Ubuntu/Debian
sudo apt install gcc

# macOS
xcode-select --install

# Or disable CGO (some features may not work)
CGO_ENABLED=0 go build ./cmd/wunderpus
```

## Provider Issues

### "model not found" Error

**Causes:**
- Incorrect model identifier
- Wrong protocol prefix
- Model not available on provider

**Solution:**
- Verify model name: `openai/gpt-4o` (not just `gpt-4o`)
- Check protocol prefix matches the API
- For OpenRouter: use full path `openrouter/deepseek/deepseek-r1`

### Authentication Failures

**Solution:**
- Confirm API key is correct and active
- Check environment variables aren't conflicting
- Verify sufficient credits/quota

```bash
wunderpus auth status
```

### Rate Limiting

**Solution:**
- Configure fallback models
- Increase timeout in config
- Use providers with higher rate limits

## Channel Issues

### Telegram Bot Not Responding

**Solution:**
- Verify bot token is correct
- Check bot is not blocked by Telegram
- Ensure Message Content Intent is enabled

### Discord Bot Not Receiving Messages

**Solution:**
- Enable Message Content Intent in Discord Developer Portal
- Enable Guild Messages intent
- Invite bot with correct permissions

### WebSocket Connection Refused

**Solution:**
- Check port is not already in use
- Verify `channels.websocket.enabled: true` in config
- Check firewall rules

## Database Issues

### SQLite Lock

```
database is locked
```

**Solution:**
- Ensure only one instance is running
- Check file permissions on `.db` files
- SQLite WAL mode is enabled by default for concurrent reads

### Database File Not Found

```
no such table: mem_sessions
```

**Solution:**
- Tables are created automatically on first access
- Check that the DB path is writable
- Verify `db.Open()` is called during bootstrap

## Tool Issues

### Tool Execution Timed Out

**Solution:**
- Increase `tools.timeout_seconds` in config
- Check if the command is hanging (e.g., waiting for input)
- Verify the tool is in the shell whitelist

### "tool denied by user"

**Solution:**
- This is expected for `RequiresApproval` tools
- Approve the tool execution via TUI or channel
- To auto-approve, change the tool's `ApprovalLevel` to `AutoExecute`

### "tool blocked by policy"

**Solution:**
- The tool is classified as `Blocked`
- Check the tool's `ApprovalLevel()` implementation
- Review policy configuration

## Memory Issues

### Context Too Long

**Solution:**
- Increase `agent.max_context_tokens`
- Summarization triggers automatically at 80% capacity
- Check tiktoken is loading correctly (falls back to heuristic if not)

### RAG Not Returning SOPs

**Solution:**
- Verify an embedding-capable provider is configured
- Check `EnhancedStore` is initialized (requires embedder)
- Verify SOPs are stored: `enhanced.StoreSOP(ctx, title, content)`

## Health Check Issues

### /health Returns "unhealthy"

**Solution:**
- Check the `components` field in the response to identify which component is down
- Common causes:
  - Database connection lost
  - Provider not responding
  - Channel disconnected

### /ready Returns "not ready"

**Solution:**
- Same as above — check component status
- Readiness uses the same aggregation as health

## Logging

### Enable Debug Logging

```bash
wunderpus -v
```

Or in config:

```yaml
logging:
  level: "debug"
```

### JSON Logs for Log Aggregation

```yaml
logging:
  level: "info"
  format: "json"
  output: "/var/log/wunderpus/app.log"
```
