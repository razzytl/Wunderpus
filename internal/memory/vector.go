package memory

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wunderpus/wunderpus/internal/provider"
)

// SOP represents a Standard Operating Procedure - a learned workflow.
type SOP struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Embedding    []float64 `json:"-"` // Not stored in DB, computed on demand
	CreatedAt    time.Time `json:"created_at"`
	SuccessCount int       `json:"success_count"`
}

// VectorStore handles vector embeddings and similarity search.
type VectorStore struct {
	mu        sync.RWMutex
	vectors   map[string][]float64 // id -> embedding
	documents map[string]*SOP      // id -> SOP document
	dimension int
}

// NewVectorStore creates a new in-memory vector store.
func NewVectorStore(dimension int) *VectorStore {
	return &VectorStore{
		vectors:   make(map[string][]float64),
		documents: make(map[string]*SOP),
		dimension: dimension,
	}
}

// Add adds a document with its embedding to the store.
func (v *VectorStore) Add(id string, embedding []float64, doc *SOP) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(embedding) != v.dimension {
		return // Skip invalid dimensions
	}

	v.vectors[id] = embedding
	v.documents[id] = doc
}

// Search finds the top-k most similar documents using cosine similarity.
// Returns both the SOPs and their similarity scores.
func (v *VectorStore) Search(query []float64, topK int) []*SOP {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(query) != v.dimension || len(v.vectors) == 0 {
		return nil
	}

	// Calculate similarities
	type scoredDoc struct {
		id    string
		score float64
		doc   *SOP
	}

	results := make([]scoredDoc, 0, len(v.vectors))
	for id, vec := range v.vectors {
		sim := cosineSimilarity(query, vec)
		results = append(results, scoredDoc{
			id:    id,
			score: sim,
			doc:   v.documents[id],
		})
	}

	// Sort by similarity (descending) using efficient sort.Slice
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Return top k
	if topK > len(results) {
		topK = len(results)
	}
	ret := make([]*SOP, topK)
	for i := 0; i < topK; i++ {
		ret[i] = results[i].doc
	}
	return ret
}

