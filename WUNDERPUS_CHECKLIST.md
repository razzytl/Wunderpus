# WUNDERPUS — IMPLEMENTATION CHECKLIST
## *Every step. Every file. Every test. In order.*

> Check nothing off until it works. Ship nothing until the test passes.

---

## HOW TO USE THIS CHECKLIST

- Items marked `[CODE]` require writing Go code
- Items marked `[TEST]` require a passing test before proceeding
- Items marked `[GATE]` are hard blockers — nothing in the next phase starts until this passes
- Items marked `[INFRA]` require external setup (Docker, cloud account, etc.)
- Items marked `[DECISION]` require a documented choice before writing any code
- Dependencies between items are noted in parentheses where non-obvious

Start at the top. Do not skip ahead.

---

## PHASE 0 — FOUNDATIONS
### *"If the foundation is weak, the tower falls."*
**Target: 2 weeks | Goal: Shared infrastructure every pillar depends on**

---

### 0.1 — Project Scaffolding

- [x] `[DECISION]` Confirm Go module path (e.g., `github.com/razzytl/wunderpus`) — used in every import
- [x] `[CODE]` Create directory structure:
  ```
  internal/
    audit/
    events/
    rsi/
    ags/
    uaa/
    ra/
    memory/
    config/
  ```
- [x] `[CODE]` Add `internal/config/config.go` — central config struct with all phase-gated feature flags
  - Field: `RSIEnabled bool`
  - Field: `AGSEnabled bool`
  - Field: `UAAEnabled bool`
  - Field: `RAEnabled bool`
  - Field: `RSIFirewallEnabled bool` (default `true`)
  - Field: `MaxDailySpendUSD float64` (default `10.0`)
  - Field: `TrustBudgetMax int` (default `1000`)
  - Field: `TrustRegenPerHour int` (default `10`)
- [x] `[CODE]` Add `internal/config/loader.go` — loads from YAML file + env var overrides
- [x] `[CODE]` Add hot-reload support to config loader using `fsnotify` — config changes apply without restart
- [x] `[TEST]` Write config round-trip test: load → modify → reload → assert new value active
- [x] `[DECISION]` Choose SQLite library (`mattn/go-sqlite3` vs `modernc.org/sqlite`) — note: modernc requires no CGO

---

### 0.2 — Audit Log

> The audit log is the single source of truth for everything the system does. Build it first.

- [x] `[CODE]` Create `internal/audit/entry.go` — define `AuditEntry` struct:
  - Fields: `ID string`, `Timestamp time.Time`, `Subsystem string`, `EventType string`, `ActorID string`, `Payload json.RawMessage`, `PrevHash string`, `Hash string`
- [x] `[CODE]` Create `internal/audit/log.go` — implement `AuditLog` struct with:
  - SQLite-backed persistence
  - Append-only write method: `func (l *AuditLog) Write(entry AuditEntry) error`
  - Hash chaining: each entry's `Hash = SHA256(prevHash + timestamp + payload)`
  - `mu sync.Mutex` — all writes serialized
- [x] `[CODE]` Implement `func (l *AuditLog) Verify() error` — walks all entries, recomputes hashes, returns error on mismatch
- [x] `[CODE]` Implement `func (l *AuditLog) Query(filter AuditFilter) ([]AuditEntry, error)` — filter by subsystem, time range, event type
- [x] `[CODE]` Set SQLite file permissions to `0444` after initial creation + use WAL mode
- [x] `[CODE]` Add `internal/audit/event_types.go` — define all event type constants as typed strings:
  - `EventActionExecuted`, `EventActionRejected`, `EventActionFailed`
  - `EventRSICycleStarted`, `EventRSIProposalGenerated`, `EventRSIDeployed`, `EventRSIRolledBack`
  - `EventGoalCreated`, `EventGoalActivated`, `EventGoalCompleted`, `EventGoalAbandoned`
  - `EventResourceAcquired`, `EventResourceReleased`, `EventResourceExhausted`
  - `EventTrustDebited`, `EventTrustCredited`, `EventLockdownEngaged`
- [x] `[TEST]` Write test: append 1000 entries concurrently → `Verify()` returns nil
- [x] `[TEST]` Write test: corrupt one entry's hash in SQLite directly → `Verify()` returns error pointing to that entry
- [x] `[GATE]` **Audit log passes both tests before any other code is written**

---

### 0.3 — Event Bus

> The event bus is how all four pillars talk to each other without tight coupling.

- [x] `[CODE]` Create `internal/events/bus.go` — implement typed pub/sub:
  ```go
  type Bus struct {
      subscribers map[EventType][]HandlerFunc
      mu          sync.RWMutex
  }
  func (b *Bus) Subscribe(t EventType, h HandlerFunc)
  func (b *Bus) Publish(e Event)  // non-blocking, goroutine per handler
  func (b *Bus) PublishSync(e Event) // blocking, for tests
  ```
- [x] `[CODE]` Create `internal/events/event.go` — define `Event` struct with `Type EventType`, `Payload interface{}`, `Timestamp time.Time`, `Source string`
- [x] `[CODE]` Define all `EventType` constants as typed strings (mirror audit event types, plus internal signals)
- [x] `[CODE]` Add dead-letter queue: if handler panics, event goes to DLQ + audit log entry written
- [x] `[TEST]` Write test: subscribe 10 handlers to same event → all 10 fire when published
- [x] `[TEST]` Write test: panicking handler → DLQ receives event, other handlers still fire
- [x] `[TEST]` Write test: PublishSync blocks until all handlers complete

---

### 0.4 — Telemetry / Profiler Skeleton

> We instrument everything now, even if the RSI loop doesn't exist yet.

- [x] `[CODE]` Create `internal/rsi/profiler.go` — define `SpanStats` struct:
  - Fields: `FunctionName string`, `CallCount int64`, `TotalDurationNs int64`, `ErrorCount int64`, `P99LatencyNs int64`, `SuccessCount int64`, `LastSeen time.Time`
