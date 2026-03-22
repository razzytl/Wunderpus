package agents

import (
	"context"
	"testing"
	"time"
)

func TestAgentManager_SpawnAndCollect(t *testing.T) {
	runner := func(ctx context.Context, goalID string) (*GoalResult, error) {
		time.Sleep(50 * time.Millisecond)
		return &GoalResult{
			GoalID:   goalID,
			Success:  true,
			Output:   "goal completed",
			Duration: 50 * time.Millisecond,
		}, nil
	}

	mgr := NewAgentManager(5, runner)

	config := AgentConfig{
		ID:         "agent-1",
		GoalID:     "goal-1",
		GoalTitle:  "Test Goal",
		TimeBudget: 5 * time.Second,
	}

	agent, err := mgr.Spawn(config)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	if agent.Status != AgentStatusRunning {
		t.Fatalf("expected running, got %s", agent.Status)
	}

	result, err := mgr.Collect("agent-1")
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if !result.Success {
		t.Fatal("expected success")
	}
	if result.Output != "goal completed" {
		t.Fatalf("expected 'goal completed', got '%s'", result.Output)
	}
}

func TestAgentManager_MaxAgentsEnforced(t *testing.T) {
	runner := func(ctx context.Context, goalID string) (*GoalResult, error) {
		time.Sleep(1 * time.Second)
		return &GoalResult{Success: true}, nil
	}

	mgr := NewAgentManager(2, runner) // max 2

	mgr.Spawn(AgentConfig{ID: "a1", GoalID: "g1", TimeBudget: 5 * time.Second})
	mgr.Spawn(AgentConfig{ID: "a2", GoalID: "g2", TimeBudget: 5 * time.Second})

	_, err := mgr.Spawn(AgentConfig{ID: "a3", GoalID: "g3", TimeBudget: 5 * time.Second})
	if err == nil {
		t.Fatal("should reject spawn when max agents reached")
	}

	// Cleanup running agents
	mgr.Kill("a1")
	mgr.Kill("a2")
}

func TestAgentManager_Kill(t *testing.T) {
	runner := func(ctx context.Context, goalID string) (*GoalResult, error) {
		time.Sleep(10 * time.Second) // long-running
		return &GoalResult{Success: true}, nil
	}

	mgr := NewAgentManager(5, runner)
	mgr.Spawn(AgentConfig{ID: "agent-k", GoalID: "g1", TimeBudget: 30 * time.Second})

	// Kill immediately
	err := mgr.Kill("agent-k")
	if err != nil {
		t.Fatalf("Kill: %v", err)
	}

	// Collect should return killed error
	_, err = mgr.Collect("agent-k")
	if err == nil {
		t.Fatal("should return error for killed agent")
	}
}

func TestAgentManager_Count(t *testing.T) {
	runner := func(ctx context.Context, goalID string) (*GoalResult, error) {
		return &GoalResult{Success: true}, nil
	}

	mgr := NewAgentManager(5, runner)

	mgr.Spawn(AgentConfig{ID: "c1", GoalID: "g1", TimeBudget: 5 * time.Second})
	mgr.Spawn(AgentConfig{ID: "c2", GoalID: "g2", TimeBudget: 5 * time.Second})

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	running := mgr.Count(AgentStatusRunning)
	completed := mgr.Count(AgentStatusCompleted)

	if running+completed != 2 {
		t.Fatalf("expected 2 total agents, got running=%d completed=%d", running, completed)
	}
}

func TestAgentManager_Cleanup(t *testing.T) {
	runner := func(ctx context.Context, goalID string) (*GoalResult, error) {
		return &GoalResult{Success: true}, nil
	}

	mgr := NewAgentManager(5, runner)
	mgr.Spawn(AgentConfig{ID: "cl1", GoalID: "g1", TimeBudget: 5 * time.Second})

	time.Sleep(200 * time.Millisecond)

	removed := mgr.Cleanup()
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}

	if len(mgr.List()) != 0 {
		t.Fatal("should have 0 agents after cleanup")
	}
}
