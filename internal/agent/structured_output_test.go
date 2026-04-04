package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/wunderpus/wunderpus/internal/provider"
)

func TestStructuredOutputEnforcer_ValidateJSON(t *testing.T) {
	e := NewStructuredOutputEnforcer(2)

	validJSON := `{"name": "test", "value": 42}`
	invalidJSON := `{"name": "test", "value": 42,}`

	result, valid := e.ValidateAndFix(validJSON, OutputFormat{Type: "json"})
	if !valid {
		t.Error("expected valid JSON to pass")
	}
	if result != validJSON {
		t.Errorf("expected %q, got %q", validJSON, result)
	}

	result, valid = e.ValidateAndFix(invalidJSON, OutputFormat{Type: "json"})
	if valid {
		t.Error("expected invalid JSON to fail")
	}
}

func TestStructuredOutputEnforcer_TextAlwaysPasses(t *testing.T) {
	e := NewStructuredOutputEnforcer(2)

	result, valid := e.ValidateAndFix("any text at all", OutputFormat{Type: "text"})
	if !valid {
		t.Error("expected text format to always pass")
	}
	if result != "any text at all" {
		t.Errorf("expected %q, got %q", "any text at all", result)
	}

	// Empty format also passes
	result, valid = e.ValidateAndFix("any text", OutputFormat{})
	if !valid {
		t.Error("expected empty format to always pass")
	}
}

func TestStructuredOutputEnforcer_ValidationWithSchema(t *testing.T) {
	e := NewStructuredOutputEnforcer(2)

	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name", "age"]
	}`

	validJSON := `{"name": "Alice", "age": 30}`
	_, valid := e.ValidateAndFix(validJSON, OutputFormat{Type: "json", JSONSchema: schema})
	if !valid {
		t.Error("expected valid JSON to pass schema check")
	}

	invalidJSON := `not json at all`
	_, valid = e.ValidateAndFix(invalidJSON, OutputFormat{Type: "json", JSONSchema: schema})
	if valid {
		t.Error("expected invalid JSON to fail")
	}
}

func TestStructuredOutputEnforcer_CorrectionPrompt(t *testing.T) {
	e := NewStructuredOutputEnforcer(2)

	// Without schema
	prompt := e.CorrectionPrompt("bad response", OutputFormat{Type: "json"})
	if !contains(prompt, "not valid JSON") {
		t.Errorf("expected 'not valid JSON' in prompt, got: %s", prompt)
	}

	// With schema
	schema := `{"type": "object"}`
	prompt = e.CorrectionPrompt("bad response", OutputFormat{Type: "json", JSONSchema: schema})
	if !contains(prompt, schema) {
		t.Errorf("expected schema in prompt, got: %s", prompt)
	}
	if !contains(prompt, "bad response") {
		t.Errorf("expected original response in prompt, got: %s", prompt)
	}
}

func TestStructuredOutputEnforcer_ExecuteWithValidation_Success(t *testing.T) {
	e := NewStructuredOutputEnforcer(2)

	validJSON := `{"result": "ok"}`
	callCount := 0

	completeFn := func(ctx context.Context, messages []provider.Message) (*provider.CompletionResponse, error) {
		callCount++
		return &provider.CompletionResponse{Content: validJSON}, nil
	}

	messages := []provider.Message{{Role: provider.RoleUser, Content: "give me json"}}
	resp, retries, err := e.ExecuteWithValidation(context.Background(), completeFn, messages, OutputFormat{Type: "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retries != 0 {
		t.Errorf("expected 0 retries, got %d", retries)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
	if resp.Content != validJSON {
		t.Errorf("expected %q, got %q", validJSON, resp.Content)
	}
}

func TestStructuredOutputEnforcer_ExecuteWithValidation_RetrySuccess(t *testing.T) {
	e := NewStructuredOutputEnforcer(3)

	callCount := 0
	completeFn := func(ctx context.Context, messages []provider.Message) (*provider.CompletionResponse, error) {
		callCount++
		if callCount == 1 {
			return &provider.CompletionResponse{Content: "not valid json {{{"}, nil
		}
		return &provider.CompletionResponse{Content: `{"retry": "success"}`}, nil
	}

	messages := []provider.Message{{Role: provider.RoleUser, Content: "give me json"}}
	resp, retries, err := e.ExecuteWithValidation(context.Background(), completeFn, messages, OutputFormat{Type: "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retries != 1 {
		t.Errorf("expected 1 retry, got %d", retries)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
	if resp.Content != `{"retry": "success"}` {
		t.Errorf("unexpected content: %q", resp.Content)
	}
}

func TestStructuredOutputEnforcer_ExecuteWithValidation_AllRetriesExhausted(t *testing.T) {
	e := NewStructuredOutputEnforcer(2)

	callCount := 0
	completeFn := func(ctx context.Context, messages []provider.Message) (*provider.CompletionResponse, error) {
		callCount++
		return &provider.CompletionResponse{Content: "always invalid"}, nil
	}

	messages := []provider.Message{{Role: provider.RoleUser, Content: "give me json"}}
	resp, retries, err := e.ExecuteWithValidation(context.Background(), completeFn, messages, OutputFormat{Type: "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retries != 2 {
		t.Errorf("expected 2 retries (exhausted), got %d", retries)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls (1 initial + 2 retries), got %d", callCount)
	}
	if resp.Content != "always invalid" {
		t.Errorf("expected last response, got %q", resp.Content)
	}
}

func TestStructuredOutputEnforcer_DefaultMaxRetries(t *testing.T) {
	e := NewStructuredOutputEnforcer(0)
	if e.maxRetries != 2 {
		t.Errorf("expected default maxRetries=2, got %d", e.maxRetries)
	}

	e2 := NewStructuredOutputEnforcer(-1)
	if e2.maxRetries != 2 {
		t.Errorf("expected default maxRetries=2 for negative, got %d", e2.maxRetries)
	}

	e3 := NewStructuredOutputEnforcer(5)
	if e3.maxRetries != 5 {
		t.Errorf("expected maxRetries=5, got %d", e3.maxRetries)
	}
}

func TestStructuredOutputEnforcer_ErrorMessage(t *testing.T) {
	e := NewStructuredOutputEnforcer(2)

	expectedErr := "provider error"
	completeFn := func(ctx context.Context, messages []provider.Message) (*provider.CompletionResponse, error) {
		return nil, &testError{msg: expectedErr}
	}

	messages := []provider.Message{{Role: provider.RoleUser, Content: "test"}}
	_, _, err := e.ExecuteWithValidation(context.Background(), completeFn, messages, OutputFormat{Type: "json"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func contains(s, substr string) bool {
	return len(s) >= len(substr) && search(s, substr)
}

func search(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test that OutputFormat can be marshaled/unmarshaled
func TestOutputFormat_JSONRoundTrip(t *testing.T) {
	of := OutputFormat{
		Type:        "json",
		JSONSchema:  `{"type":"object"}`,
		Description: "A test schema",
	}

	data, err := json.Marshal(of)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded OutputFormat
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Type != of.Type {
		t.Errorf("type mismatch: got %q, want %q", decoded.Type, of.Type)
	}
	if decoded.JSONSchema != of.JSONSchema {
		t.Errorf("schema mismatch: got %q, want %q", decoded.JSONSchema, of.JSONSchema)
	}
}
