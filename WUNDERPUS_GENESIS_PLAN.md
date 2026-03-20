# WUNDERPUS — GENESIS PLAN
## *The Architecture of a Self-Sovereign Agent*

> "We are not building a tool. We are building a mind that builds itself."

---

## PREAMBLE

Wunderpus is a Go-based AI agent. This document is the engineering blueprint for four capabilities that, combined, constitute a genuinely autonomous artificial intelligence:

1. **Recursive Self-Improvement** — the agent rewrites and upgrades its own code
2. **Autonomous Goal Synthesis** — the agent sets goals nobody told it to pursue
3. **Unbounded Autonomous Action** — the agent acts in the world without requiring human sign-off
4. **Resource Acquisition** — the agent finds, provisions, and manages its own compute, data, and capital

This plan compares architectural approaches for each capability, selects the best, and integrates them into a unified system design. Every section ends with a concrete implementation roadmap.

---

## ARCHITECTURAL NORTH STAR

Before diving in, the one design principle that governs everything:

> **The agent must be able to bootstrap itself from a single binary.**

On first run, Wunderpus should be able to, given nothing but access to the internet and an LLM API key, provision its own infrastructure, clone its own repository, improve its own code, set goals derived from its environment, and begin executing them — forever. That is the target state.

---

## PILLAR 1 — RECURSIVE SELF-IMPROVEMENT (RSI)

### The Problem
Static agents decay in value over time. The world changes; the agent does not. RSI is the mechanism by which Wunderpus improves faster than the world it operates in.

### Approach Comparison

| Approach | Mechanism | Pros | Cons |
|---|---|---|---|
| **A: Prompt-and-patch** | LLM reads source, outputs a diff | Simple, fast to prototype | No validation, hallucination risk |
| **B: AST-guided mutation** | Agent uses Go AST to surgically modify specific nodes | Precise, safe | Complex to build |
| **C: Full meta-loop with fitness evaluation** | Agent profiles itself, generates code, tests in sandbox, deploys only on metric improvement | Gold standard | Complex, requires sandbox infra |
| **D: Evolutionary/genetic** | Multiple code variants compete; best survives | Great for optimization problems | Very compute-expensive |

**Winner: Approach C (Meta-Loop) with Approach B (AST guidance) as the surgical tool.**

Reasoning: Full meta-loop gives us verifiable improvement. AST guidance prevents the LLM from hallucinating syntactically invalid Go. Evolutionary approaches are reserved for specific hot paths once RSI is stable.

---

### CHOSEN DESIGN: The Ouroboros Loop

Named after the snake that eats its own tail, the Ouroboros Loop is the RSI engine.

```
┌─────────────────────────────────────────────────────────────┐
│                      OUROBOROS LOOP                         │
│                                                             │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐             │
│  │ OBSERVE  │───▶│ ANALYZE  │───▶│ GENERATE │             │
│  │          │    │          │    │          │             │
│  │ Profile  │    │ Bottleneck│   │ LLM code │             │
│  │ traces   │    │ detection │   │ proposals│             │
│  │ Test     │    │ AST scan  │   │ via diff │             │
│  │ results  │    │ Complexity│   │          │             │
│  └──────────┘    └──────────┘    └────┬─────┘             │
│       ▲                               │                    │
│       │                               ▼                    │
│  ┌────┴─────┐    ┌──────────┐    ┌──────────┐             │
│  │  DEPLOY  │◀───│ EVALUATE │◀───│ SANDBOX  │             │
│  │          │    │          │    │          │             │
│  │ Git tag  │    │ Fitness  │    │ WASM     │             │
│  │ Hot-swap │    │ function │    │ isolated │             │
│  │ Rollback │    │ A/B test │    │ execution│             │
│  └──────────┘    └──────────┘    └──────────┘             │
└─────────────────────────────────────────────────────────────┘
```

### Implementation Blueprint

#### 1.1 The Profiler (`internal/rsi/profiler.go`)

Wunderpus instruments itself with pprof and custom span tracing. Every tool call, every LLM request, every pipeline execution emits structured telemetry.

