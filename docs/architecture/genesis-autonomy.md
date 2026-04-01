# Genesis Autonomy System

Genesis is Wunderpus's autonomous operation framework — four pillars that enable the agent to self-improve, set its own goals, act independently, and acquire resources.

## Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    GENESIS SYSTEM                            │
├──────────────┬──────────────┬──────────────┬────────────────┤
│     RSI      │     AGS      │     UAA      │      RA        │
│  Recursive   │  Autonomous  │  Unbounded   │  Resource      │
│  Self-       │  Goal        │  Autonomous  │  Acquisition   │
│  Improvement │  Synthesis   │  Action      │                │
├──────────────┼──────────────┼──────────────┼────────────────┤
│ Agent writes │ Agent sets   │ Agent acts   │ Agent provisions│
│ & deploys    │ its own      │ independently│ compute & keys  │
│ code changes │ goals        │ with trust   │ for operation   │
│              │              │ budget       │                 │
└──────────────┴──────────────┴──────────────┴────────────────┘
         │              │              │              │
         └──────────────┴──────────────┴──────────────┘
                        │
              Shared Infrastructure:
              • Event Bus (pub/sub)
              • Audit Log (hash-chained)
              • Profiler (P99 tracking)
```

## Pillar 1: RSI — Recursive Self-Improvement

The agent profiles its own performance, identifies weaknesses, proposes code improvements, tests them in sandbox, and deploys the best version.

### RSI Loop

```
Every N tasks or hourly:
    │
    ▼
┌─────────────────────────────────────────┐
│ 1. Profile                              │
│    - Track call count, duration, errors │
│    - P99 latency via ring buffer        │
│    - Persist to SQLite every 5 min      │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│ 2. Analyze Weaknesses                    │
│    - Composite score:                   │
│      error_rate × 0.5                   │
│      + p99 × 0.3                        │
│      + complexity × 0.2                 │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│ 3. Propose (3 candidates)               │
│    - Temperature 0.2 (conservative)     │
│    - Temperature 0.5 (moderate)         │
│    - Temperature 0.8 (creative)         │
│    - Parallel LLM calls                 │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│ 4. Sandbox Test                          │
│    - Copy repo to temp dir              │
│    - Apply diff                         │
│    - go build, go test, go test -race   │
│    - Optional: Docker isolation         │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│ 5. Fitness Evaluate                      │
│    - Score: latency_delta × 0.6         │
│           + error_delta × 0.4           │
│    - Gate: tests must pass              │
│    - Gate: no race conditions           │
│    - Threshold: configurable (default   │
│      0.05)                              │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│ 6. Deploy                                │
│    - Apply winning diff via git         │
│    - Create branch + rollback tag       │
│    - Rebuild binary                     │
│    - Monitor 10 min, auto-rollback if   │
│      error rate increases >20%          │
└─────────────────────────────────────────┘
```

### RSI Firewall

```yaml
genesis:
  rsi_firewall_enabled: true  # Can ONLY modify internal/ directory
```

The deployer validates all diffs — modifications outside `internal/` are rejected.

### Configuration

```yaml
genesis:
  rsi_enabled: false
  rsi_firewall_enabled: true
  rsi_self_referential_enabled: false  # Phase 4: allow RSI to modify itself
  rsi_fitness_threshold: 0.05
```

## Pillar 2: AGS — Autonomous Goal Synthesis

The agent analyzes its memory patterns and weakness reports to generate its own goals, prioritizes them, and executes them.

### Goal Hierarchy

```
Tier 0 (Immutable Root Goals)
├── "Be maximally useful"
├── "Improve own capabilities"
├── "Maintain operational continuity"
└── "Expand knowledge"
    │
    ▼
Tier 1 (Derived from Tier 0)
├── "Add support for new provider"
├── "Optimize slow tool execution"
└── "Fix recurring error patterns"
    │
    ▼
Tier 2 (Specific tasks)
├── "Implement Groq provider adapter"
├── "Add response caching to HTTP tool"
└── "Fix null pointer in file_read"
```

### Synthesis Loop

```
Every 60 minutes:
    │
    ▼
1. Analyze memory patterns (error rates, slow tasks)
2. Review weakness reports from RSI
3. LLM generates proposed goals
4. Validate:
   - Tier 1-3 only
   - Parent must be Tier 0
   - Expected value 0-1
5. Deduplicate (Jaccard similarity >0.85)
6. Score by priority
7. Save to store
```

### Execution Loop

```
Every 5 minutes:
    │
    ▼
