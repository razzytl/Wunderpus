# Memory & Knowledge System

Wunderpus implements a multi-tier memory architecture combining in-memory context, persistent SQLite storage, vector-based retrieval, and a knowledge graph.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MEMORY SYSTEM                              │
├──────────────┬──────────────┬────────────────────────────────┤
│  Tier 1      │  Tier 2      │  Tier 3                        │
│  In-Memory   │  Persistent  │  Vector (RAG)                  │
│  Context     │  Storage     │                                │
│              │              │                                │
│ • tiktoken   │ • SQLite     │ • Embeddings via provider      │
│ • Truncation │ • AES-256    │ • SOP retrieval                │
│ • Summarize  │ • Sessions   │ • Hybrid fallback (SQL search) │
│ • Branching  │ • Branches   │                                │
└──────────────┴──────────────┴────────────────────────────────┘
```

## Tier 1: In-Memory Context

### Token Management

Uses tiktoken (cl100k_base encoding) for accurate token counting:

```yaml
agent:
  max_context_tokens: 8000  # Maximum tokens in conversation
```

### Context Lifecycle

| Event | Action |
|---|---|
| New message | Add to context, count tokens |
| > 80% capacity | Trigger LLM summarization |
| > 100% capacity | Truncate oldest (keep min 2 messages) |
| Session end | Persist to SQLite |

### Summarization

When context reaches 80% capacity:
1. Oldest messages selected for summarization
2. LLM generates concise summary
3. Original messages replaced with summary
4. Token count recalculated

## Tier 2: Persistent Storage

### SQLite Backend

All sessions persisted in SQLite with WAL mode. Tables are namespaced with `mem_` prefix in the shared `wunderpus.db`:

```sql
CREATE TABLE mem_sessions (
    id TEXT PRIMARY KEY,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT 'New Conversation'
);

CREATE TABLE mem_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    tool_call_id TEXT DEFAULT '',
    tool_calls TEXT DEFAULT '',
    timestamp TEXT NOT NULL,
    encrypted INTEGER DEFAULT 0,
    parent_message_id INTEGER DEFAULT NULL,
    branch_id TEXT DEFAULT 'main',
    FOREIGN KEY (session_id) REFERENCES mem_sessions(id) ON DELETE CASCADE
);
```

### Encryption

Messages encrypted with AES-256-GCM before storage:

```yaml
security:
  encryption:
    enabled: true
    key: "<base64-encoded-32-byte-key>"
```

### Conversation Branching

Messages support `parent_message_id` and `branch_id` for branching:

```go
// Create a new branch from message N
branchID, err := store.CreateBranch(sessionID, messageID)

// Load messages from a specific branch
messages, err := store.LoadSessionBranch(sessionID, branchID, encKey)

// Get all branches for a session
branches, err := store.GetBranches(sessionID)
```

## Tier 3: Vector Search (RAG)

### Enhanced Store

Extends basic storage with Retrieval-Augmented Generation:

```go
enhanced := memory.NewEnhancedStore(baseStore, embedder)
```

The embedder is auto-selected from the provider router — the first provider that supports embeddings is used.

### SOPs (Standard Operating Procedures)

Learned workflows stored as vectors:

```go
// Store a new SOP
enhanced.StoreSOP(ctx, "API Design", "Steps: 1. Define schema...")

// Retrieve relevant SOPs for a task
sops := enhanced.GetRelevantSOPs(ctx, "Build a REST API", 3)
```

### Retrieval Flow

```
Task: "Build a REST API"
    │
    ▼
1. Embed query via LLM embedder
    │
    ▼
2. Cosine similarity search over SOP vectors
    │
    ▼
3. Return top-k most relevant SOPs
    │
    ▼
4. Inject into system prompt
    │
    ▼
5. LLM generates response with SOP context
```

### Hybrid Fallback

If embedder unavailable:
- SQL keyword search as fallback
- Combines vector + keyword results
- Ensures retrieval even during provider outages

## World Model (Knowledge Graph)

A persistent knowledge graph that learns from every interaction. Tables are namespaced with `wm_` prefix.

### Entity Types

| Type | Description |
|---|---|
| Person | Individuals mentioned in conversations |
| Organization | Companies, projects, teams |
| Product | Software, tools, services |
| API | API endpoints and services |
| Technology | Frameworks, languages, libraries |
| Concept | Abstract concepts |
| File | Referenced files |
| URL | Web resources |
| Tool | Agent tools |

### Confidence Scoring

Each entity/relation has a confidence score based on source:

| Source | Confidence |
|---|---|
| Direct API call | 0.95 |
| LLM (authoritative) | 0.80 |
| User statement | 0.70 |
| LLM inference | 0.60 |

### Confidence Decay

Entities not observed for 30 days lose 10% confidence per day.

### Queries

Cypher-like query syntax:

```
// Find path between entities
FIND PATH FROM "OpenAI" TO "Sam Altman" MAX 3 HOPS

// Query entities by type
MATCH (n:Technology) WHERE n.confidence > 0.7
```

## Persistence Model

Wunderpus uses exactly 2 SQLite databases:

| Database | Purpose | Namespaces |
|---|---|---|
| `wunderpus.db` | Core data | `mem_` (memory), `wm_` (world model), `cost_` (cost tracking), `task_checkpoints` |
| `wunderpus-audit.db` | Tamper-evident audit log | `audit_log` (hash-chained entries) |
