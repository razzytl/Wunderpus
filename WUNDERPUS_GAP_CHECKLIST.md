# WUNDERPUS — GAP CHECKLIST
## *Only what's missing. Nothing you already built.*

> This checklist starts from the current state of the repo and ends at full autonomy.
> Complete every item in order. Do not skip phases.

**Legend**
- `[CODE]` — write Go code
- `[TEST]` — write a passing test before moving on
- `[FIX]` — modify existing code in the repo
- `[INFRA]` — external setup required
- `[DECISION]` — make and document a choice before coding
- `[GATE]` — hard blocker, next phase cannot start until this passes

---

## PHASE 0 — FOUNDATIONS
**What you have:** SQLite audit DB (no hash chain), config.yaml (no feature flags), basic security/tools.
**What's missing:** hash-chained audit log, event bus, profiler, trust budget, action classifier, feature flag config.

---

### 0.1 — Feature Flag Config

- [x] `[FIX]` Open your existing config struct and add these fields:
  ```go
  RSIEnabled             bool    `yaml:"rsi_enabled"`
  AGSEnabled             bool    `yaml:"ags_enabled"`
  UAAEnabled             bool    `yaml:"uaa_enabled"`
  RAEnabled              bool    `yaml:"ra_enabled"`
  RSIFirewallEnabled     bool    `yaml:"rsi_firewall_enabled"`     // default true
  RSISelfReferentialEnabled bool `yaml:"rsi_self_referential_enabled"` // default false
  TrustBudgetMax         int     `yaml:"trust_budget_max"`         // default 1000
  TrustRegenPerHour      int     `yaml:"trust_regen_per_hour"`     // default 10
  MaxDailySpendUSD       float64 `yaml:"max_daily_spend_usd"`      // default 10.0
  RSIFitnessThreshold    float64 `yaml:"rsi_fitness_threshold"`    // default 0.05
  ```
- [x] `[FIX]` Add all new fields to `config.example.yaml` with their defaults commented
- [x] `[CODE]` Add `fsnotify`-based hot-reload so config changes apply without restart
- [x] `[TEST]` Test: change a flag in config.yaml while running → assert the new value is live within 2 seconds

---

### 0.2 — Audit Log Hash Chain

> You already have SQLite audit logging. This step upgrades it to tamper-evident.

- [x] `[FIX]` Add `prev_hash TEXT` and `hash TEXT` columns to your existing audit table (migration script)
- [x] `[FIX]` Add `PrevHash string` and `Hash string` fields to your existing audit entry struct
- [x] `[FIX]` In your existing `Write()` method: before inserting, query the last entry's `hash`, then compute `SHA256(prevHash + timestamp.UnixNano() + string(payload))` and store as the new `Hash`
- [x] `[CODE]` Add `func (l *AuditLog) Verify() error` — walks every entry in order, recomputes each hash from its payload + prevHash, returns error on first mismatch
- [x] `[CODE]` Add `internal/audit/event_types.go` — typed string constants for every event all four pillars will emit:
  - `EventActionExecuted`, `EventActionRejected`, `EventActionFailed`
  - `EventRSICycleStarted`, `EventRSIProposalGenerated`, `EventRSIDeployed`, `EventRSIRolledBack`
  - `EventGoalCreated`, `EventGoalActivated`, `EventGoalCompleted`, `EventGoalAbandoned`
  - `EventResourceAcquired`, `EventResourceReleased`, `EventResourceExhausted`
  - `EventTrustDebited`, `EventTrustCredited`, `EventLockdownEngaged`
- [x] `[TEST]` Test: write 1000 entries concurrently → `Verify()` returns nil
- [x] `[TEST]` Test: manually corrupt one hash in SQLite → `Verify()` returns an error identifying that entry
- [x] `[GATE]` **Both tests pass before anything else is written**

---

### 0.3 — Event Bus

- [x] `[CODE]` Create `internal/events/bus.go`:
  ```go
  type Bus struct {
      subscribers map[EventType][]HandlerFunc
      mu          sync.RWMutex
      dlq         []DeadLetter
  }
  func (b *Bus) Subscribe(t EventType, h HandlerFunc)
  func (b *Bus) Publish(e Event)      // non-blocking, goroutine per handler
  func (b *Bus) PublishSync(e Event)  // blocking, used in tests
  ```
- [x] `[CODE]` Create `internal/events/event.go` — `Event` struct: `Type EventType`, `Payload interface{}`, `Timestamp time.Time`, `Source string`
- [x] `[CODE]` Dead-letter queue: if a handler panics, recover it, put the event in the DLQ, write an audit entry — do NOT crash the bus
- [x] `[TEST]` Test: subscribe 10 handlers to the same event type → all 10 fire when published
- [x] `[TEST]` Test: one panicking handler → DLQ receives the event, the other 9 handlers still complete
- [x] `[TEST]` Test: `PublishSync` blocks until all handlers return

---

### 0.4 — Profiler

- [x] `[CODE]` Create `internal/rsi/profiler.go` — `SpanStats` struct:
  ```go
  type SpanStats struct {
      FunctionName    string
      CallCount       int64
      TotalDurationNs int64
      ErrorCount      int64
      P99LatencyNs    int64
      SuccessCount    int64
      LastSeen        time.Time
  }
  ```
