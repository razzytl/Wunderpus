package provider

import (
	"context"
	"encoding/json"
)

// Role constants for messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Message represents a single chat message.
type Message struct {
	Role         string         `json:"role"`
	Content      string         `json:"content,omitempty"`
	MultiContent []ContentPart  `json:"multi_content,omitempty"` // For vision/multimodal
	ToolCallID   string         `json:"tool_call_id,omitempty"`  // for tool result messages
	ToolCalls    []ToolCallInfo `json:"tool_calls,omitempty"`    // for assistant messages with tool calls
}

// ContentPart represents a part of a multimodal message.
type ContentPart struct {
	Type     string    `json:"type"` // "text", "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents a base64 or hosted image for vision models.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// CompletionRequest is the input for an LLM completion call.
type CompletionRequest struct {
	Messages    []Message
	Model       string
	MaxTokens   int
	Temperature float64
	Tools       []ToolSchema // tool definitions for function calling
	ToolChoice  any          `json:"tool_choice,omitempty"` // "auto", "none", "required", or {"type": "function", "function": {"name": "..."}}
}

// ToolSchema is an OpenAI-compatible tool/function definition.
type ToolSchema struct {
	Type     string         `json:"type"` // "function"
	Function FunctionSchema `json:"function"`
}

// FunctionSchema describes a callable function.
type FunctionSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// CompletionResponse is the output from an LLM completion call.
type CompletionResponse struct {
	Content      string
	Model        string
	FinishReason string
	PromptTokens int
	CompTokens   int
	ToolCalls    []ToolCallInfo // parsed tool calls from the LLM
}

// ToolCallInfo holds a single tool invocation from the LLM.
type ToolCallInfo struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function ToolCallFunc `json:"function"`
}

// ToolCallFunc is the function name + arguments.
type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
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
