package planning

import "context"

// LLMCaller for LLM calls.
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

// GoalManager for AGS goal management.
type GoalManager interface {
	CreateGoal(ctx context.Context, title, description string) (string, error)
}

// WorldModelQuery for context.
type WorldModelQuery interface {
	Ask(ctx context.Context, question string) (string, error)
}
