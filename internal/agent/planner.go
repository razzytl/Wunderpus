package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wunderpus/wunderpus/internal/provider"
)

// Subtask represents a single actionable step toward a larger goal.
type Subtask struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Dependencies []string `json:"dependencies"` // IDs of tasks that must complete before this one
	Type        string   `json:"type"`         // io, compute, general
}

// TaskGraph represents a decomposed goal.
type TaskGraph struct {
	Goal     string    `json:"goal"`
	Subtasks []Subtask `json:"subtasks"`
}

// TaskPlanner uses an LLM to decompose a complex goal into smaller subtasks.
type TaskPlanner struct {
	provider provider.Provider
	model    string
}

// NewTaskPlanner creating a new planner.
func NewTaskPlanner(p provider.Provider, model string) *TaskPlanner {
	return &TaskPlanner{
		provider: p,
		model:    model,
	}
}

// Decompose takes a complex objective and breaks it into a dependency graph using a strict JSON schema Tool call approach.
func (p *TaskPlanner) Decompose(ctx context.Context, goal string) (*TaskGraph, error) {
	systemPrompt := `You are an expert systems planner. Your job is to take a user's complex goal and break it down into a dependency graph of subtasks.
Each subtask must have a unique ID, a clear description, and a list of dependency IDs that must be completed before it can start.
Classify the task type as one of: "io" (reading/writing/web), "compute" (math/logic/heavy lifting), or "general" (reasoning/glue operations).
Call the provided "submit_plan" tool to submit your plan. You MUST call this tool. Do not just output text.`

	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: systemPrompt},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Goal: %s", goal)},
	}

	// Define strict schema for the graph
	schema := []provider.ToolSchema{
		{
			Type: "function",
			Function: provider.FunctionSchema{
				Name:        "submit_plan",
				Description: "Submit the decomposed task graph",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"subtasks": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"id": { "type": "string", "description": "Unique task ID, like 'task_1'" },
									"description": { "type": "string", "description": "Clear step description" },
									"dependencies": { "type": "array", "items": { "type": "string" }, "description": "IDs of tasks that must complete before this" },
									"type": { "type": "string", "enum": ["io", "compute", "general"] }
								},
								"required": ["id", "description", "dependencies", "type"]
							}
						}
					},
					"required": ["subtasks"]
				}`),
			},
		},
	}

	req := &provider.CompletionRequest{
		Messages:    messages,
		Model:       p.model,
		Temperature: 0.1,
		Tools:       schema,
	}

	resp, err := p.provider.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("planner completion error: %w", err)
	}

	if len(resp.ToolCalls) == 0 {
		return nil, fmt.Errorf("planner failed to call submit_plan tool, returned text: %s", resp.Content)
	}

	var rawPlan struct {
		Subtasks []Subtask `json:"subtasks"`
	}

	// Find the submit_plan tool call
	for _, tc := range resp.ToolCalls {
		if tc.Function.Name == "submit_plan" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &rawPlan); err != nil {
				return nil, fmt.Errorf("failed to parse plan json: %w", err)
			}
			return &TaskGraph{
				Goal:     goal,
				Subtasks: rawPlan.Subtasks,
			}, nil
		}
	}

	return nil, fmt.Errorf("the model returned tools but not the 'submit_plan' tool")
}
