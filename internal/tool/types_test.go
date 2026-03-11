package tool

import (
	"testing"
)

func TestParameterDef(t *testing.T) {
	tests := []struct {
		name string
		def  ParameterDef
	}{
		{
			name: "string parameter",
			def: ParameterDef{
				Name:        "prompt",
				Type:        "string",
				Description: "The prompt to send",
				Required:    true,
			},
		},
		{
			name: "number parameter",
			def: ParameterDef{
				Name:        "count",
				Type:        "number",
				Description: "Number of items",
				Required:    false,
			},
		},
		{
			name: "boolean parameter",
			def: ParameterDef{
				Name:        "verbose",
				Type:        "boolean",
				Description: "Enable verbose output",
				Required:    false,
			},
		},
		{
			name: "object parameter",
			def: ParameterDef{
				Name:        "options",
				Type:        "object",
				Description: "Additional options",
				Required:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.def.Name == "" {
				t.Error("expected non-empty name")
			}
			if tt.def.Type == "" {
				t.Error("expected non-empty type")
			}
		})
	}
}

func TestResult(t *testing.T) {
	tests := []struct {
		name   string
		result Result
	}{
		{
			name: "successful result",
			result: Result{
				Output: "test output",
			},
		},
		{
			name: "error result",
			result: Result{
				Output: "",
				Error:  "something went wrong",
			},
		},
		{
			name:   "empty result",
			result: Result{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.result.Output
			_ = tt.result.Error
		})
	}
}

func TestToolCall(t *testing.T) {
	call := ToolCall{
		ID:   "call-123",
		Name: "web_search",
		Args: map[string]any{
			"query": "test query",
		},
	}

	if call.ID != "call-123" {
		t.Errorf("expected ID 'call-123', got %q", call.ID)
	}

	if call.Name != "web_search" {
		t.Errorf("expected Name 'web_search', got %q", call.Name)
	}

	if call.Args["query"] != "test query" {
		t.Errorf("expected Args['query'] 'test query', got %v", call.Args["query"])
	}
}

func TestBuildSchema(t *testing.T) {
	// Test basic schema building
	schema := ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        "test_func",
			Description: "A test function",
		},
	}

	if schema.Type != "function" {
		t.Errorf("expected Type 'function', got %q", schema.Type)
	}

	if schema.Function.Name != "test_func" {
		t.Errorf("expected Function.Name 'test_func', got %q", schema.Function.Name)
	}
}
