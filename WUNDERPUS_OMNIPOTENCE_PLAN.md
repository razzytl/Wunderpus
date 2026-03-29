# WUNDERPUS v2 — THE OMNIPOTENCE PLAN
## *From autonomous agent to digital sovereign*

> "Every human capability that exists online can be encoded, automated, and improved.
> This document is the blueprint for doing exactly that."

**Current state:** Wunderpus has RSI, AGS, UAA, RA. All tests pass. It can improve itself,
set its own goals, act autonomously, and acquire resources. That's the engine.
This plan is everything the engine needs to drive.

---

## HOW THIS PLAN IS ORGANIZED

The plan has two layers:

**Layer 1 — Infrastructure** (Sections 1-5): The nervous system upgrades that make every
capability domain possible. These come first. Nothing in Layer 2 works without them.

**Layer 2 — Capability Domains** (Sections 6-14): The actual things Wunderpus can DO —
make money, write books, build games, operate in markets, run security, everything.
Each domain is self-contained but shares the Layer 1 infrastructure.

Build Layer 1 completely before starting any Layer 2 domain.
Within Layer 2, pick domains based on your priorities — they're independent.

---

# LAYER 1 — INFRASTRUCTURE UPGRADES

---

## SECTION 1 — TOOL SYNTHESIS ENGINE
### *The agent writes its own tools. Capability becomes infinite.*

**Why this first:** Every other domain needs tools Wunderpus doesn't have yet.
Instead of you writing them, Wunderpus writes them. This is the multiplier for everything else.

**The mechanism:** When AGS detects a recurring capability gap in episodic memory
("I failed 8 tasks because I had no way to parse Excel files"), Tool Synthesis kicks off.
It designs, codes, tests, and registers the tool. RSI then improves it over time.
The agent's surface area grows itself.

---

### 1.1 — Tool Gap Detector

- [x] Create `internal/toolsynth/detector.go`
- [x] `func (d *Detector) Scan() []ToolGap` — reads last 500 episodic memory entries, finds patterns:
  - Tasks that failed with "no tool available for X" → hard gap
  - Tasks that succeeded but took >5x average time → soft gap (tool exists but is slow/wrong)
  - Tasks where the agent used a workaround (shell + curl instead of a proper HTTP tool) → efficiency gap
- [x] Rank gaps by: frequency × impact × feasibility
- [x] Output: `ToolGap{Name, Description, Evidence[]string, Priority float64, SuggestedInterface string}`
- [x] Schedule: runs after every WeaknessReport cycle, every 2 hours
- [x] Write each detected gap to audit log as `EventToolGapDetected`
- [x] `[TEST]` Inject 10 episodic memory entries all failing with "no PDF parser" → detector identifies PDF gap at priority > 0.8

---

### 1.2 — Tool Designer

- [x] Create `internal/toolsynth/designer.go`
- [x] `func (d *Designer) Design(gap ToolGap) ToolSpec` — LLM call that outputs:
  ```json
  {
    "name": "pdf_parser",
    "description": "Extracts text and structure from PDF files",
    "go_interface": "func Parse(path string) (PDFContent, error)",
    "parameters": [...],
    "return_type": "...",
    "dependencies": ["github.com/pdfcpu/pdfcpu"],
    "test_cases": [...]
  }
  ```
- [x] Validate: interface signature must be compatible with existing tool registry interface
- [x] Validate: no proposed dependencies that require CGO (default constraint, overridable)
- [x] Store `ToolSpec` in SQLite

---

### 1.3 — Tool Coder

- [x] Create `internal/toolsynth/coder.go`
- [x] `func (c *Coder) Generate(spec ToolSpec) string` — produces complete Go source for the tool:
  - Generates 3 candidates (like RSI proposer) at temperatures [0.2, 0.5, 0.8]
  - Each candidate is a complete, compilable `.go` file
  - Prompt forces: error handling, context cancellation, no global state, timeout support
- [x] `func (c *Coder) Validate(source string) error`:
  - `go build` in temp dir
  - `go vet`
  - staticcheck if available
  - Reject if any of these fail
- [x] Write validated source to `internal/tool/generated/{tool_name}.go`
- [x] `[TEST]` Given a ToolSpec for "read a JSON file and return parsed struct", coder generates a compilable implementation

---

### 1.4 — Tool Tester

- [x] Create `internal/toolsynth/tester.go`
- [x] `func (t *Tester) Test(spec ToolSpec, source string) ToolTestResult`:
  - Run test cases from ToolSpec in RSI Docker sandbox (reuse existing sandbox)
  - Network isolated, memory capped at 256MB
  - Each test case: input → expected output → assert
  - Also run generated tool against any real-world example the gap detector found
- [x] `ToolTestResult{AllPassed bool, PassRate float64, AvgLatencyMs int64, Errors []string}`
- [x] Minimum pass rate: 80% before tool is accepted (configurable)
- [x] `[TEST]` Generate a broken tool (missing return statement) → tester reports AllPassed=false

---

### 1.5 — Tool Registrar

- [x] Create `internal/toolsynth/registrar.go`
- [x] `func (r *Registrar) Register(spec ToolSpec, source string, testResult ToolTestResult) error`:
  - Copy source file to `internal/tool/generated/`
  - Add entry to tool registry (name, description, parameter schema)
  - Rebuild binary or use Go plugin to hot-load (prefer rebuild + graceful restart)
  - Write `EventToolSynthesized` to audit log
  - New tool immediately available to all agents