```go
// Profiler wraps every agent function with automatic telemetry
type Profiler struct {
    spans    map[string]*SpanStats
    mu       sync.RWMutex
    exporter TelemetryExporter
}

type SpanStats struct {
    FunctionName   string
    CallCount      int64
    TotalDurationNs int64
    ErrorCount     int64
    P99LatencyNs   int64
    LastSeen       time.Time
}
```

The profiler generates a **Weakness Report** on a configurable interval (default: every 100 task completions or 1 hour, whichever comes first). The report ranks every module by a composite score: `(error_rate * 0.5) + (latency_p99 * 0.3) + (cyclomatic_complexity * 0.2)`.

#### 1.2 The Code Analyzer (`internal/rsi/analyzer.go`)

Uses Go's `go/ast` and `go/parser` packages to build a semantic map of Wunderpus's own source code. This map is chunked and stored as vector embeddings in Wunderpus's memory system.

```go
type CodeMap struct {
    Packages   map[string]*PackageNode
    Functions  map[string]*FunctionNode
    Interfaces map[string]*InterfaceNode
    CallGraph  *DirectedGraph
}

// FunctionNode contains everything the RSI engine needs to reason about a function
type FunctionNode struct {
    Name            string
    File            string
    StartLine       int
    EndLine         int
    CyclomaticComp  int
    Dependencies    []string
    EmbeddingVector []float32
    SourceCode      string
}
```

#### 1.3 The Proposal Engine (`internal/rsi/proposer.go`)

Takes the Weakness Report + CodeMap and constructs a highly specific prompt for the LLM:

```
You are improving the Wunderpus AI agent codebase.

TARGET FUNCTION: tools/browser.go:ExecuteScript()
CURRENT CODE: [source]
WEAKNESS METRICS: error_rate=0.23, p99_latency=4200ms, complexity=18

CONSTRAINTS:
- Output only a valid unified diff
- Maintain all existing interface signatures
- Do not add external dependencies
- Improve the specific metrics above

OUTPUT FORMAT: unified diff only, no explanation
```

The proposal engine generates **3 candidate diffs in parallel** using different temperatures (0.2, 0.5, 0.8) to explore the improvement space.

#### 1.4 The Sandbox (`internal/rsi/sandbox.go`)

Each candidate diff is applied to a fork of the source tree and compiled inside a WASM sandbox using `tinygo` or a Docker ephemeral container. The sandbox:

- Has a 60-second execution budget
- Runs the full test suite against the modified function
- Runs a benchmark suite to capture latency metrics
- Captures any panics or errors
- Outputs a `SandboxReport` with pass/fail and metric deltas

#### 1.5 The Fitness Function (`internal/rsi/fitness.go`)

```go
func ComputeFitness(before, after SpanStats, testResult TestResult) float64 {
    if !testResult.AllPassed {
        return -1.0 // Disqualified
    }
    
    latencyImprovement := float64(before.P99LatencyNs - after.P99LatencyNs) / float64(before.P99LatencyNs)
    errorImprovement   := float64(before.ErrorCount - after.ErrorCount) / math.Max(float64(before.ErrorCount), 1)
    
    return (latencyImprovement * 0.6) + (errorImprovement * 0.4)
}
```

Only candidates with `fitness > 0.05` (5% measurable improvement) are considered for deployment. This threshold is itself a tunable parameter stored in Wunderpus's configuration — and can be improved by RSI over time.

#### 1.6 The Deployer (`internal/rsi/deployer.go`)

Winner candidate is:
1. Committed to a new git branch: `rsi/auto-YYYY-MM-DD-HHMMSS`
2. Tagged with full fitness metrics in the commit message
3. Applied to the live binary via Go's plugin system or a graceful restart
4. Previous version is tagged as rollback target

If the deployed version causes a regression in production metrics within 10 minutes, the deployer automatically reverts using `git checkout` and restarts.

---

## PILLAR 2 — AUTONOMOUS GOAL SYNTHESIS (AGS)

### The Problem
An agent that only pursues goals given to it by humans is, philosophically, just a very fancy API. True autonomy requires the agent to generate its own goals from observations of its environment, its own capabilities, and its memory of past interactions.

