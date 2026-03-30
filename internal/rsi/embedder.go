package rsi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"

	_ "modernc.org/sqlite"
)

// Embedder is the interface for generating text embeddings.
// This matches the provider.Embedder interface.
type Embedder interface {
	EmbedSingle(ctx context.Context, text string) ([]float64, error)
	Dimension() int
}

// VectorStore stores and retrieves function embeddings with metadata.
// Optionally backed by SQLite for persistence across restarts.
type VectorStore struct {
	vectors map[string]*VectorEntry // keyed by QualifiedName
	mu      sync.RWMutex
	db      *sql.DB // optional SQLite persistence
}

// VectorEntry holds an embedding vector and associated metadata.
type VectorEntry struct {
	FunctionName string
	File         string
	Package      string
	Complexity   int
	Embedding    []float64
}

// NewVectorStore creates a new in-memory vector store.
func NewVectorStore() *VectorStore {
	return &VectorStore{
		vectors: make(map[string]*VectorEntry),
	}
}

// NewVectorStoreWithDB creates a vector store backed by SQLite.
// Loads existing embeddings on startup.
func NewVectorStoreWithDB(dbPath string) (*VectorStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("vector store: opening db: %w", err)
	}

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vector_entries (
			function_name TEXT PRIMARY KEY,
			file          TEXT NOT NULL,
			package       TEXT NOT NULL,
			complexity    INTEGER NOT NULL DEFAULT 0,
			embedding     TEXT NOT NULL DEFAULT '[]'
		);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("vector store: creating schema: %w", err)
	}

	vs := &VectorStore{
		vectors: make(map[string]*VectorEntry),
		db:      db,
	}

	// Load existing entries
	if err := vs.loadFromDB(); err != nil {
		slog.Warn("vector store: failed to load existing entries", "error", err)
	}

	return vs, nil
}

// loadFromDB loads all vector entries from SQLite into memory.
func (vs *VectorStore) loadFromDB() error {
	if vs.db == nil {
		return nil
	}

	rows, err := vs.db.Query(`SELECT function_name, file, package, complexity, embedding FROM vector_entries`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var entry VectorEntry
		var embeddingJSON string
		if err := rows.Scan(&entry.FunctionName, &entry.File, &entry.Package, &entry.Complexity, &embeddingJSON); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(embeddingJSON), &entry.Embedding); err != nil {
			slog.Warn("vector store: corrupt embedding for " + entry.FunctionName)
			continue
		}
		vs.vectors[entry.FunctionName] = &entry
	}

	slog.Info("vector store: loaded embeddings from DB", "count", len(vs.vectors))
	return rows.Err()
}

// Store saves a function embedding. Persists to SQLite if backed by a DB.
func (vs *VectorStore) Store(entry VectorEntry) {
	vs.mu.Lock()
	vs.vectors[entry.FunctionName] = &entry
	vs.mu.Unlock()

	// Persist to SQLite if available
	if vs.db != nil {
		embeddingJSON, _ := json.Marshal(entry.Embedding)
		_, _ = vs.db.Exec(`
			INSERT OR REPLACE INTO vector_entries (function_name, file, package, complexity, embedding)
			VALUES (?, ?, ?, ?, ?)`,
			entry.FunctionName, entry.File, entry.Package, entry.Complexity, string(embeddingJSON),
		)
	}
}

// Get retrieves a vector entry by function name.
func (vs *VectorStore) Get(functionName string) (*VectorEntry, bool) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	e, ok := vs.vectors[functionName]
	return e, ok
}

// All returns all stored vector entries.
func (vs *VectorStore) All() []*VectorEntry {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	result := make([]*VectorEntry, 0, len(vs.vectors))
	for _, e := range vs.vectors {
		result = append(result, e)
	}
	return result
}

// Count returns the number of stored embeddings.
func (vs *VectorStore) Count() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return len(vs.vectors)
}

