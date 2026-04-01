# Memory & Knowledge System

Wunderpus implements a three-tier memory architecture combining in-memory context, persistent storage, and vector-based retrieval.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MEMORY SYSTEM                              │
├──────────────┬──────────────┬────────────────────────────────┤
│  Tier 1      │  Tier 2      │  Tier 3                        │
│  In-Memory   │  Persistent  │  Vector (RAG)                  │
│  Context     │  Storage     │                                │
│              │              │                                │
│ • tiktoken   │ • SQLite     │ • Cosine similarity            │
│ • Truncation │ • AES-256    │ • SOP embeddings               │
│ • Summarize  │ • Sessions   │ • Hybrid fallback              │
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

All sessions persisted in SQLite with WAL mode:

```yaml
agent:
  memory_db_path: "wonderpus_memory.db"
```

### Schema

| Table | Purpose |
|---|---|
| `sessions` | Session metadata (ID, created, provider) |
| `messages` | Individual messages (role, content, tokens) |
| `preferences` | User preferences per session |

### Encryption

Messages encrypted with AES-256-GCM before storage:

```go
// Encryption key derived from config
encrypted := encryption.Encrypt(message, key)
store.SaveMessage(sessionID, encrypted)
```

## Tier 3: Vector Search (RAG)

### Enhanced Store

Extends basic storage with Retrieval-Augmented Generation:

```go
enhanced := memory.NewEnhancedStore(baseStore, vectorStore, embedder)
```

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

A persistent knowledge graph that learns from every interaction.

### Entity Types

| Type | Description |
|---|---|
| Person | Individuals mentioned in conversations |
| Organization | Companies, projects, teams |
| Product | Software, tools, services |
| API | API endpoints and services |
| Technology | Frameworks, languages, libraries |
| Market | Markets, sectors, industries |
| Event | Notable events |
| Concept | Abstract concepts |
| File | Referenced files |
| URL | Web resources |
| Price | Pricing information |
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

```
Entities not observed for 30 days: -10% confidence/day
```

### Queries

Cypher-like query syntax:

```
// Find path between entities
FIND PATH FROM "OpenAI" TO "Sam Altman" MAX 3 HOPS

// Query entities by type
MATCH (n:Technology) WHERE n.confidence > 0.7
```

## Memory Configuration

```yaml
agent:
  memory_db_path: "wonderpus_memory.db"
  max_context_tokens: 8000

security:
  audit_db_path: "wonderpus_audit.db"
```

## Persistence Model Summary

| Database | Purpose | Encryption |
|---|---|---|
| `wonderpus_memory.db` | Sessions, messages, preferences | AES-256-GCM |
| `wonderpus_audit.db` | Tamper-evident audit log | Optional |
| `wonderpus_worldmodel.db` | Knowledge graph | No |
| `wonderpus_cost.db` | Cost tracking | No |
| `wunderpus_profiler.db` | RSI metrics | No |
| `wunderpus_trust.db` | Trust budget | No |
| `wunderpus_resources.db` | Provisioned resources | AES-256-GCM |