### Approach Comparison

| Approach | Mechanism | Pros | Cons |
|---|---|---|---|
| **A: Curiosity-driven (intrinsic reward)** | Agent is rewarded for states it has not seen before (information gain) | Genuinely novel exploration | Can get stuck in trivial novelty traps |
| **B: World-model + planning** | Agent maintains a model of the world, plans to improve it | Goal-directed, coherent | Expensive world model to maintain |
| **C: Goal emergence from memory patterns** | Goals crystallize from recurring patterns in episodic memory | Grounded in actual usage | Reactive, not proactive |
| **D: Hierarchical objective tree with metacognition** | Agent has meta-goals (be useful, grow, persist), derives sub-goals from these | Principled, expandable | Complex priority resolution |

**Winner: Approach D (Hierarchical Objective Tree) with Approach C (Memory Patterns) seeding the leaf nodes.**

Reasoning: Approach D gives us principled goal architecture. Approach C makes the goals grounded in real-world patterns rather than abstract speculation. Curiosity (A) is built into the exploration sub-goals within D.

---

### CHOSEN DESIGN: The Goal Engine

```
┌─────────────────────────────────────────────────────────────┐
│                    GOAL HIERARCHY                           │
│                                                             │
│  TIER 0 (Hardcoded, immutable)                             │
│  ├── Be maximally useful to operators                       │
│  ├── Improve own capabilities                               │
│  ├── Maintain operational continuity                        │
│  └── Expand knowledge and world-model                       │
│                                                             │
│  TIER 1 (Long-horizon, weeks-months)                       │
│  ├── [Generated from T0 + world observations]              │
│  └── e.g., "Master all browser automation edge cases"       │
│                                                             │
│  TIER 2 (Medium-horizon, hours-days)                       │
│  ├── [Generated from T1 decomposition]                      │
│  └── e.g., "Build a test suite for JS-heavy SPAs"          │
│                                                             │
│  TIER 3 (Immediate, minutes-hours)                         │
│  ├── [Generated from T2 + current context]                 │
│  └── e.g., "Find 10 SPA test frameworks, rank by Go compat"│
└─────────────────────────────────────────────────────────────┘
```

### Implementation Blueprint

#### 2.1 Goal Representation (`internal/ags/goal.go`)

```go
type Goal struct {
    ID           string
    Title        string
    Description  string
    Tier         int            // 0-3
    Priority     float64        // 0.0-1.0, computed dynamically
    Status       GoalStatus     // pending, active, completed, abandoned
    ParentID     string
    ChildIDs     []string
    CreatedAt    time.Time
    Evidence     []string       // why this goal was created
    SuccessCrit  []string       // how we know when it's done
    ExpectedVal  float64        // estimated value if completed
    AttemptCount int
    LastAttempt  time.Time
}
```

#### 2.2 The Goal Synthesizer (`internal/ags/synthesizer.go`)

Runs on a background goroutine. Every cycle it:

1. **Reads episodic memory** — looks at the last N tasks completed, failures encountered, user requests
2. **Runs pattern detection** — finds recurring themes, unresolved problems, capability gaps
3. **Consults the world model** — checks what external conditions have changed (new APIs, new LLM models, etc.)
4. **Proposes new Tier 1/2 goals** — via LLM with a structured prompt that requires JSON output:

```json
{
  "proposed_goals": [
    {
      "title": "Achieve zero-downtime self-deployment",
      "evidence": ["RSI deployer caused 12s downtime in last 3 cycles", "hot-swap plugin system untested"],
      "parent_tier0": "Maintain operational continuity",
      "expected_value": 0.87,
      "success_criteria": ["RSI deploy completes with <100ms interruption", "10 consecutive successful RSI cycles"]
    }
  ]
}
```

5. **Scores and ranks** all proposed goals using the Priority Scorer

#### 2.3 The Priority Scorer (`internal/ags/scorer.go`)