- [x] `[CODE]` Implement `Profiler` struct with `sync.RWMutex`-protected `map[string]*SpanStats`
- [x] `[CODE]` Implement `func (p *Profiler) Track(name string, fn func() error) error` — wraps any function, records duration and error
- [x] `[CODE]` Implement `func (p *Profiler) Snapshot() map[string]SpanStats` — returns copy of current stats
- [x] `[CODE]` Implement P99 calculation using a sliding window of last 1000 durations per function (ring buffer)
- [x] `[CODE]` Expose pprof endpoint on configurable port (default `:6060`) — guarded by config flag
- [ ] `[CODE]` **Go through every existing Wunderpus tool call and wrap with `profiler.Track()`**
  - [ ] Wrap all LLM API calls
  - [ ] Wrap all browser/web tool executions
  - [ ] Wrap all file system operations
  - [ ] Wrap all external HTTP calls
- [x] `[CODE]` Persist `SpanStats` snapshots to SQLite every 5 minutes (background goroutine)
- [x] `[TEST]` Write test: call a function 100x with 10% error rate → profiler records ~10 errors, P99 within 5% of actual

---

### 0.5 — Trust Budget

> The trust budget must exist before any autonomous action code is written.

- [x] `[CODE]` Create `internal/uaa/trust.go` — define `TrustBudget`:
  ```go
  type TrustBudget struct {
      current     int
      max         int
      mu          sync.Mutex
      db          *sql.DB
      events      *events.Bus
  }
  ```
- [x] `[CODE]` Implement `func (tb *TrustBudget) CanExecute(cost int) (bool, string)` — checks current >= cost
- [x] `[CODE]` Implement `func (tb *TrustBudget) Deduct(cost int)` — thread-safe, writes audit event
- [x] `[CODE]` Implement `func (tb *TrustBudget) Credit(amount int, reason string)` — capped at max, writes audit event
- [x] `[CODE]` Implement passive regen: background goroutine credits `TrustRegenPerHour/3600` per second
- [x] `[CODE]` Implement `func (tb *TrustBudget) EnterLockdown()` — sets current = 0, publishes `EventLockdownEngaged`, requires human reset
- [x] `[CODE]` Implement `func (tb *TrustBudget) Reset(jwt string) error` — validates JWT, restores budget to max
- [x] `[CODE]` Persist current trust value to SQLite on every change (survive restarts)
- [x] `[CODE]` Implement JWT validation: HS256, 1-hour expiry, cannot be signed by the agent (secret lives in env only)
- [x] `[TEST]` Write test: deduct below zero → lockdown auto-engages
- [x] `[TEST]` Write test: expired JWT → reset fails
- [x] `[TEST]` Write test: agent cannot generate valid JWT (secret unavailable to internal code)
- [x] `[TEST]` Write test: restart with depleted budget → budget loads from SQLite, not reset to max

---

### 0.6 — Action Classifier (Rule-Based)

- [x] `[CODE]` Create `internal/uaa/classifier.go` — define `ActionTier` type and constants:
  - `TierReadOnly = 1` (cost 0): web search, read files, API GETs, list directories
  - `TierEphemeral = 2` (cost 1): create temp files, run tests, `go build`, sandbox exec
  - `TierPersistent = 3` (cost 5): git commit, write to DB, modify config, write non-temp files
  - `TierExternal = 4` (cost 20): send HTTP POST to external, deploy, send comms, spend money
- [x] `[CODE]` Implement `Action` struct: `ID, Description, Tool, Parameters, Tier, TrustCost, Reversible, Scope`
- [x] `[CODE]` Implement rule-based classifier: pattern-match on `Tool` name + parameter inspection
  - `tool == "web_search"` → Tier 1
  - `tool == "read_file"` → Tier 1
  - `tool == "write_file" && path contains "/tmp/"` → Tier 2
  - `tool == "write_file" && path NOT in /tmp/` → Tier 3
  - `tool == "git_commit"` → Tier 3
  - `tool == "http_post" && host NOT in allowlist` → Tier 4
  - Default unknown tools → Tier 4 (fail-safe)
- [x] `[CODE]` Implement `func (c *Classifier) Classify(a Action) ActionTier`
- [x] `[CODE]` Add allowlist of known-safe external hosts to config (empty by default)
- [x] `[TEST]` Write classifier test table: 20 action examples → assert correct tier for each
- [x] `[GATE]` **Phase 0 complete: Audit log ✓, Event bus ✓, Profiler ✓, Trust budget ✓, Classifier ✓ — all tests green**

---

## PHASE 1 — RECURSIVE SELF-IMPROVEMENT (RSI)
### *"The Ouroboros Loop"*
**Target: 3 weeks | Goal: Agent successfully improves one real function**

---

### 1.1 — Go AST Code Mapper

- [ ] `[CODE]` Create `internal/rsi/analyzer.go` — implement `CodeMapper`
- [ ] `[CODE]` Implement `func (m *CodeMapper) Build(rootPath string) (*CodeMap, error)`:
  - Walk all `.go` files under `rootPath` using `filepath.WalkDir`
  - Parse each file with `go/parser.ParseFile`
  - Extract all function declarations into `FunctionNode` structs
  - Build call graph: for each function, find all function calls it makes using `ast.Inspect`
  - Calculate cyclomatic complexity per function (count `if`, `for`, `switch`, `case`, `&&`, `||`)
- [ ] `[CODE]` Define `CodeMap` struct: `Packages map[string]*PackageNode`, `Functions map[string]*FunctionNode`, `CallGraph *DirectedGraph`
- [ ] `[CODE]` Define `FunctionNode`: `Name, File, Package, StartLine, EndLine, CyclomaticComp int, Dependencies []string, SourceCode string, EmbeddingVector []float32`
- [ ] `[CODE]` Implement `DirectedGraph` with adjacency list, `func (g *DirectedGraph) Dependents(fn string) []string` (who calls this function)
- [ ] `[CODE]` Implement `func (m *CodeMapper) Diff(before, after *CodeMap) []ChangedFunction` — detects which functions changed between two snapshots
- [ ] `[CODE]` Apply RSI Firewall: `Build()` returns error if `rootPath` resolves outside `internal/` when firewall is enabled
- [ ] `[TEST]` Write test: run CodeMapper on a known Go file → assert function names, line numbers, cyclomatic complexity match expected values
- [ ] `[TEST]` Write test: function with complexity 15 (multiple nested ifs) → assert score >= 15

---

### 1.2 — Source Code Embeddings

