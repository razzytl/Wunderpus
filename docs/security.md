# Security Guide

Wunderpus includes comprehensive security features designed for enterprise deployments. This guide covers configuration options and best practices for securing your installation.

## Security Architecture

The security system consists of several interconnected components:

```
┌─────────────────────────────────────────────────────────────┐
│                    Security Layer                           │
├──────────────┬──────────────┬──────────────┬────────────────┤
│  Encryption  │ Audit Logging│ Rate Limiting│ Sanitization  │
│              │              │              │                │
│  - At Rest   │  - SQLite   │  - Token     │  - Input       │
│  - Config    │  - Actions  │    Bucket    │  - Output      │
│  - Keys      │  - Search   │  - Per-User  │  - Shell       │
└──────────────┴──────────────┴──────────────┴────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Tool Execution Layer                       │
├──────────────┬──────────────┬───────────────────────────────┤
│   Shell      │   File       │        HTTP                  │
│  Sandbox     │  Operations  │     SSRF Protection          │
│              │              │                               │
│  - Allowlist │ - Workspace  │  - Blocklist                  │
│  - Patterns  │ - Restricted │  - Private IP Check           │
└──────────────┴──────────────┴───────────────────────────────┘
```

## Encryption

Wunderpus supports AES-256-GCM encryption for sensitive data at rest.

### Configuration

```yaml
security:
  encryption:
    enabled: true
    key: "base64-encoded-32-byte-key"
```

### Generating a Key

Generate a secure encryption key:

```bash
# Using OpenSSL
openssl rand -base64 32
```

Or programmatically:

```go
import "crypto/rand"

// Generate 32-byte key
key := make([]byte, 32)
rand.Read(key)
// Convert to base64 for config
```

### Encrypted Fields

When enabled, the following are encrypted:
- API keys in configuration
- Provider credentials
- Channel tokens
- Custom secrets

### Key Management

**Best Practices:**

1. **Use Environment Variables**: Store the key in an environment variable
   ```yaml
   security:
     encryption:
       enabled: true
       key: "${ENCRYPTION_KEY}"
   ```

2. **Rotate Keys Periodically**: Plan for key rotation
   ```bash
   # Generate new key
   export NEW_ENCRYPTION_KEY=$(openssl rand -base64 32)
   
   # Re-encrypt data
   wunderpus security reencrypt --old-key "$OLD_KEY" --new-key "$NEW_ENCRYPTION_KEY"
   ```

3. **Secure Storage**: Use a secrets manager for production
   - HashiCorp Vault
   - AWS Secrets Manager
   - Azure Key Vault

## Audit Logging

All agent actions are logged to a SQLite database for compliance and debugging.

### Configuration

```yaml
security:
  # Path to audit database
  audit_db_path: "wunderpus_audit.db"
  
  # Enable/disable (default: enabled when tools enabled)
  audit_enabled: true
  
  # Log retention (days)
  audit_retention_days: 90
```

### Logged Events

The audit log captures:

| Event Type | Description |
|------------|-------------|
| `message` | User messages to agent |
| `response` | Agent responses |
| `tool_execution` | Tool invocations |
| `tool_result` | Tool execution results |
| `channel_message` | Channel communications |
| `auth` | Authentication events |
| `config_change` | Configuration modifications |
| `error` | Error conditions |

### Audit Schema

```sql
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    session_id TEXT,
    event_type TEXT NOT NULL,
    event_data TEXT,  -- JSON
    user_id TEXT,
    channel TEXT,
    provider TEXT,
    success INTEGER,
    error_message TEXT
);

CREATE INDEX idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_session ON audit_log(session_id);
CREATE INDEX idx_audit_type ON audit_log(event_type);
```

### Querying Audit Logs

```bash
# Using SQLite
sqlite3 wunderpus_audit.db "SELECT * FROM audit_log WHERE event_type='tool_execution' LIMIT 10;"

# Using CLI
wunderpus audit query --type tool_execution --limit 10
```

## Rate Limiting

Protect against abuse with configurable rate limits.

### Configuration

```yaml
security:
  rate_limit:
    # Global rate limit
    requests_per_minute: 60
    burst: 10
    
    # Per-user limits
    per_user:
      requests_per_minute: 20
      burst: 5
      
    # Per-provider limits
    per_provider:
      requests_per_minute: 30
      burst: 5
```

### How Rate Limiting Works

Wunderpus uses token bucket algorithm:

1. **Requests per minute**: Maximum sustained rate
2. **Burst**: Maximum concurrent requests

When limits are exceeded:
- Requests are queued
- Client receives HTTP 429
- Retry-After header indicates wait time

## Shell Sandboxing

Commands executed by the agent are sandboxed for safety.

### Configuration

