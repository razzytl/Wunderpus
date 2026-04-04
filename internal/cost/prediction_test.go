package cost

import (
	"testing"

	"github.com/wunderpus/wunderpus/internal/provider"
)

func TestEstimateCost_BasicMessages(t *testing.T) {
	tracker := &Tracker{
		prices: map[string]ModelPrice{
			"gpt-4o": {InputPrice: 2.50, OutputPrice: 10.00},
		},
	}

	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: "You are a helpful assistant."},
		{Role: provider.RoleUser, Content: "Hello, how are you?"},
	}

	prediction := tracker.EstimateCost(messages, "gpt-4o", 0.5)

	if prediction.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", prediction.Model)
	}
	if prediction.InputTokens <= 0 {
		t.Errorf("expected positive input tokens, got %d", prediction.InputTokens)
	}
	if prediction.MinCost <= 0 {
		t.Errorf("expected positive min cost, got %f", prediction.MinCost)
	}
	if prediction.MaxCost <= prediction.MinCost {
		t.Errorf("expected max cost > min cost, got max=%f, min=%f", prediction.MaxCost, prediction.MinCost)
	}
	if prediction.EstOutputTokens <= 0 {
		t.Errorf("expected positive output tokens, got %d", prediction.EstOutputTokens)
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	tracker := &Tracker{
		prices: map[string]ModelPrice{
			"gpt-4o": {InputPrice: 2.50, OutputPrice: 10.00},
		},
	}

	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "test"},
	}

	prediction := tracker.EstimateCost(messages, "unknown-model", 0.5)

	// Should use default pricing
	if prediction.MinCost <= 0 {
		t.Errorf("expected positive cost with default pricing, got %f", prediction.MinCost)
	}
}

func TestEstimateCost_EmptyMessages(t *testing.T) {
	tracker := &Tracker{
		prices: map[string]ModelPrice{
			"gpt-4o": {InputPrice: 2.50, OutputPrice: 10.00},
		},
	}

	prediction := tracker.EstimateCost(nil, "gpt-4o", 0.5)

	if prediction.InputTokens != 0 {
		t.Errorf("expected 0 input tokens for empty messages, got %d", prediction.InputTokens)
	}
	if prediction.MinCost != 0 {
		t.Errorf("expected 0 min cost for empty messages, got %f", prediction.MinCost)
	}
}

func TestEstimateCost_ToolCalls(t *testing.T) {
	tracker := &Tracker{
		prices: map[string]ModelPrice{
			"gpt-4o": {InputPrice: 2.50, OutputPrice: 10.00},
		},
	}

	messages := []provider.Message{
		{
			Role:    provider.RoleAssistant,
			Content: "Let me check the file.",
			ToolCalls: []provider.ToolCallInfo{
				{
					ID:   "call_1",
					Type: "function",
					Function: provider.ToolCallFunc{
						Name:      "file_read",
						Arguments: `{"path": "/etc/passwd"}`,
					},
				},
			},
		},
	}

	prediction := tracker.EstimateCost(messages, "gpt-4o", 0.5)

	// Tool calls should add tokens
	if prediction.InputTokens <= 0 {
		t.Errorf("expected positive input tokens with tool calls, got %d", prediction.InputTokens)
	}
}

func TestEstimateCost_CustomOutputRatio(t *testing.T) {
	tracker := &Tracker{
		prices: map[string]ModelPrice{
			"gpt-4o": {InputPrice: 2.50, OutputPrice: 10.00},
		},
	}

	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "Hello world this is a test message with some content"},
	}

	pred1 := tracker.EstimateCost(messages, "gpt-4o", 0.3)
	pred2 := tracker.EstimateCost(messages, "gpt-4o", 0.8)

	// Higher output ratio should produce higher estimated output tokens
	if pred2.EstOutputTokens <= pred1.EstOutputTokens {
		t.Errorf("expected higher output ratio to produce more output tokens: 0.8=%d, 0.3=%d",
			pred2.EstOutputTokens, pred1.EstOutputTokens)
	}

	// Higher output ratio should produce higher max cost
	if pred2.MaxCost <= pred1.MaxCost {
		t.Errorf("expected higher output ratio to produce higher max cost: 0.8=%f, 0.3=%f",
			pred2.MaxCost, pred1.MaxCost)
	}
}

func TestEstimateCost_ZeroOutputRatio(t *testing.T) {
	tracker := &Tracker{
		prices: map[string]ModelPrice{
			"gpt-4o": {InputPrice: 2.50, OutputPrice: 10.00},
		},
	}

	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "Hello"},
	}

	prediction := tracker.EstimateCost(messages, "gpt-4o", 0)

	// Should default to 0.5
	if prediction.EstOutputTokens <= 0 {
		t.Errorf("expected positive output tokens with zero ratio (should default to 0.5), got %d", prediction.EstOutputTokens)
	}
}
