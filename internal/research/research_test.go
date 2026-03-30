package research

import (
	"testing"
)

func TestAgenticRAG_Query(t *testing.T) {
	rag := &AgenticRAG{
		maxIter: 3,
		llm:     &mockLLM{},
	}

	if rag.maxIter != 3 {
		t.Errorf("Expected maxIter 3, got %d", rag.maxIter)
	}
	if rag.llm == nil {
		t.Error("Expected llm to be set")
	}
}

func TestCitation_Create(t *testing.T) {
	c := Citation{
		Source: "https://example.com",
		URL:    "https://example.com/article",
		Trust:  0.85,
	}

	if c.Source != "https://example.com" {
		t.Errorf("Expected source 'https://example.com', got %s", c.Source)
	}
	if c.URL != "https://example.com/article" {
		t.Errorf("Expected URL 'https://example.com/article', got %s", c.URL)
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

	if result.Question != "What is AI?" {
		t.Errorf("Expected question 'What is AI?', got %s", result.Question)
	}
	if result.Answer != "AI is artificial intelligence." {
		t.Errorf("Expected answer 'AI is artificial intelligence.', got %s", result.Answer)
	}
	if result.Iterations != 2 {
		t.Errorf("Expected 2 iterations, got %d", result.Iterations)
	}
	if len(result.Citations) != 1 {
		t.Errorf("Expected 1 citation, got %d", len(result.Citations))
	}
}

type mockLLM struct{}

func (m *mockLLM) Complete(req LLMRequest) (string, error) {
	return "mock response", nil
}
