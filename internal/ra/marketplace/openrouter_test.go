package marketplace

import (
	"context"
	"errors"
	"testing"
)

func TestOpenRouterAdapter_RouteWithFallback(t *testing.T) {
	callCount := 0
	mockComplete := func(ctx context.Context, model, prompt string) (string, error) {
		callCount++
		// First model (deepseek-coder) always fails
		if model == "deepseek/deepseek-coder" {
			return "", errors.New("model unavailable")
		}
		return "response from " + model, nil
	}

	adapter := NewOpenRouterAdapter("test-key", mockComplete)

	result, err := adapter.Route(context.Background(), TaskCodeGeneration, "write a function", 0.01)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}

	// Should have tried deepseek first (failed), then succeeded with claude
	if callCount != 2 {
		t.Fatalf("expected 2 model calls (1 fail + 1 success), got %d", callCount)
	}
	if result != "response from anthropic/claude-3.5-sonnet" {
		t.Fatalf("expected claude response, got: %s", result)
	}
}

func TestOpenRouterAdapter_BudgetFilter(t *testing.T) {
	mockComplete := func(ctx context.Context, model, prompt string) (string, error) {
		return "ok", nil
	}

	adapter := NewOpenRouterAdapter("test-key", mockComplete)

	// Set very low budget — only cheapest model qualifies
	_, err := adapter.Route(context.Background(), TaskCodeGeneration, "test", 0.0001)
	if err == nil {
		t.Fatal("should fail when no model is within budget")
	}
}

func TestOpenRouterAdapter_AllModelsFail(t *testing.T) {
	mockComplete := func(ctx context.Context, model, prompt string) (string, error) {
		return "", errors.New("all down")
	}

	adapter := NewOpenRouterAdapter("test-key", mockComplete)

	_, err := adapter.Route(context.Background(), TaskTextGeneration, "test", 1.0)
	if err == nil {
		t.Fatal("should fail when all models fail")
	}
}

func TestOpenRouterAdapter_SpendTracking(t *testing.T) {
	mockComplete := func(ctx context.Context, model, prompt string) (string, error) {
		return "ok", nil
	}

	adapter := NewOpenRouterAdapter("test-key", mockComplete)

	adapter.Route(context.Background(), TaskEmbedding, "test", 1.0)

	total := adapter.TotalSpend()
	if total <= 0 {
		t.Fatal("should track spend")
	}

	spend := adapter.GetSpend()
	if len(spend) == 0 {
		t.Fatal("should have per-model spend entries")
	}
}