// SearchWithScores finds the top-k most similar documents with their scores.
func (v *VectorStore) SearchWithScores(query []float64, topK int) []SearchResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(query) != v.dimension || len(v.vectors) == 0 {
		return nil
	}

	type scoredDoc struct {
		id    string
		score float64
		doc   *SOP
	}

	results := make([]scoredDoc, 0, len(v.vectors))
	for id, vec := range v.vectors {
		sim := cosineSimilarity(query, vec)
		results = append(results, scoredDoc{
			id:    id,
			score: sim,
			doc:   v.documents[id],
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if topK > len(results) {
		topK = len(results)
	}

	ret := make([]SearchResult, topK)
	for i := 0; i < topK; i++ {
		ret[i] = SearchResult{
			SessionID: results[i].doc.ID,
			Content:   results[i].doc.Content,
			Score:     results[i].score,
			SOP:       results[i].doc,
		}
	}
	return ret
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	var dotProduct, normA, normB float64

	for i := 0; i < len(a) && i < len(b); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// SearchResult represents a match in memory.
type SearchResult struct {
	SessionID string
	Content   string
	Score     float64
	SOP       *SOP // RAG result
}

// EnhancedStore extends the base Store with vector search capabilities.
type EnhancedStore struct {
	*Store
	vectorStore *VectorStore
	embedder    provider.Embedder
}

// NewEnhancedStore creates a new store with vector search capabilities using the shared core DB connection.
func NewEnhancedStore(db *sql.DB, embedder provider.Embedder) (*EnhancedStore, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, err
	}

	// Add SOP table if not present
	if err := AddSOPTable(db); err != nil {
		return nil, fmt.Errorf("memory: adding sop table: %w", err)
	}

	dimension := 1536 // Default for OpenAI embeddings
	if embedder != nil {
		dimension = embedder.Dimension()
	}

	return &EnhancedStore{
		Store:       store,
		vectorStore: NewVectorStore(dimension),
		embedder:    embedder,
	}, nil
}

// StoreSOP saves a Standard Operating Procedure to memory.
func (s *EnhancedStore) StoreSOP(ctx context.Context, title, content string) (*SOP, error) {
	sop := &SOP{
		ID:           uuid.New().String(),
		Title:        title,
		Content:      content,
		CreatedAt:    time.Now(),
		SuccessCount: 1,
	}

	// Generate embedding if embedder is available
	if s.embedder != nil && len(content) > 0 {
		emb, err := s.embedder.EmbedSingle(ctx, content)
		if err == nil {
			sop.Embedding = emb
			s.vectorStore.Add(sop.ID, emb, sop)
		}
	}

	// Also save to SQLite for persistence
	_, err := s.Store.db.Exec(`
		INSERT INTO mem_sops (id, title, content, created_at, success_count)
		VALUES (?, ?, ?, ?, ?)`,
		sop.ID, sop.Title, sop.Content, sop.CreatedAt.Format(time.RFC3339), sop.SuccessCount)

	return sop, err
}

// SearchMemories performs vector-based semantic search over SOPs.
func (s *EnhancedStore) SearchMemories(ctx context.Context, query string, encKey []byte) ([]SearchResult, error) {
	results := make([]SearchResult, 0)

	// If we have an embedder, do vector search with actual scores
	if s.embedder != nil && len(query) > 0 {
		queryEmbedding, err := s.embedder.EmbedSingle(ctx, query)
		if err == nil {
			// Use SearchWithScores to get actual similarity scores
			scoredResults := s.vectorStore.SearchWithScores(queryEmbedding, 5)
			results = append(results, scoredResults...)
		}
	}

	// Also do keyword fallback (with lower priority)
	rows, err := s.Store.db.QueryContext(ctx, `
		SELECT session_id, content, encrypted 
		FROM mem_messages 
		WHERE content LIKE ? 
		LIMIT 10`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sID, content string
		var encrypted int
		if err := rows.Scan(&sID, &content, &encrypted); err != nil {
			continue
		}

		if encrypted == 1 && len(encKey) > 0 {
			// Decrypt for preview if possible
		}

		// Only add if not already found via vector search
		exists := false
		for _, r := range results {
			if r.SessionID == sID && r.Content == content {
				exists = true
				break
			}
		}
		if !exists {
			results = append(results, SearchResult{
				SessionID: sID,
				Content:   content,
				Score:     0.5, // Lower priority than vector results
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// GetRelevantSOPs retrieves SOPs relevant to the current task using RAG.
func (s *EnhancedStore) GetRelevantSOPs(ctx context.Context, taskDescription string, topK int) ([]string, error) {
	if s.embedder == nil || len(taskDescription) == 0 {
		return nil, nil // No embedder available
	}

	queryEmbedding, err := s.embedder.EmbedSingle(ctx, taskDescription)
	if err != nil {
		return nil, err
	}

	sops := s.vectorStore.Search(queryEmbedding, topK)
	if len(sops) == 0 {
		return nil, nil
	}

	relevant := make([]string, len(sops))
	for i, sop := range sops {
		relevant[i] = fmt.Sprintf("SOP: %s\n%s", sop.Title, sop.Content)
	}

	return relevant, nil
}

// RecordSuccess increments the success count for an SOP.
func (s *EnhancedStore) RecordSuccess(sopID string) error {
	_, err := s.Store.db.Exec(`
		UPDATE mem_sops SET success_count = success_count + 1 WHERE id = ?`, sopID)
	return err
}

// AddSOPTable adds the SOP table to an existing database.
// Tables are namespaced with mem_ to prevent collisions.
func AddSOPTable(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS mem_sops (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TEXT NOT NULL,
		success_count INTEGER DEFAULT 1
	);
	
	CREATE INDEX IF NOT EXISTS idx_mem_sops_title ON mem_sops(title);
	`
	_, err := db.Exec(schema)
	return err
}