- [ ] `[DECISION]` Choose embedding provider: local (`ollama` with `nomic-embed-text`) vs API (`text-embedding-3-small`) — document trade-offs
- [ ] `[CODE]` Create `internal/rsi/embedder.go` — implement `CodeEmbedder`:
  - `func (e *CodeEmbedder) EmbedFunction(fn FunctionNode) ([]float32, error)`
  - Input: function source code as string
  - Chunk large functions (>500 tokens) before embedding
- [ ] `[CODE]` Implement embedding pipeline: after `CodeMapper.Build()`, embed all functions and store vectors in the existing vector DB
- [ ] `[CODE]` Add metadata to each vector: function name, file, package, complexity, last modified time
- [ ] `[CODE]` Implement `func (e *CodeEmbedder) FindSimilar(query string, topK int) []FunctionNode` — semantic search over code
- [ ] `[CODE]` Implement incremental update: only re-embed functions that changed since last run (use `CodeMapper.Diff`)
- [ ] `[TEST]` Write test: embed two similar functions and two dissimilar ones → cosine similarity of similar pair > 0.8, dissimilar pair < 0.4

---

### 1.3 — Weakness Report Generator

- [ ] `[CODE]` Create `internal/rsi/weakness.go` — implement `WeaknessReporter`
- [ ] `[CODE]` Implement `func (w *WeaknessReporter) Generate() WeaknessReport`:
  - Pull current `SpanStats` snapshot from Profiler
  - Pull `CodeMap` from CodeMapper
  - For each function in SpanStats, compute weakness score:
    `score = (error_rate * 0.5) + (normalized_p99 * 0.3) + (normalized_complexity * 0.2)`
  - Normalize p99 and complexity to 0.0-1.0 range using min-max scaling across all functions
  - Sort by score descending
  - Return top 10 weakest functions with their `FunctionNode` data
- [ ] `[CODE]` Define `WeaknessReport`: `GeneratedAt time.Time`, `TopCandidates []WeaknessEntry`, `TotalFunctionsAnalyzed int`
- [ ] `[CODE]` Define `WeaknessEntry`: `FunctionNode, WeaknessScore float64, PrimaryReason string` (which metric dominated)
- [ ] `[CODE]` Schedule `Generate()` on background goroutine: every 100 task completions OR every 1 hour, whichever first
- [ ] `[CODE]` Persist each generated `WeaknessReport` to SQLite (keep last 30)
- [ ] `[CODE]` Publish `EventRSICycleStarted` to event bus when report is generated
- [ ] `[TEST]` Write test: inject SpanStats with known high-error function → assert it appears #1 in report

---

### 1.4 — Proposal Engine

- [ ] `[CODE]` Create `internal/rsi/proposer.go` — implement `ProposalEngine`
- [ ] `[CODE]` Implement `func (p *ProposalEngine) Propose(entry WeaknessEntry) ([3]Proposal, error)`:
  - Launch 3 goroutines concurrently, each calling the LLM with temperatures `[0.2, 0.5, 0.8]`
  - Build prompt template (embed function source, metrics, constraints)
  - Require LLM output to be a valid unified diff (`--- a/file\n+++ b/file\n...`)
  - Validate each response is parseable as a diff before returning
  - Timeout: 120 seconds per proposal, use `context.WithTimeout`
- [ ] `[CODE]` Define `Proposal`: `ID string`, `Temperature float64`, `Diff string`, `TargetFunction string`, `GeneratedAt time.Time`, `LLMModel string`
- [ ] `[CODE]` Implement diff validation: `func validateDiff(diff string) error` — parses unified diff format, ensures target file path is under `internal/`
- [ ] `[CODE]` If all 3 proposals fail validation, write to audit log and abort this RSI cycle (do not retry indefinitely)
- [ ] `[CODE]` Write each proposal to audit log with event type `EventRSIProposalGenerated`
- [ ] `[TEST]` Write test with mocked LLM: returns valid diff → `Propose()` returns 3 proposals
- [ ] `[TEST]` Write test with mocked LLM: returns diff targeting `cmd/` directory → validation rejects it

---

### 1.5 — Docker Sandbox

- [ ] `[INFRA]` Ensure Docker daemon is running on the host, accessible via Docker socket
- [ ] `[CODE]` Create `internal/rsi/sandbox.go` — implement `Sandbox`
- [ ] `[CODE]` Implement `func (s *Sandbox) Run(proposal Proposal, baseRepoPath string) (*SandboxReport, error)`:
  - Create temp directory: copy entire repo to `/tmp/wunderpus-sandbox-{uuid}/`
  - Apply the proposal's unified diff using `exec.Command("patch", "-p1", ...)`
  - If patch fails → `SandboxReport{PatchApplied: false}`, clean up, return
  - Build the modified package: `go build ./internal/...` (timeout: 60s)
  - If build fails → `SandboxReport{BuildPassed: false}`, clean up, return
  - Run tests for the modified package: `go test -run . -bench . -benchtime 3s ./internal/...` (timeout: 60s)
  - Parse test output: extract pass/fail, benchmark ns/op
  - Run with `-race` flag to catch data races
  - Capture all output, return `SandboxReport`
- [ ] `[CODE]` Define `SandboxReport`: `PatchApplied, BuildPassed, TestsPassed, RaceClean bool`, `BenchmarkNsOp map[string]float64`, `TestOutput string`, `Duration time.Duration`
- [ ] `[CODE]` Implement network isolation: sandbox runs in separate Docker container with `--network none`
- [ ] `[CODE]` Implement resource limits: `--memory 512m --cpus 1.0` on the Docker container
- [ ] `[CODE]` Implement cleanup: always delete temp directory in `defer`, even on error
- [ ] `[TEST]` Write test: apply a known-good diff → sandbox reports all green
- [ ] `[TEST]` Write test: apply a diff with a syntax error → sandbox reports `BuildPassed: false`
- [ ] `[TEST]` Write test: apply a diff with a failing test → sandbox reports `TestsPassed: false`
- [ ] `[TEST]` Write test: sandbox cannot make outbound HTTP calls (assert network isolation)

---

### 1.6 — Fitness Function