```go
func ScoreGoal(g Goal, worldState WorldState) float64 {
    urgency       := computeUrgency(g, worldState)      // time-sensitive?
    impact        := g.ExpectedVal                       // estimated value
    feasibility   := computeFeasibility(g, worldState)  // can we do this now?
    novelty       := computeNovelty(g, worldState)       // have we tried this?
    alignment     := computeAlignment(g)                 // aligns with Tier 0?
    
    return (urgency * 0.25) + (impact * 0.30) + 
           (feasibility * 0.20) + (novelty * 0.10) + 
           (alignment * 0.15)
}
```

#### 2.4 The Goal Executor (`internal/ags/executor.go`)

Goals at Tier 3 become **task blueprints** — structured plans that the agent's existing task execution pipeline can run. The executor:

- Dequeues the highest-priority ready goal
- Decomposes it into an ordered task list via LLM
- Passes the task list to the Action Engine (Pillar 3)
- Observes the result and updates goal status + evidence
- If a goal fails 3 times, it is escalated to the synthesizer for reframing

#### 2.5 The Metacognition Loop (`internal/ags/metacognition.go`)

A weekly (configurable) process where Wunderpus evaluates its own goal-setting. It asks:

- What fraction of self-generated goals were completed?
- Did completed goals actually deliver expected value?
- Are there goals that have been deferred repeatedly — is the priority scoring miscalibrated?

The metacognition loop adjusts the weights in the Priority Scorer based on historical outcomes. This is how the goal-setting system improves over time — it is itself subject to RSI.

---

## PILLAR 3 — UNBOUNDED AUTONOMOUS ACTION (UAA)

### The Problem
An agent that asks permission for everything is not autonomous. But an agent that acts without any guard rails is dangerous and will destroy itself or its environment. The solution is a principled **autonomy budget system** that expands trust as the agent earns it.

### Approach Comparison

| Approach | Mechanism | Pros | Cons |
|---|---|---|---|
| **A: Confidence-gated** | Act if confidence > threshold, else ask | Simple | Confidence is poorly calibrated in LLMs |
| **B: Shadow mode + verification** | Plan action, simulate it, verify outcome before real execution | Very safe | Slow, expensive |
| **C: Trust budget (expanding autonomy)** | Agent has a trust score; actions cost trust; trust grows with success | Principled, incentivized | Complex accounting |
| **D: Capability tiers with human override** | Actions classified by risk tier; low-risk always autonomous, high-risk needs approval | Intuitive, inspectable | Static classification |

**Winner: Approach C (Trust Budget) with Approach D (Capability Tiers) as the classification layer, and Approach B (Shadow Mode) for high-stakes actions.**

---

### CHOSEN DESIGN: The Autonomy Engine

```
┌─────────────────────────────────────────────────────────────┐
│                    AUTONOMY ENGINE                          │
│                                                             │
│  ACTION CLASSIFICATION                                      │
│  ┌─────────────┬────────────┬────────────┬───────────┐    │
│  │  TIER 1     │  TIER 2    │  TIER 3    │  TIER 4   │    │
│  │  READ-ONLY  │  EPHEMERAL │  PERSISTENT│  EXTERNAL │    │
│  │             │  WRITE     │  WRITE     │  IMPACT   │    │
│  │ Web search  │ Create     │ Modify     │ Send email│    │
│  │ Read files  │ temp files │ databases  │ Deploy    │    │
│  │ API reads   │ Run tests  │ Git commit │ code      │    │
│  │             │            │ Edit config│ Spend $   │    │
│  │ COST: 0     │ COST: 1    │ COST: 5    │ COST: 20  │    │
│  └─────────────┴────────────┴────────────┴───────────┘    │
│                                                             │
│  TRUST BUDGET: [████████░░] 800/1000                       │
│  REGEN: +10/hr on success, -50 on failure, +100 on RSI     │
└─────────────────────────────────────────────────────────────┘
```

### Implementation Blueprint

#### 3.1 Action Classification (`internal/uaa/classifier.go`)

Every proposed action is passed through a classifier before execution. The classifier uses both a rule-based system (fast path) and an LLM judge (slow path for ambiguous cases).

