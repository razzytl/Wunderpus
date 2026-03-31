package subagent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/provider"
)

// Status represents the status of a subagent
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "canceled"
)

// SubAgent represents a spawned sub-agent
type SubAgent struct {
	ID          string
	SessionID   string
	Task        string
	Status      Status
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Result      string
	Error       string
	Agent       *agent.Agent
	mu          sync.RWMutex
}

// GetTask returns the task of the sub-agent
func (s *SubAgent) GetTask() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Task
}

// Manager manages sub-agent instances
type Manager struct {
	agentMgr *agent.Manager
	router   *provider.Router

	mu        sync.RWMutex
	subAgents map[string]*SubAgent
}

// NewManager creates a new sub-agent manager
func NewManager(agentMgr *agent.Manager, router *provider.Router) *Manager {
	return &Manager{
		agentMgr:  agentMgr,
		router:    router,
		subAgents: make(map[string]*SubAgent),
	}
}

// Spawn creates and starts a new sub-agent for a task
func (m *Manager) Spawn(ctx context.Context, task string, systemPrompt string) (*SubAgent, error) {
	subID := uuid.New().String()
	sessionID := fmt.Sprintf("subagent_%s", subID[:8])

	sub := &SubAgent{
		ID:        subID,
		SessionID: sessionID,
		Task:      task,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}

	// Create a new agent with independent context (no history)
	// If systemPrompt is provided, create a new agent with custom prompt
	// Otherwise, get a fresh agent with default system prompt from config
	ag := m.agentMgr.GetAgent(sessionID)
	if systemPrompt != "" {
		// Override system prompt for subagent with custom prompt
		ag = agent.NewAgent(
			m.router,
			nil, // sanitizer - inherit from parent
			nil, // audit - inherit from parent
			nil, // store - new in-memory
			nil, // registry - inherit from parent
			nil, // executor - inherit from parent
			nil, // skillsLoader - inherit from parent
			systemPrompt,
			8000, // maxContextTokens
			0.7,  // temperature
			sessionID,
		)
	}

	sub.Agent = ag
	sub.Status = StatusRunning
	now := time.Now()
	sub.StartedAt = &now

	m.mu.Lock()
	m.subAgents[subID] = sub
	m.mu.Unlock()

	slog.Info("subagent spawned", "id", subID[:8], "task", task)

	// Execute task asynchronously
	go m.executeTask(ctx, sub)

	return sub, nil
}

// SpawnWithCallback creates a subagent and calls the callback when done
func (m *Manager) SpawnWithCallback(ctx context.Context, task string, systemPrompt string, callback func(*SubAgent)) (*SubAgent, error) {
	sub, err := m.Spawn(ctx, task, systemPrompt)
	if err != nil {
		return nil, err
	}

	// Start a goroutine to watch for completion and call callback
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sub.mu.RLock()
				status := sub.Status
				sub.mu.RUnlock()

				if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
					callback(sub)
					return
				}
			case <-ctx.Done():
				m.Cancel(sub.ID)
				return
			}
		}
	}()

	return sub, nil
}

// executeTask runs the subagent task
func (m *Manager) executeTask(ctx context.Context, sub *SubAgent) {
	result, err := sub.Agent.HandleMessage(ctx, sub.Task)

	sub.mu.Lock()
	defer sub.mu.Unlock()

	if err != nil {
		sub.Status = StatusFailed
		sub.Error = err.Error()
		slog.Error("subagent task failed", "id", sub.ID[:8], "error", err)
	} else {
		sub.Status = StatusCompleted
		sub.Result = result
		slog.Info("subagent task completed", "id", sub.ID[:8])
	}

	now := time.Now()
	sub.CompletedAt = &now
}

// Get retrieves a sub-agent by ID
func (m *Manager) Get(id string) (*SubAgent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sub, ok := m.subAgents[id]
	return sub, ok
}

// List returns all sub-agents
func (m *Manager) List() []*SubAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*SubAgent, 0, len(m.subAgents))
	for _, sub := range m.subAgents {
		result = append(result, sub)
	}
	return result
}

// ListByStatus returns sub-agents filtered by status
func (m *Manager) ListByStatus(status Status) []*SubAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*SubAgent
	for _, sub := range m.subAgents {
		if sub.Status == status {
			result = append(result, sub)
		}
	}
	return result
}

// Cancel cancels a running sub-agent
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, ok := m.subAgents[id]
	if !ok {
		return fmt.Errorf("subagent not found: %s", id)
	}

	if sub.Status != StatusRunning && sub.Status != StatusPending {
		return fmt.Errorf("subagent cannot be canceled (status: %s)", sub.Status)
	}

	sub.Status = StatusCancelled
	now := time.Now()
	sub.CompletedAt = &now

	slog.Info("subagent canceled", "id", id[:8])
	return nil
}

// Remove removes a completed sub-agent from tracking
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, ok := m.subAgents[id]
	if !ok {
		return fmt.Errorf("subagent not found: %s", id)
	}

	if sub.Status == StatusRunning || sub.Status == StatusPending {
		return fmt.Errorf("cannot remove running subagent: %s", id)
	}

	delete(m.subAgents, id)
	slog.Debug("subagent removed", "id", id[:8])
	return nil
}

// GetStatus returns the current status of a sub-agent
func (s *SubAgent) GetStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// GetResult returns the result of a completed sub-agent
func (s *SubAgent) GetResult() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Result
}

// GetError returns the error of a failed sub-agent
func (s *SubAgent) GetError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Error
}

func (m *Manager) SendMessage(ctx context.Context, subID string, message string) (string, error) {
	sub, ok := m.Get(subID)
	if !ok {
		return "", fmt.Errorf("subagent not found: %s", subID)
	}

	sub.mu.RLock()
	status := sub.Status
	sub.mu.RUnlock()

	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		return "", fmt.Errorf("subagent is not running (status: %s)", status)
	}

	return sub.Agent.HandleMessage(ctx, message)
}

// GetStats returns statistics about sub-agents
func (m *Manager) GetStats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"total":     len(m.subAgents),
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
		"canceled":  0,
	}

	for _, sub := range m.subAgents {
		switch sub.Status {
		case StatusPending:
			stats["pending"]++
		case StatusRunning:
			stats["running"]++
		case StatusCompleted:
			stats["completed"]++
		case StatusFailed:
			stats["failed"]++
		case StatusCancelled:
			stats["canceled"]++
		}
	}

	return stats
}