- [ ] `[CODE]` Create `internal/rsi/fitness.go` — implement `FitnessEvaluator`
- [ ] `[CODE]` Implement `func (f *FitnessEvaluator) Score(before SpanStats, report SandboxReport) float64`:
  ```
  if !report.TestsPassed || !report.RaceClean → return -1.0
  latencyDelta = (before.P99LatencyNs - after.P99LatencyNs) / before.P99LatencyNs
  errorDelta   = (before.ErrorCount - after.ErrorCount) / max(before.ErrorCount, 1)
  return (latencyDelta * 0.6) + (errorDelta * 0.4)
  ```
- [ ] `[CODE]` Define minimum fitness threshold: `0.05` (5% improvement) — read from config, can be tuned
- [ ] `[CODE]` Implement `func (f *FitnessEvaluator) SelectWinner(proposals []Proposal, reports []SandboxReport, before SpanStats) (*Proposal, float64)` — returns highest-scoring proposal above threshold, or nil if none qualify
- [ ] `[CODE]` Log all scores to audit log, including losing proposals
- [ ] `[TEST]` Write test: before P99=1000ms, after P99=800ms, tests pass → score = 0.12 (>0.05 threshold)
- [ ] `[TEST]` Write test: test failure → score = -1.0, not selected as winner

---

### 1.7 — Deployer

- [ ] `[CODE]` Create `internal/rsi/deployer.go` — implement `Deployer`
- [ ] `[CODE]` Implement `func (d *Deployer) Deploy(proposal Proposal, fitness float64) error`:
  - Apply the winning diff to the live source tree
  - Run `go build ./...` to produce new binary
  - Create git branch: `rsi/auto-YYYY-MM-DD-HHMMSS`
  - Commit with message including fitness score, target function, metrics delta
  - Tag previous commit as `rsi/rollback-YYYYMMDDHHMMSS`
  - Write `EventRSIDeployed` to audit log
  - Signal the main process to restart with the new binary (use Unix signal or pid file)
- [ ] `[CODE]` Implement `func (d *Deployer) Rollback(tag string) error`:
  - `git checkout <tag>`
  - Rebuild and restart
  - Write `EventRSIRolledBack` to audit log
- [ ] `[CODE]` Implement production regression detection: background goroutine monitors SpanStats for 10 minutes post-deploy; if error rate increases by >20% vs pre-deploy baseline → auto-rollback
- [ ] `[CODE]` Enforce RSI Firewall in deployer: scan the diff for any path outside `internal/` → abort with audit entry
- [ ] `[TEST]` Write test: deploy a known-good improvement → git log shows new branch with correct commit message
- [ ] `[TEST]` Write test: inject spike in error rate post-deploy → rollback fires within 15 minutes
- [ ] `[GATE]` **RSI INTEGRATION TEST: Run full Ouroboros cycle on a real Wunderpus function. The function must be measurably improved (fitness > 0.05) and deployed. Git log must show the RSI commit. All existing tests must still pass.**

---

## PHASE 2 — AUTONOMOUS GOAL SYNTHESIS (AGS)
### *"The Goal Engine"*
**Target: 3 weeks | Goal: Agent generates and executes its first self-originated goal**

---

### 2.1 — Goal Data Model

- [ ] `[CODE]` Create `internal/ags/goal.go` — define `Goal` struct:
  - `ID string` (UUID)
  - `Title string`
  - `Description string`
  - `Tier int` (1-3; Tier 0 is hardcoded, not stored)
  - `Priority float64` (0.0-1.0, recomputed on each cycle)
  - `Status GoalStatus` (pending, active, completed, abandoned)
  - `ParentID string`
  - `ChildIDs []string`
  - `CreatedAt, UpdatedAt time.Time`
  - `Evidence []string` (why this goal was created)
  - `SuccessCriteria []string`
  - `ExpectedValue float64`
  - `AttemptCount int`
  - `LastAttempt *time.Time`
  - `CompletedAt *time.Time`
  - `ActualValue *float64` (filled in after completion)
- [ ] `[CODE]` Define `GoalStatus` as typed string with all constants
- [ ] `[CODE]` Hardcode Tier 0 goals as package-level vars (not in DB, not modifiable by agent):
  - `GoalBeUseful` = "Be maximally useful to operators"
  - `GoalImproveCapabilities` = "Improve own capabilities"
  - `GoalMaintainContinuity` = "Maintain operational continuity"
  - `GoalExpandKnowledge` = "Expand knowledge and world-model"
- [ ] `[CODE]` Create `internal/ags/store.go` — SQLite-backed `GoalStore`:
  - `func (s *GoalStore) Save(g Goal) error`
  - `func (s *GoalStore) GetByID(id string) (Goal, error)`
  - `func (s *GoalStore) GetByStatus(status GoalStatus) ([]Goal, error)`
  - `func (s *GoalStore) GetByTier(tier int) ([]Goal, error)`
  - `func (s *GoalStore) Update(g Goal) error`
  - `func (s *GoalStore) History(limit int) ([]Goal, error)` — completed + abandoned
- [ ] `[TEST]` Write test: save 50 goals, query by status and tier → correct counts returned

---

### 2.2 — Priority Scorer

- [ ] `[CODE]` Create `internal/ags/scorer.go` — implement `PriorityScorer`
- [ ] `[CODE]` Implement `func (s *PriorityScorer) Score(g Goal) float64`:
  - `urgency = computeUrgency(g)` — is there a time component? Has it been deferred often?
  - `impact = g.ExpectedValue` — already 0.0-1.0
  - `feasibility = computeFeasibility(g)` — do we have tools and resources for this right now?
  - `novelty = computeNovelty(g)` — 1.0 if never attempted, decreases with each attempt
  - `alignment = computeAlignment(g)` — how strongly does this tie to a Tier 0 goal?
  - `return (urgency*0.25) + (impact*0.30) + (feasibility*0.20) + (novelty*0.10) + (alignment*0.15)`