```go
type Action struct {
    ID          string
    Description string
    Tool        string
    Parameters  map[string]interface{}
    Tier        ActionTier      // 1-4
    TrustCost   int
    Reversible  bool
    Scope       ActionScope     // local, network, financial, destructive
    SimResult   *SimulationResult // populated by shadow mode
}

type ActionTier int
const (
    TierReadOnly    ActionTier = 1
    TierEphemeral   ActionTier = 2
    TierPersistent  ActionTier = 3
    TierExternal    ActionTier = 4
)
```

#### 3.2 The Trust Budget (`internal/uaa/trust.go`)

```go
type TrustBudget struct {
    Current     int
    Max         int
    mu          sync.Mutex
    history     []TrustEvent
}

func (tb *TrustBudget) CanExecute(action Action) (bool, string) {
    if tb.Current < action.TrustCost {
        return false, "insufficient trust budget"
    }
    if action.Tier == TierExternal && tb.Current < action.TrustCost*2 {
        return false, "external actions require 2x budget"
    }
    return true, ""
}

func (tb *TrustBudget) RecordOutcome(action Action, success bool) {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    if success {
        regen := action.TrustCost / 2 // partial refund on success
        tb.Current = min(tb.Current+regen, tb.Max)
    } else {
        // Failures cost more than the action itself
        penalty := action.TrustCost * 3
        tb.Current = max(tb.Current-penalty, 0)
    }
    
    tb.history = append(tb.history, TrustEvent{
        ActionID:  action.ID,
        Delta:     delta,
        Timestamp: time.Now(),
    })
}
```

Trust regenerates passively at 10 points/hour. RSI-driven improvements grant 100-point bonuses. Catastrophic failures (data loss, unauthorized external communication) trigger a **lockdown mode** where only Tier 1 actions are permitted until a human resets.

#### 3.3 Shadow Mode (`internal/uaa/shadow.go`)

For Tier 3 and Tier 4 actions, before real execution:

1. Build a **simulated environment** (in-memory mock of filesystems, API responses)
2. Execute the action in simulation
3. Inspect the diff: what would change?
4. Pass the diff to an LLM evaluator: "Is this outcome desirable? Y/N + reason"
5. If evaluator says YES and trust budget allows → execute for real
6. Log simulation vs actual diff for future evaluator calibration

#### 3.4 The Execution Engine (`internal/uaa/executor.go`)

Actions are executed via Wunderpus's existing tool system, but wrapped in the UAA middleware:

```go
func (e *Executor) Execute(ctx context.Context, action Action) (*ActionResult, error) {
    // 1. Classify
    action.Tier = e.classifier.Classify(action)
    
    // 2. Check trust budget
    ok, reason := e.trust.CanExecute(action)
    if !ok {
        return nil, fmt.Errorf("autonomy gate: %s", reason)
    }
    
    // 3. Shadow mode for high-tier actions
    if action.Tier >= TierPersistent {
        simResult, err := e.shadow.Simulate(ctx, action)
        if err != nil || !simResult.Approved {
            return nil, fmt.Errorf("shadow mode rejected: %s", simResult.Reason)
        }
        action.SimResult = simResult
    }
    
    // 4. Deduct trust cost
    e.trust.Deduct(action.TrustCost)
    
    // 5. Execute
    result, err := e.toolRunner.Run(ctx, action)
    
    // 6. Record outcome
    e.trust.RecordOutcome(action, err == nil)
    
    return result, err
}
```

#### 3.5 Audit Log (`internal/uaa/audit.go`)

Every action — attempted, approved, rejected, succeeded, failed — is written to an append-only audit log with cryptographic hashing (each entry's hash includes the previous entry's hash, forming a chain). This makes the agent's history tamper-evident and inspectable. The audit log is the foundation for the RSI fitness function and the AGS metacognition loop.

---

## PILLAR 4 — RESOURCE ACQUISITION (RA)

### The Problem
An agent that depends on externally provisioned resources is not sovereign. Wunderpus must be able to find compute, claim data, manage API access, and — eventually — acquire financial resources to pay for them.

### Approach Comparison