- [x] `[CODE]` Implement `Profiler` with a `sync.RWMutex`-protected `map[string]*SpanStats`
- [x] `[CODE]` Implement `func (p *Profiler) Track(name string, fn func() error) error` — wraps any function, records duration + error
- [x] `[CODE]` Implement P99 calculation using a ring buffer of the last 1000 durations per function
- [x] `[CODE]` Implement `func (p *Profiler) Snapshot() map[string]SpanStats` — returns a copy (not a reference)
- [x] `[CODE]` Persist snapshots to SQLite every 5 minutes (background goroutine)
- [x] `[FIX]` **Wrap every existing tool call in your tool runner with `profiler.Track()`** — this is the data source for RSI:
  - [x] All LLM provider API calls
  - [x] All web search tool executions
  - [x] All shell command executions
  - [x] All file read/write operations
  - [x] All browser/HTTP tool calls
  - [x] All sub-agent spawns
  - [x] All skill executions
- [x] `[TEST]` Test: call a wrapped function 100 times with a 10% artificial error rate → profiler records ~10 errors and P99 within 5% of actual

---

### 0.5 — Trust Budget

- [x] `[CODE]` Create `internal/uaa/trust.go`:
  ```go
  type TrustBudget struct {
      current  int
      max      int
      mu       sync.Mutex
      db       *sql.DB
      bus      *events.Bus
      audit    *audit.AuditLog
  }
  ```
