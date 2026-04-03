package tool

import (
	"context"
	"encoding/json"
)

// ApprovalLevel determines how a tool is executed.
type ApprovalLevel int

const (
	AutoExecute      ApprovalLevel = iota // Run immediately, no approval needed
	NotifyOnly                            // Run but notify the user
	RequiresApproval                      // Pause and wait for user approval
	Blocked                               // Never execute, reject immediately
)

// Tool is the interface every tool must implement.
type Tool interface {
	// Name returns the tool identifier.
	Name() string
	// Description returns a human-readable description for the LLM.
	Description() string
	// Parameters returns JSON Schema-style parameter definitions.
	Parameters() []ParameterDef
	// Execute runs the tool with the given arguments.
	Execute(ctx context.Context, args map[string]any) (*Result, error)
	// Sensitive returns true if this tool requires user approval before execution.
	Sensitive() bool
	// ApprovalLevel returns the policy-based approval level for this tool.
	ApprovalLevel() ApprovalLevel
	// Version returns the tool version string.
	Version() string
	// Dependencies returns a list of other tool names this tool relies on.
	Dependencies() []string
}

// ParameterDef describes a single tool parameter.
type ParameterDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, number, boolean, object
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// Result is the output of a tool execution.
type Result struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// ToolCall represents a parsed tool invocation from the LLM.
type ToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// ToolSchema is the OpenAI-compatible function schema sent to the LLM.
type ToolSchema struct {
	Type     string         `json:"type"` // "function"
	Function FunctionSchema `json:"function"`
}

// FunctionSchema describes a function for the LLM.
type FunctionSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// BuildSchema generates an OpenAI-compatible tool schema from a Tool.
func BuildSchema(t Tool) ToolSchema {
	params := t.Parameters()

	properties := make(map[string]any)
	var required []string

	for _, p := range params {
		properties[p.Name] = map[string]string{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	raw, _ := json.Marshal(schema)

	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  raw,
		},
	}
}