1. Get highest-priority pending goals
2. Execute via agent or sub-agent
3. Update status: completed/failed/abandoned
4. Record actual vs expected value
```

### Metacognition

Weekly self-assessment of the goal system:
- Are goals being completed?
- Are completed goals useful?
- Should scoring weights be adjusted?

### Configuration

```yaml
genesis:
  ags_enabled: false
```

## Pillar 3: UAA — Unbounded Autonomous Action

The agent acts independently through a trust budget system with 4-tier action classification.

### Action Tiers

| Tier | Type | Cost | Examples |
|---|---|---|---|
| 1 | Read-only | 0 | web_search, file_read, file_list, calculator, system_info |
| 2 | Ephemeral | 1 | temp file writes, go build, go test, sandbox exec |
| 3 | Persistent | 5 | file writes, git commit, DB modifications |
| 4 | External | 20 | HTTP POST, deploy, send comms, spend money |

### Autonomy Gate

```
Action Proposed
    │
    ▼
Classify (rule-based tier assignment)
    │
    ▼
Trust Check (can afford cost?)
    │  (no → lockdown, only Tier 1)
    ▼
Shadow Simulation (Tier 3+ only)
    │  LLM judges potential outcomes
    ▼
Execute
    │
    ▼
Record outcome
    ├── Success → refund 50% cost
    └── Failure → penalize 3x cost
```

### Trust Budget

```yaml
genesis:
  trust_budget_max: 1000        # Maximum points
  trust_regen_per_hour: 10     # Passive regeneration (1 per 360s)
```

**Mechanics:**
- TOCTOU-safe atomic deduct (check + deduct in single lock)
- Zero trust = lockdown (Tier 1 only)
- Human reset via externally-signed JWT
- Agent cannot self-issue JWT

### Configuration

```yaml
genesis:
  uaa_enabled: false
  trust_budget_max: 1000
  trust_regen_per_hour: 10
  jwt_secret_env: "WUNDERPUS_JWT_SECRET"
```

## Pillar 4: RA — Resource Acquisition

The agent provisions cloud resources, manages API keys, and forecasts resource needs.

### Resource Types

| Type | Description |
|---|---|
| Compute | VMs, containers, serverless functions |
| Storage | Block storage, object storage |
| API Key | LLM provider keys, service credentials |
| Financial | Payment methods, billing accounts |
| Data | Datasets, subscriptions |

### Cloud Providers

| Provider | Package |
|---|---|
| AWS | `ra/cloud/` |
| GCP | `ra/cloud/` |
| DigitalOcean | `ra/cloud/` |
| Vast.ai | `ra/cloud/` |
| Local | `ra/cloud/` |

### Security

- All credentials encrypted with AES-256-GCM
- Spend cap enforced before any provision call
- Resource lifecycle tracked in SQLite registry

### Configuration

```yaml
genesis:
  ra_enabled: false
  max_daily_spend_usd: 10.0
```

## Shared Infrastructure

### Event Bus

All pillars communicate via typed pub/sub:

```go
// RSI publishes
bus.Publish(audit.RSIProposalGenerated, event)

// AGS subscribes
bus.Subscribe(audit.RSIProposalGenerated, handler)
```

Features:
- Non-blocking (handlers in separate goroutines)
- Panic recovery per handler
- Dead-letter queue (max 1000)

### Audit Log

Every Genesis action logged with:
- SHA-256 hash chain (tamper-evident)
- Subsystem identifier
- Event type
- Actor ID
- Payload

### Profiler

Ring-buffer P99 latency tracking:
- Per-function call count
- Duration percentiles
- Error/success counts
- Persisted to SQLite every 5 minutes

## Progressive Enablement

Genesis systems should be enabled progressively:

```yaml
genesis:
  # Start with infrastructure
  toolsynth_enabled: true
  worldmodel_enabled: true
  perception_enabled: true
  swarm_enabled: true

  # Core autonomy — enable when ready
  rsi_enabled: false        # Test RSI in isolation first
  ags_enabled: false        # Requires RSI data
  uaa_enabled: false        # Requires trust budget setup
  ra_enabled: false         # Requires cloud credentials
```

## Safety Guarantees

| Guarantee | Mechanism |
|---|---|
| RSI can't break core code | Firewall (internal/ only) + sandbox testing |
| Agent can't spend unlimitedly | Spend cap + trust budget |
| Agent can't modify itself | `rsi_self_referential_enabled: false` |
| Autonomous actions are limited | 4-tier classification + shadow simulation |
| All actions are auditable | Hash-chained audit log |
| Human can always intervene | JWT trust reset + manual override |
