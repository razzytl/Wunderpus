package ags

import (
	"context"
	"testing"

	"github.com/wunderpus/wunderpus/internal/provider"
)

func TestGoalExecutor_SelectNext(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer(nil)

	// Create 3 pending goals
	for i := 0; i < 3; i++ {
		g := NewGoal("Goal", "desc", 2, GoalBeUseful.Title, nil, []string{"done"}, 0.5)
		store.Save(g)
	}

	executor := NewGoalExecutor(store, scorer, nil, nil, nil, nil)
	next, err := executor.SelectNext()
	if err != nil {
		t.Fatalf("SelectNext: %v", err)
	}

	if next == nil {
		t.Fatal("expected a goal, got nil")
	}
	if next.Status != GoalStatusPending {
		t.Fatalf("expected pending, got %s", next.Status)
	}
}

type mockProvider struct {
	completeFn func(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error)
}

func (m *mockProvider) Name() string { return "mock" }

func (m *mockProvider) Complete(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return m.completeFn(ctx, req)
}

func (m *mockProvider) Stream(ctx context.Context, req *provider.CompletionRequest) (<-chan provider.StreamChunk, error) {
	return nil, nil
}

func TestGoalExecutor_ExecuteSuccess(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer(nil)

	g := NewGoal("Simple goal", "do something", 3, GoalBeUseful.Title,
		nil, []string{"task completed"}, 0.8)
	store.Save(g)

	p := &mockProvider{
		completeFn: func(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
			return &provider.CompletionResponse{
				Content: `[{"step_num":1,"description":"test task","tool":"calculator","parameters":{},"expected_outcome":"done","depends_on":[]}]`,
			}, nil
		},
	}

	mockTaskExec := func(ctx context.Context, task TaskBlueprint) (string, error) {
		return "task completed successfully", nil
	}

	mockJudge := func(ctx context.Context, criteria []string, outcomes []string) (bool, float64) {
		return true, 0.85
	}

	executor := NewGoalExecutor(store, scorer, p, nil, mockTaskExec, mockJudge)

	tasks, err := executor.Decompose(context.Background(), g)
	if err != nil {
		t.Fatalf("Decompose: %v", err)
	}

	err = executor.Execute(context.Background(), g, tasks)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Verify goal is completed
	updated, _ := store.GetByID(g.ID)
	if updated.Status != GoalStatusCompleted {
		t.Fatalf("expected completed, got %s", updated.Status)
	}
}

func TestGoalExecutor_ExecuteAbandonAfter3Failures(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer(nil)

	g := NewGoal("Failing goal", "will fail", 2, GoalBeUseful.Title,
		nil, []string{"impossible"}, 0.5)
	store.Save(g)

	p := &mockProvider{
		completeFn: func(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
			return &provider.CompletionResponse{
				Content: `[{"step_num":1,"description":"fail task","tool":"test","parameters":{},"expected_outcome":"fail","depends_on":[]}]`,
			}, nil
		},
	}

	mockTaskExec := func(ctx context.Context, task TaskBlueprint) (string, error) {
		return "task result", nil
	}

	mockJudge := func(ctx context.Context, criteria []string, outcomes []string) (bool, float64) {
		return false, 0.0
	}

	executor := NewGoalExecutor(store, scorer, p, nil, mockTaskExec, mockJudge)

	// Execute via loop-like logic
	for i := 0; i < 3; i++ {
		// Manual orchestration of loop logic
		goal, _ := executor.SelectNext()
		goal.AttemptCount++
		store.Update(*goal)
		tasks, _ := executor.Decompose(context.Background(), *goal)
		executor.Execute(context.Background(), *goal, tasks)
	}

	updated, _ := store.GetByID(g.ID)
	if updated.Status != GoalStatusAbandoned {
		t.Fatalf("expected abandoned, got %s", updated.Status)
	}
}