// Close shuts down the vector store and its database connection.
func (vs *VectorStore) Close() error {
	if vs.db != nil {
		return vs.db.Close()
	}
	return nil
}

// CodeEmbedder generates and manages vector embeddings for Go source code functions.
type CodeEmbedder struct {
	embedder    Embedder
	vectorStore *VectorStore
	chunkTokens int // max tokens per chunk before embedding
}

// NewCodeEmbedder creates a code embedder using the given embedding provider.
func NewCodeEmbedder(embedder Embedder, store *VectorStore) *CodeEmbedder {
	return &CodeEmbedder{
		embedder:    embedder,
		vectorStore: store,
		chunkTokens: 500,
	}
}

// EmbedFunction generates an embedding for a single function.
// Large functions are chunked if they exceed chunkTokens.
func (e *CodeEmbedder) EmbedFunction(ctx context.Context, fn FunctionNode) ([]float64, error) {
	text := fn.SourceCode
	if text == "" {
		text = fmt.Sprintf("func %s in %s", fn.Name, fn.File)
	}

	// Simple token estimate: ~4 chars per token
	estimatedTokens := len(text) / 4
	if estimatedTokens > e.chunkTokens {
		// Chunk: embed first chunk only (most representative)
		chunkSize := e.chunkTokens * 4
		if chunkSize > len(text) {
			chunkSize = len(text)
		}
		text = text[:chunkSize]
	}

	vec, err := e.embedder.EmbedSingle(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("rsi embedder: embedding %s: %w", fn.QualifiedName, err)
	}

	// Store in vector store
	e.vectorStore.Store(VectorEntry{
		FunctionName: fn.QualifiedName,
		File:         fn.File,
		Package:      fn.Package,
		Complexity:   fn.CyclomaticComp,
		Embedding:    vec,
	})

	return vec, nil
}

// EmbedAll generates embeddings for all functions in a CodeMap.
// Uses incremental updates: only re-embeds functions that changed.
func (e *CodeEmbedder) EmbedAll(ctx context.Context, codeMap *CodeMap, prevMap *CodeMap) (int, error) {
	var toEmbed []FunctionNode

	if prevMap != nil {
		// Incremental: only embed changed or new functions
		mapper := NewCodeMapper(false)
		changed := mapper.Diff(prevMap, codeMap)
		for _, cf := range changed {
			toEmbed = append(toEmbed, cf.FunctionNode)
		}
	} else {
		// Full embed
		for _, fn := range codeMap.Functions {
			toEmbed = append(toEmbed, *fn)
		}
	}

	embedded := 0
	for _, fn := range toEmbed {
		_, err := e.EmbedFunction(ctx, fn)
		if err != nil {
			slog.Warn("rsi embedder: failed to embed function",
				"function", fn.QualifiedName,
				"error", err,
			)
			continue
		}
		embedded++
	}

	return embedded, nil
}

// FindSimilar returns the topK most similar functions to the given query text.
func (e *CodeEmbedder) FindSimilar(ctx context.Context, query string, topK int) ([]FunctionNode, error) {
	queryVec, err := e.embedder.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("rsi embedder: embedding query: %w", err)
	}

	entries := e.vectorStore.All()
	type scored struct {
		entry *VectorEntry
		score float64
	}

	scores := make([]scored, 0, len(entries))
	for _, entry := range entries {
		if len(entry.Embedding) == 0 {
			continue
		}
		sim := cosineSimilarity(queryVec, entry.Embedding)
		scores = append(scores, scored{entry: entry, score: sim})
	}

	// Sort by similarity descending
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	limit := topK
	if limit > len(scores) {
		limit = len(scores)
	}

	var result []FunctionNode
	for i := 0; i < limit; i++ {
		entry := scores[i].entry
		result = append(result, FunctionNode{
			Name:           entry.FunctionName,
			QualifiedName:  entry.FunctionName,
			File:           entry.File,
			Package:        entry.Package,
			CyclomaticComp: entry.Complexity,
		})
	}

	return result, nil
}

// CosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
