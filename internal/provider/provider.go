package provider

import "context"

// Role constants for messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message represents a single chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest is the input for an LLM completion call.
type CompletionRequest struct {
	Messages    []Message
	Model       string
	MaxTokens   int
	Temperature float64
}

// CompletionResponse is the output from an LLM completion call.
type CompletionResponse struct {
	Content      string
	Model        string
	FinishReason string
	PromptTokens int
	CompTokens   int
}

// StreamChunk is a single chunk from a streaming response.
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// Provider is the interface every LLM backend must implement.
type Provider interface {
	// Name returns the provider identifier (e.g. "openai", "anthropic").
	Name() string
	// Complete sends a request and returns the full response.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
	// Stream sends a request and returns a channel of streaming chunks.
	Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)
}
