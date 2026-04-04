package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// AgentStatus is the lifecycle state of a sub-agent.
type AgentStatus string

const (
	AgentStatusRunning   AgentStatus = "running"
	AgentStatusCompleted AgentStatus = "completed"
	AgentStatusFailed    AgentStatus = "failed"
	AgentStatusKilled    AgentStatus = "killed"
)

// AgentConfig specifies how to spawn a sub-agent.
type AgentConfig struct {
	ID         string
	GoalID     string
	GoalTitle  string
	TimeBudget time.Duration
}

// SubAgent represents a running sub-agent instance.
type SubAgent struct {
	Config    AgentConfig
	Status    AgentStatus
	StartTime time.Time
	EndTime   *time.Time
	Result    *GoalResult
	Error     error
	cancel    context.CancelFunc // cancels the goroutine
}

// GoalResult contains the outcome of a sub-agent's work.
type GoalResult struct {
	GoalID   string
	Success  bool
	Output   string
	Duration time.Duration
}

// TaskRunnerFn is the function a sub-agent runs to accomplish its goal.
type TaskRunnerFn func(ctx context.Context, goalID string) (*GoalResult, error)

// AgentManager spawns and manages sub-agents that can work on goals independently.
type AgentManager struct {
	agents    map[string]*SubAgent
	mu        sync.RWMutex
	maxAgents int
	runner    TaskRunnerFn
}

// NewAgentManager creates a new agent manager.
func NewAgentManager(maxAgents int, runner TaskRunnerFn) *AgentManager {
	return &AgentManager{
		agents:    make(map[string]*SubAgent),
		maxAgents: maxAgents,
		runner:    runner,
	}
}

// Spawn starts a new sub-agent for the given goal configuration.
func (m *AgentManager) Spawn(config AgentConfig) (*SubAgent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.agents) >= m.maxAgents {
		return nil, fmt.Errorf("agents: max agent count reached (%d)", m.maxAgents)
	}

	if _, exists := m.agents[config.ID]; exists {
		return nil, fmt.Errorf("agents: agent %s already exists", config.ID)
	}

	agent := &SubAgent{
		Config:    config,
		Status:    AgentStatusRunning,
		StartTime: time.Now().UTC(),
	}
	m.agents[config.ID] = agent

	// Run the agent in a goroutine with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.TimeBudget)

	agent.cancel = cancel

	go func() {
		defer cancel()

		slog.Info("agents: sub-agent started", "id", config.ID, "goal", config.GoalTitle)

		result, err := m.runner(ctx, config.GoalID)

		m.mu.Lock()
		defer m.mu.Unlock()

		// Only update if still running (not already killed)
		if agent.Status == AgentStatusKilled {
			return // killed externally, don't overwrite status
		}

		now := time.Now().UTC()
		agent.EndTime = &now

		if err != nil {
			agent.Status = AgentStatusFailed
			agent.Error = err
			slog.Warn("agents: sub-agent failed", "id", config.ID, "error", err)
		} else {
			agent.Status = AgentStatusCompleted
			agent.Result = result
			slog.Info("agents: sub-agent completed", "id", config.ID, "duration", now.Sub(agent.StartTime))
		}
	}()

	return agent, nil
}

// Collect waits for a sub-agent to complete and returns its result.
func (m *AgentManager) Collect(agentID string) (*GoalResult, error) {
	// Poll until the agent is no longer running
	for {
		m.mu.RLock()
		agent, ok := m.agents[agentID]
		m.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("agents: agent %s not found", agentID)
		}

		switch agent.Status {
		case AgentStatusCompleted:
			return agent.Result, nil
		case AgentStatusFailed:
			return nil, fmt.Errorf("agents: agent %s failed: %w", agentID, agent.Error)
		case AgentStatusKilled:
			return nil, fmt.Errorf("agents: agent %s was killed (exceeded budget)", agentID)
		case AgentStatusRunning:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Kill forcefully terminates a sub-agent by canceling its context.
func (m *AgentManager) Kill(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[agentID]
	if !ok {
		return fmt.Errorf("agents: agent %s not found", agentID)
	}

	if agent.Status == AgentStatusRunning {
		agent.Status = AgentStatusKilled
		now := time.Now().UTC()
		agent.EndTime = &now

		// Cancel the goroutine's context to actually stop it
		if agent.cancel != nil {
			agent.cancel()
		}

		slog.Warn("agents: sub-agent killed", "id", agentID)
	}

	return nil
}

// List returns all managed sub-agents.
func (m *AgentManager) List() []*SubAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*SubAgent, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, a)
	}
	return result
}

// Count returns the number of agents in the given status.
func (m *AgentManager) Count(status AgentStatus) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, a := range m.agents {
		if a.Status == status {
			count++
		}
	}
	return count
}

// Cleanup removes completed/failed/killed agents from the manager.
func (m *AgentManager) Cleanup() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := 0
	for id, a := range m.agents {
		if a.Status != AgentStatusRunning {
			delete(m.agents, id)
			removed++
		}
	}
	return removed
}