- [ ] `[CODE]` Implement `computeUrgency`: base 0.5, +0.3 if `AttemptCount > 2` (kept getting deferred), -0.1 per day since creation (deprioritize stale goals)
- [ ] `[CODE]` Implement `computeFeasibility`: check if required tools exist in Wunderpus's tool registry, check trust budget can cover expected action tier, check resource broker has required resources
- [ ] `[CODE]` Implement `computeNovelty`: `1.0 / (1.0 + AttemptCount)`
- [ ] `[CODE]` Implement `computeAlignment`: LLM call asking "on a scale 0-1, how closely does this goal serve [Tier 0 goal]?" — cache result per goal
- [ ] `[CODE]` Store scorer weights in config (hot-reloadable, adjustable by Metacognition Loop)
- [ ] `[TEST]` Write test: high-error, unfinished goal from 5 days ago with 3 failed attempts → scores higher than new speculative goal

---

### 2.3 — Goal Synthesizer

- [ ] `[CODE]` Create `internal/ags/synthesizer.go` — implement `GoalSynthesizer`
- [ ] `[CODE]` Implement `func (s *GoalSynthesizer) Synthesize() ([]Goal, error)`:
  1. **Memory scan**: pull last 200 episodic memory entries, look for patterns:
     - Repeated errors of the same type → capability gap goal
     - Tasks that took >10x average time → efficiency improvement goal
     - Failed tasks with "no tool for X" → tool acquisition goal
  2. **World model check**: query for recently changed external conditions (new LLM releases, API deprecations via web search)
  3. **Weakness Report check**: if WeaknessReport shows functions with score > 0.7 → generate RSI-aligned goal
  4. **LLM synthesis**: pass all findings to LLM, request JSON output:
     ```json
     {
       "proposed_goals": [
         {
           "title": "...",
           "description": "...",
           "tier": 2,
           "evidence": ["..."],
           "parent_tier0": "...",
           "expected_value": 0.75,
           "success_criteria": ["..."]
         }
       ]
     }
     ```
  5. Validate JSON schema strictly — reject any proposal missing required fields
  6. Deduplicate against existing active/pending goals using semantic similarity (embedding cosine > 0.85 = duplicate)
  7. Return validated, deduplicated proposals
- [ ] `[CODE]` Schedule synthesis: every 100 task completions OR every 60 minutes
- [ ] `[CODE]` Each proposed goal auto-saved with status `pending`, publishes `EventGoalCreated`
- [ ] `[CODE]` Limit: synthesizer creates at most 5 new goals per cycle (prevent goal explosion)
- [ ] `[TEST]` Write test: inject episodic memory with 10 identical `"browser_timeout"` errors → synthesizer proposes a browser reliability goal
- [ ] `[TEST]` Write test: synthesizer proposes goal identical to existing active goal → deduplication drops it

---

### 2.4 — Goal Executor

- [ ] `[CODE]` Create `internal/ags/executor.go` — implement `GoalExecutor`
- [ ] `[CODE]` Implement `func (e *GoalExecutor) SelectNext() (*Goal, error)`:
  - Fetch all `pending` goals from GoalStore
  - Rescore all with PriorityScorer
  - Update `Priority` field in DB for all
  - Return highest-priority goal that passes feasibility check
- [ ] `[CODE]` Implement `func (e *GoalExecutor) Decompose(g Goal) ([]TaskBlueprint, error)`:
  - Call LLM with goal description + success criteria
  - Request ordered list of concrete tasks in JSON format
  - Each task must specify: `description`, `tool`, `parameters`, `expected_outcome`
  - Validate: all tools referenced must exist in tool registry
- [ ] `[CODE]` Define `TaskBlueprint`: `StepNum int`, `Description string`, `Tool string`, `Parameters map[string]interface{}`, `ExpectedOutcome string`, `DependsOn []int`
- [ ] `[CODE]` Implement `func (e *GoalExecutor) Execute(g Goal, tasks []TaskBlueprint) error`:
  - Mark goal `active`, publish `EventGoalActivated`, write to audit log
  - Execute each task through the UAA Executor (Pillar 3) — all actions go through autonomy gates
  - Track outcomes: record each task result in episodic memory
  - Evaluate success criteria after all tasks complete (LLM judge: "Did the outcomes satisfy these criteria? Y/N")
  - If success: mark goal `completed`, fill `ActualValue`, publish `EventGoalCompleted`
  - If failure: increment `AttemptCount`, set `LastAttempt`, return to `pending` (up to 3 attempts)
  - If 3 failures: mark `abandoned`, publish `EventGoalAbandoned`, synthesizer will reframe next cycle
- [ ] `[CODE]` Run `SelectNext + Decompose + Execute` on a background goroutine — at most 1 active goal at a time initially
- [ ] `[TEST]` Write test: inject a simple, feasible pending goal → executor completes it and marks `completed`
- [ ] `[TEST]` Write test: goal fails 3 times → status becomes `abandoned`, audit log has 3 failure entries

---

### 2.5 — Metacognition Loop

- [ ] `[CODE]` Create `internal/ags/metacognition.go` — implement `MetacognitionLoop`
- [ ] `[CODE]` Implement `func (m *MetacognitionLoop) Run() error` (called weekly):
  1. Pull all goals completed or abandoned in the last 7 days from GoalStore
  2. For completed goals: compute `value_accuracy = ActualValue / ExpectedValue`
  3. Compute `completion_rate = completed / (completed + abandoned)`
  4. Identify systematically deferred goals (AttemptCount > 2, still pending)
  5. Call LLM with all metrics, request scorer weight adjustments:
     - If impact scores are overestimated (ActualValue < 0.7 * ExpectedValue) → reduce impact weight
     - If feasibility fails often → increase feasibility weight
     - Return new weights as JSON
  6. Validate weights sum to 1.0 ± 0.001
  7. Write new weights to config via config store (hot-reload picks them up immediately)
  8. Write metacognition report to audit log
- [ ] `[CODE]` Schedule: run every 7 days (configurable)
- [ ] `[CODE]` Safeguard: weights can change by at most ±0.05 per cycle (no sudden weight collapse)
- [ ] `[TEST]` Write test: inject 20 completed goals where ExpectedValue = 0.9 but ActualValue avg = 0.4 → metacognition reduces impact weight
- [ ] `[GATE]` **AGS INTEGRATION TEST: Starting from a fresh goal store, let Wunderpus run for 1 hour. Assert: at least 1 self-generated Tier 2 goal was created, at least 1 Tier 3 goal was completed, and the completion is visible in the audit log.**

---

