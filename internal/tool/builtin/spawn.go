package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/wunderpus/wunderpus/internal/subagent"
	"github.com/wunderpus/wunderpus/internal/tool"
)

// SpawnTool spawns a new sub-agent for long-running tasks
type SpawnTool struct {
	subAgentMgr *subagent.Manager
}

// NewSpawnTool creates a new spawn tool
func NewSpawnTool(subAgentMgr *subagent.Manager) *SpawnTool {
	return &SpawnTool{
		subAgentMgr: subAgentMgr,
	}
}

// Name returns the tool name
func (t *SpawnTool) Name() string {
	return "spawn"
}

// Description returns the tool description
func (t *SpawnTool) Description() string {
	return "Spawns a new independent sub-agent to execute a task asynchronously. Use this for long-running tasks that should run in the background. Returns the sub-agent ID immediately, use message tool to communicate or get results."
}

// Parameters returns the tool parameters
func (t *SpawnTool) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{
			Name:        "task",
			Type:        "string",
			Description: "The task description for the sub-agent to execute",
			Required:    true,
		},
		{
			Name:        "system_prompt",
			Type:        "string",
			Description: "Optional custom system prompt for the sub-agent",
			Required:    false,
		},
	}
}

// Execute runs the spawn tool
func (t *SpawnTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	task, ok := args["task"].(string)
	if !ok || task == "" {
		return &tool.Result{Error: "task is required"}, nil
	}

	systemPrompt := ""
	if sp, ok := args["system_prompt"].(string); ok {
		systemPrompt = sp
	}

	sub, err := t.subAgentMgr.Spawn(ctx, task, systemPrompt)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("failed to spawn sub-agent: %v", err)}, nil
	}

	return &tool.Result{
		Output: fmt.Sprintf("Sub-agent spawned with ID: %s\nTask: %s\nUse 'message' tool to communicate or get results.", sub.ID[:8], task),
	}, nil
}

// Sensitive returns whether this tool requires approval
func (t *SpawnTool) Sensitive() bool {
	return false
}

// ApprovalLevel returns the policy-based approval level for this tool.
func (t *SpawnTool) ApprovalLevel() tool.ApprovalLevel { return tool.RequiresApproval }

// Version returns the tool version
func (t *SpawnTool) Version() string {
	return "1.0.0"
}

// Dependencies returns tool dependencies
func (t *SpawnTool) Dependencies() []string {
	return nil
}

// MessageTool allows communication with a sub-agent
type MessageTool struct {
	subAgentMgr *subagent.Manager
}

// NewMessageTool creates a new message tool
func NewMessageTool(subAgentMgr *subagent.Manager) *MessageTool {
	return &MessageTool{
		subAgentMgr: subAgentMgr,
	}
}

// Name returns the tool name
func (t *MessageTool) Name() string {
	return "message"
}

// Description returns the tool description
func (t *MessageTool) Description() string {
	return "Send a message to a running sub-agent or get its status/result. Use the sub-agent ID returned from spawn."
}

// Parameters returns the tool parameters
func (t *MessageTool) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{
			Name:        "subagent_id",
			Type:        "string",
			Description: "The ID of the sub-agent (first 8 characters from spawn response)",
			Required:    true,
		},
		{
			Name:        "message",
			Type:        "string",
			Description: "Message to send to the sub-agent (optional - omit to just get status)",
			Required:    false,
		},
		{
			Name:        "wait",
			Type:        "boolean",
			Description: "Wait for sub-agent to complete and return result (default: false)",
			Required:    false,
		},
	}
}

// Execute runs the message tool
func (t *MessageTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	subID, ok := args["subagent_id"].(string)
	if !ok || subID == "" {
		return &tool.Result{Error: "subagent_id is required"}, nil
	}

	// Find the subagent by ID prefix
	var sub *subagent.SubAgent
	for _, s := range t.subAgentMgr.List() {
		if len(s.ID) >= 8 && s.ID[:8] == subID {
			sub = s
			break
		}
	}

	if sub == nil {
		return &tool.Result{Error: fmt.Sprintf("sub-agent not found: %s", subID)}, nil
	}

	status := sub.GetStatus()
	result := sub.GetResult()
	errMsg := sub.GetError()

	// Check if we should wait for completion
	wait, _ := args["wait"].(bool)

	// If wait is true and agent is still running, wait for it
	if wait && status == subagent.StatusRunning {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(5 * time.Minute) // 5 minute default timeout
		for {
			select {
			case <-ticker.C:
				status = sub.GetStatus()
				result = sub.GetResult()
				errMsg = sub.GetError()

				if status != subagent.StatusRunning {
					goto doneWaiting
				}
			case <-timeout:
				return &tool.Result{Error: "wait timeout - sub-agent still running"}, nil
			case <-ctx.Done():
				return &tool.Result{Error: "context cancelled"}, nil
			}
		}
	doneWaiting:
	}

	// If no message provided, just return status
	message, hasMessage := args["message"].(string)
	if !hasMessage || message == "" {
		// Return status
		output := fmt.Sprintf("Sub-agent %s - Status: %s\nTask: %s", subID, status, sub.GetTask())
		if status == subagent.StatusCompleted {
			output += fmt.Sprintf("\n\nResult:\n%s", result)
		} else if status == subagent.StatusFailed {
			output += fmt.Sprintf("\n\nError: %s", errMsg)
		}
		return &tool.Result{Output: output}, nil
	}

	// Send message
	response, err := t.subAgentMgr.SendMessage(ctx, subID, message)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("failed to send message: %v", err)}, nil
	}

	return &tool.Result{Output: response}, nil
}

// Sensitive returns whether this tool requires approval
func (t *MessageTool) Sensitive() bool {
	return false
}

// ApprovalLevel returns the policy-based approval level for this tool.
func (t *MessageTool) ApprovalLevel() tool.ApprovalLevel { return tool.AutoExecute }

// Version returns the tool version
func (t *MessageTool) Version() string {
	return "1.0.0"
}

// Dependencies returns tool dependencies
func (t *MessageTool) Dependencies() []string {
	return nil
}
