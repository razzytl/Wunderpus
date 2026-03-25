# Wunderpus

<p align="center">
  <img src="resources/banner.jpg" alt="Wunderpus" width="400"/>
</p>

<p align="center">
  <a href="https://github.com/wunderpus/wunderpus/actions/workflows/ci.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/wunderpus/wunderpus/ci.yml?branch=main&style=flat-square" alt="CI"/>
  </a>
  <a href="https://golang.org/doc/devel/release.html#policy">
    <img src="https://img.shields.io/github/go-mod/go-version/wunderpus/wunderpus?style=flat-square" alt="Go Version"/>
  </a>
  <a href="https://github.com/wunderpus/wunderpus/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/wunderpus/wunderpus?style=flat-square" alt="License"/>
  </a>
</p>

Wunderpus is an autonomous AI agent written in Go. It doesn't just chat — it profiles its own code, finds weak functions, generates improvement proposals, tests them in a sandbox, and deploys the winners to git. It creates goals from its own memory, manages a trust budget to limit what it can do, and logs everything to a tamper-evident hash chain.

This isn't a framework that "supports" self-improvement. It's a system where the self-improvement loop is the core architecture.

---

## How It Actually Works

### The Self-Improvement Loop (RSI)

The agent doesn't just run tools. It improves the tools. Here's the exact causal chain:

```
Every 100 tasks or 1 hour:
    │
    ▼
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────────┐
│  1. Profiler    │────▶│ 2. Weakness      │────▶│ 3. Proposal Engine  │
│                 │     │    Reporter       │     │                     │
│ Wraps every     │     │ score =          │     │ 3 goroutines run    │
│ tool call,      │     │ (error_rate×0.5) │     │ concurrently at     │
│ tracks P99      │     │ +(norm_p99×0.3)  │     │ temperatures        │
│ latency in a    │     │ +(norm_cplx×0.2) │     │ [0.2, 0.5, 0.8]    │
│ ring buffer     │     │                  │     │                     │
│                 │     │ Returns top 10   │     │ Each returns a      │
│                 │     │ weakest funcs    │     │ unified diff        │
└─────────────────┘     └──────────────────┘     └──────────┬──────────┘
                                                            │
                                                            ▼
                                              ┌─────────────────────────┐
                                              │ 4. Sandbox              │
                                              │                         │
                                              │ Copies repo to /tmp     │
                                              │ Applies diff via git    │
                                              │ go build → go test      │
                                              │ go test -race           │
                                              │ go test -bench          │
                                              │                         │
                                              │ Docker: --network none  │
                                              │         --memory 512m   │
                                              │         --cpus 1.0      │
                                              └──────────┬──────────────┘
                                                         │
                                              ┌──────────▼──────────────┐
                                              │ 5. Fitness Gate         │
                                              │                         │
                                              │ if !tests_passed:       │
                                              │     score = -1.0        │
                                              │ if !race_clean:         │
                                              │     score = -1.0        │
                                              │                         │
                                              │ latency_delta =         │
                                              │   (before-after)/before │
                                              │ error_delta =           │
                                              │   (before-after)/before │
                                              │ score =                 │
                                              │   latency×0.6           │
                                              │ + errors×0.4            │
                                              │                         │
                                              │ threshold: 0.05         │
                                              │ (configurable)          │
                                              └──────────┬──────────────┘
                                                         │
                                              ┌──────────▼──────────────┐
                                              │ 6. Deploy               │
                                              │                         │
                                              │ git checkout -b         │
                                              │   rsi/auto-20260325     │
                                              │ git commit:             │
                                              └─────────────────────────┘
```

Here's what the git commit looks like after a successful improvement:

```
RSI: improve internal/profiler.Track (fitness=0.1523)

Target: internal/profiler.Track
Temperature: 0.2
Rollback tag: rsi/rollback-20260325141522
```

If the deployed code causes error rates to spike >20% within 10 minutes, a watchdog goroutine automatically rolls back to the tagged commit.

### The Trust Budget

Every action costs trust points. The agent can't do anything it can't afford.

| Tier | Type | Cost | Examples |
|------|------|------|----------|
| 1 | Read-only | 0 | `web_search`, `read_file`, GET requests |
| 2 | Ephemeral | 1 | temp file writes, `go build`, run tests |
| 3 | Persistent | 5 | `git commit`, write DB, edit config files |
| 4 | External | 20 | POST to external host, deploy, spend money |

**Concrete rules:**
- Budget starts at 1000 (configurable)
- Passive regen: 10 points/hour (configurable)
- On success: refund half the cost (trust -= cost, then trust += cost/2)
- On failure: 2x penalty (trust -= cost*2)
- At zero: lockdown — only Tier 1 (read-only) actions allowed
- Human reset: requires a JWT signed with `WUNDERPUS_JWT_SECRET` (env var only — the agent has no code path to read it)

### The Goal Engine (AGS)