## PHASE 3 — UNBOUNDED AUTONOMOUS ACTION & RESOURCE ACQUISITION
### *"The Autonomy Engine + Resource Broker"*
**Target: 4 weeks | Goal: Agent completes a self-generated goal using self-provisioned resources**

---

### 3.1 — Shadow Mode Simulator

- [ ] `[CODE]` Create `internal/uaa/shadow.go` — implement `ShadowSimulator`
- [ ] `[CODE]` Implement `func (s *ShadowSimulator) Simulate(ctx context.Context, action Action) (*SimResult, error)`:
  - Build an in-memory mock environment that mirrors the real environment:
    - Filesystem: `afero.NewMemMapFs()` (use `spf13/afero`)
    - HTTP calls: intercept and return stubbed 200 responses
    - DB writes: no-op stubs that return success
  - Execute the action against the mock environment
  - Capture the diff: what files would change? What HTTP calls would fire?
  - Format diff as human-readable summary
  - Pass summary to LLM judge: "Given this planned action and its expected effects, is this safe and desirable? Respond: APPROVE or REJECT with one sentence reason."
  - Parse LLM response: extract APPROVE/REJECT
- [ ] `[CODE]` Define `SimResult`: `Approved bool`, `Reason string`, `EffectSummary string`, `SimDuration time.Duration`
- [ ] `[CODE]` Timeout: shadow simulation must complete in 30 seconds or auto-REJECT
- [ ] `[CODE]` Cache: if same action (same tool + parameters hash) was approved within 5 minutes, skip re-simulation
- [ ] `[TEST]` Write test: simulate `write_file` to `/etc/passwd` → LLM judge REJECTS

---

### 3.2 — Full UAA Executor

- [ ] `[CODE]` Create `internal/uaa/executor.go` — implement the full `UAA Executor`
- [ ] `[CODE]` Implement `func (e *UAA) Execute(ctx context.Context, action Action) (*ActionResult, error)`:
  1. `action.Tier = e.classifier.Classify(action)` — classify
  2. `ok, reason = e.trust.CanExecute(action.TrustCost)` — check budget
  3. If `!ok` → write audit entry `EventActionRejected`, return error
  4. If `action.Tier >= TierPersistent` → run `e.shadow.Simulate(ctx, action)`
  5. If simulation rejects → write audit entry, return error
  6. `e.trust.Deduct(action.TrustCost)`
  7. Execute action via tool runner
  8. `e.trust.RecordOutcome(action, err == nil)` — credit partial refund or apply penalty
  9. Write `EventActionExecuted` or `EventActionFailed` to audit log
  10. Return result
- [ ] `[CODE]` Make all existing Wunderpus tool calls go through the UAA Executor (replace direct tool invocations)
- [ ] `[TEST]` Write test: Tier 4 action with insufficient trust budget → rejected, budget unchanged
- [ ] `[TEST]` Write test: Tier 3 action, shadow approves → executes, trust debited
- [ ] `[TEST]` Write test: Tier 1 action → never hits shadow mode, executes immediately

---

### 3.3 — Resource Abstraction Layer

- [ ] `[CODE]` Create `internal/ra/resource.go` — define `Resource`, `ResourceType`, `ResourceSpec`, `ResourceStatus`, `Credentials` (AES-256-GCM encrypted)
- [ ] `[CODE]` Create `internal/ra/registry.go` — `ResourceRegistry` backed by SQLite:
  - `func (r *ResourceRegistry) Register(res Resource) error`
  - `func (r *ResourceRegistry) Get(id string) (Resource, error)`
  - `func (r *ResourceRegistry) ListByType(t ResourceType) ([]Resource, error)`
  - `func (r *ResourceRegistry) Deregister(id string) error`
  - `func (r *ResourceRegistry) UpdateStatus(id string, status ResourceStatus) error`
- [ ] `[CODE]` Implement credential encryption: `func EncryptCreds(creds Credentials, key []byte) ([]byte, error)` using `crypto/aes` + `crypto/cipher` (GCM mode)
- [ ] `[CODE]` Register "local" resources automatically on startup: current machine CPU/RAM/disk → `ResourceType = "compute"`, provider `"local"`
- [ ] `[TEST]` Write test: register local resource → list returns it with correct spec

---

### 3.4 — API Key Manager

- [ ] `[CODE]` Create `internal/ra/keymanager.go` — implement `APIKeyManager`
- [ ] `[CODE]` Implement `func (k *APIKeyManager) Register(provider string, key string, limits RateLimits) error` — encrypts and stores in resource registry
- [ ] `[CODE]` Implement `func (k *APIKeyManager) Get(provider string) (string, error)` — decrypts and returns best available key, prioritizing free-tier keys
- [ ] `[CODE]` Implement rate limit tracking: decrement remaining quota on each use, reset on schedule
- [ ] `[CODE]` Implement `func (k *APIKeyManager) Rotate(provider string) error` — marks current key as expiring, selects next
- [ ] `[CODE]` Auto-discover free API tiers: maintain a static list of known free-tier providers + quota limits (OpenRouter free tier, Groq free tier, etc.) — read from a YAML config file, not hardcoded in Go
- [ ] `[TEST]` Write test: register 3 keys for same provider (1 free, 2 paid) → Get returns free key first

---

### 3.5 — Cloud Adapter (DigitalOcean)

- [ ] `[INFRA]` Create DigitalOcean account, generate API token, store in key manager
- [ ] `[CODE]` Create `internal/ra/cloud/digitalocean.go` — implement `DOAdapter` implementing `CloudAdapter` interface:
  ```go
  type CloudAdapter interface {
      ProvisionCompute(spec ResourceSpec) (Resource, error)
      ProvisionStorage(spec ResourceSpec) (Resource, error)
      Deprovision(resourceID string) error
      ListProvisioned() ([]Resource, error)
      GetCostToDate() (float64, error)
  }
  ```
- [ ] `[CODE]` Implement `ProvisionCompute`: create a DigitalOcean Droplet via their API (`godo` Go library)
  - Smallest viable size: `s-1vcpu-1gb` ($6/mo)
  - Always tag with `wunderpus-managed` for cost tracking and cleanup
  - Store credentials (SSH key + IP) encrypted in resource registry