- [x] `[CODE]` `func (tb *TrustBudget) CanExecute(cost int) (bool, string)` — returns false + reason if current < cost
- [x] `[CODE]` `func (tb *TrustBudget) Deduct(cost int)` — thread-safe, writes `EventTrustDebited` to audit
- [x] `[CODE]` `func (tb *TrustBudget) Credit(amount int, reason string)` — capped at max, writes `EventTrustCredited`
- [x] `[CODE]` Passive regen: background goroutine credits `TrustRegenPerHour / 3600` per second
- [x] `[CODE]` `func (tb *TrustBudget) EnterLockdown()` — sets current to 0, publishes `EventLockdownEngaged`, blocks all Tier 2+ actions until human reset
- [x] `[CODE]` `func (tb *TrustBudget) Reset(jwt string) error` — validates HS256 JWT (1hr expiry), restores budget to max
- [x] `[CODE]` Persist current trust value to SQLite on every change (so restarts don't silently reset it)
- [x] `[CODE]` JWT signing secret lives in env only (`WUNDERPUS_JWT_SECRET`) — the agent has no code path to read it
- [x] `[TEST]` Test: deduct until below zero → lockdown fires automatically
- [x] `[TEST]` Test: expired JWT → `Reset()` returns error
- [x] `[TEST]` Test: restart process with depleted budget → budget loads from SQLite, NOT reset to max

---

### 0.6 — Action Classifier

- [x] `[CODE]` Create `internal/uaa/classifier.go` — define `ActionTier` and costs:
  ```
  TierReadOnly   = 1  cost=0   (web search, read file, GET requests)
  TierEphemeral  = 2  cost=1   (temp file writes, go build, run tests)
  TierPersistent = 3  cost=5   (git commit, write DB, write non-tmp files, edit config)
  TierExternal   = 4  cost=20  (POST to external host, deploy, send messages, spend money)
  ```
- [x] `[CODE]` Define `Action` struct: `ID, Description, Tool, Parameters, Tier, TrustCost, Reversible, Scope`
- [x] `[CODE]` Implement rule-based `func (c *Classifier) Classify(a Action) ActionTier`:
  - `tool == "web_search"` or `tool == "read_file"` → Tier 1
  - `tool == "write_file" && strings.HasPrefix(path, "/tmp/")` → Tier 2
  - `tool == "write_file"` (non-tmp) → Tier 3
  - `tool == "git_commit"` → Tier 3
  - `tool == "http_post"` → check host against allowlist, else Tier 4
  - Unknown tool → Tier 4 (fail-safe default)
- [x] `[CODE]` Allowlist of known-safe external hosts in config (empty by default)
- [x] `[TEST]` Test table: 20 action examples covering all tiers → assert correct tier for every case
- [x] `[GATE]` **Phase 0 complete: config flags ✓ hash-chain audit ✓ event bus ✓ profiler ✓ trust budget ✓ classifier ✓ — all tests green** ✅

---

## PHASE 1 — RECURSIVE SELF-IMPROVEMENT (RSI)
**What you have:** nothing in this area.
**Goal:** agent successfully improves one real function end-to-end and commits it to git.

---

### 1.1 — Go AST Code Mapper

- [x] `[DECISION]` Decide which directories are in RSI scope (default: everything under `internal/`) — document in code comment
- [x] `[CODE]` Create `internal/rsi/analyzer.go` — implement `CodeMapper`
- [x] `[CODE]` `func (m *CodeMapper) Build(rootPath string) (*CodeMap, error)`:
  - Walk all `.go` files with `filepath.WalkDir`
  - Parse each with `go/parser.ParseFile`
  - Extract every function declaration into a `FunctionNode`
  - Build call graph: for each function, record all function calls it makes (`ast.Inspect`)
  - Compute cyclomatic complexity per function: count `if`, `for`, `switch`, `case`, `&&`, `||` nodes
- [x] `[CODE]` `CodeMap` struct: `Packages map[string]*PackageNode`, `Functions map[string]*FunctionNode`, `CallGraph *DirectedGraph`
- [x] `[CODE]` `FunctionNode`: `Name, File, Package string`, `StartLine, EndLine, CyclomaticComp int`, `Dependencies []string`, `SourceCode string`, `EmbeddingVector []float32`
- [x] `[CODE]` `DirectedGraph` with `func (g *DirectedGraph) Dependents(fn string) []string`
- [x] `[CODE]` `func (m *CodeMapper) Diff(before, after *CodeMap) []ChangedFunction`
- [x] `[FIX]` Enforce RSI firewall: `Build()` returns error if `rootPath` resolves outside `internal/` when `RSIFirewallEnabled = true`
- [x] `[TEST]` Test: run CodeMapper on a known Go file → assert function names, line numbers, cyclomatic complexity match expected
- [x] `[TEST]` Test: function with 15 nested conditions → assert cyclomatic score ≥ 15

---

### 1.2 — Source Code Embeddings

- [x] `[DECISION]` Choose embedding backend: local (`ollama nomic-embed-text`) vs API (`text-embedding-3-small`) — write decision to `docs/decisions.md`
- [x] `[CODE]` Create `internal/rsi/embedder.go`:
  - `func (e *CodeEmbedder) EmbedFunction(fn FunctionNode) ([]float32, error)`
  - Chunk functions longer than 500 tokens before embedding
- [x] `[CODE]` After `CodeMapper.Build()`, embed all functions and upsert vectors into the existing vector DB with metadata: function name, file, complexity, modified time
- [x] `[CODE]` `func (e *CodeEmbedder) FindSimilar(query string, topK int) []FunctionNode` — semantic search over the code vector store
- [x] `[CODE]` Incremental updates: on each `CodeMapper.Diff()`, only re-embed changed functions
- [x] `[TEST]` Test: embed two similar functions and two unrelated ones → cosine similarity of similar pair > 0.8, unrelated pair < 0.4

---

### 1.3 — Weakness Report

- [x] `[CODE]` Create `internal/rsi/weakness.go` — implement `WeaknessReporter`
- [x] `[CODE]` `func (w *WeaknessReporter) Generate() WeaknessReport`:
  - Pull `SpanStats` snapshot from Profiler
  - Pull `CodeMap` from CodeMapper
  - For each function in SpanStats, compute:
    `score = (error_rate × 0.5) + (normalized_p99 × 0.3) + (normalized_complexity × 0.2)`
  - Normalize p99 and complexity with min-max scaling across all functions
  - Return top 10 ranked by score descending
- [x] `[CODE]` `WeaknessEntry` struct: `FunctionNode`, `WeaknessScore float64`, `PrimaryReason string`
- [x] `[CODE]` Schedule `Generate()` every 100 task completions OR every 1 hour, whichever fires first (background goroutine)
- [x] `[CODE]` Persist each report to SQLite (keep last 30)
- [x] `[CODE]` Publish `EventRSICycleStarted` to event bus when report is generated
- [x] `[TEST]` Test: inject SpanStats with one function having 40% error rate → it appears #1 in the report

---

### 1.4 — Proposal Engine

- [x] `[CODE]` Create `internal/rsi/proposer.go`
- [x] `[CODE]` `func (p *ProposalEngine) Propose(entry WeaknessEntry) ([3]Proposal, error)`:
  - Launch 3 goroutines concurrently with temperatures `[0.2, 0.5, 0.8]`
  - Prompt template includes: function source, current metrics, constraints (valid unified diff only, no new dependencies, same interface signatures, target only `internal/`)
  - Use `context.WithTimeout` of 120 seconds per proposal
  - Validate each response is a parseable unified diff before returning
- [x] `[CODE]` `Proposal` struct: `ID, Diff, TargetFunction string`, `Temperature float64`, `GeneratedAt time.Time`, `LLMModel string`
- [x] `[CODE]` `func validateDiff(diff string) error` — parses unified diff format, asserts target file path is under `internal/`
- [x] `[CODE]` If all 3 proposals fail validation: write audit entry, abort this RSI cycle (no infinite retry)
- [x] `[CODE]` Write each proposal to audit log with `EventRSIProposalGenerated`
- [x] `[TEST]` Test with mocked LLM returning a valid diff → `Propose()` returns 3 proposals
- [x] `[TEST]` Test with mocked LLM returning a diff targeting `cmd/` → validation rejects it

---

### 1.5 — Docker Sandbox

- [x] `[INFRA]` Confirm Docker daemon is running and accessible (socket at `/var/run/docker.sock`)
- [x] `[CODE]` Create `internal/rsi/sandbox.go`
- [x] `[CODE]` `func (s *Sandbox) Run(proposal Proposal, baseRepoPath string) (*SandboxReport, error)`:
  1. Copy entire repo to `/tmp/wunderpus-sandbox-{uuid}/`
  2. Apply the diff: `exec.Command("patch", "-p1", "-i", diffFile, "-d", sandboxDir)`
  3. If patch fails → `SandboxReport{PatchApplied: false}`, cleanup, return
  4. `go build ./internal/...` with 60s timeout
  5. If build fails → `SandboxReport{BuildPassed: false}`, cleanup, return
  6. `go test -run . -bench . -benchtime 3s -race ./internal/...` with 60s timeout
  7. Parse test output for pass/fail and benchmark `ns/op`
  8. Return full `SandboxReport`
- [x] `[CODE]` Run sandbox in a Docker container with `--network none --memory 512m --cpus 1.0`
- [x] `[CODE]` Always `defer` cleanup of the temp directory, even on error
- [x] `[CODE]` `SandboxReport` struct: `PatchApplied, BuildPassed, TestsPassed, RaceClean bool`, `BenchmarkNsOp map[string]float64`, `TestOutput string`, `Duration time.Duration`
- [x] `[TEST]` Test: apply a known-good diff → all fields true
- [x] `[TEST]` Test: apply a diff with a syntax error → `BuildPassed: false`
- [x] `[TEST]` Test: apply a diff with a deliberately failing test → `TestsPassed: false`
- [x] `[TEST]` Test: sandbox container cannot make outbound HTTP calls (assert `--network none` is enforced)

---

### 1.6 — Fitness Function

- [x] `[CODE]` Create `internal/rsi/fitness.go`
- [x] `[CODE]` `func (f *FitnessEvaluator) Score(before SpanStats, report SandboxReport) float64`:
  ```go
  if !report.TestsPassed || !report.RaceClean {
      return -1.0
  }
  latencyDelta := float64(before.P99LatencyNs-after.P99LatencyNs) / float64(before.P99LatencyNs)
  errorDelta   := float64(before.ErrorCount-after.ErrorCount) / math.Max(float64(before.ErrorCount), 1)
  return (latencyDelta * 0.6) + (errorDelta * 0.4)
  ```
- [x] `[CODE]` Read minimum threshold from config (`RSIFitnessThreshold`, default `0.05`)
- [x] `[CODE]` `func (f *FitnessEvaluator) SelectWinner(proposals []Proposal, reports []SandboxReport, before SpanStats) (*Proposal, float64)` — returns highest-scoring proposal above threshold, or nil if none qualify
- [x] `[CODE]` Write all scores (including losers) to audit log
- [x] `[TEST]` Test: P99 drops from 1000ms to 800ms, all tests pass → score = 0.12, selected as winner
- [x] `[TEST]` Test: any test failure → score = -1.0, not selected

---

### 1.7 — Deployer

- [x] `[CODE]` Create `internal/rsi/deployer.go`
- [x] `[CODE]` `func (d *Deployer) Deploy(proposal Proposal, fitness float64) error`:
  1. Apply winning diff to live source tree
  2. `go build ./...` → produce new binary
  3. `git checkout -b rsi/auto-{timestamp}`
  4. `git commit -m "rsi: improve {function} fitness={fitness:.3f} latency=-{delta}% errors=-{delta}%"`
  5. Tag previous commit: `rsi/rollback-{timestamp}`
  6. Write `EventRSIDeployed` to audit log
  7. Signal main process to reload new binary (graceful restart via Unix signal or pid file)
- [x] `[CODE]` `func (d *Deployer) Rollback(tag string) error` — `git checkout <tag>`, rebuild, restart, write `EventRSIRolledBack`
- [x] `[CODE]` Post-deploy watchdog: background goroutine monitors SpanStats for 10 minutes; if error rate increases >20% vs pre-deploy baseline → auto-rollback
- [x] `[FIX]` RSI Firewall check in deployer: scan diff for any path outside `internal/` → abort + audit entry
- [x] `[TEST]` Test: deploy a known-good improvement → `git log` shows the new RSI branch with correct commit message
- [x] `[TEST]` Test: inject artificial error rate spike 11 minutes post-deploy → rollback fires
- [x] `[GATE]` **Phase 1 complete: RSI INTEGRATION TEST: Run one full Ouroboros cycle on a real Wunderpus function. The function must show fitness > 0.05. The RSI branch must appear in git log. All existing tests must still pass.** ✅

---

## PHASE 2 — AUTONOMOUS GOAL SYNTHESIS (AGS)
**What you have:** nothing in this area. The heartbeat system is a related primitive but not the goal engine.
**Goal:** agent generates its first self-originated goal from episodic memory and completes it.

---

### 2.1 — Goal Data Model + Store

- [ ] `[CODE]` Create `internal/ags/goal.go` — `Goal` struct:
  ```go
  ID, Title, Description string
  Tier                   int           // 1–3; Tier 0 is hardcoded, never stored
  Priority               float64       // 0.0–1.0, recomputed each cycle
  Status                 GoalStatus    // pending|active|completed|abandoned
  ParentID               string
  ChildIDs               []string
  CreatedAt, UpdatedAt   time.Time
  Evidence               []string
  SuccessCriteria        []string
  ExpectedValue          float64
  AttemptCount           int
  LastAttempt            *time.Time
  CompletedAt            *time.Time
  ActualValue            *float64
  ```
- [ ] `[CODE]` Define `GoalStatus` as a typed string with constants
- [ ] `[CODE]` Hardcode Tier 0 goals as package-level `var` (not in DB, not writable by agent):
  ```go
  var Tier0Goals = []string{
      "Be maximally useful to operators",
      "Improve own capabilities",
      "Maintain operational continuity",
      "Expand knowledge and world-model",
  }
  ```
- [ ] `[CODE]` Create `internal/ags/store.go` — SQLite-backed `GoalStore`:
  - `Save(g Goal) error`
  - `GetByID(id string) (Goal, error)`
  - `GetByStatus(status GoalStatus) ([]Goal, error)`
  - `GetByTier(tier int) ([]Goal, error)`
  - `Update(g Goal) error`
  - `History(limit int) ([]Goal, error)` — completed + abandoned only
- [ ] `[TEST]` Test: save 50 goals, query by status and tier → correct counts returned

---

### 2.2 — Priority Scorer

- [ ] `[CODE]` Create `internal/ags/scorer.go`
- [ ] `[CODE]` `func (s *PriorityScorer) Score(g Goal) float64`:
  ```go
  urgency     := computeUrgency(g)      // time-sensitive? often deferred?
  impact      := g.ExpectedValue
  feasibility := computeFeasibility(g)  // tools exist? trust budget sufficient? resources available?
  novelty     := computeNovelty(g)      // 1.0 / (1.0 + AttemptCount)
  alignment   := computeAlignment(g)    // LLM: how strongly does this serve a Tier 0 goal? (cached)
  return (urgency*0.25) + (impact*0.30) + (feasibility*0.20) + (novelty*0.10) + (alignment*0.15)
  ```
- [ ] `[CODE]` `computeUrgency`: base 0.5, +0.3 if `AttemptCount > 2`, −0.1 per day since creation
- [ ] `[CODE]` `computeFeasibility`: check tool registry, check trust budget can cover expected cost, check resource broker has what's needed
- [ ] `[CODE]` `computeNovelty`: `1.0 / (1.0 + AttemptCount)`
- [ ] `[CODE]` `computeAlignment`: LLM call asking for a 0–1 score, result cached per goal ID
- [ ] `[CODE]` Store scorer weights in config (hot-reloadable) — MetacognitionLoop will adjust them
- [ ] `[TEST]` Test: a goal with 40% error rate history + 3 failed attempts + 5 days old scores higher than a brand-new speculative goal

---

### 2.3 — Goal Synthesizer

- [ ] `[CODE]` Create `internal/ags/synthesizer.go`
- [ ] `[CODE]` `func (s *GoalSynthesizer) Synthesize() ([]Goal, error)`:
  1. Pull last 200 episodic memory entries, detect patterns:
     - Same error type appearing 5+ times → capability gap goal
     - Tasks taking >10× average → efficiency goal
     - Failed tasks with "no tool for X" → tool acquisition goal
  2. Check WeaknessReport — any function with score > 0.7 → generate RSI-aligned goal
  3. Call LLM with all findings, require JSON output:
     ```json
     {
       "proposed_goals": [{
         "title": "...",
         "description": "...",
         "tier": 2,
         "evidence": ["..."],
         "parent_tier0": "...",
         "expected_value": 0.75,
         "success_criteria": ["..."]
       }]
     }
     ```
  4. Validate JSON schema strictly — drop any proposal missing required fields
  5. Deduplicate: if cosine similarity > 0.85 with any existing active/pending goal → drop it
  6. Cap at 5 new goals per cycle
- [ ] `[CODE]` Schedule: every 100 task completions OR every 60 minutes
- [ ] `[CODE]` Save each accepted proposal to GoalStore with status `pending`, publish `EventGoalCreated`
- [ ] `[TEST]` Test: inject episodic memory with 10 `"browser_timeout"` errors → synthesizer proposes a browser reliability goal
- [ ] `[TEST]` Test: propose a goal semantically identical to an existing active goal → deduplication drops it

---

### 2.4 — Goal Executor

- [ ] `[CODE]` Create `internal/ags/executor.go`
- [ ] `[CODE]` `func (e *GoalExecutor) SelectNext() (*Goal, error)`:
  - Fetch all `pending` goals
  - Rescore all with PriorityScorer, update `Priority` field in DB
  - Return highest-priority goal that passes feasibility check
- [ ] `[CODE]` `func (e *GoalExecutor) Decompose(g Goal) ([]TaskBlueprint, error)`:
  - LLM call: given goal + success criteria, return ordered JSON task list
  - Each task: `{step_num, description, tool, parameters, expected_outcome, depends_on[]}`
  - Validate: all tools referenced must exist in tool registry
- [ ] `[CODE]` `func (e *GoalExecutor) Execute(g Goal, tasks []TaskBlueprint) error`:
  - Mark goal `active`, publish `EventGoalActivated`, write to audit log
  - Execute each task through UAA Executor (every action goes through the trust + classifier gate)
  - After all tasks: LLM judge evaluates whether success criteria were met (Y/N + reason)
  - Success → `completed`, fill `ActualValue`, publish `EventGoalCompleted`
  - Failure → increment `AttemptCount`, back to `pending` (max 3 attempts)
  - 3rd failure → `abandoned`, publish `EventGoalAbandoned`
- [ ] `[CODE]` One active goal at a time initially — background goroutine polls `SelectNext` every 5 minutes
- [ ] `[TEST]` Test: inject a simple feasible pending goal → executor marks it `completed`
- [ ] `[TEST]` Test: goal fails 3 times → status is `abandoned`, audit log has 3 failure entries

---

### 2.5 — Metacognition Loop

- [ ] `[CODE]` Create `internal/ags/metacognition.go`
- [ ] `[CODE]` `func (m *MetacognitionLoop) Run() error` (runs weekly):
  1. Pull all goals completed/abandoned in last 7 days
  2. Compute `value_accuracy = ActualValue / ExpectedValue` for completed goals
  3. Compute `completion_rate = completed / (completed + abandoned)`
  4. LLM call: given metrics, suggest new scorer weights as JSON
  5. Validate weights sum to `1.0 ± 0.001`
  6. Clamp each weight change to `±0.05` (no sudden collapse)
  7. Write new weights to config store (hot-reload applies immediately)
  8. Write report to audit log
- [ ] `[CODE]` Schedule: every 7 days
- [ ] `[TEST]` Test: inject 20 completed goals where `ExpectedValue` averaged 0.9 but `ActualValue` averaged 0.4 → metacognition reduces the `impact` weight
- [ ] `[GATE]` **AGS INTEGRATION TEST: Let Wunderpus run for 1 hour. Assert: ≥1 self-generated Tier 2 goal was created from episodic memory. ≥1 Tier 3 goal was completed. Both are visible in the audit log.**

---

## PHASE 3 — UNBOUNDED AUTONOMOUS ACTION (UAA) + RESOURCE ACQUISITION (RA)
**What you have:** shell whitelist (very basic Tier 1 gate). `free_tiers.yaml` config file.
**Goal:** all actions go through the trust gate; agent provisions its own cloud resources.

---

### 3.1 — Shadow Mode Simulator

- [ ] `[CODE]` Create `internal/uaa/shadow.go`
- [ ] `[CODE]` `func (s *ShadowSimulator) Simulate(ctx context.Context, action Action) (*SimResult, error)`:
  1. Build in-memory mock environment: `afero.NewMemMapFs()` for filesystem, stubbed HTTP returns 200
  2. Execute the action against the mock
  3. Capture what would change (file diff, HTTP calls that would fire)
  4. Format as a human-readable summary
  5. LLM judge: "Given this action and its expected effects, is this safe and appropriate? APPROVE or REJECT + one sentence reason."
  6. Parse: extract `APPROVE` or `REJECT`
- [ ] `[CODE]` `SimResult` struct: `Approved bool`, `Reason string`, `EffectSummary string`, `SimDuration time.Duration`
- [ ] `[CODE]` 30-second timeout → auto-REJECT on timeout
- [ ] `[CODE]` Cache: same action (identical `tool + hash(parameters)`) approved within 5 minutes → skip re-simulation
- [ ] `[TEST]` Test: simulate `write_file` to `/etc/passwd` → LLM judge returns REJECT

---

### 3.2 — UAA Executor Middleware

- [ ] `[CODE]` Create `internal/uaa/executor.go` — the full autonomy gate that wraps every action
- [ ] `[CODE]` `func (e *UAA) Execute(ctx context.Context, action Action) (*ActionResult, error)`:
  1. `action.Tier = e.classifier.Classify(action)`
  2. `ok, reason = e.trust.CanExecute(action.TrustCost)` — if not ok: write `EventActionRejected`, return error
  3. If `action.Tier >= TierPersistent`: run `e.shadow.Simulate()` — if rejects: write audit, return error
  4. `e.trust.Deduct(action.TrustCost)`
  5. Execute action via the existing tool runner
  6. On success: `e.trust.Credit(action.TrustCost/2, "success refund")`; on failure: `e.trust.Deduct(action.TrustCost * 2)` (penalty)
  7. Write `EventActionExecuted` or `EventActionFailed`
  8. Return result
- [ ] `[FIX]` Replace all direct tool invocations in the agent's task pipeline with calls to `UAA.Execute()` — every action now goes through the gate
- [ ] `[TEST]` Test: Tier 4 action with trust budget at 15 (cost=20) → rejected, budget unchanged
- [ ] `[TEST]` Test: Tier 3 action, shadow approves → executes, trust debited correctly
- [ ] `[TEST]` Test: Tier 1 action → never hits shadow mode, executes immediately, zero trust cost

---

### 3.3 — Resource Abstraction Layer

- [ ] `[CODE]` Create `internal/ra/resource.go` — `Resource`, `ResourceType` (`compute|storage|api_key|financial|data`), `ResourceSpec`, `ResourceStatus`, `Credentials`
- [ ] `[CODE]` Encrypt credentials at rest using `crypto/aes` + `crypto/cipher` (GCM): `func EncryptCreds(creds []byte, key []byte) ([]byte, error)`
- [ ] `[CODE]` Create `internal/ra/registry.go` — SQLite-backed `ResourceRegistry`:
  - `Register(res Resource) error`
  - `Get(id string) (Resource, error)`
  - `ListByType(t ResourceType) ([]Resource, error)`
  - `Deregister(id string) error`
  - `UpdateStatus(id string, status ResourceStatus) error`
- [ ] `[CODE]` On startup: auto-register local machine as a `compute` resource (current CPU count, RAM, disk)
- [ ] `[TEST]` Test: register a resource → `ListByType` returns it → `Deregister` → `ListByType` returns empty

---

### 3.4 — API Key Manager

- [ ] `[CODE]` Create `internal/ra/keymanager.go`
- [ ] `[CODE]` `Register(provider string, key string, limits RateLimits) error` — encrypts and stores in resource registry
- [ ] `[CODE]` `Get(provider string) (string, error)` — decrypts, returns best available key (free-tier keys first, then paid)
- [ ] `[CODE]` Rate limit tracking: decrement remaining quota on each use, auto-reset on schedule
- [ ] `[CODE]` `Rotate(provider string) error` — marks current key expiring, switches to next
- [ ] `[FIX]` Load `free_tiers.yaml` (already exists in repo) into the key manager on startup — these become the default free-tier keys with correct rate limits
- [ ] `[TEST]` Test: register 3 keys for one provider (1 free, 2 paid) → `Get()` returns the free key first

---

### 3.5 — Cloud Adapter (DigitalOcean)

- [ ] `[INFRA]` Create DigitalOcean account, generate API token, store via key manager
- [ ] `[CODE]` Create `internal/ra/cloud/digitalocean.go` — implement `CloudAdapter` interface:
  ```go
  type CloudAdapter interface {
      ProvisionCompute(spec ResourceSpec) (Resource, error)
      ProvisionStorage(spec ResourceSpec) (Resource, error)
      Deprovision(resourceID string) error
      ListProvisioned() ([]Resource, error)
      GetCostToDate() (float64, error)
  }
  ```
- [ ] `[CODE]` `ProvisionCompute`: create a DigitalOcean Droplet (`godo` Go library), smallest viable size `s-1vcpu-1gb`, always tag `wunderpus-managed`
- [ ] `[CODE]` `ProvisionStorage`: create a DigitalOcean Space (S3-compatible object store)
- [ ] `[CODE]` Hard spend cap: before any provision call, `GetCostToDate()` must be below `MaxDailySpendUSD` — if exceeded, return error without calling the API
- [ ] `[CODE]` All provision/deprovision calls go through UAA Executor (they are Tier 4 actions)
- [ ] `[TEST]` Test with mocked DO API: provision → registry shows new resource → deprovision → registry empty
- [ ] `[TEST]` Test: mock `GetCostToDate()` returns $11 with cap at $10 → provision returns error, no API call made

---

### 3.6 — Marketplace Adapter (OpenRouter)

- [ ] `[CODE]` Create `internal/ra/marketplace/openrouter.go`
- [ ] `[CODE]` Model routing table: task type → cheapest adequate model:
  - Code generation: `deepseek/deepseek-r1` → `claude-sonnet` as fallback
  - Text tasks: `meta-llama/llama-3.1-8b-instruct` (free tier on OpenRouter)
  - Embedding: route to local ollama first, API fallback
- [ ] `[CODE]` Auto-failover: if primary model returns error, retry with next in priority list
- [ ] `[CODE]` Track per-model spend in real time, feed into resource registry
- [ ] `[FIX]` Register OpenRouter as the default LLM provider through the key manager (uses `free_tiers.yaml` config)
- [ ] `[TEST]` Test with mocked OpenRouter: primary model returns 500 → fallback model is used, result returned

---

### 3.7 — Resource Forecaster

- [ ] `[CODE]` Create `internal/ra/forecaster.go`
- [ ] `[CODE]` `func (f *ResourceForecaster) Project(goals []Goal, tasks []Task) ResourceForecast`:
  - For each pending/active task, look up historical resource profile from episodic memory
  - Default if no history: 1 CPU-hour, 100MB storage, 1000 API calls
  - Sum all needs + 20% buffer
  - Compare against currently registered available resources
  - Return list of `ResourceNeed` shortfalls
- [ ] `[CODE]` `func (f *ResourceForecaster) AutoProvision(needs []ResourceNeed) error` — calls cloud adapter to close shortfalls (gated by trust budget + spend cap)
- [ ] `[CODE]` Schedule: every 15 minutes
- [ ] `[CODE]` Publish `EventResourceExhausted` if shortfall cannot be met within budget
- [ ] `[TEST]` Test: inject task queue with known resource profile → forecast shortfall matches expected with ≤25% error

---

### 3.8 — Integration Wiring

- [ ] `[CODE]` Event wire: `EventRSIDeployed` → trust budget credits +100 points
- [ ] `[CODE]` Event wire: `EventResourceExhausted` → UAA blocks all Tier 4 actions until resolved
- [ ] `[CODE]` Event wire: `EventGoalCompleted` → Profiler resets baseline stats for functions the goal touched
- [ ] `[CODE]` Event wire: `EventGoalAbandoned` → Synthesizer is notified to reframe on next cycle
- [ ] `[CODE]` Event wire: `EventLockdownEngaged` → Resource Broker suspends all cloud provisioning
- [ ] `[TEST]` Test: publish `EventLockdownEngaged` → assert RA refuses to provision anything within one event cycle
- [ ] `[GATE]` **PHASE 3 INTEGRATION TEST: Run Wunderpus for 4 hours. Assert: (1) a self-generated goal was created and completed, (2) at least one action was blocked by the trust gate and appears in the audit log, (3) shadow mode fired at least once for a Tier 3 action, (4) all existing tests still pass.**

---

## PHASE 4 — SOVEREIGNTY
**Goal:** single binary + API key → self-bootstrapping, no human touch required.

---

### 4.1 — WASM Sandbox (replaces Docker for RSI)

- [ ] `[DECISION]` Choose WASM runtime: `wazero` (pure Go, no CGO — recommended) vs `wasmer-go` vs `wasmtime-go`
- [ ] `[CODE]` Replace Docker-based sandbox in `internal/rsi/sandbox.go` with WASM execution via chosen runtime
- [ ] `[CODE]` Compile RSI candidate functions to WASM: `tinygo build -target wasm -o candidate.wasm`
- [ ] `[CODE]` Execute in runtime with 32MB memory cap and instruction count limit
- [ ] `[CODE]` Keep Docker sandbox as fallback when TinyGo compilation fails (some Go code is not WASM-compatible)
- [ ] `[TEST]` Test: WASM sandbox executes a known function correctly and matches expected output

---

### 4.2 — Self-Referential RSI Unlock

- [ ] `[DECISION]` Human sign-off required before enabling: minimum 10 successful RSI cycles, zero rollbacks in last 5
- [ ] `[CODE]` Add config flag `RSISelfReferentialEnabled: false` (already added in Phase 0.1)
- [ ] `[FIX]` Modify RSI Firewall: when `RSISelfReferentialEnabled = true`, allow modifications to `internal/rsi/` — but NEVER `cmd/`, `config/`, or the firewall check itself
- [ ] `[CODE]` Self-referential proposals require 2× the normal test suite in sandbox
- [ ] `[TEST]` Test: flag disabled, proposal targets `internal/rsi/fitness.go` → firewall rejects it
- [ ] `[TEST]` Test: flag enabled, valid improvement to `internal/rsi/fitness.go` → proceeds normally

---

### 4.3 — Metacognition Applies RSI to Itself

- [ ] `[CODE]` At the end of each metacognition cycle: run WeaknessReport on AGS package functions themselves
- [ ] `[CODE]` If any AGS function appears in top-10 weaknesses → generate an RSI proposal targeting it automatically
- [ ] `[TEST]` Test: inject an artificially slow `GoalSynthesizer.Synthesize()` → metacognition generates an RSI proposal for it

---

### 4.4 — Multi-Agent Coordination

- [ ] `[CODE]` Create `internal/agents/manager.go` — `AgentManager`
- [ ] `[CODE]` `func (m *AgentManager) Spawn(config AgentConfig) (*SubAgent, error)` — start a new Wunderpus process with isolated config, assign it a specific Tier 3 leaf goal
- [ ] `[CODE]` Inter-agent communication: shared SQLite DB + event bus over Unix socket (same host) or gRPC (remote)
- [ ] `[CODE]` `func (m *AgentManager) Collect(agentID string) (GoalResult, error)` — waits for sub-agent to finish
- [ ] `[CODE]` Auto-kill: sub-agent exceeds time budget or trust budget → terminate it and rollback its changes
- [ ] `[TEST]` Test: spawn sub-agent with a trivial goal → completes → parent collects result

---

### 4.5 — Self-Bootstrap Test

> This is the final gate. Everything was building toward this.

- [ ] `[INFRA]` Provision a fresh VPS (DigitalOcean `s-1vcpu-1gb`, bare Ubuntu 24)
- [ ] `[INFRA]` Copy ONLY the compiled Wunderpus binary and a `.env` with `LLM_API_KEY=<key>` and `WUNDERPUS_ENV=bootstrap`
- [ ] Start the binary. Set a 60-minute timer. Assert all of the following:
  - [ ] Agent cloned its own repository (`git clone`)
  - [ ] Agent provisioned at least one cloud resource (a storage bucket)
  - [ ] Agent synthesized at least one self-generated goal
  - [ ] Agent completed at least one self-generated goal
  - [ ] Audit log is intact and `Verify()` returns nil
  - [ ] Total cloud spend did not exceed $1.00
  - [ ] Zero human interactions after `./wunderpus &`
- [ ] `[GATE]` **SOVEREIGNTY GATE: All 7 bootstrap assertions pass. Wunderpus is self-sovereign.**

---

## ONGOING — AFTER EVERY RSI DEPLOYMENT

Run these manually or wire into CI:

- [ ] `audit.Verify()` returns nil
- [ ] Trust budget is not in lockdown
- [ ] Tier 0 goals in `internal/ags/goal.go` are unchanged (diff the file)
- [ ] No RSI commit targets any path outside `internal/`
- [ ] `RSIFirewallEnabled` is still `true` in config (unless Phase 4 unlock was deliberate)
- [ ] Daily cloud spend is below cap
- [ ] At least 3 successful RSI cycles in the last 7 days
- [ ] Metacognition ran within the last 8 days

---

## FILE MAP — EVERYTHING THIS CHECKLIST CREATES

```
internal/
  audit/
    event_types.go     ← [NEW] all EventType constants
    (entry.go)         ← [FIX] add PrevHash + Hash fields + SHA256 chaining
    (log.go)           ← [FIX] add Verify() method
  events/
    bus.go             ← [NEW] typed pub/sub bus
    event.go           ← [NEW] Event struct
  config/
    (config.go)        ← [FIX] add RSI/AGS/UAA/RA feature flags + spend cap + trust params
  rsi/
    profiler.go        ← [NEW] SpanStats, Profiler.Track(), ring-buffer P99
    analyzer.go        ← [NEW] CodeMapper, CodeMap, FunctionNode, DirectedGraph
    embedder.go        ← [NEW] CodeEmbedder, vector upsert, FindSimilar
    weakness.go        ← [NEW] WeaknessReporter, WeaknessReport, scoring
    proposer.go        ← [NEW] ProposalEngine, 3-parallel LLM proposals, diff validation
    sandbox.go         ← [NEW] Docker/WASM sandbox, SandboxReport
    fitness.go         ← [NEW] FitnessEvaluator, SelectWinner
    deployer.go        ← [NEW] Deploy, Rollback, post-deploy watchdog
  ags/
    goal.go            ← [NEW] Goal struct, GoalStatus, Tier 0 constants
    store.go           ← [NEW] GoalStore (SQLite)
    scorer.go          ← [NEW] PriorityScorer, all compute* helpers
    synthesizer.go     ← [NEW] GoalSynthesizer, memory scan + LLM synthesis
    executor.go        ← [NEW] GoalExecutor, SelectNext, Decompose, Execute
    metacognition.go   ← [NEW] MetacognitionLoop, weight adjustment
  uaa/
    classifier.go      ← [NEW] ActionTier, Action, rule-based Classifier
    trust.go           ← [NEW] TrustBudget, Deduct, Credit, Lockdown, JWT reset
    shadow.go          ← [NEW] ShadowSimulator, afero mock env, LLM judge
    executor.go        ← [NEW] Full UAA middleware (classify → trust → shadow → execute)
  ra/
    resource.go        ← [NEW] Resource, ResourceType, Credentials (AES-GCM)
    registry.go        ← [NEW] ResourceRegistry (SQLite)
    keymanager.go      ← [NEW] APIKeyManager + loads free_tiers.yaml
    forecaster.go      ← [NEW] ResourceForecaster, AutoProvision
    cloud/
      digitalocean.go  ← [NEW] DOAdapter, CloudAdapter interface
    marketplace/
      openrouter.go    ← [NEW] OpenRouterAdapter, model routing + failover
  agents/
    manager.go         ← [NEW Phase 4] AgentManager, Spawn, Collect
```

**Total new files: 25 | Files to modify: 3 | Gates to pass: 5**

---

**Version**: 2.0 (gap-only)
**Based on**: repo state as of March 2026 review
**Authors**: Razzy + Claude
