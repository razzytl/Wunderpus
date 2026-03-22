package marketplace

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// ModelRoute describes a model configuration for a task type.
type ModelRoute struct {
	Model   string
	Quality string  // "high", "medium", "low"
	CostPer float64 // cost per 1K tokens
}

// TaskType classifies the type of LLM task.
type TaskType string

const (
	TaskCodeGeneration TaskType = "code_generation"
	TaskTextGeneration TaskType = "text_generation"
	TaskEmbedding      TaskType = "embedding"
	TaskClassification TaskType = "classification"
)

// CompletionFn abstracts LLM completion for the marketplace adapter.
type CompletionFn func(ctx context.Context, model, prompt string) (string, error)

// OpenRouterAdapter routes LLM requests through OpenRouter with automatic
// model selection and fallback.
type OpenRouterAdapter struct {
	apiKey       string
	completeFn   CompletionFn
	spendTracker map[string]float64 // model → total spend
	mu           sync.RWMutex
}

// NewOpenRouterAdapter creates an OpenRouter adapter.
func NewOpenRouterAdapter(apiKey string, completeFn CompletionFn) *OpenRouterAdapter {
	return &OpenRouterAdapter{
		apiKey:       apiKey,
		completeFn:   completeFn,
		spendTracker: make(map[string]float64),
	}
}

// modelTable maps task types to prioritized model lists.
var modelTable = map[TaskType][]ModelRoute{
	TaskCodeGeneration: {
		{Model: "deepseek/deepseek-coder", Quality: "high", CostPer: 0.001},
		{Model: "anthropic/claude-3.5-sonnet", Quality: "high", CostPer: 0.003},
		{Model: "openai/gpt-4o-mini", Quality: "medium", CostPer: 0.0015},
	},
	TaskTextGeneration: {
		{Model: "google/gemma-2-9b", Quality: "medium", CostPer: 0.0002},
		{Model: "meta-llama/llama-3-70b", Quality: "high", CostPer: 0.0009},
		{Model: "openai/gpt-4o-mini", Quality: "medium", CostPer: 0.0015},
	},
	TaskEmbedding: {
		{Model: "openai/text-embedding-3-small", Quality: "high", CostPer: 0.00002},
	},
	TaskClassification: {
		{Model: "google/gemma-2-9b", Quality: "medium", CostPer: 0.0002},
		{Model: "openai/gpt-4o-mini", Quality: "medium", CostPer: 0.0015},
	},
}

// Route selects the best model for a task type and budget, then completes.
// Falls back to next model if primary fails.
func (o *OpenRouterAdapter) Route(ctx context.Context, taskType TaskType, prompt string, maxCost float64) (string, error) {
	models, ok := modelTable[taskType]
	if !ok {
		return "", fmt.Errorf("ra marketplace: unknown task type %s", taskType)
	}

	var lastErr error
	for _, route := range models {
		// Check budget
		if route.CostPer > maxCost {
			continue
		}

		slog.Debug("ra marketplace: trying model", "model", route.Model, "task", taskType)

		result, err := o.completeFn(ctx, route.Model, prompt)
		if err != nil {
			lastErr = err
			slog.Warn("ra marketplace: model failed, trying fallback", "model", route.Model, "error", err)
			continue
		}

		// Track spend (thread-safe)
		o.mu.Lock()
		o.spendTracker[route.Model] += route.CostPer
		o.mu.Unlock()

		return result, nil
	}

	return "", fmt.Errorf("ra marketplace: all models failed for task %s: %w", taskType, lastErr)
}

// GetSpend returns the total spend across all models.
func (o *OpenRouterAdapter) GetSpend() map[string]float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	cp := make(map[string]float64)
	for k, v := range o.spendTracker {
		cp[k] = v
	}
	return cp
}

// TotalSpend returns the sum of all model spend.
func (o *OpenRouterAdapter) TotalSpend() float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	total := 0.0
	for _, v := range o.spendTracker {
		total += v
	}
	return total
}