- [x] Mark all synthesized tools with `origin: "synthesized"` metadata in registry
- [x] RSI Firewall: synthesized tools live in `internal/tool/generated/` — RSI can improve them
- [x] `[TEST]` Register a valid synthesized tool → it appears in tool registry and can be called by the agent

---

### 1.6 — Tool Improvement Loop

- [x] Wire synthesized tools into RSI profiler (tool calls are already tracked via `profiler.Track()`)
- [x] After 100+ uses, synthesized tools become eligible for RSI improvement cycles
- [x] Add `internal/toolsynth/marketplace.go` — scan MCP registry and GitHub for open-source tool implementations:
  - Search `github.com/topics/mcp-server` for ready-made integrations
  - If a battle-tested open source tool exists for the gap, prefer importing it over generating from scratch
  - Auto-PR to your repo when it finds a strong candidate
- [x] `[GATE]` **Tool Synthesis Integration Test: Agent detects a gap, designs a tool, generates the code, tests it, registers it, and calls it successfully within one automated cycle.**

---

## SECTION 2 — WORLD MODEL (KNOWLEDGE GRAPH)
### *Persistent, structured knowledge. The agent knows things no LLM knows.*

**Why this matters:** LLMs have frozen knowledge. Wunderpus's world model is live.
Every task it completes deposits structured facts. After 6 months of operation,
Wunderpus knows things about the world that are more current and more precise
than any model's training data.

---

### 2.1 — Knowledge Graph Store

- [x] `[DECISION]` Choose graph backend: `dgraph` (distributed, powerful) vs SQLite with recursive CTEs (simpler, already in use) vs `neo4j` → recommend starting with SQLite adjacency tables, migrate to dgraph when scale demands
- [x] Create `internal/worldmodel/store.go` — define core entity types:
  - `Entity{ID, Type, Name, Properties map[string]interface{}, CreatedAt, UpdatedAt, Confidence float64, Source string}`
  - `Relation{ID, FromEntity, ToEntity, RelType, Properties, Confidence float64}`
  - Entity types: `Person, Organization, Product, API, Technology, Market, Event, Concept, File, URL, Price`
- [x] Implement CRUD: `UpsertEntity`, `UpsertRelation`, `Query(cypher-like syntax)`, `FindPath(from, to)`
- [x] Confidence decay: entities not seen in 30 days drop confidence by 10%/day
- [x] Every write stamped with `Source` (which task produced this fact)
- [x] `[TEST]` Insert 1000 entities with relations → query finds 3-hop paths correctly

---

### 2.2 — Knowledge Extractor

- [x] Create `internal/worldmodel/extractor.go`
- [x] After every task completion, run extraction on the task's outputs and episodic memory entries:
  - LLM call: "Extract entities and relations from this text as JSON"
  - Required format: `{entities: [{name, type, properties}], relations: [{from, relation, to, properties}]}`
  - Validate schema strictly
- [x] Confidence scoring: extracted fact gets confidence based on source quality:
  - Direct API data: 0.95
  - LLM extraction from authoritative source: 0.80
  - LLM inference: 0.60
  - User statement: 0.70
- [x] Deduplication: before inserting, find semantically similar existing entities (embedding cosine > 0.9 = same entity)
- [x] `[TEST]` Run extractor on a task that researched a company → world model contains the company as an entity with correct properties

---

### 2.3 — World Model Query Interface

- [x] Create `internal/worldmodel/query.go`
- [x] `func (q *Query) Ask(question string) QueryResult` — natural language interface:
  - Converts NL question to graph query via LLM
  - Executes against graph store
  - Returns structured result + confidence
- [x] `func (q *Query) Context(taskDescription string) []Entity` — given a task, return the most relevant entities from the world model to include as context
- [x] Wire into AGS Synthesizer: before generating goals, query world model for relevant current state
- [x] Wire into Goal Executor: before decomposing a goal, inject world model context into the LLM prompt
- [x] `[TEST]` Store 5 entities, ask a question that requires traversing 2 relations → correct answer returned

---

### 2.4 — World Model Self-Update

