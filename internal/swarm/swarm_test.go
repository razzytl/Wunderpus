package swarm

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// --- Profile Tests ---

func TestMatchSpecialist_Research(t *testing.T) {
	name, config := MatchSpecialist("Research the latest AI trends")
	if name != "researcher" {
		t.Errorf("expected 'researcher', got %q", name)
	}
	if config.Name != "researcher" {
		t.Errorf("config name mismatch")
	}
}

func TestMatchSpecialist_Code(t *testing.T) {
	name, _ := MatchSpecialist("Code review and debug the implementation")
	if name != "coder" {
		t.Errorf("expected 'coder', got %q", name)
	}
}

func TestMatchSpecialist_Write(t *testing.T) {
	name, _ := MatchSpecialist("Write a blog post about Go")
	if name != "writer" {
		t.Errorf("expected 'writer', got %q", name)
	}
}

func TestMatchSpecialist_NoMatch(t *testing.T) {
	name, _ := MatchSpecialist("Do something completely unknown")
	if name != "" {
		t.Errorf("expected no match, got %q", name)
	}
}

func TestMatchAllSpecialists(t *testing.T) {
	// "research X and write a report" should match researcher AND writer
	matches := MatchAllSpecialists("Research the market and write a report about it")
	if len(matches) < 2 {
		t.Errorf("expected at least 2 matches, got %d: %v", len(matches), specialistNames(matches))
	}
}

func TestGetSpecialist(t *testing.T) {
	config, ok := GetSpecialist("coder")
	if !ok {
		t.Fatal("coder should exist")
	}
	if config.Name != "coder" {
		t.Errorf("expected 'coder', got %q", config.Name)
	}
	if len(config.Tools) == 0 {
		t.Error("coder should have tools")
	}
}

func TestGetSpecialist_NotFound(t *testing.T) {
	_, ok := GetSpecialist("nonexistent")
	if ok {
		t.Error("should not find nonexistent specialist")
	}
}

func TestListSpecialists(t *testing.T) {
	names := ListSpecialists()
	if len(names) != 7 {
		t.Errorf("expected 7 specialists, got %d", len(names))
	}
}

// --- Orchestrator Tests ---

func TestOrchestratorDispatch_SingleSpecialist(t *testing.T) {
	executor := func(ctx context.Context, goal Goal, config AgentConfig) (*SpecialistResult, error) {
		return &SpecialistResult{
			Output:  fmt.Sprintf("Research result for: %s", goal.Title),
			Success: true,
		}, nil
	}

	orch := NewOrchestrator(executor, nil)
	goal := Goal{ID: "g1", Title: "Research AI", Description: "Research the latest in AI technology"}

	result, err := orch.Dispatch(context.Background(), goal)
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}
	if len(result.Results) == 0 {
		t.Error("expected at least 1 result")
	}
	if result.Synthesis == "" {
		t.Error("expected synthesis output")
	}
}

func TestOrchestratorDispatch_MultipleSpecialists(t *testing.T) {
	executor := func(ctx context.Context, goal Goal, config AgentConfig) (*SpecialistResult, error) {
		return &SpecialistResult{
			Specialist: config.Name,
			Output:     fmt.Sprintf("Output from %s", config.Name),
			Success:    true,
		}, nil
	}

	orch := NewOrchestrator(executor, nil)
	goal := Goal{ID: "g2", Title: "Research and Write", Description: "Research the market and write a report"}

	result, err := orch.Dispatch(context.Background(), goal)
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}
	if len(result.Specialists) < 2 {
		t.Errorf("expected at least 2 specialists, got %d", len(result.Specialists))
	}
}

func TestOrchestratorDispatch_ExecutorError(t *testing.T) {
	executor := func(ctx context.Context, goal Goal, config AgentConfig) (*SpecialistResult, error) {
		return nil, fmt.Errorf("executor failed")
	}

	orch := NewOrchestrator(executor, nil)
	goal := Goal{ID: "g3", Title: "Will fail", Description: "Do something"}

	result, err := orch.Dispatch(context.Background(), goal)
	if err != nil {
		t.Fatalf("dispatch should not return error, got: %v", err)
	}

	if result.Success {
		t.Error("should not be successful when executor fails")
	}
}

func TestOrchestratorDispatch_NoMatch(t *testing.T) {
	executor := func(ctx context.Context, goal Goal, config AgentConfig) (*SpecialistResult, error) {
		return &SpecialistResult{Output: "default handler", Success: true}, nil
	}

	orch := NewOrchestrator(executor, nil)
	goal := Goal{ID: "g4", Title: "Random task", Description: "xyzzy unknown task type"}

	result, err := orch.Dispatch(context.Background(), goal)
	if err != nil {
		t.Fatalf("should fallback to operator: %v", err)
	}

	// Should use default operator
	if len(result.Specialists) != 1 {
		t.Errorf("expected 1 specialist (default), got %d", len(result.Specialists))
	}
}

func TestOrchestratorDispatch_WithSynthesizer(t *testing.T) {
	executor := func(ctx context.Context, goal Goal, config AgentConfig) (*SpecialistResult, error) {
		return &SpecialistResult{Output: "specialist output", Success: true}, nil
	}
	synthesizer := func(ctx context.Context, goal Goal, results []SpecialistResult) (string, error) {
		return "Synthesized: " + results[0].Output, nil
	}

	orch := NewOrchestrator(executor, synthesizer)
	goal := Goal{ID: "g5", Title: "Research", Description: "Research AI trends"}

	result, err := orch.Dispatch(context.Background(), goal)
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}

	if result.Synthesis != "Synthesized: specialist output" {
		t.Errorf("unexpected synthesis: %q", result.Synthesis)
	}
}

func TestOrchestratorDispatch_ParallelExecution(t *testing.T) {
	callOrder := []string{}
	var mu sync.Mutex

	executor := func(ctx context.Context, goal Goal, config AgentConfig) (*SpecialistResult, error) {
		mu.Lock()
		callOrder = append(callOrder, config.Name)
		mu.Unlock()
		return &SpecialistResult{Output: "done", Success: true}, nil
	}

	orch := NewOrchestrator(executor, nil)
	goal := Goal{ID: "g6", Title: "Research and Write", Description: "Research data and write article"}

	_, _ = orch.Dispatch(context.Background(), goal)

	// Both should have been called
	if len(callOrder) < 2 {
		t.Errorf("expected parallel execution of at least 2 specialists, got %d", len(callOrder))
	}
}

// --- Integration Tests ---

func TestInitSwarm(t *testing.T) {
	system, err := InitSwarm(Config{Enabled: true}, func(ctx context.Context, goal Goal, config AgentConfig) (*SpecialistResult, error) {
		return &SpecialistResult{Success: true}, nil
	}, nil)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if system == nil {
		t.Error("system should not be nil")
	}
	if system.Orchestrator == nil {
		t.Error("orchestrator should not be nil")
	}
}

func TestInitSwarm_Disabled(t *testing.T) {
	system, err := InitSwarm(Config{Enabled: false}, nil, nil)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if system != nil {
		t.Error("system should be nil when disabled")
	}
}
