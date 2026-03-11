package provider

import (
	"testing"
	"time"
)

func TestCompletionRequest(t *testing.T) {
	req := &CompletionRequest{
		Model:    "gpt-4",
		Messages: []Message{},
	}

	if req.Model != "gpt-4" {
		t.Errorf("expected Model 'gpt-4', got %q", req.Model)
	}
}

func TestCompletionResponse(t *testing.T) {
	resp := &CompletionResponse{
		Content: "Test response",
		Model:   "gpt-4",
	}

	if resp.Content != "Test response" {
		t.Errorf("expected Content 'Test response', got %q", resp.Content)
	}
}

func TestMessage_RoleContent(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello",
	}

	if msg.Role != "user" {
		t.Errorf("expected Role 'user', got %q", msg.Role)
	}

	if msg.Content != "Hello" {
		t.Errorf("expected Content 'Hello', got %q", msg.Content)
	}
}

func TestToolCallInfo(t *testing.T) {
	tc := ToolCallInfo{
		ID:   "call-123",
		Type: "function",
		Function: ToolCallFunc{
			Name:      "web_search",
			Arguments: `{"query": "test"}`,
		},
	}

	if tc.ID != "call-123" {
		t.Errorf("expected ID 'call-123', got %q", tc.ID)
	}

	if tc.Function.Name != "web_search" {
		t.Errorf("expected Function.Name 'web_search', got %q", tc.Function.Name)
	}
}

func TestContentPart(t *testing.T) {
	part := ContentPart{
		Type: "text",
		Text: "Hello world",
	}

	if part.Type != "text" {
		t.Errorf("expected Type 'text', got %q", part.Type)
	}

	if part.Text != "Hello world" {
		t.Errorf("expected Text 'Hello world', got %q", part.Text)
	}
}

func TestImageURL(t *testing.T) {
	url := ImageURL{
		URL:    "https://example.com/image.jpg",
		Detail: "auto",
	}

	if url.URL != "https://example.com/image.jpg" {
		t.Errorf("expected URL 'https://example.com/image.jpg', got %q", url.URL)
	}

	if url.Detail != "auto" {
		t.Errorf("expected Detail 'auto', got %q", url.Detail)
	}
}

func TestStreamChunk(t *testing.T) {
	chunk := StreamChunk{
		Content: "Hello",
		Done:    false,
	}

	if chunk.Content != "Hello" {
		t.Errorf("expected Content 'Hello', got %q", chunk.Content)
	}

	if chunk.Done {
		t.Error("expected Done to be false")
	}
}

func TestResponseCache(t *testing.T) {
	cache := NewResponseCache(time.Minute)
	if cache == nil {
		t.Error("expected non-nil cache")
	}
}

func TestCooldownTracker(t *testing.T) {
	tracker := NewCooldownTracker()
	if tracker == nil {
		t.Error("expected non-nil tracker")
	}

	tracker.StartCooldown("test-provider", time.Minute)
	if !tracker.IsInCooldown("test-provider") {
		t.Error("expected provider to be in cooldown")
	}
}

func TestErrorClassifier(t *testing.T) {
	classifier := &ErrorClassifier{}

	tests := []struct {
		errStr   string
		expected FailoverReason
	}{
		{"rate limit exceeded", FailoverReasonRateLimit},
		{"timeout", FailoverReasonTimeout},
		{"500 internal server error", FailoverReasonServerError},
		{"quota exceeded", FailoverReasonQuotaExceeded},
		{"400 bad request", FailoverReasonInvalidRequest},
		{"some random error", FailoverReasonRetriableError},
	}

	for _, tt := range tests {
		t.Run(tt.errStr, func(t *testing.T) {
			reason := classifier.Classify(&testError{tt.errStr})
			if reason != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, reason)
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
