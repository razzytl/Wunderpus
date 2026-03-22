package ags

import (
	"context"
	"testing"
)

func TestGoalExecutor_SelectNext(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	// Create 3 pending goals
	for i := 0; i < 3; i++ {
		g := NewGoal("Goal", "desc", 2, GoalBeUseful.Title, nil, []string{"done"}, 0.5)
		store.Save(g)
	}

	executor := NewGoalExecutor(store, scorer, nil, nil, nil)
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

func TestGoalExecutor_ExecuteSuccess(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	g := NewGoal("Simple goal", "do something", 3, GoalBeUseful.Title,
		nil, []string{"task completed"}, 0.8)
	store.Save(g)

	mockLLM := func(ctx context.Context, sys, usr string) (string, error) {
		return `[{"step_num":1,"description":"test task","tool":"calculator","parameters":{},"expected_outcome":"done","depends_on":[]}]`, nil
	}

	mockTaskExec := func(ctx context.Context, task TaskBlueprint) (string, error) {
		return "task completed successfully", nil
	}

	mockJudge := func(ctx context.Context, criteria []string, outcomes []string) (bool, float64) {
		return true, 0.85
	}

	executor := NewGoalExecutor(store, scorer, mockLLM, mockTaskExec, mockJudge)

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
	if updated.ActualValue == nil || *updated.ActualValue != 0.85 {
		t.Fatalf("expected actual value 0.85, got %v", updated.ActualValue)
	}
	if updated.CompletedAt == nil {
		t.Fatal("completed_at should be set")
	}
}

func TestGoalExecutor_ExecuteAbandonAfter3Failures(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	g := NewGoal("Failing goal", "will fail", 2, GoalBeUseful.Title,
		nil, []string{"impossible"}, 0.5)
	store.Save(g)

	mockLLM := func(ctx context.Context, sys, usr string) (string, error) {
		return `[{"step_num":1,"description":"fail task","tool":"test","parameters":{},"expected_outcome":"fail","depends_on":[]}]`, nil
	}

	mockTaskExec := func(ctx context.Context, task TaskBlueprint) (string, error) {
		return "task result", nil // succeeds but...
	}

	mockJudge := func(ctx context.Context, criteria []string, outcomes []string) (bool, float64) {
		return false, 0.0 // ...judge always says fail
	}

	executor := NewGoalExecutor(store, scorer, mockLLM, mockTaskExec, mockJudge)

	// Execute 3 times
	for i := 0; i < 3; i++ {
		tasks, _ := executor.Decompose(context.Background(), g)
		executor.Execute(context.Background(), g, tasks)
		g, _ = store.GetByID(g.ID)
	}

	if g.Status != GoalStatusAbandoned {
		t.Fatalf("expected abandoned after 3 failures, got %s", g.Status)
	}
	if g.AttemptCount != 3 {
		t.Fatalf("expected 3 attempts, got %d", g.AttemptCount)
	}
}

func TestGoalExecutor_ExecuteRetryOnFailure(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	g := NewGoal("Retry goal", "will fail once", 2, GoalBeUseful.Title,
		nil, []string{"done"}, 0.5)
	store.Save(g)

	mockLLM := func(ctx context.Context, sys, usr string) (string, error) {
		return `[{"step_num":1,"description":"task","tool":"test","parameters":{},"expected_outcome":"ok","depends_on":[]}]`, nil
	}

	mockTaskExec := func(ctx context.Context, task TaskBlueprint) (string, error) {
		return "ok", nil
	}

	mockJudge := func(ctx context.Context, criteria []string, outcomes []string) (bool, float64) {
		return false, 0.0 // fail
	}

	executor := NewGoalExecutor(store, scorer, mockLLM, mockTaskExec, mockJudge)

	tasks, _ := executor.Decompose(context.Background(), g)
	executor.Execute(context.Background(), g, tasks)

	updated, _ := store.GetByID(g.ID)
	if updated.Status != GoalStatusPending {
		t.Fatalf("expected pending after first failure, got %s", updated.Status)
	}
	if updated.AttemptCount != 1 {
		t.Fatalf("expected 1 attempt, got %d", updated.AttemptCount)
	}
}

func TestGoalExecutor_SelectNextNilWhenEmpty(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()

	executor := NewGoalExecutor(store, NewPriorityScorer(), nil, nil, nil)
	next, _ := executor.SelectNext()
	if next != nil {
		t.Fatal("expected nil when no pending goals")
	}
}
