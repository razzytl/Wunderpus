package rsi

import (
	"context"
	"math"
	"testing"
)

// mockEmbedder implements the Embedder interface for testing.
type mockEmbedder struct {
	dim int
}

func (m *mockEmbedder) EmbedSingle(ctx context.Context, text string) ([]float64, error) {
	// Generate a deterministic "embedding" based on text length
	vec := make([]float64, m.dim)
	for i := range vec {
		vec[i] = float64(len(text)+i) / 100.0
	}
	return vec, nil
}

func (m *mockEmbedder) Dimension() int {
	return m.dim
}

func TestCodeEmbedder_EmbedFunction(t *testing.T) {
	store := NewVectorStore()
	emb := &mockEmbedder{dim: 8}
	embedder := NewCodeEmbedder(emb, store)

	fn := FunctionNode{
		Name:           "TestFunc",
		QualifiedName:  "pkg.TestFunc",
		File:           "test.go",
		Package:        "pkg",
		CyclomaticComp: 5,
		SourceCode:     "func TestFunc() { return }",
	}

	vec, err := embedder.EmbedFunction(context.Background(), fn)
	if err != nil {
		t.Fatalf("EmbedFunction: %v", err)
	}

	if len(vec) != 8 {
		t.Fatalf("expected 8-dim vector, got %d", len(vec))
	}

	// Should be stored in vector store
	entry, ok := store.Get("pkg.TestFunc")
	if !ok {
		t.Fatal("function not stored in vector store")
	}
	if entry.Complexity != 5 {
		t.Fatalf("expected complexity 5, got %d", entry.Complexity)
	}
}

func TestCosineSimilarity(t *testing.T) {
	// Identical vectors → similarity = 1.0
	a := []float64{1.0, 2.0, 3.0}
	sim := cosineSimilarity(a, a)
	if math.Abs(sim-1.0) > 0.001 {
		t.Fatalf("identical vectors should have similarity 1.0, got %f", sim)
	}

	// Orthogonal vectors → similarity ≈ 0
	b := []float64{1.0, 0.0}
	c := []float64{0.0, 1.0}
	sim = cosineSimilarity(b, c)
	if math.Abs(sim) > 0.001 {
		t.Fatalf("orthogonal vectors should have similarity ~0, got %f", sim)
	}

	// Similar vectors → high similarity
	d := []float64{1.0, 2.0, 3.0}
	e := []float64{1.1, 2.1, 3.1}
	sim = cosineSimilarity(d, e)
	if sim < 0.8 {
		t.Fatalf("similar vectors should have similarity > 0.8, got %f", sim)
	}

	// Dissimilar vectors → low similarity
	f := []float64{1.0, 0.0, 0.0}
	g := []float64{0.0, 0.0, 1.0}
	sim = cosineSimilarity(f, g)
	if sim > 0.1 {
		t.Fatalf("dissimilar vectors should have similarity < 0.1, got %f", sim)
	}
}

func TestCodeEmbedder_FindSimilar(t *testing.T) {
	store := NewVectorStore()
	emb := &mockEmbedder{dim: 4}
	embedder := NewCodeEmbedder(emb, store)

	// Embed two functions
	fn1 := FunctionNode{
		Name:          "Short",
		QualifiedName: "pkg.Short",
		SourceCode:    "ab", // very short
	}
	fn2 := FunctionNode{
		Name:          "Long",
		QualifiedName: "pkg.Long",
		SourceCode:    "abcdefghijklmnop", // very long
	}

	embedder.EmbedFunction(context.Background(), fn1)
	embedder.EmbedFunction(context.Background(), fn2)

	results, err := embedder.FindSimilar(context.Background(), "short text", 2)
	if err != nil {
		t.Fatalf("FindSimilar: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}
