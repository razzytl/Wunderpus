# Security Guide

Wunderpus implements a defense-in-depth security model with five layers of protection.

## Security Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Security Layers                           │
├──────────────┬──────────────┬──────────────┬────────────────┤
│  Input       │  Execution   │  Network     │  Autonomy      │
│  Sanitization│  Sandbox     │  SSRF Block  │  Trust Budget  │
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

## Layer 4: Trust Budget (UAA)

The Unbounded Autonomous Action system limits what the agent can do without human approval:

### Action Tiers

| Tier | Type | Cost | Examples |
|---|---|---|---|
| 1 | Read-only | 0 | Web search, file read, calculator |
| 2 | Ephemeral | 1 | Temp file writes, go build, go test |
| 3 | Persistent | 5 | File writes, git commit, DB modifications |
| 4 | External | 20 | HTTP POST, deploy, send communications, spend money |

### Trust Budget

```yaml
genesis:
  trust_budget_max: 1000        # Maximum trust points
  trust_regen_per_hour: 10     # Passive regeneration
```

**Mechanics:**
- Successful actions refund 50% of cost
- Failed actions penalize 3x cost
- At zero trust: only Tier 1 actions allowed (lockdown)
- Human reset via JWT (agent cannot self-issue)

### Shadow Simulation

Tier 3+ actions are simulated before execution:
- LLM judges potential outcomes
- SHA-256 keyed cache (5-min TTL)
- High-risk actions require human approval

## Layer 5: Storage Protection

### Encryption

AES-256-GCM encryption for sensitive data:

```yaml
# Generate key: openssl rand -base64 32
security:
  encryption_key: "${WUNDERPUS_ENCRYPTION_KEY}"
```

**Encrypted at rest:**
- Session messages
- Audit log entries
- Resource credentials (RA system)
- API keys (when prefixed with `enc:`)

### Audit Log

SHA-256 hash-chained, tamper-evident audit log:

```yaml
security:
  audit_db_path: "wunderpus_audit.db"
```

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
# Default: 60 requests/minute, burst of 10
```

## RSI Firewall

The Recursive Self-Improvement system can only modify files within `internal/`:

```yaml
genesis:
  rsi_firewall_enabled: true  # NEVER disable in production
```

The deployer validates all diffs against this restriction before applying.

## Security Checklist

Before deploying to production:

- [ ] `sanitization_enabled: true`
- [ ] `restrict_to_workspace: true`
- [ ] `rsi_firewall_enabled: true`
- [ ] Encryption key configured
- [ ] Audit log path set
- [ ] Shell whitelist reviewed
- [ ] SSRF blocklist updated for your network
- [ ] Trust budget configured appropriately
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
  audit_db_path: "/var/lib/wunderpus/audit.db"

agents:
  defaults:
    restrict_to_workspace: true
    workspace: "/var/lib/wunderpus/workspace"

genesis:
  rsi_firewall_enabled: true
  uaa_enabled: false  # Disable autonomy until thoroughly tested
  trust_budget_max: 100
```

## Threat Model

| Threat | Mitigation |
|---|---|
| Prompt injection | Input sanitization (9 patterns) |
| File system escape | Workspace sandbox |
| SSRF attacks | Network blocklist |
| Autonomous damage | Trust budget + shadow simulation |
| Data theft | AES-256-GCM encryption |
| Audit tampering | SHA-256 hash chain |
| Resource exhaustion | Rate limiting |
| Self-modification abuse | RSI firewall |
| Credential exposure | Encrypted storage |