```yaml
tools:
  enabled: true
  
  # Allowed shell commands (whitelist approach)
  shell_whitelist:
    - git
    - go
    - npm
    - cargo
    - docker
    - make
    
  # Additional patterns to block
  shell_blocklist:
    - "^rm -rf"          # Recursive delete
    - "^dd "             # Disk write
    - "^mkfs"            # Filesystem format
    - ":(){:|:&};:"      # Fork bomb
    - "curl.*| sh"       # Pipe to shell
    
  # Timeout for commands
  timeout_seconds: 30
```

### Default Allowlist

The default shell whitelist includes:

```
cat, cd, cp, date, echo, env, grep, head, ls, mkdir, pwd,
rm, rmdir, sed, sort, tail, tee, test, touch, uname, wc,
git, go, cargo, npm, yarn, pnpm, docker, docker-compose,
make, cmake, gcc, g++, clang, python, python3, pip, pip3,
node, npm, bun, deno, curl, wget, tar, gzip, zip, unzip,
ssh, scp, rsync, kubectl, helm, terraform
```

### Customizing the Allowlist

```yaml
tools:
  shell_whitelist:
    # Keep defaults
    - git
    - go
    - npm
    # Add project-specific
    - my-script.sh
```

## Workspace Isolation

Restrict file operations to a designated workspace directory.

### Configuration

```yaml
agents:
  defaults:
    # Workspace directory
    workspace: "/home/user/projects"
    
    # Enable workspace isolation
    restrict_to_workspace: true
```

### Behavior

When `restrict_to_workspace: true`:
- File operations limited to workspace directory
- Path traversal attempts are blocked
- Commands cannot access paths outside workspace

### Disabling Isolation

```yaml
agents:
  defaults:
    restrict_to_workspace: false  # DANGER: Allows file access outside workspace
```

## SSRF Protection

Prevent the agent from making requests to internal or sensitive network destinations.

### Configuration

```yaml
tools:
  # Default: localhost and private IPs blocked
  ssrf_protection_enabled: true
  
  # Additional domains/IPs to block
  ssrf_blocklist:
    - "internal.company.com"
    - "admin.local"
    - "169.254.169.254"  # Cloud metadata endpoints
    - "metadata.google.internal"
    
  # Custom allowlist (bypasses blocklist)
  ssrf_allowlist:
    - "api.example.com"
```

### Default Blocked Addresses

The following are always blocked:

| Category | Examples |
|----------|----------|
| Localhost | 127.0.0.1, localhost, ::1 |
| Private Ranges | 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 |
| Link-local | 169.254.0.0/16 |
| Documentation | 192.0.2.0/24, 198.51.100.0/24 |
| Benchmark | 192.18.0.0/15 |

## Input Sanitization

All user input is sanitized before processing.

### Configuration

```yaml
security:
  sanitization_enabled: true
  
  # Additional patterns to strip
  sanitization_patterns:
    - "\\$\\(.*\\)"     # Command substitution
    - "`.*`"            # Backtick commands
```

### What Gets Sanitized

- Null bytes
- Control characters
- Common injection patterns
- Excessively long inputs

## API Key Security

### Environment Variables (Recommended)

Store API keys in environment variables rather than config files:

```yaml
# config.yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
```

```bash
# .env (add to .gitignore)
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
```

### Configuration Files

If using config files:

1. **Restrict file permissions**
   ```bash
   chmod 600 config.yaml
   ```

2. **Add to .gitignore**
   ```
   config.yaml
   *.db
   *.log
   ```

3. **Use separate secrets file**
   ```yaml
   # config.yaml (committed)
   include: secrets.yaml
   ```

## Network Security

### TLS/SSL

For production deployments, always use TLS:

```yaml
server:
  # Health check server TLS
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    
  # HTTP to HTTPS redirect
  https_redirect: true
```

### Firewall Configuration

Restrict access to Wunderpus ports:

```bash
# Allow specific IPs
iptables -A INPUT -p tcp -s 10.0.0.0/8 --dport 8080 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j DROP
```

## Security Checklist

Before deploying to production:

- [ ] Enable encryption for API keys
- [ ] Configure audit logging
- [ ] Set up rate limiting
- [ ] Review shell whitelist
- [ ] Enable workspace isolation
- [ ] Configure SSRF protection
- [ ] Use TLS for external connections
- [ ] Restrict file permissions on config
- [ ] Rotate API keys periodically
- [ ] Monitor audit logs regularly

## Monitoring Security Events

Set up alerts for security events:

```yaml
monitoring:
  alerts:
    - type: "rate_limit_exceeded"
      threshold: 10
      window: 1m
      action: "notify"
      
    - type: "tool_blocked"
      threshold: 5
      window: 5m
      action: "notify"
      
    - type: "auth_failure"
      threshold: 3
      window: 1m
      action: "notify"
```

## Compliance

For compliance requirements:

1. **Audit Logs**: Retain according to policy (90 days default)
2. **Encryption**: Enable for PII/credentials
3. **Access Control**: Implement per-user restrictions
4. **Logging**: Integrate with SIEM systems
5. **Incident Response**: Document procedures