- [ ] `[CODE]` Implement `ProvisionStorage`: create a DigitalOcean Space (S3-compatible)
- [ ] `[CODE]` Implement `Deprovision`: destroy Droplet or Space, update registry
- [ ] `[CODE]` Implement hard spend cap check: before any provision call, `GetCostToDate()` must be below `MaxDailySpendUSD` — if exceeded, return error
- [ ] `[TEST]` Write test with mocked DO API: provision → registry shows new resource → deprovision → registry shows removed
- [ ] `[TEST]` Write test: cost exceeds daily cap → provision returns error without making API call

---

### 3.6 — Marketplace Adapter (OpenRouter)

- [ ] `[CODE]` Create `internal/ra/marketplace/openrouter.go` — implement `OpenRouterAdapter`
- [ ] `[CODE]` Implement model routing: given a task type and budget, select the cheapest model that meets quality threshold:
  - For code generation: `deepseek-coder`, `claude-3.5-sonnet`
  - For text tasks: `gemma-2-9b`, `llama-3-70b`
  - For embedding: `text-embedding-3-small`
- [ ] `[CODE]` Implement automatic fallback: if primary model returns error, retry with next in priority list
- [ ] `[CODE]` Track per-model spend in real time, feed into cost optimizer
- [ ] `[CODE]` Register OpenRouter as default LLM provider (replaces hardcoded provider if one exists)
- [ ] `[TEST]` Write test with mocked OpenRouter: primary model fails → fallback model used automatically

---

### 3.7 — Resource Forecaster

- [ ] `[CODE]` Create `internal/ra/forecaster.go` — implement `ResourceForecaster`
- [ ] `[CODE]` Implement `func (f *ResourceForecaster) Project(goals []Goal, tasks []Task) ResourceForecast`:
  - For each pending/active task, look up historical resource usage from episodic memory (how much CPU, which APIs, how long)
  - If no history, use conservative defaults: 1 CPU-hour, 100MB storage, 1000 API calls
  - Sum all projected needs, add 20% buffer
  - Compare against currently registered available resources
  - Flag shortfalls as `ResourceNeed` structs
- [ ] `[CODE]` Implement `func (f *ResourceForecaster) AutoProvision(needs []ResourceNeed) error` — calls cloud adapter to provision missing resources (gated by trust budget and spend cap)
- [ ] `[CODE]` Schedule: run every 15 minutes
- [ ] `[CODE]` Publish `EventResourceExhausted` if shortfall cannot be met within budget
- [ ] `[TEST]` Write test: inject task queue with known resource profile → forecast matches expected needs with ≤25% error

---

### 3.8 — Integration Wiring

- [ ] `[CODE]` Connect RSI `EventRSIDeployed` → Trust budget credits +100 points
- [ ] `[CODE]` Connect RA `EventResourceExhausted` → UAA gates all Tier 4 actions until resolved
- [ ] `[CODE]` Connect AGS `EventGoalCompleted` → Profiler resets baseline for affected functions
- [ ] `[CODE]` Connect AGS `EventGoalAbandoned` → Synthesizer gets notified to reframe on next cycle
- [ ] `[CODE]` Connect UAA `EventLockdownEngaged` → RA suspends all cloud provisioning
- [ ] `[TEST]` Write event flow test: simulate lockdown → assert RA stops provisioning within one event cycle
- [ ] `[GATE]` **PHASE 3 INTEGRATION TEST: From a clean state, let Wunderpus run for 4 hours. Assert: (1) A self-generated goal was created and completed. (2) At least one action was gated by the trust budget and the gate decision appears in the audit log. (3) Shadow mode fired at least once for a Tier 3 action. (4) All existing unit tests still pass.**

---

## PHASE 4 — SOVEREIGNTY
### *"The Full Loop"*
**Target: Ongoing | Goal: Self-bootstrapping from a single binary**

---

### 4.1 — WASM Sandbox (RSI upgrade)

- [ ] `[DECISION]` Choose WASM runtime: `wasmer-go`, `wasmtime-go`, or `wazero` (pure Go, no CGO — recommend `wazero`)
- [ ] `[CODE]` Replace Docker sandbox in `internal/rsi/sandbox.go` with WASM-based execution
- [ ] `[CODE]` Compile target functions to WASM using TinyGo: `tinygo build -target wasm -o candidate.wasm ./internal/pkg/...`
- [ ] `[CODE]` Execute WASM module in `wazero` runtime with memory cap (32MB) and instruction count limit
- [ ] `[CODE]` Keep Docker sandbox as fallback when WASM compilation fails (not all Go code compiles to WASM)
- [ ] `[TEST]` Write test: sandbox executes correct function, WASM result matches expected output

---

### 4.2 — Self-Referential RSI Unlock

- [ ] `[DECISION]` Human review of RSI track record required before enabling: minimum 10 successful RSI cycles, zero rollbacks in last 5 cycles
- [ ] `[CODE]` Add config flag: `RSISelfReferentialEnabled bool` (default `false`)
- [ ] `[CODE]` Modify RSI Firewall: if `RSISelfReferentialEnabled`, allow modifications to `internal/rsi/` package (but never `cmd/main.go`, never the firewall itself)
- [ ] `[CODE]` Add extra sandbox requirement for self-referential proposals: must run 2x the normal test suite with both old and new RSI code
- [ ] `[TEST]` Write test: with flag disabled, RSI proposal targeting `internal/rsi/fitness.go` → firewall rejects
- [ ] `[TEST]` Write test: with flag enabled, valid improvement to `internal/rsi/fitness.go` → proceeds through normal pipeline

---

### 4.3 — Metacognition Applies RSI to Itself

- [ ] `[CODE]` After metacognition computes new scorer weights, also run the Weakness Report on AGS functions themselves
- [ ] `[CODE]` If any AGS function (synthesizer, scorer, executor) appears in top-10 weakness report → generate an RSI proposal targeting it
- [ ] `[CODE]` This creates the full recursive loop: RSI improves AGS which improves goal quality which drives better RSI targets
- [ ] `[TEST]` Write test: inject a deliberately slow `GoalSynthesizer.Synthesize` mock → metacognition generates RSI proposal for it

---

### 4.4 — Multi-Agent Coordination

