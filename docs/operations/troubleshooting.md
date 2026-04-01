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
make build
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

# Or disable CGO
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
- Configure fallback providers
- Use cheaper models for simple tasks
- Enable response caching

```yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    fallback_models:
      - "groq/llama-3.3-70b-versatile"
```

## Channel Issues

### Channel Fails to Start

**Solution:**
- Verify bot token is correct
- Check network connectivity
- Ensure bot has necessary permissions

```bash
wunderpus gateway -v 2>&1 | grep -i telegram
```

### Messages Not Delivered

**Solution:**
1. Check channel status: `wunderpus status`
2. Verify rate limits not exceeded
3. Check bot permissions on the platform

## Memory Issues

### High Memory Usage

**Solution:**
- Reduce `max_context_tokens`
- Limit concurrent sessions
- Check for memory leaks in tools

```yaml
agent:
  max_context_tokens: 4000  # Reduce from 8000
```

### Database Lock Errors

**Solution:**
- Ensure only one Wunderpus instance accesses the database
- Use WAL mode (enabled by default)
- Check file permissions

```bash
chmod 644 *.db
```

## Tool Issues

### Tool Execution Timeout

```
tool execution timed out after 30s
```

**Solution:**
- Increase timeout: `tools.timeout_seconds: 60`
- Check if command is hanging
- Verify tool dependencies are installed

### Shell Command Not Allowed

```
command 'rm' is not in the whitelist
```

**Solution:**
- Add command to whitelist:

```yaml
tools:
  shell_whitelist:
    - git
    - go
    - rm  # Add the command
```

## Genesis Issues

### RSI Sandbox Failures

**Solution:**
- Ensure Go toolchain is available in sandbox
- Check Docker is running (if using Docker isolation)
- Verify `internal/` directory exists

### Trust Budget Lockdown

**Solution:**
- Check trust budget status
- Reset via JWT if locked down
- Reduce autonomous action frequency

```bash
# Check status
wunderpus status
```

## Debug Mode

Enable verbose logging for detailed diagnostics:

```bash
wunderpus -v
wunderpus agent -v -m "test message"
wunderpus gateway -v
```

### Log Levels

| Level | Use Case |
|---|---|
| `debug` | Detailed debugging |
| `info` | Normal operation (default) |
| `warn` | Warning conditions |
| `error` | Error conditions only |

```yaml
logging:
  level: "debug"  # Maximum verbosity
```

## Getting Help

- **GitHub Issues:** [Report bugs](https://github.com/wunderpus/wunderpus/issues)
- **Documentation:** [docs/INDEX.md](../INDEX.md)
- **CLI Help:** `wunderpus --help`