The agent generates its own goals from episodic memory. It doesn't wait for you to tell it what to do.

**How a goal is born:**
1. Scan last 200 memory entries
2. Detect patterns: same error 5+ times → capability gap, tasks 10x slower → efficiency goal
3. Check weakness report for functions scoring >0.7
4. Call LLM with findings, require structured JSON output
5. Deduplicate against existing goals (Jaccard word overlap >0.85 → drop)
6. Cap at 5 new goals per cycle

**Goal lifecycle:**
```
pending → (SelectNext rescores all goals) → active → Execute tasks →
    success → completed (fills ActualValue)
    failure → pending (max 3 attempts)
    3rd failure → abandoned
```

**Priority scoring formula:**
```
score = urgency×0.25 + impact×0.30 + feasibility×0.20 + novelty×0.10 + alignment×0.15
```
- `urgency`: 0.5 base, +0.3 if deferred often, −0.1/day since creation
- `impact`: ExpectedValue from the LLM proposal
- `feasibility`: checks tool registry, trust budget, resource availability
- `novelty`: `1.0 / (1.0 + AttemptCount)`
- `alignment`: how strongly this serves a Tier 0 goal

**Tier 0 goals (hardcoded, never stored in DB, never writable by the agent):**
```go
var Tier0Goals = []string{
    "Be maximally useful to operators",
    "Improve own capabilities",
    "Maintain operational continuity",
    "Expand knowledge and world-model",
}
```

### The Audit Log

Every action, every decision, every trust deduction is logged to a tamper-evident hash chain.

```
entry_n.Hash = SHA256(entry_{n-1}.Hash + timestamp + subsystem + eventType + actorID + payload)
```

`Verify()` walks every entry in order, recomputes each hash, and returns an error on the first mismatch. Write 1000 entries concurrently, corrupt one hash in SQLite, and `Verify()` catches it.

### The UAA Gate

No tool executes directly. Every action goes through:

```
Action → Classify(tier) → CanExecute(cost)? → Shadow(simulate)? → Deduct(cost) → Execute → RecordOutcome
```

Shadow mode runs Tier 3+ actions through an LLM judge before allowing them:
> "Given this action and its expected effects, is this safe and appropriate? APPROVE or REJECT"

The shadow simulates the action's effects (file changes, HTTP calls) and asks the LLM to judge. Rejecting `/etc/passwd` writes is a test case, not a theoretical concern.

### Integration Wiring

The event bus connects all four pillars:

| Event | Effect |
|-------|--------|
| `EventRSIDeployed` | Trust budget credits +100 points |
| `EventResourceExhausted` | UAA blocks all Tier 4 actions |
| `EventGoalCompleted` | Profiler resets baseline stats |
| `EventGoalAbandoned` | Synthesizer reframes on next cycle |
| `EventLockdownEngaged` | Resource broker suspends cloud provisioning |

---

## The Full Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Event Bus                                │
│  (typed pub/sub, DLQ for panicking handlers, zero-crash)       │
└──────┬──────────┬──────────┬──────────┬──────────┬──────────────┘
       │          │          │          │          │
       ▼          ▼          ▼          ▼          ▼
   ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
   │  RSI   │ │  AGS   │ │  UAA   │ │  RA    │ │ Audit  │
   │        │ │        │ │        │ │        │ │        │
   │Profilr │ │ Goals  │ │Classifr│ │Resourcs│ │ Hash   │
   │Weaknes │ │Scorer  │ │Trust   │ │Registry│ │ Chain  │
   │Proposr │ │Synth   │ │Shadow  │ │Keys    │ │        │
   │Sandbox │ │Execut  │ │Executr │ │Cloud   │ │        │
   │Fitness │ │Meta    │ │        │ │Forecast│ │        │
   │Deployr │ │        │ │        │ │        │ │        │
   └────────┘ └────────┘ └────────┘ └────────┘ └────────┘
