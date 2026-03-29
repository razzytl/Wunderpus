package social

import "context"

// LLMCaller interface for LLM.
type LLMCaller interface {
	Complete(req LLMRequest) (string, error)
}

// LLMRequest for LLM calls.
type LLMRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
}

// WorldModelQuery for context.
type WorldModelQuery interface {
	Ask(ctx context.Context, question string) (string, error)
}

// BrowserAgent for automation.
type BrowserAgent interface {
	Execute(ctx context.Context, goal, url string) (string, error)
}
