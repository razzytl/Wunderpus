package heartbeat

import (
	"context"
	"log/slog"

	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/subagent"
)

// HeartbeatExecutor executes heartbeat tasks using the agent manager
type HeartbeatExecutor struct {
	agentMgr    *agent.Manager
	subAgentMgr *subagent.Manager
}

// NewHeartbeatExecutor creates a new heartbeat executor
func NewHeartbeatExecutor(agentMgr *agent.Manager, subAgentMgr *subagent.Manager) *HeartbeatExecutor {
	return &HeartbeatExecutor{
		agentMgr:    agentMgr,
		subAgentMgr: subAgentMgr,
	}
}

// ExecuteQuickTask executes a quick task immediately
func (e *HeartbeatExecutor) ExecuteQuickTask(ctx context.Context, task HeartbeatTask) (string, error) {
	slog.Info("executing quick heartbeat task", "task", task.Content)

	// Get or create a session for heartbeat tasks
	sessionID := "heartbeat_quick"
	ag := e.agentMgr.GetAgent(sessionID)

	result, err := ag.HandleMessage(ctx, task.Content)
	if err != nil {
		return "", err
	}

	return result, nil
}

// ExecuteLongTask executes a long task by spawning a subagent
func (e *HeartbeatExecutor) ExecuteLongTask(ctx context.Context, task HeartbeatTask) (string, error) {
	slog.Info("spawning subagent for long heartbeat task", "task", task.Content)

	// Spawn a subagent for long-running tasks
	sub, err := e.subAgentMgr.Spawn(ctx, task.Content, "")
	if err != nil {
		return "", err
	}

	// Return the subagent ID so the task can be tracked
	return "Sub-agent spawned: " + sub.ID[:8], nil
}

// Ensure HeartbeatExecutor implements TaskExecutor
var _ TaskExecutor = (*HeartbeatExecutor)(nil)
