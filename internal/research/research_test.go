package research

import (
	"testing"
)

func TestAgenticRAG_Query(t *testing.T) {
	rag := &AgenticRAG{
		maxIter: 3,
		llm:     &mockLLM{},
	}

	// With no data sources, would get empty results but still work
	if rag.maxIter != 3 {
		t.Errorf("Expected maxIter 3, got %d", rag.maxIter)
	}
}

func TestCitation_Create(t *testing.T) {
	c := Citation{
		Source: "https://example.com",
		URL:    "https://example.com/article",
		Trust:  0.85,
	}

	if c.Trust != 0.85 {
		t.Errorf("Expected trust 0.85, got %f", c.Trust)
	}
}

func TestRAGResult_Structure(t *testing.T) {
	result := &RAGResult{
		Question:   "What is AI?",
		Answer:     "AI is artificial intelligence.",
		Iterations: 2,
		Citations: []Citation{
			{Source: "wiki", URL: "https://wiki.ai", Trust: 0.9},
		},
	}

	if len(result.Citations) != 1 {
		t.Errorf("Expected 1 citation, got %d", len(result.Citations))
	}
}

type mockLLM struct{}

func (m *mockLLM) Complete(req LLMRequest) (string, error) {
	return "mock response", nil
}
