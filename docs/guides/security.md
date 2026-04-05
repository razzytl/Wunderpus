# Security Guide

Wunderpus implements a defense-in-depth security model with five layers of protection.

## Security Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Security Layers                           │
├──────────────┬──────────────┬──────────────┬────────────────┤
│  Input       │  Execution   │  Network     │  Approval      │
│  Sanitization│  Sandbox     │  SSRF Block  │  Policy Gates  │
├──────────────┴──────────────┴──────────────┴────────────────┤
│                    Storage Protection                         │
│              AES-256-GCM + Hash-Chained Audit                │
└─────────────────────────────────────────────────────────────┘
```

## Layer 1: Input Sanitization

All user input passes through the sanitizer before processing:

- **Unicode normalization** — Prevents homoglyph attacks
- **9-pattern injection detection** — Blocks prompt injection attempts:
  - Instruction override
  - Role reassignment
  - System prompt extraction
  - Identity override
  - Safety bypass
  - Raw template injection
  - ReAct pattern injection
  - Obfuscated payloads
- **Control character stripping** — Removes non-printable characters

### Configuration

```yaml
security:
  sanitization_enabled: true
```

## Layer 2: Execution Sandbox

File and shell operations are restricted to the workspace:

### Workspace Isolation

```yaml
agents:
  defaults:
    workspace: "/path/to/workspace"
    restrict_to_workspace: true  # Default: enabled
```

When enabled:
- All file operations restricted to workspace directory
- Path traversal (`../`) blocked
- Commands cannot access paths outside workspace

### Command Sandboxing

```yaml
tools:
  shell_whitelist:
    - ls
    - git
    - go
    - cat
    - echo
    - grep
    - find
```

**Blocked patterns:**
- Command chaining: `;`, `&&`, `||`, `|`, backticks, `$()`
- Redirects: `>`, `>>`
- Directory escapes via `cd`

## Layer 3: Network Protection (SSRF)

The HTTP tool blocks requests to internal networks by default:

**Always blocked:**
- Localhost: `127.0.0.1`, `localhost`, `::1`
- Private ranges: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- Link-local: `169.254.0.0/16`
- Cloud metadata: `169.254.169.254`

### Custom Blocklist

```yaml
tools:
  ssrf_blocklist:
    - "internal.company.com"
    - "admin.local"
```

## Layer 4: Policy-Based Approval Gates

Tools are classified by their `ApprovalLevel()`:

| Level | Behavior | Examples |
|---|---|---|
| `AutoExecute` | Run immediately | file_read, calculator, system_info |
| `NotifyOnly` | Run with log | file_list, file_glob |
| `RequiresApproval` | Pause for human approval | shell_exec, http_request, file_write |
| `Blocked` | Reject immediately | Any tool blocked by policy |

When a tool requires approval:
1. Execution pauses
2. Human is notified (via TUI or channel)
3. Human approves or denies
4. Execution continues or aborts

## Layer 5: Storage Protection

### Encryption

AES-256-GCM encryption for sensitive data:

```yaml
# Generate key: openssl rand -base64 32
security:
  encryption:
    enabled: true
    key: "${WUNDERPUS_ENCRYPTION_KEY}"
```

**Encrypted at rest:**
- Session messages
- Audit log entries
- API keys (when prefixed with `enc:`)

### Audit Log

SHA-256 hash-chained, tamper-evident audit log in `wunderpus-audit.db`:

Every action logged with:
- Timestamp
- Subsystem
- Event type
- Actor ID
- Payload
- Previous hash (chain integrity)

### Rate Limiting

Sliding window rate limiter per session:

```yaml
security:
  rate_limit:
    enabled: true
    window_sec: 60
    max_requests: 60
    cleanup_interval_sec: 300
```

## Security Checklist

Before deploying to production:

- [ ] `sanitization_enabled: true`
- [ ] `restrict_to_workspace: true`
- [ ] Encryption key configured
- [ ] Audit log path set
- [ ] Shell whitelist reviewed
- [ ] SSRF blocklist updated for your network
- [ ] API keys in environment variables (not config files)
- [ ] Config file permissions: `chmod 600 config.yaml`
- [ ] Database files excluded from version control

## Best Practices

### API Key Management

```bash
# ✅ Good: Environment variables
export OPENAI_API_KEY="sk-..."

# ❌ Bad: Hardcoded in config
providers:
  openai:
    api_key: "sk-actual-key"
```

### File Permissions

```bash
chmod 600 config.yaml
chmod 600 .env
chmod 644 *.db  # SQLite databases
```

### Production Configuration

```yaml
security:
  sanitization_enabled: true
  encryption:
    enabled: true
    key: "${WUNDERPUS_ENCRYPTION_KEY}"

agents:
  defaults:
    restrict_to_workspace: true
    workspace: "/var/lib/wunderpus/workspace"
```

## Threat Model

| Threat | Mitigation |
|---|---|
| Prompt injection | Input sanitization (9 patterns) |
| File system escape | Workspace sandbox |
| SSRF attacks | Network blocklist |
| Unauthorized tool execution | Policy-based approval gates |
| Data theft | AES-256-GCM encryption |
| Audit tampering | SHA-256 hash chain |
| Resource exhaustion | Rate limiting |
| Credential exposure | Encrypted storage |