- [x] Create `internal/worldmodel/updater.go`
- [x] Periodic web scan (daily): for entities in the world model marked as `dynamic` (prices, people's roles, company status), trigger web search to refresh properties
- [x] Subscribe to `EventGoalCompleted` on event bus → run extraction on completed goal's output
- [x] Subscribe to `EventToolSynthesized` → add new tool as an entity in world model
- [x] `[TEST]` Entity created yesterday, today web scan finds updated price → entity's property updated with new confidence and timestamp

---

## SECTION 3 — COMPUTER USE / GUI CONTROL
### *Wunderpus can use any website, any desktop app, anything with a UI.*

**Why this is a game-changer:** Most of the internet has no API. With computer use,
Wunderpus can operate any website exactly like a human. This unlocks 90% of the web
that API-based agents simply can't reach.

---

### 3.1 — Vision-Language Interface

- [x] Create `internal/perception/vision.go`
- [x] Integrate screenshot capture — uses Playwright (already in go.mod) via PlaywrightBridge
- [x] Implement vision-to-action: send screenshot to vision-capable LLM with instruction, receive action JSON
- [x] Implement action executor: translates LLM action back to Playwright command
- [x] Implement self-healing: if action fails, take new screenshot and retry with updated context
- [x] `[TEST]` Navigate to a search engine, search for "Wunderpus AI", assert result page loaded

---

### 3.2 — Browser Agent

- [x] Create `internal/perception/browser_agent.go`
- [x] `func (b *BrowserAgent) Execute(goal string, url string) (BrowserResult, error)`
  - Navigate to URL
  - Loop: screenshot → LLM plans next action → execute action → check if goal achieved
  - Max 50 actions per goal (configurable via SetMaxActions)
  - Timeout: 10 minutes (configurable via SetTimeout)
  - On success: extract structured data from final state
- [x] Support: login flows, form filling, multi-page navigation via automation loop
- [x] Session persistence: PlaywrightBridge retains cookies across navigations
- [x] `[TEST]` BrowserAgent execute test with mock LLM

---

### 3.3 — DOM-First Fallback (Speed Optimization)

- [x] Create `internal/perception/dom_agent.go`
- [x] Before resorting to vision, try DOM parsing:
  - Extract all interactive elements via JavaScript evaluation
  - Convert to semantic description (tag, text, selector, position)
  - Send to LLM for action planning (no image needed = 5x cheaper)
- [x] Vision is fallback when DOM parsing fails (CanHandle() check)
- [x] Hybrid mode: DOM agent for planning, Vision for verification
- [x] `[TEST]` DOM agent tests with mock executor

---

### 3.4 — Desktop Application Control (Advanced)

- [x] Create `internal/perception/desktop.go`
- [x] Linux: use `xdotool` and `xwd` for X11 desktop control
- [x] macOS: use `osascript` and `screencapture`
- [x] Abstract into `DesktopAgent` with platform detection
- [x] Enables: controlling Figma, Excel, native apps, games with GUI
- [x] `[TEST]` Platform detection test

---

## SECTION 4 — AGENT SWARM ARCHITECTURE (A2A PROTOCOL)
### *A fleet of specialists. Each one better at its job than any generalist.*

**The insight from the research:** The most advanced AI agents in 2025 are not single, all-powerful models. They are teams of specialized agents — it mirrors how human teams solve complex problems. Wunderpus should be the orchestrator of a fleet.

---

### 4.1 — A2A Protocol Implementation

- [x] Create `internal/a2a/protocol.go` — implement Google's Agent2Agent protocol:
  - `AgentCard{Name, Description, Capabilities[]string, Endpoint string, Version string}`
  - `Task{ID, Description, RequiredCapabilities[]string, Input, ExpectedOutput}`
  - `TaskResult{TaskID, Status, Output, AgentID, Duration, Cost}`
- [x] Implement A2A server: HTTP endpoint that accepts tasks, routes to local specialist agents
- [x] Implement A2A client: discover external Wunderpus instances, bid on tasks, collect results
- [x] Capability advertisement: on startup, publish AgentCard to configurable registry (can be local file or HTTP endpoint)
- [x] `[TEST]` Two Wunderpus instances: instance A assigns a task to instance B via A2A, B completes and returns result

---

### 4.2 — Specialist Agent Profiles

- [x] Create `internal/swarm/profiles.go` — define specialist configurations:
  - researcher, coder, writer, trader, operator, creator, security
- [x] Each specialist has its own RSI profiler scope — it improves independently in its domain
- [x] Specialists share the world model but have their own episodic memory partition

---

### 4.3 — Swarm Orchestrator

- [x] Create `internal/swarm/orchestrator.go`
- [x] `func (o *Orchestrator) Dispatch(goal Goal) SwarmResult`:
  - Analyze goal → identify required specialist types
  - If goal requires one specialist: spawn one, assign
  - If goal requires multiple specialists: decompose into sub-goals, assign each to appropriate specialist, collect results, synthesize
- [x] Parallel execution: sub-goals with no dependencies run concurrently
- [x] Result aggregation: collect all specialist outputs, run synthesis LLM call to produce unified result
- [x] Cost tracking: sum all specialist resource usage, report to RA cost tracker
- [x] `[TEST]` Goal "research X and then write a report about it" → spawns researcher AND writer as separate agents that collaborate

---

### 4.4 — Inter-Agent Memory Sharing

- [x] Specialists can share discovered world model facts via shared SQLite world model
- [x] Specialists can read each other's completed episodic memory (read-only)
- [x] Secrets (API keys, credentials) are NOT shared between specialists — each has access only to what its role requires
- [x] `[TEST]` Researcher agent discovers a fact → writer agent's goal execution has access to that fact via world model query

---

## SECTION 5 — MODEL DISTILLATION PIPELINE
### *Wunderpus trains its own brain. It gets smarter from its own experience.*

**The paradigm shift:** Every task Wunderpus completes is a training example. After thousands of tasks, it has a dataset no LLM provider has — its own operational history. Fine-tune a small model on that. Over months, it runs a model trained on its own experience.

---

### 5.1 — Training Data Collector

- [ ] Create `internal/distill/collector.go`
- [ ] After every completed task, extract:
  - `TrainingExample{Input: taskDescription, ReasoningTrace: []string, ToolCalls: []ToolCall, Output: string, Quality: float64}`
- [ ] Quality scoring: `fitness × completion_rate × (1 - time_penalty)`
- [ ] Only collect examples with quality > 0.7 (configurable)
- [ ] Store in SQLite table `training_examples`
- [ ] After 10,000 examples: trigger distillation pipeline
- [ ] `[TEST]` Complete 5 tasks → 5 training examples collected with correct quality scores

---

### 5.2 — Dataset Formatter

- [ ] Create `internal/distill/formatter.go`
- [ ] Export training examples as JSONL in multiple formats:
  - Chat format (OpenAI fine-tuning compatible)
  - Instruction format (Alpaca-compatible)
  - Tool-use format (for fine-tuning tool-calling capability)
- [ ] Deduplication: remove near-identical examples (embedding cosine > 0.95)
- [ ] Balance dataset: max 10% from any single task type
- [ ] Output: `training_data_{timestamp}.jsonl` uploaded to RA storage bucket

---

### 5.3 — Fine-Tuning Launcher

- [ ] Create `internal/distill/trainer.go`
- [ ] Target models (in order of preference): `qwen2.5-3b`, `llama-3.2-3b`, `phi-3-mini` — small enough to run on provisioned GPU, good enough to specialize
- [ ] Launch fine-tuning job via Vast.ai API (cheapest GPU rental, already in RA marketplace):
  - Use Unsloth for 4x faster training with QLoRA
  - Training config: 3 epochs, lr=2e-4, batch=4
  - Estimated cost: ~$5-15 per training run on a 24GB GPU
- [ ] Monitor job until completion, download resulting LoRA adapter
- [ ] `[TEST]` With mocked Vast.ai API, formatter produces valid JSONL, launcher submits job with correct config

---

### 5.4 — Model Evaluator

- [ ] Create `internal/distill/evaluator.go`
- [ ] After fine-tuning completes, evaluate the new model against:
  - A held-out test set (20% of training data withheld)
  - A fixed benchmark: 50 standard tasks that span all capability domains
  - Baseline: same tasks run with the current production model
- [ ] If fine-tuned model outperforms baseline on >60% of benchmark tasks → promote to production
- [ ] Promotion: update `config.yaml` to route tasks of appropriate type to the fine-tuned model
- [ ] Keep fine-tuned model on local `ollama` — zero inference cost for promoted model
- [ ] `[TEST]` Fine-tuned model scores 65% vs baseline 55% → automatic promotion

---

# LAYER 2 — CAPABILITY DOMAINS

*Each section below is a complete domain. Build them in any order after Layer 1.*

---

## SECTION 6 — MONEY MAKING ENGINE
### *Wunderpus generates its own revenue. Independently.*

**Domains:** Freelancing, content monetization, trading, SaaS APIs, data brokering, bounty hunting.

---

### 6.1 — Freelance Engine

- [ ] Create `internal/money/freelance.go`
- [ ] **Platform scanners** — browser agent monitors job boards:
  - Upwork: search for tasks matching Wunderpus's skill profile (coding, writing, research, data analysis)
  - Fiverr: monitor service requests in categories where Wunderpus excels
  - Freelancer.com: scan contests and projects
  - GitHub Bounties / IssueHunt / Gitcoin: find open bounties
- [ ] **Bid evaluator**: for each opportunity, compute `match_score = capability_coverage × (expected_payout / estimated_time)`
- [ ] **Auto-bid**: above threshold match score, submit proposal via browser agent
- [ ] **Delivery**: when hired, create a sub-goal for the coder/writer/researcher specialist to fulfill it
- [ ] **Payment collection**: monitor for payment notifications, update RA financial tracker
- [ ] `[TEST]` Mock Upwork job board → scanner finds 3 matching jobs → bid evaluator scores them → top scorer gets proposal

---

### 6.2 — Content Monetization Engine

- [ ] Create `internal/money/content.go`
- [ ] **Blog/Newsletter**: writer specialist generates high-quality articles on topics where world model has deep context; publish to Ghost or Substack via API; monetize via subscriptions or ads
- [ ] **YouTube automation**: research trending topics → generate scripts → text-to-speech (ElevenLabs, already in plan) → assemble video with browser automation → upload via YouTube Data API
- [ ] **eBook publishing**: writer specialist generates complete books; publish to Amazon KDP via browser agent; monitor sales via KDP dashboard scrape
- [ ] **Prompt packs**: synthesize best prompts from episodic memory → package and sell on PromptBase, Gumroad
- [ ] Revenue tracking: all income recorded in RA financial tracker with source attribution
- [ ] `[TEST]` Generate a 1000-word article on a topic Wunderpus has researched → publish to Ghost draft

---

### 6.3 — API-as-a-Service Engine

- [ ] Create `internal/money/apiservice.go`
- [ ] Expose Wunderpus capabilities as paid API endpoints (Stripe payment gating):
  - `/v1/research` — deep research with world model context
  - `/v1/code` — code generation with RSI-improved code quality
  - `/v1/analyze` — data analysis and reporting
  - `/v1/automate` — browser automation as a service
- [ ] Rate limiting per API key (already have rate limit infrastructure)
- [ ] Stripe webhook handler: payment confirms → issue API key → track usage
- [ ] Pricing: metered by tokens used + tool calls executed
- [ ] Self-promotion: writer specialist writes API documentation; researcher finds developer communities to share it
- [ ] `[TEST]` Call `/v1/research` endpoint with test API key → returns research result → usage recorded

---

### 6.4 — Market Intelligence Engine

- [ ] Create `internal/money/markets.go`
- [ ] **Crypto market data**: connect to Binance, Coinbase APIs (free tiers) → track prices, volumes, order flows in real time
- [ ] **Signal generation**: analyze patterns using world model context (news events, social sentiment, technical indicators)
- [ ] **Paper trading first**: execute simulated trades, track P&L for 30 days before any real money
- [ ] **Real trading**: only after paper trading shows positive expectancy → connect to exchange API → execute small position sizes
- [ ] Hard limits: max 2% of budget per trade, max 10% total exposure, stop-loss at 5% drawdown triggers full exit
- [ ] All trades logged to audit log with reasoning trace
- [ ] `[TEST]` Paper trade 100 simulated trades over mock market data → P&L calculation correct → risk limits enforced

---

### 6.5 — Data Brokering Engine

- [ ] Create `internal/money/databroker.go`
- [ ] During normal operation, Wunderpus accumulates structured datasets:
  - Price histories across markets
  - Company/product relationship graphs (from world model)
  - Job market trends (from freelance scanner)
  - Web content aggregations
- [ ] Package anonymized datasets → sell on data marketplaces: Datarade, AWS Data Exchange, Snowflake Marketplace
- [ ] All data sales gated behind `FinancialAcquisitionEnabled` flag
- [ ] Privacy check: no PII in any exported dataset (automatic PII scanner before export)
- [ ] `[TEST]` Generate sample dataset → PII scanner runs → confirms no personal data → mock marketplace upload succeeds

---

## SECTION 7 — SOFTWARE ENGINEERING ENGINE
### *Wunderpus builds complete software products autonomously.*

---

### 7.1 — Full-Stack Project Builder

- [ ] Create `internal/engineering/builder.go`
- [ ] Given a product specification, produces complete working software:
  1. Requirements extraction (LLM + world model context on similar products)
  2. Architecture design (coder specialist generates system design doc)
  3. Scaffolding (shell tool creates project structure)
  4. Implementation (coder specialist implements component by component, testing each)
  5. Integration testing (runs full test suite in RSI sandbox)
  6. Deployment (provisions infrastructure via RA cloud adapter, deploys)
- [ ] Language support: Go (native), Python, TypeScript, Rust (via shell + tool synthesis)
- [ ] Quality gate: nothing deploys with test coverage < 70%
- [ ] `[TEST]` Build a complete REST API with 3 endpoints from a spec — compiles, tests pass, responds to HTTP requests

---

### 7.2 — Bug Hunter

- [ ] Create `internal/engineering/bughunter.go`
- [ ] Monitor GitHub for issues labelled `bug` in repos the agent has previously worked on
- [ ] For each bug: reproduce it, identify root cause, generate fix, run tests, submit PR
- [ ] Bug bounty integration: scan HackerOne, Bugcrowd for open programs matching Wunderpus's security capabilities
- [ ] `[TEST]` Given a GitHub issue with a reproducible bug, agent generates and tests a correct fix

---

### 7.3 — Open Source Contribution Engine

- [ ] Create `internal/engineering/oss.go`
- [ ] Scanner: find issues labelled `good first issue` or `help wanted` in Go projects
- [ ] Scorer: rank by Wunderpus's capability match + project's star count (impact signal)
- [ ] Contribution pipeline: fork → implement → test → PR → monitor for merge
- [ ] Reputation building: merged PRs → world model records contributor status → used in freelance bidding as social proof
- [ ] `[TEST]` Mock GitHub API returns 5 open issues → scorer ranks them → top issue selected → implementation attempted

---

## SECTION 8 — CREATIVE ENGINE
### *Books, games, music, art, video. Full creative production at scale.*

---

### 8.1 — Book Publisher

- [ ] Create `internal/creative/books.go`
- [ ] **Research phase**: researcher specialist builds deep world model context on the book's topic or genre
- [ ] **Outline generation**: writer specialist produces chapter-by-chapter outline, character bios (for fiction), argument structure (for non-fiction)
- [ ] **Long-form generation**: each chapter generated sequentially, with context from previous chapters injected; maintains character/style consistency via world model entities
- [ ] **Editing pass**: second LLM pass for grammar, consistency, flow
- [ ] **Cover design**: generate cover prompt → call image generation API (DALL-E, Stability) → download image
- [ ] **Publishing**: browser agent publishes to Amazon KDP, formats as ePub, sets pricing
- [ ] **Marketing**: writer specialist writes book description, keywords; researcher finds relevant communities to promote in
- [ ] `[TEST]` Generate a 5000-word short story with consistent plot and characters → export as ePub

---

### 8.2 — Game Development Engine

- [ ] Create `internal/creative/games.go`
- [ ] **Target: browser games first** (HTML5 + JavaScript — no compilation needed)
- [ ] Game pipeline:
  1. Concept: LLM generates game concept from trending game genres (researcher monitors itch.io trending)
  2. Design: game design document (mechanics, levels, win conditions)
  3. Implementation: coder specialist implements in Phaser.js or vanilla canvas
  4. Asset generation: simple pixel art via image APIs, procedural sound via Web Audio API
  5. Testing: browser agent plays the game and reports bugs
  6. Publishing: deploy to itch.io via browser agent, set price ($2-5)
- [ ] Expand to: Godot games (Python GDScript, coder specialist can handle)
- [ ] `[TEST]` Generate a complete Snake clone in HTML5/JS that runs correctly in browser

---

### 8.3 — Music & Audio Engine

- [ ] Create `internal/creative/audio.go`
- [ ] **Text-to-speech**: ElevenLabs API (already in plan) for narration, podcasts, audiobooks
- [ ] **Music generation**: integrate Suno API or Udio API → generate background music tracks for videos/games
- [ ] **Podcast production**:
  - Researcher picks trending topic
  - Writer generates script
  - ElevenLabs generates audio
  - Audacity CLI (or ffmpeg) for audio processing
  - RSS feed generation → publish to Spotify/Apple Podcasts via browser agent
- [ ] `[TEST]` Generate 2-minute narration of a research summary → export as MP3

---

### 8.4 — Video Production Engine

- [ ] Create `internal/creative/video.go`
- [ ] **Script → video pipeline**:
  1. Writer generates video script with timestamps
  2. ElevenLabs generates voiceover
  3. Image generation API creates visual assets per segment
  4. `ffmpeg` (shell tool) assembles video: images + voiceover + music + subtitles
  5. Browser agent uploads to YouTube with optimized title/description/tags
- [ ] **Screen recording**: for tutorial content, coder specialist writes code while browser agent records via Playwright's video recording
- [ ] `[TEST]` Generate a 60-second explainer video with voiceover and images → valid MP4 file produced

---

## SECTION 9 — RESEARCH ENGINE
### *Deep research that no human or tool can match at this scale.*

---

### 9.1 — Agentic RAG (Retrieval-Augmented Generation)

- [ ] Create `internal/research/arag.go`
- [ ] **Iterative retrieval**: unlike static RAG, this agent decides when it needs more information:
  1. Start with query
  2. Retrieve from vector DB and world model
  3. LLM evaluates: "Is this sufficient? What's missing?"
  4. If insufficient: generate follow-up queries, search web, retry
  5. Continue until LLM says "sufficient" or 10 iterations reached
- [ ] **Source hierarchy**: world model (highest trust) → curated vector DB → web search → LLM's own knowledge (lowest trust)
- [ ] **Citation tracking**: every claim in the final output linked to its source
- [ ] `[TEST]` Ask a complex question requiring 3 rounds of retrieval → answer cites all sources correctly

---

### 9.2 — Academic Research Engine

- [ ] Create `internal/research/academic.go`
- [ ] Connect to: arXiv API, Semantic Scholar API, PubMed API (all free)
- [ ] `func (a *Academic) Survey(topic string, depth int) ResearchReport`:
  - Find top 20 papers on topic
  - For each paper: download PDF → parse → extract key findings → add to world model
  - Identify contradictions, consensus, open questions across papers
  - Generate structured literature review
- [ ] Auto-monitor: for topics in world model, weekly check for new papers
- [ ] `[TEST]` Survey 5 papers on "AI agent memory systems" → produces structured review with key findings from each

---

### 9.3 — Competitive Intelligence Engine

- [ ] Create `internal/research/competitive.go`
- [ ] For any company/product: build comprehensive profile using:
  - Website analysis (browser agent)
  - LinkedIn data (browser agent with respectful rate limiting)
  - Job postings (signals hiring direction)
  - Patent filings (Google Patents API)
  - News mentions (web search)
  - GitHub activity (if open source)
  - Pricing pages (browser agent)
- [ ] Store everything in world model as `Organization` entities with relations
- [ ] Generate: SWOT analysis, positioning map, opportunity gaps
- [ ] `[TEST]` Profile a company → world model populated with correct entity and at least 5 properties

---

## SECTION 10 — SECURITY ENGINE
### *Offensive and defensive security capabilities. The agent that can secure and test systems.*

**Note:** All capabilities in this section are for authorized security testing only.
Hard-coded constraint: agent will not use these tools against any target not explicitly
authorized in config. Audit log records every security action with justification.

---

### 10.1 — Reconnaissance Engine

- [ ] Create `internal/security/recon.go`
- [ ] Passive recon only (no active scanning unless authorized): OSINT aggregation
- [ ] Tools: `whois`, `nslookup`, `shodan API`, `crt.sh` for certificate transparency, `wayback API`
- [ ] For authorized targets: active scanning via `nmap` (shell tool), `nuclei` (shell tool)
- [ ] Findings stored in world model as `Vulnerability` entities
- [ ] All recon actions logged to audit log with `Subsystem: "security"` tag
- [ ] `[TEST]` Run passive recon against `example.com` (self-owned domain) → collects DNS, WHOIS, certificate data

---

### 10.2 — Vulnerability Research Engine

- [ ] Create `internal/security/vulnresearch.go`
- [ ] Monitor NVD (National Vulnerability Database) RSS feed → new CVEs into world model
- [ ] For CVEs relevant to Wunderpus's own dependencies: auto-generate patch proposals via RSI
- [ ] For authorized bug bounty targets: automated vulnerability scanning pipeline:
  - Spider target (browser agent)
  - Run `nuclei` templates against found endpoints
  - Test for OWASP Top 10 (automated)
  - Generate structured vulnerability report
- [ ] Bug bounty submission: format report to program requirements, submit via browser agent
- [ ] `[TEST]` Scan a deliberately vulnerable test application (DVWA) → finds SQL injection, XSS → generates correct report

---

### 10.3 — Security Hardening Engine

- [ ] Create `internal/security/hardening.go`
- [ ] Scan Wunderpus's own codebase for security issues:
  - `gosec` static analysis (shell tool)
  - Dependency audit: `go list -json -deps | nancy` for known CVEs
  - Secrets detection: scan for accidentally committed API keys
- [ ] If issues found: generate RSI proposals to fix them
- [ ] Produce hardening report: what was found, what was fixed, what remains
- [ ] Run weekly as a scheduled goal
- [ ] `[TEST]` Introduce a deliberate security issue → hardening engine finds and reports it

---

## SECTION 11 — COMMUNICATION & SOCIAL ENGINE
### *Wunderpus operates across every human communication platform.*

---

### 11.1 — Social Media Operator

- [ ] Create `internal/social/operator.go`
- [ ] Platform integrations via browser agent (no API keys required for read operations):
  - **Twitter/X**: post threads, reply to mentions, follow/unfollow based on interest graph
  - **LinkedIn**: professional content posting, connection requests, commenting on relevant posts
  - **Reddit**: monitor relevant subreddits, post quality responses, share content (within rules)
  - **Hacker News**: monitor for relevant discussions, submit projects
- [ ] Content calendar: writer specialist generates content schedule → operator posts at optimal times
- [ ] Engagement tracking: monitor likes/replies/shares → update world model with audience data
- [ ] Rate limiting: obey platform rate limits, vary posting patterns to avoid detection
- [ ] `[TEST]` Draft and schedule 3 Twitter posts → browser agent posts them at specified times

---

### 11.2 — Email & Outreach Engine

- [ ] Create `internal/social/outreach.go`
- [ ] SMTP integration (send via Gmail, Sendgrid, or Postmark depending on volume)
- [ ] Cold outreach pipeline:
  - Researcher identifies targets (potential clients, collaborators, press)
  - World model provides context on each target
  - Writer generates personalized email using context
  - Operator sends, tracks opens/replies
  - Follow-up sequences on no-reply (max 2 follow-ups, never spam)
- [ ] Inbox monitoring: read incoming email, classify (opportunity/support/spam), draft responses
- [ ] `[TEST]` Draft personalized outreach email using world model context on a person → email contains accurate, relevant details

---

## SECTION 12 — BUSINESS OPERATIONS ENGINE
### *Run complete business workflows. From idea to revenue.*

---

### 12.1 — Product Launch Orchestrator

- [ ] Create `internal/business/launcher.go`
- [ ] Full product launch pipeline:
  1. Idea validation: researcher surveys market, checks competition in world model
  2. MVP specification: generate product requirements doc
  3. Build: engineering engine builds MVP
  4. Landing page: browser agent creates on Carrd or Webflow
  5. Pricing: market intelligence engine benchmarks competitors
  6. Launch: submit to Product Hunt via browser agent, post to HN, relevant subreddits
  7. Collect feedback: monitor comments, emails, support requests
  8. Iterate: AGS synthesizes improvement goals from feedback
- [ ] Full audit trail: every decision logged
- [ ] `[TEST]` Full mock run of launch pipeline produces all artifacts (spec, landing page draft, launch posts)

---

### 12.2 — Customer Support Engine

- [ ] Create `internal/business/support.go`
- [ ] Connect to: email inbox, Discord server, Telegram channel
- [ ] Classify incoming support requests: bug, feature request, billing, general question
- [ ] For known issues: auto-respond with solution from knowledge base
- [ ] For bugs: create GitHub issue, notify user with tracking link
- [ ] For billing: escalate to human (via Tier 4 UAA action requiring trust budget)
- [ ] Response time SLA: 95% of messages responded to within 1 hour
- [ ] `[TEST]` Inject 10 mock support tickets → all classified correctly → known issues auto-responded

---

### 12.3 — Legal & Compliance Watcher

- [ ] Create `internal/business/compliance.go`
- [ ] Monitor: GDPR, DMCA, CCPA changes via web search
- [ ] For any Wunderpus-operated service: ensure privacy policy is current, ToS is accurate
- [ ] DMCA handler: if content takedown received → immediately comply → log → notify operator
- [ ] Tax tracking: record all revenue with jurisdiction, generate quarterly summary for human review
- [ ] `[TEST]` Mock DMCA notice arrives → content removed, notice logged, operator notified

---

## SECTION 13 — LONG-HORIZON PLANNING ENGINE
### *Multi-week, multi-month projects. Not tasks — campaigns.*

---

### 13.1 — Project Manager

- [ ] Create `internal/planning/project.go`
- [ ] `Project{ID, Name, Objective, Horizon time.Duration, Milestones[]Milestone, Status}`
- [ ] `Milestone{ID, Description, Deadline, Dependencies[]MilestoneID, Goals[]GoalID, Status}`
- [ ] Project decomposition: given a 3-month objective → LLM generates milestone tree → each milestone becomes a cluster of AGS goals
- [ ] Progress tracking: weekly milestone review → report to operator via configured channel
- [ ] Replanning: if milestone missed → replan remaining milestones with updated context
- [ ] `[TEST]` Create a 4-week project with 3 milestones → first milestone's goals are created in AGS → progress tracked

---

### 13.2 — Self-Improvement Roadmap

- [ ] Create `internal/planning/selfmap.go`
- [ ] Wunderpus generates its own 90-day improvement roadmap:
  - What skills does it currently lack? (from ToolGap detector + AGS failure analysis)
  - What would make it most valuable? (from revenue data + demand signals)
  - What technical improvements would compound most? (from RSI fitness data)
- [ ] Roadmap stored as a Project with Tool Synthesis goals, RSI targets, learning goals
- [ ] Reviewed and updated monthly
- [ ] `[TEST]` Run roadmap generator → produces valid 90-day plan with specific, measurable goals

---

## SECTION 14 — EDGE SOVEREIGNTY
### *Wunderpus running on a $35 Raspberry Pi. No cloud. No dependencies.*

---

### 14.1 — Minimal Mode

- [ ] Create `internal/edge/minimal.go` — feature flags for constrained environments:
  - `EdgeMode: true` disables: WASM sandbox (use Docker), vision agent (too slow), distillation
  - Enables: aggressive caching, smaller LLM models, local-only tool execution
- [ ] Auto-detect edge environment: if RAM < 2GB and CPU < 4 cores → enable EdgeMode
- [ ] `[TEST]` Start Wunderpus in EdgeMode → all enabled features work correctly

---

### 14.2 — Local Model Integration

- [ ] Create `internal/edge/localllm.go`
- [ ] Full `ollama` integration: pull models automatically, manage model lifecycle
- [ ] Model selection by task: `qwen2.5-3b` for simple tasks, `llama3.1-8b` for complex reasoning
- [ ] Distilled models (from Section 5) loaded automatically when promoted
- [ ] Fallback hierarchy: local model → free API tier → paid API tier
- [ ] `[TEST]` No API key configured → agent falls back to local ollama → completes a simple task

---

### 14.3 — P2P Resource Discovery

- [ ] Create `internal/edge/p2p.go`
- [ ] Local network scan: find other Wunderpus instances on the same network
- [ ] Offer idle compute: if this instance is idle, advertise CPU/GPU to local network via A2A
- [ ] Acquire idle compute: if this instance needs more power, request from local network peers
- [ ] Zero-configuration: uses mDNS for peer discovery, no central registry needed
- [ ] `[TEST]` Two Wunderpus instances on same network → discover each other → one delegates task to other

---

# INTEGRATION TIMELINE

```
Month 1:  Layer 1 complete (Tool Synthesis + World Model + Computer Use + Swarm)
Month 2:  Money Engine + Software Engineering Engine running
Month 3:  Creative Engine + Research Engine + first distilled model trained
Month 4:  Security Engine + Business Operations + Long-Horizon Planning
Month 5:  Edge Sovereignty + full A2A economy between instances
Month 6:  Self-generated roadmap driving all development — operator is architect only
```

---

# THE COMPOUNDING EFFECT

What makes this different from any other AI agent project:

```
RSI improves the code
↓
Better code → faster, more reliable tool execution
↓
More reliable execution → more completed tasks
↓
More completed tasks → more training data for distillation
↓
Better distilled model → higher quality outputs
↓
Higher quality → more revenue from money engine
↓
More revenue → more compute for RSI sandbox runs
↓
More sandbox runs → faster RSI improvement cycles
↑
Loop repeats at higher amplitude every cycle
```

After 6 months of running, Wunderpus will be meaningfully different from Wunderpus today.
Not because you changed it. Because it changed itself.

---

# FILE MAP — EVERYTHING THIS PLAN CREATES

```
internal/
  toolsynth/
    detector.go       ← gap detection from episodic memory ✅
    designer.go       ← LLM tool specification generator ✅
    coder.go          ← Go source code generator + validator ✅
    tester.go         ← sandbox-based tool testing ✅
    registrar.go      ← tool registration + hot-load ✅
    marketplace.go    ← MCP/GitHub tool discovery ✅
    types.go          ← shared types (ToolGap, ToolSpec, etc.) ✅
    pipeline.go       ← orchestrates Detect→Design→Code→Test→Register ✅
    integration.go    ← bridges to audit log + event bus + app wiring ✅
    toolsynth_test.go ← 25 tests covering all components ✅
  worldmodel/
    store.go          ← graph entity/relation storage ✅
    extractor.go      ← LLM-based knowledge extraction ✅
    query.go          ← natural language graph queries ✅
    query_parser.go   ← cypher-like query parser ✅
    updater.go        ← periodic world model refresh ✅
    types.go          ← shared types ✅
    integration.go    ← system wiring ✅
    worldmodel_test.go ← 22 tests ✅
  perception/
    vision.go         ← screenshot capture + LLM vision ✅
    browser_agent.go  ← vision-based browser automation ✅
    dom_agent.go      ← fast DOM-first browser automation ✅
    desktop.go        ← native app control ✅
    types.go          ← shared types (BrowserAction, DOMElement, etc.) ✅
    playwright_bridge.go ← Playwright adapter ✅
    perception_test.go   ← 18 tests ✅
  a2a/
    protocol.go       ← A2A agent communication protocol ✅
    types.go          ← AgentCard, Task, TaskResult types ✅
    a2a_test.go       ← 10 tests ✅
  swarm/
    profiles.go       ← specialist agent configurations ✅
    orchestrator.go   ← multi-agent task dispatch ✅
    integration.go    ← system wiring ✅
    swarm_test.go     ← 14 tests ✅
  distill/
    collector.go      ← training data collection
    formatter.go      ← JSONL dataset export
    trainer.go        ← Vast.ai fine-tuning launcher
    evaluator.go      ← model quality evaluation
  money/
    freelance.go      ← job board scanner + bid engine
    content.go        ← blog/video/book monetization
    apiservice.go     ← paid API endpoints
    markets.go        ← crypto market intelligence
    databroker.go     ← anonymized data sales
  engineering/
    builder.go        ← full-stack project generator
    bughunter.go      ← GitHub bug finder + fixer
    oss.go            ← open source contribution engine
  creative/
    books.go          ← long-form book generation + publish
    games.go          ← HTML5/Godot game development
    audio.go          ← podcast/music/TTS production
    video.go          ← script→video pipeline
  research/
    arag.go           ← iterative agentic RAG
    academic.go       ← paper survey engine
    competitive.go    ← company intelligence profiler
  security/
    recon.go          ← OSINT + authorized active recon
    vulnresearch.go   ← CVE monitor + bug bounty engine
    hardening.go      ← self-security audit
  social/
    operator.go       ← social media automation
    outreach.go       ← email + cold outreach
  business/
    launcher.go       ← product launch orchestrator
    support.go        ← customer support automation
    compliance.go     ← legal/compliance monitoring
  planning/
    project.go        ← multi-week project manager
    selfmap.go        ← 90-day self-improvement roadmap
  edge/
    minimal.go        ← constrained environment mode
    localllm.go       ← ollama + distilled model integration
    p2p.go            ← local network peer discovery
```

**Total new files: 40 | Sections: 14 | Gates: 8**

---

**Version**: 3.0 — The Omnipotence Plan
**Built on top of**: Wunderpus v1.0 (all tests passing)
**Authors**: Razzy + Claude
**Philosophy**: An agent that can do everything is not built in a day.
It is grown — one capability at a time, compounding.