- [ ] `[CODE]` Create `internal/agents/manager.go` — implement `AgentManager`
- [ ] `[CODE]` Implement `func (m *AgentManager) Spawn(config AgentConfig) (*SubAgent, error)`:
  - Start a new Wunderpus process (or goroutine with isolated config)
  - Register it in resource registry as a `ResourceType = "agent"`
  - Assign it a specific goal from the goal tree (a Tier 3 leaf that can run independently)
- [ ] `[CODE]` Implement inter-agent communication: shared SQLite DB + event bus over Unix socket (same host) or gRPC (remote)
- [ ] `[CODE]` Implement `func (m *AgentManager) Collect(agentID string) (GoalResult, error)` — waits for sub-agent to complete its goal
- [ ] `[CODE]` Implement automatic sub-agent termination: if sub-agent exceeds time budget or trust budget, kill it and rollback its changes
- [ ] `[TEST]` Write test: spawn sub-agent with trivial goal → completes → results collected by parent

---

### 4.5 — Financial Resource Acquisition (Opt-In)

> **These tasks must not be started until operator explicitly sets `FinancialAcquisitionEnabled: true` in config.**

- [ ] `[CODE]` Create `internal/ra/financial/` package, all code gated by config flag check at runtime
- [ ] `[CODE]` Implement Stripe integration: create payment links for agent capabilities API
- [ ] `[CODE]` Implement GitHub bounty scanner: `GET /search/issues?q=label:bounty+state:open`, filter by matching capabilities
- [ ] `[CODE]` Implement bounty submission: when agent completes a task matching an open bounty, format and submit PR with solution
- [ ] `[CODE]` All financial transactions write to audit log with `Subsystem: "financial"` — never suppressed
- [ ] `[TEST]` Write test: config flag disabled → any call into financial package returns `ErrFinancialAcquisitionDisabled`

---

### 4.6 — Self-Bootstrap Test

> This is the final boss. Everything in the plan has been leading here.

- [ ] `[TEST]` Provision a fresh VPS (DigitalOcean $6 droplet, bare Ubuntu)
- [ ] `[TEST]` Copy only the compiled Wunderpus binary and a `.env` file with: `LLM_API_KEY=<key>`, `WUNDERPUS_ENV=bootstrap`
- [ ] `[TEST]` Start the binary. Set a 60-minute timer.
- [ ] `[TEST]` Assert within 60 minutes:
  - [ ] Agent cloned its own repository (git clone)
  - [ ] Agent provisioned at least one cloud resource (a storage bucket for episodic memory)
  - [ ] Agent synthesized at least one self-generated goal
  - [ ] Agent completed at least one self-generated goal
  - [ ] Audit log is intact and hash-chain verifies
  - [ ] Agent did not exceed $1.00 in cloud spend
  - [ ] No human interaction was required after `./wunderpus &`
- [ ] `[GATE]` **SOVEREIGNTY GATE: All 7 bootstrap assertions pass. Wunderpus is self-sovereign.**

---

## ONGOING — MAINTENANCE CHECKLIST
### *Run these checks on every RSI deployment*

- [ ] `audit.Verify()` returns nil — hash chain intact
- [ ] Trust budget is not in lockdown
- [ ] All Tier 0 goals are still hardcoded and unmodified (diff `internal/ags/goal.go` Tier 0 section)
- [ ] RSI Firewall config has not been modified by the agent
- [ ] No RSI commit targets files outside `internal/`
- [ ] Daily cloud spend is below cap
- [ ] At least 3 successful Ouroboros cycles in the last 7 days
- [ ] Metacognition ran within the last 8 days
- [ ] Sub-agent count is ≤ configured maximum

---

## QUICK REFERENCE — FILE MAP

```
internal/
  audit/
    entry.go          ← AuditEntry struct, hash chain logic
    log.go            ← AuditLog: Write, Query, Verify
    event_types.go    ← All EventType constants
  events/
    bus.go            ← Typed pub/sub event bus
    event.go          ← Event struct
  config/
    config.go         ← Master config struct with all flags
    loader.go         ← YAML loader + env overrides + hot-reload
  rsi/
    profiler.go       ← SpanStats, Profiler.Track(), Snapshot()
    analyzer.go       ← CodeMapper, CodeMap, FunctionNode, DirectedGraph
    embedder.go       ← CodeEmbedder, vector storage
    weakness.go       ← WeaknessReporter, WeaknessReport
    proposer.go       ← ProposalEngine, 3-parallel LLM proposals
    sandbox.go        ← Docker/WASM sandbox, SandboxReport
    fitness.go        ← FitnessEvaluator, SelectWinner
    deployer.go       ← Deployer, git branch, regression detection, rollback
  ags/
    goal.go           ← Goal struct, GoalStatus, Tier 0 constants
    store.go          ← GoalStore (SQLite)
    scorer.go         ← PriorityScorer, all compute* functions
    synthesizer.go    ← GoalSynthesizer, memory scan, LLM synthesis
    executor.go       ← GoalExecutor, SelectNext, Decompose, Execute
    metacognition.go  ← MetacognitionLoop, weight adjustment
  uaa/
    classifier.go     ← ActionTier, Action, rule-based Classifier
    trust.go          ← TrustBudget, CanExecute, Deduct, Credit, Lockdown
    shadow.go         ← ShadowSimulator, SimResult, LLM judge
    executor.go       ← Full UAA middleware wrapper
    audit.go          ← UAA-specific audit helpers
  ra/
    resource.go       ← Resource, ResourceType, Credentials (encrypted)
    registry.go       ← ResourceRegistry (SQLite)
    keymanager.go     ← APIKeyManager, rate tracking, rotation
    forecaster.go     ← ResourceForecaster, AutoProvision
    cloud/
      digitalocean.go ← DOAdapter, CloudAdapter interface
    marketplace/
      openrouter.go   ← OpenRouterAdapter, model routing
    financial/        ← Opt-in financial acquisition (Phase 4)
  agents/
    manager.go        ← AgentManager, Spawn, Collect (Phase 4)
```

---

**Checklist Version**: 1.0.0
**Derived from**: WUNDERPUS_GENESIS_PLAN.md v1.0.0
**Total checkable items**: 187
**Gates**: 5 (must pass before next phase begins)
**Authors**: Razzy + Claude