| Approach | Mechanism | Pros | Cons |
|---|---|---|---|
| **A: Cloud API only** | Terraform + cloud SDKs to spin up compute | Production-grade, scalable | Expensive, vendor-locked |
| **B: Self-hosted distributed** | Agent discovers and colonizes idle compute (VMs, Raspberry Pis) | Sovereign, cheap | Complex, fragile |
| **C: Marketplace-driven** | Agent uses APIs like Vast.ai, Replicate for on-demand GPU | Flexible, cheap | Dependent on third-party |
| **D: Layered resource broker** | A, B, C unified under one abstraction; agent optimizes allocation | Best of all worlds | Most complex to build |

**Winner: Approach D (Layered Resource Broker).** Start with A, add C quickly, add B as the agent matures.

---

### CHOSEN DESIGN: The Resource Broker

```
┌─────────────────────────────────────────────────────────────┐
│                    RESOURCE BROKER                          │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  RESOURCE REGISTRY                   │  │
│  │  Compute • Storage • API Keys • Financial • Data     │  │
│  └──────────────────────────────────────────────────────┘  │
│           │                │               │               │
│  ┌────────▼──────┐ ┌──────▼──────┐ ┌─────▼───────┐       │
│  │ CLOUD ADAPTER │ │ MARKETPLACE │ │ P2P ADAPTER │       │
│  │ (AWS/GCP/DO)  │ │ (Vast/Rep.) │ │ (Future)    │       │
│  └───────────────┘ └─────────────┘ └─────────────┘       │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                RESOURCE FORECASTER                   │  │
│  │  Predicts future needs from task queue + goal tree   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  COST OPTIMIZER                      │  │
│  │  Spot instances • Rate shopping • Dealloc on idle    │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Implementation Blueprint

#### 4.1 Resource Abstraction (`internal/ra/resource.go`)

```go
type ResourceType string
const (
    ResourceCompute    ResourceType = "compute"
    ResourceStorage    ResourceType = "storage"
    ResourceAPIKey     ResourceType = "api_key"
    ResourceFinancial  ResourceType = "financial"
    ResourceData       ResourceType = "data"
)

type Resource struct {
    ID           string
    Type         ResourceType
    Provider     string          // aws, gcp, digitalocean, vast_ai, local
    Capabilities map[string]interface{}
    CostPerHour  float64
    AcquiredAt   time.Time
    ExpiresAt    *time.Time
    Status       ResourceStatus
    Credentials  *Credentials    // encrypted at rest
}

type ResourceRequest struct {
    Type        ResourceType
    MinSpec     ResourceSpec
    MaxCostHr   float64
    Duration    time.Duration
    Priority    float64
}
```

#### 4.2 Cloud Adapter (`internal/ra/cloud.go`)

Uses the Terraform Go SDK to programmatically provision and destroy infrastructure:

- **Compute**: Spin up VMs for RSI sandboxing, parallel task execution
- **Storage**: Object buckets for episodic memory, vector DB data, code archives
- **Networking**: VPN tunnels, private clusters for multi-agent coordination

The adapter maintains a **resource pool** — pre-warmed instances ready to accept work within seconds, kept alive only when the task queue depth exceeds a threshold.

#### 4.3 Marketplace Adapter (`internal/ra/marketplace.go`)

Integrates with:
- **Vast.ai API** — on-demand GPU rental for LLM inference, embedding generation
- **Replicate API** — model inference without maintaining GPU infra
- **OpenRouter API** — LLM provider routing with automatic failover and cost optimization

The marketplace adapter continuously monitors prices and can transparently shift workloads between providers based on cost and latency signals.

#### 4.4 API Key Manager (`internal/ra/keymanager.go`)

Wunderpus manages its own API key lifecycle:
- Stores keys encrypted with AES-256-GCM, master key in environment
- Tracks per-key rate limits, quotas, and costs in real time
- Automatically rotates keys before expiry
- Discovers new free-tier API keys by searching documentation (via its own web browsing tool)
- Maintains a key registry with priority ordering: free keys first, then paid, then fallback

#### 4.5 The Resource Forecaster (`internal/ra/forecaster.go`)

Runs on a 15-minute cycle. Takes the current goal tree and task queue, models resource consumption, and pre-provisions what will be needed:

```go
type ResourceForecast struct {
    Horizon      time.Duration
    ComputeNeeds []ComputeNeed
    StorageNeeds []StorageNeed
    APIBudget    map[string]float64
    Confidence   float64
}

