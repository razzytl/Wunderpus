package research

import (
	"context"
	"log/slog"
	"time"
)

// Citation represents a source citation.
type Citation struct {
	Source  string  `json:"source"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Trust   float64 `json:"trust"` // 0-1
}

// RetrievalResult represents a single retrieval result.
type RetrievalResult struct {
	Content   string    `json:"content"`
	Source    string    `json:"source"`
	URL       string    `json:"url"`
	Trust     float64   `json:"trust"`
	Retrieved time.Time `json:"retrieved"`
}

// AgenticRAG performs iterative retrieval-augmented generation.
type AgenticRAG struct {
	vectorDB   VectorStore
	worldModel WorldModel
	webSearch  WebSearch
	llm        LLMCaller
	maxIter    int
}

// VectorStore interface for vector database operations.
type VectorStore interface {
	Search(ctx context.Context, query string, topK int) ([]RetrievalResult, error)
}

// WorldModel interface for world model queries.
type WorldModel interface {
	Ask(ctx context.Context, question string) (string, error)
}

// WebSearch interface for web search.
type WebSearch interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

// SearchResult represents a web search result.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// LLMCaller interface for LLM operations.
type LLMCaller interface {
	Complete(req LLMRequest) (string, error)
}

// LLMRequest represents an LLM request.
type LLMRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
}

// RAGConfig holds configuration for Agentic RAG.
type RAGConfig struct {
	Enabled      bool
	MaxIteration int
	TopK         int
}

// NewAgenticRAG creates a new Agentic RAG system.
func NewAgenticRAG(cfg RAGConfig, vecDB VectorStore, wm WorldModel, web WebSearch, llm LLMCaller) *AgenticRAG {
	return &AgenticRAG{
		vectorDB:   vecDB,
		worldModel: wm,
		webSearch:  web,
		llm:        llm,
		maxIter:    cfg.MaxIteration,
	}
}

// Query performs iterative RAG on a question.
func (r *AgenticRAG) Query(ctx context.Context, question string) (*RAGResult, error) {
	slog.Info("research: Agentic RAG query", "question", question)

	// Priority order: world model > vector DB > web > LLM knowledge
	var allResults []RetrievalResult

	// First: world model (highest trust)
	if r.worldModel != nil {
		wmResult, err := r.worldModel.Ask(ctx, question)
		if err == nil && wmResult != "" {
			allResults = append(allResults, RetrievalResult{
				Content:   wmResult,
				Source:    "world_model",
				Trust:     0.95,
				Retrieved: time.Now(),
			})
		}
	}

	// Second: vector DB
	if r.vectorDB != nil {
		vecResults, err := r.vectorDB.Search(ctx, question, 5)
		if err == nil {
			allResults = append(allResults, vecResults...)
		}
	}

	// Third: web search (if needed)
	needsMore, err := r.evaluate(ctx, question, allResults)
	if err == nil && needsMore && r.webSearch != nil {
		webResults, err := r.webSearch.Search(ctx, question)
		if err == nil {
			for _, res := range webResults {
				allResults = append(allResults, RetrievalResult{
					Content:   res.Snippet,
					Source:    res.URL,
					URL:       res.URL,
					Trust:     0.6, // Lower trust for web
					Retrieved: time.Now(),
				})
			}
		}
	}

	// Final: synthesize answer with citations
	answer, citations := r.synthesize(ctx, question, allResults)

	result := &RAGResult{
		Question:   question,
		Answer:     answer,
		Iterations: 1,
		Citations:  citations,
	}

	slog.Info("research: RAG complete", "iterations", result.Iterations, "citations", len(citations))

	return result, nil
}

func (r *AgenticRAG) evaluate(ctx context.Context, question string, results []RetrievalResult) (bool, error) {
	// Ask LLM if current results are sufficient
	prompt := "Given the question: '" + question + "'\n\nAnd these results:\n"
	for i, res := range results {
		prompt += sprintf("%d. %s (trust: %.2f)\n", i+1, res.Content, res.Trust)
	}
	prompt += "\nIs this sufficient to answer the question thoroughly? Reply YES or NO and explain briefly."

	req := LLMRequest{
		SystemPrompt: "You evaluate whether search results are sufficient.",
		UserPrompt:   prompt,
		Temperature:  0.3,
		MaxTokens:    100,
	}

	resp, err := r.llm.Complete(req)
	if err != nil {
		return true, err // Default to needing more if evaluation fails
	}

	// Simple check - if response starts with NO, need more
	return len(resp) > 0 && resp[0] == 'N', nil
}

func (r *AgenticRAG) synthesize(ctx context.Context, question string, results []RetrievalResult) (string, []Citation) {
	// Build prompt with all results
	prompt := "Based on these sources, answer the question thoroughly with citations:\n\nQuestion: " + question + "\n\nSources:\n"

	citations := make([]Citation, 0, len(results))
	for _, res := range results {
		prompt += "- " + res.Content + " [Source: " + res.Source + "]\n"
		citations = append(citations, Citation{
			Source: res.Source,
			URL:    res.URL,
			Trust:  res.Trust,
		})
	}

	prompt += "\nProvide a comprehensive answer with inline citations."

	req := LLMRequest{
		SystemPrompt: "You are a research assistant. Answer with proper citations.",
		UserPrompt:   prompt,
		Temperature:  0.5,
		MaxTokens:    2000,
	}

	answer, _ := r.llm.Complete(req)

	return answer, citations
}

// RAGResult represents the result of an Agentic RAG query.
type RAGResult struct {
	Question   string
	Answer     string
	Iterations int
	Citations  []Citation
}

func sprintf(format string, a ...interface{}) string {
	// Simple placeholder
	return "[sprintf]"
}