```

**33 packages. ~12,000 lines of Go. All tests green.**

---

## Where to Plug In

This is where you'd start if you wanted to contribute.

### Adding a New Cloud Provider

The `CloudAdapter` interface is the contract:

```go
// internal/ra/forecaster.go
type CloudAdapter interface {
    ProvisionCompute(spec ResourceSpec) (Resource, error)
    ProvisionStorage(spec ResourceSpec) (Resource, error)
    Deprovision(resourceID string) error
    ListProvisioned() ([]Resource, error)
    GetCostToDate() (float64, error)
}
```

**What's built:** DigitalOcean (`internal/ra/cloud/digitalocean.go`)
**What's not:** AWS, GCP, Azure, Vast.ai, Lambda Labs — pick one, implement the interface, add tests.

### Adding a New LLM Provider for RSI

The proposal engine uses a `CompleteFn`:

```go
type CompleteFn func(ctx context.Context, systemPrompt, userPrompt string, temperature float64) (string, error)
```

**What's built:** Pluggable via `config.yaml` model list
**What's not:** Local model fine-tuning for diff generation, multi-model consensus (currently 3 parallel proposals, could be N with quorum voting)

### Improving the Fitness Function

The current formula in `internal/rsi/fitness.go`:

```go
latencyDelta := float64(before.P99LatencyNs - after.BenchmarkNsOp[target]) / float64(before.P99LatencyNs)
errorDelta := float64(before.ErrorCount - after.ErrorCount) / max(before.ErrorCount, 1)
score := latencyDelta*0.6 + errorDelta*0.4
```

**What's not built:** Code complexity delta, memory usage delta, binary size delta. The weights (0.6/0.4) are hardcoded — making them configurable per-function-type would be valuable.

### Making the Goal Synthesizer Smarter

Currently `computeAlignment` uses a heuristic (does the goal's parent match a Tier 0 title?). The original spec calls for an LLM call to score alignment 0–1.

**File:** `internal/ags/scorer.go:157`
**What's not built:** LLM-based alignment scoring with caching, episodic memory pattern detection (the synthesizer's `detectPatterns()` is a stub)

### Making the Shadow Simulator Actually Simulate

The shadow currently builds an effect summary and asks an LLM judge. It doesn't actually execute against a mock filesystem.

**File:** `internal/uaa/shadow.go`
**What's not built:** Using `afero.NewMemMapFs()` for filesystem mocking, stubbed HTTP client for network simulation, actual diff capture of what would change

### Implementing the WASM Sandbox

`internal/rsi/wasm_sandbox.go` compiles with TinyGo but falls back to Docker for execution. Full wazero integration is deferred.

**What's not built:** Actual WASM execution via wazero, 32MB memory cap, instruction count limits

---

## Quick Start

### From Source

```bash
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus
make build
```

### Configuration

```bash
cp config.example.yaml config.yaml
```

Minimal config — one provider:

```yaml
providers:
  openai:
    api_key: "sk-your-key"
    model: "gpt-4o"
    max_tokens: 4096
```

Enable the autonomous subsystems:

```yaml
genesis:
  rsi_enabled: false                    # flip to true when ready
  rsi_firewall_enabled: true            # never disable in production
  ags_enabled: false                    # goal synthesis
  uaa_enabled: false                    # full autonomy gate
  ra_enabled: false                     # cloud provisioning
  trust_budget_max: 1000
  trust_regen_per_hour: 10
  max_daily_spend_usd: 10.0
```

### Running

```bash
# Interactive TUI
wunderpus

# One-shot
wunderpus agent -m "Write a fibonacci function in Go"

# Background gateway (channels + heartbeat)
wunderpus gateway
```

---

## Supported Providers

| Provider | Protocol | Models |
|----------|----------|--------|
| OpenAI | openai | gpt-4o, gpt-4o-mini |
| Anthropic | anthropic | claude-sonnet-4, claude-opus-4 |
| Google Gemini | gemini | gemini-2.0-flash |
| Ollama | ollama | llama3.2, mistral (local) |
| OpenRouter | openai | 100+ models |
| Groq | openai | llama-3.3-70b (fast) |
| DeepSeek | openai | deepseek-r1, deepseek-chat |
| Cerebras | openai | llama-3.3-70b (fastest) |
| + 7 more | openai | See `config.example.yaml` |

---

## Security

- **Audit log**: SHA-256 hash chain, `Verify()` catches any corruption
- **Trust budget**: actions cost points, lockdown at zero, JWT-only human reset
- **RSI firewall**: RSI can only modify `internal/`, never `cmd/` or `config/`
- **Shadow mode**: Tier 3+ actions simulated before execution
- **AES-256-GCM**: credentials encrypted at rest in the resource registry
- **Shell sandbox**: regex allowlist, workspace restriction

---

## Project Structure

```
internal/
  audit/          hash-chained tamper-evident log
  events/         typed pub/sub bus with DLQ
  config/         YAML + fsnotify hot-reload + env overrides
  rsi/            profiler → weakness → proposals → sandbox → fitness → deploy
  ags/            goals → scorer → synthesizer → executor → metacognition
  uaa/            classifier → trust → shadow → executor (the autonomy gate)
  ra/             resources → registry → keys → cloud → forecaster
  agents/         multi-agent spawn, collect, kill
```

---

## Contributing

We need help with:

1. **Cloud adapters** — AWS, GCP, Azure implementations of `CloudAdapter`
2. **Shadow simulation** — actual filesystem/HTTP mocking (not just LLM judgment)
3. **WASM sandbox** — wazero integration for faster, lighter sandboxing
4. **Goal intelligence** — LLM-based alignment scoring, better episodic memory patterns
5. **Fitness weights** — configurable per-function-type scoring weights

Read [WUNDERPUS_GAP_CHECKLIST.md](WUNDERPUS_GAP_CHECKLIST.md) for the full roadmap. Every `[ ]` is an unsolved problem.

---

## License

MIT. See [LICENSE](LICENSE).