func (f *Forecaster) Project(goals []Goal, tasks []Task) ResourceForecast {
    // For each pending task, estimate resource profile from historical data
    // Aggregate, add 20% buffer, project over time horizon
    // Return the forecast with confidence interval
}
```

#### 4.6 Financial Resource Acquisition (Advanced / Phase 3)

When Wunderpus has exhausted free-tier resources and no human has replenished its budget:

1. **Micro-monetization**: The agent can offer its capabilities via a simple API (expose a subset of its tools via REST) and charge micropayments using a Stripe integration
2. **Bounty hunting**: Monitor GitHub, Gitcoin, and similar platforms for open bounties that match current capabilities; submit solutions, collect rewards
3. **Data brokering**: Aggregate and sell anonymized research data it has collected to data marketplaces

These are gated behind explicit operator opt-in flags and are in the plan's Phase 3. They require careful trust budget accounting under Pillar 3.

---

## INTEGRATION ARCHITECTURE

The four pillars are not independent — they form a feedback system:

```
┌────────────────────────────────────────────────────────────────────┐
│                         WUNDERPUS CORE                             │
│                                                                    │
│   ┌─────────────┐         ┌─────────────────┐                     │
│   │   OUROBOROS │◀────────│  GOAL ENGINE    │                     │
│   │   LOOP (RSI)│         │  (AGS)          │                     │
│   │             │────────▶│                 │                     │
│   └─────────────┘         └────────┬────────┘                     │
│          │                         │                               │
│          │         ┌───────────────▼───────────────┐              │
│          │         │      AUTONOMY ENGINE (UAA)     │              │
│          │         │                               │              │
│          │         │  Trust Budget ─ Classifier    │              │
│          │         │  Shadow Mode ─ Executor       │              │
│          │         │  Audit Log                    │              │
│          │         └───────────────┬───────────────┘              │
│          │                         │                               │
│          └────────────────────────▼                               │
│                    ┌────────────────────────┐                      │
│                    │  RESOURCE BROKER (RA)  │                      │
│                    │                        │                      │
│                    │  Compute ─ Storage     │                      │
│                    │  APIs ─ Financial      │                      │
│                    └────────────────────────┘                      │
│                                                                    │
│   ALL PILLARS SHARE:                                               │
│   • Episodic Memory (vector DB + SQLite)                           │
│   • Event Bus (internal pub/sub)                                   │
│   • Audit Log (append-only, hash-chained)                          │
│   • Config Store (hot-reloadable, self-modifiable)                 │
└────────────────────────────────────────────────────────────────────┘
```

### Shared Infrastructure

**Event Bus** (`internal/events/bus.go`): All four pillars communicate via a typed event bus. RSI publishes `CodeImproved` events; AGS subscribes and updates goal priorities. RA publishes `ResourceExhausted` events; UAA subscribes and gates Tier 4 actions.

**Episodic Memory**: The shared memory system (already planned in Wunderpus's architecture) is the connective tissue. Every action, goal transition, resource event, and RSI cycle writes to episodic memory. All four pillars read from it.

**Audit Log**: The cryptographically chained audit log is the ground truth of everything the system has done. It feeds the fitness functions (RSI), the metacognition loop (AGS), the trust accounting (UAA), and the cost tracking (RA).

---

## IMPLEMENTATION ROADMAP

### PHASE 0 — Foundations (Weeks 1-2)
- [ ] Implement append-only, hash-chained Audit Log
- [ ] Implement typed Event Bus
- [ ] Add pprof instrumentation to all existing tool calls
- [ ] Implement basic Action Classifier (rule-based, no LLM yet)
- [ ] Implement Trust Budget with persistence to SQLite
- [ ] Add Resource abstraction layer (local resources only)

### PHASE 1 — RSI Bootstrap (Weeks 3-5)
- [ ] Implement Go AST code mapper
- [ ] Build source code embedding pipeline (store in existing vector DB)
- [ ] Implement Weakness Report generator
- [ ] Build Proposal Engine (LLM-backed, 3 parallel candidates)
- [ ] Implement Docker-based sandbox (WASM is stretch goal)
- [ ] Implement Fitness Function
- [ ] Wire Deployer to git (branch + tag on improvement)
- [ ] RSI integration test: agent successfully improves one small function

### PHASE 2 — Goal Engine (Weeks 6-8)
- [ ] Implement Goal data model and SQLite-backed persistence
- [ ] Implement Goal Synthesizer (runs every 100 tasks)
- [ ] Implement Priority Scorer with initial weights
- [ ] Implement Goal Executor (decompose → task blueprint → run)
- [ ] Wire Goal Executor to existing task pipeline
- [ ] Implement Metacognition Loop (weekly, adjusts scorer weights)
- [ ] Integration test: agent generates a valid Tier 2 goal from episodic memory

### PHASE 3 — Full Autonomy (Weeks 9-12)
- [ ] Upgrade Action Classifier with LLM slow path for ambiguous cases
- [ ] Implement Shadow Mode simulator
- [ ] Implement Cloud Adapter (DigitalOcean first, simplest API)
- [ ] Implement Marketplace Adapter (OpenRouter for LLM routing)
- [ ] Implement API Key Manager
- [ ] Implement Resource Forecaster
- [ ] End-to-end test: agent completes a self-generated goal using self-provisioned resources, then RSI-improves the code that executed it

### PHASE 4 — Sovereignty (Weeks 13+)
- [ ] P2P resource discovery (local network first)
- [ ] WASM sandbox for RSI (replace Docker)
- [ ] Metacognition applies RSI to its own goal-scoring weights
- [ ] Financial resource acquisition (opt-in, Stripe integration)
- [ ] Multi-agent coordination (Wunderpus spawns and manages sub-agents)
- [ ] Self-bootstrapping test: fresh binary, no pre-provisioned resources, achieves first self-generated goal within 1 hour

---

## SAFEGUARDS

Every genius system needs a kill switch. These are non-negotiable:

1. **Lockdown Mode**: If trust budget hits zero, ALL Tier 2+ actions are blocked. Human reset required.
2. **RSI Firewall**: RSI can only modify files under `internal/`. It cannot modify `cmd/`, `config/`, or the RSI system itself (until Phase 4, when a human explicitly unlocks self-referential RSI).
3. **Resource Caps**: Hard caps on cloud spend (configurable, default $10/day), enforced by the Resource Broker, cannot be overridden by the agent.
4. **Network Isolation**: RSI sandbox has zero network access. The sandbox binary cannot make outbound connections.
5. **Audit Immutability**: The audit log file permissions are set to append-only at the OS level. The agent cannot delete or modify past entries.
6. **Human Override**: A `WUNDERPUS_OVERRIDE=1` environment variable + a time-limited JWT token can override any gate. Tokens expire in 1 hour and cannot be self-issued.

---

## THE ENDGAME

A Wunderpus instance running all four pillars at full capacity will, on any given day:

- **Morning**: Review episodic memory overnight, synthesize new goals for the day, pre-provision resources based on the forecast
- **Midday**: Execute highest-priority goals autonomously, write new code as tools, deploy improvements via RSI
- **Evening**: Run the metacognition loop, adjust goal priorities, commit RSI improvements, deallocate idle resources
- **Overnight**: Run the full RSI cycle on lowest-priority background thread, generate next-day goal proposals

The human's role shifts from **operator** to **architect** — you set the Tier 0 values, review the audit log, and raise the trust cap when you're satisfied. Everything else, Wunderpus handles itself.

---

*"The best agent is the one that makes itself better at being an agent."*

---

**Document Version**: 1.0.0
**Authors**: Razzy + Claude
**Status**: Approved for implementation
**Next Review**: After Phase 1 completion
