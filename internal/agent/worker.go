package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wunderpus/wunderpus/internal/provider"
	"github.com/wunderpus/wunderpus/internal/tool"
)

// TaskType constants correspond to the types emitted by the Planner.
const (
	TaskTypeIO      = "io"
	TaskTypeCompute = "compute"
	TaskTypeGeneral = "general"
)

// WorkerArm represents an isolated execution unit for a specific subtask.
type WorkerArm struct {
	id        string
	taskType  string
	router    *provider.Router
	executor  *tool.Executor
	registry  *tool.Registry // A scoped registry just for this worker
}

// NewWorkerArm creates a new specialized worker arm.
func NewWorkerArm(id, taskType string, router *provider.Router, globalRegistry *tool.Registry, globalExecutor *tool.Executor) *WorkerArm {
	// Create a scoped registry that only contains tools relevant to the task type
	scopedRegistry := tool.NewRegistry()
	
	if globalRegistry != nil {
		allTools := globalRegistry.List()
		for _, t := range allTools {
			if shouldIncludeTool(taskType, t.Name()) {
				scopedRegistry.Register(t)
			}
		}
	}

	return &WorkerArm{
		id:       id,
		taskType: taskType,
		router:   router,
		executor: globalExecutor,
		registry: scopedRegistry,
	}
}

// shouldIncludeTool isolates tools to specific worker arms to prevent lateral drift.
func shouldIncludeTool(taskType, toolName string) bool {
	switch taskType {
	case TaskTypeIO:
		// I/O arm only gets HTTP tools (could include database reads later)
		return strings.HasPrefix(toolName, "http_")
	case TaskTypeCompute:
		// Compute arm only gets the calculator or other math/logic tools
		return toolName == "calculator"
	case TaskTypeGeneral:
		// General arm gets file operations and shell
		return strings.HasPrefix(toolName, "file_") || strings.HasPrefix(toolName, "shell_")
	default:
		// Unknown task type gets no tools by default for safety
		return false
	}
}

// ExecuteSubtask takes a Subtask and dependent results, and executes it using an LLM.
func (w *WorkerArm) ExecuteSubtask(ctx context.Context, subtask Subtask, contextData string) (string, error) {
	sysPrompt := fmt.Sprintf(`You are a specialized %s worker. 
Your objective: %s
Any relevant context provided from prior dependencies: %s

Think step by step and execute any tools required to achieve this objective. Output the final summary of what you accomplished.`, 
		w.taskType, subtask.Description, contextData)

	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: sysPrompt},
		{Role: provider.RoleUser, Content: "Begin."},
	}

	// We run a localized loop similar to the Agent loop, but strictly for this subtask
	maxIterations := 5
	for i := 0; i < maxIterations; i++ {
		prov := w.router.Active()
		req := &provider.CompletionRequest{
			Messages:    messages,
			Temperature: 0.1, // Workers should be deterministic
		}

		if w.registry != nil && w.registry.Count() > 0 {
			toolSchemas := w.registry.Schemas()
			req.Tools = make([]provider.ToolSchema, len(toolSchemas))
			for j, ts := range toolSchemas {
				req.Tools[j] = provider.ToolSchema{
					Type: ts.Type,
					Function: provider.FunctionSchema{
						Name:        ts.Function.Name,
						Description: ts.Function.Description,
						Parameters:  ts.Function.Parameters,
					},
				}
			}
		}

		resp, err := prov.Complete(ctx, req)
		if err != nil {
			return "", fmt.Errorf("worker %s error: %w", w.id, err)
		}

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		// Add assistant response to localized memory
		messages = append(messages, provider.Message{
			Role:      provider.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			var args map[string]any
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

			toolCall := tool.ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: args,
			}

			var outputStr string
			// Secondary sanity check: ensure the tool actually exists in this worker's registry
			if _, ok := w.registry.Get(tc.Function.Name); !ok {
				outputStr = fmt.Sprintf("Error: tool %s is not permitted in the %s worker arm", tc.Function.Name, w.taskType)
			} else {
				res := w.executor.Execute(ctx, toolCall)
				outputStr = res.Output
				if res.Error != "" {
					outputStr = "Error: " + res.Error
				}
			}

			// Feed tool explicit result back
			messages = append(messages, provider.Message{
				Role:       provider.RoleTool,
				Content:    outputStr,
				ToolCallID: tc.ID,
			})
		}
	}

	return "", fmt.Errorf("worker %s hit loop limits without concluding", w.id)
}
