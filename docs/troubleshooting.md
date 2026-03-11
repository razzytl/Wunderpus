# Troubleshooting Guide

This guide covers common issues encountered when running Wunderpus and their solutions.

## Quick Diagnostics

Before diving into specific issues, run these commands to gather information:

```bash
# Check version
wunderpus --version

# Verify configuration
wunderpus status

# Check authentication
wunderpus auth status

# Run with verbose output
wunderpus agent -v
```

## Provider Issues

### "model not found" or Provider API Errors

**Symptom:** Error messages like `model "openrouter/free" not found` or authentication failures.

**Cause:** The model identifier doesn't match what the provider expects.

**Solutions:**

1. **OpenRouter Models**: Use the full model identifier
   ```yaml
   # Wrong
   model: "free"
   
   # Correct
   model: "openrouter/deepseek/deepseek-r1"
   ```

2. **Verify API Key**: Check authentication
   ```bash
   wunderpus auth status
   ```

3. **Environment Variables**: Ensure API keys are set correctly
   ```bash
   # Check if variable is set
   echo $OPENAI_API_KEY
   
   # Set if missing
   export OPENAI_API_KEY="sk-your-key"
   ```

4. **Check Provider Status**: Some providers have outages
   - [OpenAI Status](https://status.openai.com)
   - [Anthropic Status](https://status.anthropic.com)
   - [OpenRouter Status](https://openrouter.statuspage.io)

### Authentication Failures

**Symptom:** `401 Unauthorized` or `Authentication failed` errors.

**Solutions:**

1. **Verify API Key Format**
   ```yaml
   # OpenAI: starts with "sk-"
   providers:
     openai:
       api_key: "sk-..."
   
   # Anthropic: starts with "sk-ant-"
   providers:
     anthropic:
       api_key: "sk-ant-..."
   ```

2. **Check for Whitespace**: Ensure no trailing newlines in config
   ```bash
   # Verify key format
   head -n 1 config.yaml | grep -o "sk-[a-zA-Z0-9]*"
   ```

3. **Confirm Key is Active**: Log into provider dashboard and verify key hasn't expired

### Provider Rate Limiting

**Symptom:** `429 Too Many Requests` errors.

**Solutions:**

1. **Add Fallback Providers**
   ```yaml
   model_list:
     - model_name: "primary"
       model: "openai/gpt-4o"
       fallback_models:
         - "anthropic/claude-sonnet-4-20250514"
   ```

2. **Configure Rate Limits**
   ```yaml
   security:
     rate_limit:
       requests_per_minute: 30
   ```

3. **Implement Backoff**: Wait before retrying

### Timeout Errors

**Symptom:** Requests hang or timeout after 30+ seconds.

**Solutions:**

1. **Increase Timeout**
   ```yaml
   providers:
     openai:
       timeout: 120s
   ```

2. **Check Network**: Test connectivity
   ```bash
   curl -v https://api.openai.com/v1/models
   ```

3. **Use Faster Models**: Switch to low-latency models
   ```yaml
   providers:
     openai:
       model: "gpt-4o-mini"  # Faster than gpt-4o
   ```

## Channel Issues

### Channel Fails to Start

**Symptom:** `Failed to start channel telegram` or similar errors.

**Diagnostic Steps:**

1. **Run with Verbose Output**
   ```bash
   wunderpus gateway -v
   ```

2. **Check Configuration**
   ```yaml
   channels:
     telegram:
       enabled: true
       bot_token: "${TELEGRAM_BOT_TOKEN}"
   ```

3. **Verify Token Format**
   - Telegram: `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`
   - Discord: Should be a valid bot token (not user token)

### Telegram Issues

**Bot Not Responding:**

1. **Check Bot Token**
   ```bash
   curl "https://api.telegram.org/bot<TOKEN>/getMe"
   ```

2. **Set Bot Commands**
   Start a conversation with @BotFather and set commands

3. **Privacy Mode**: Ensure privacy mode allows reading messages

**Webhook Issues:**

1. **Verify URL is Reachable**
   ```bash
   curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://your-domain.com/telegram"
   ```

2. **Check SSL Certificate**: Use valid SSL (Let's Encrypt works)

### Discord Issues

**Bot Offline:**

1. **Check Intent Permissions**
   - Go to Discord Developer Portal
   - Enable `MESSAGE CONTENT INTENT` in Bot settings

2. **Re-invite Bot**
   Generate new invite link with correct permissions

**Slash Commands Not Working:**

1. **Register Commands**
   Commands should register automatically on gateway start

2. **Check Permissions**
   ```yaml
   channels:
     discord:
       bot_token: "${DISCORD_BOT_TOKEN}"
       # guild_id: "optional-guild-id"  # For development
   ```

### WebSocket Connection Issues

**Client Cannot Connect:**

1. **Verify Port**
   ```yaml
   channels:
     websocket:
       enabled: true
       host: "0.0.0.0"
       port: 8081
   ```

2. **Check Firewall**
   ```bash
   # Test connectivity
   telnet your-server 8081
   ```

3. **Network Configuration**
   Ensure WebSocket endpoint is accessible

## Tool Execution Issues

### Tools Not Working

**Symptom:** Agent says "I don't have access to tools" or tool execution fails.

**Solutions:**

1. **Enable Tools**
   ```yaml
   tools:
     enabled: true
   ```

2. **Configure Shell Whitelist**
   ```yaml
   tools:
     shell_whitelist:
       - git
       - go
       - npm
       - cargo
       - docker
   ```

3. **Check Permissions**
   Ensure binary is executable
   ```bash
   chmod +x /path/to/tool
   ```

### Command Blocked by Sandbox

**Symptom:** "Command not allowed" errors.

**Cause:** Command not in whitelist or blocked by pattern.

**Solutions:**

1. **Add to Whitelist**
   ```yaml
   tools:
     shell_whitelist:
       - my-command
   ```

2. **Check Blocked Patterns**
   Default blocked patterns include:
   - `rm -rf` (recursive delete)
   - Fork bombs
   - Attempts to bypass sandbox

### File Access Denied

**Symptom:** "Access denied" when reading/writing files.

**Solutions:**

1. **Check Workspace Configuration**
   ```yaml
   agents:
     defaults:
       workspace: "/path/to/allowed/directory"
       restrict_to_workspace: true
   ```

2. **Add Allowed Paths**
   ```yaml
   tools:
     allowed_paths:
       - "/home/user/projects"
       - "/tmp"
   ```

### HTTP Requests Blocked (SSRF)

**Symptom:** HTTP tool requests fail with "blocked" error.

**Cause:** Target is in blocklist or is a private IP.

**Solutions:**

1. **Check Target**: Some destinations are always blocked:
   - localhost
   - 10.0.0.0/8
   - 172.16.0.0/12
   - 192.168.0.0/16

2. **Use Allowed Destinations**
   Only publicly accessible URLs can be accessed

## Memory and Session Issues

### Agent Forgets Conversation

**Symptom:** Agent doesn't remember previous messages in session.

**Solutions:**

1. **Check Context Token Limit**
   ```yaml
   agent:
     max_context_tokens: 8000
   ```

2. **Verify Session Storage**
   ```bash
   # Check if database is writable
   ls -la *.db
   ```

3. **Clear Session Manually**
   In TUI, use `/clear` command

### Memory Errors / Crashes

**Symptom:** Application crashes with "out of memory" or similar errors.

**Solutions:**

1. **Reduce Context Size**
   ```yaml
   agent:
     max_context_tokens: 4000
   ```

2. **Limit Concurrent Sessions**
   ```yaml
   agent:
     max_sessions: 50
   ```

3. **Check System Resources**
   ```bash
   free -h
   df -h
   ```

## Security Issues

### Encryption Errors

**Symptom:** `invalid encryption key` or garbled decrypted data.

**Solutions:**

1. **Verify Key Format**: Must be 32 bytes, base64 encoded
   ```bash
   # Generate valid key
   openssl rand -base64 32
   ```

2. **Check Configuration**
   ```yaml
   security:
     encryption:
       enabled: true
       key: "32-byte-base64-encoded-key"
   ```

3. **Re-enable Encryption**: If previously disabled, data may be incompatible

### Rate Limiting Triggered

**Symptom:** `rate limit exceeded` errors.

**Solutions:**

1. **Check Current Settings**
   ```yaml
   security:
     rate_limit:
       requests_per_minute: 60
   ```

2. **Wait for Cooldown**: Rate limits typically reset after 1 minute

3. **Adjust Limits** (if needed):
   ```yaml
   security:
     rate_limit:
       requests_per_minute: 120
   ```

## Installation Issues

### Binary Not Found

**Symptom:** `command not found` after installation.

**Solutions:**

1. **Check Installation Path**
   ```bash
   which wunderpus
   # or
   ls /usr/local/bin/wunderpus
   ```

2. **Add to PATH**
   ```bash
   export PATH=$PATH:/usr/local/bin
   # Add to ~/.bashrc for persistence
   ```

### Permission Denied

**Symptom:** `Permission denied` when running.

**Solutions:**

1. **Make Executable**
   ```bash
   chmod +x wunderpus
   ```

2. **Check Directory Permissions**

### Build Failures

**Symptom:** `go build` fails.

**Solutions:**

1. **Verify Go Version**
   ```bash
   go version  # Must be 1.25 or later
   ```

2. **Clean and Rebuild**
   ```bash
   go clean -cache
   go build -o wunderpus ./cmd/wunderpus
   ```

3. **Update Dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

## Docker Issues

### Container Won't Start

**Diagnostic Steps:**

1. **Check Logs**
   ```bash
   docker logs wunderpus
   docker-compose logs wunderpus
   ```

2. **Verify Configuration Mount**
   ```bash
   docker run --rm -v $(pwd)/config.yaml:/app/config.yaml:ro wunderpus/wunderpus:latest wunderpus status
   ```

3. **Check Port Conflicts**
   ```bash
   netstat -tuln | grep 8080
   ```

### Container Exits Immediately

**Solutions:**

1. **Run Interactively**
   ```bash
   docker run -it wunderpus/wunderpus:latest /bin/sh
   ```

2. **Check Entry Point**
   ```dockerfile
   CMD ["gateway"]
   ```

## Performance Issues

### High Memory Usage

**Symptoms:** System becomes slow, OOM errors.

**Solutions:**

1. **Reduce Context Tokens**
   ```yaml
   agent:
     max_context_tokens: 4000
   ```

2. **Limit Sessions**
   ```yaml
   agent:
     max_sessions: 50
   ```

3. **Enable Compression** (if available)

### Slow Responses

**Symptoms:** Agent takes long time to respond.

**Solutions:**

1. **Use Faster Models**
   ```yaml
   providers:
     openai:
       model: "gpt-4o-mini"
   ```

2. **Enable Parallel Probing**
   ```yaml
   agent:
     parallel_probe: true
     probe_timeout: 5s
   ```

3. **Add Fallback Providers**

## Getting Help

If these solutions don't resolve your issue:

1. **Enable Debug Logging**
   ```bash
   wunderpus agent -v
   ```

2. **Check Log Files**
   ```bash
   # Default location
   cat workspace/logs/wunderpus.log
   
   # Or with Docker
   docker logs wunderpus
   ```

3. **Gather Information**
   - Wunderpus version: `wunderpus --version`
   - Configuration (redacted): `cat config.yaml`
   - Error messages
   - Steps to reproduce

4. **Contact Support**
   - Open an issue on GitHub
   - Ask in Discord
   - Check existing issues

## Debug Mode

For advanced debugging:

```yaml
logging:
  level: "debug"
  format: "json"
  output: "debug.log"
```

Then analyze `debug.log` for detailed execution traces.
