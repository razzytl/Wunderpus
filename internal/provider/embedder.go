package provider

import "context"

// Embedding represents a vector embedding.
type Embedding []float64

// Embedder is the interface for generating text embeddings.
type Embedder interface {
	// Embed generates embeddings for the given texts.
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	// EmbedSingle generates embedding for a single text.
	EmbedSingle(ctx context.Context, text string) ([]float64, error)
	// Dimension returns the embedding dimension.
	Dimension() int
}

// EmbeddingProvider adds embedding capabilities to a Provider.
type EmbeddingProvider interface {
	// Embed generates embeddings using the provider's embedding model.
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}
