package ags

import (
	"context"
	"testing"
)

func TestRecursiveLoop_GeneratesRSIGoalsFromWeakness(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()
	synth := NewGoalSynthesizer(nil, store, scorer)

	loop := NewRecursiveLoop(scorer, store, synth)

	weaknesses := []WeaknessCandidate{
		{FunctionName: "profiler.Track", Score: 0.85, PrimaryReason: "error_rate"},
		{FunctionName: "analyzer.Build", Score: 0.3, PrimaryReason: "latency"}, // below 0.7 threshold
	}

	report := &MetacognitionReport{CompletionRate: 0.5}

	goals, err := loop.RunAfterMetacognition(context.Background(), report, weaknesses)
	if err != nil {
		t.Fatalf("RunAfterMetacognition: %v", err)
	}

	if len(goals) != 1 {
		t.Fatalf("expected 1 RSI goal (only score > 0.7), got %d", len(goals))
	}

	if goals[0].Tier != 2 {
		t.Fatalf("expected Tier 2, got %d", goals[0].Tier)
	}

	if goals[0].ParentID != GoalImproveCapabilities.Title {
		t.Fatalf("expected parent 'Improve own capabilities', got '%s'", goals[0].ParentID)
	}
}

func TestRecursiveLoop_CheckAGSHealth(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	loop := NewRecursiveLoop(scorer, store, nil)

	agsFunctions := []WeaknessCandidate{
		{FunctionName: "GoalSynthesizer", Score: 0.8, PrimaryReason: "latency"},
		{FunctionName: "PriorityScorer", Score: 0.3, PrimaryReason: "complexity"}, // below threshold
		{FunctionName: "SomeOtherFunc", Score: 0.9, PrimaryReason: "error"},       // not an AGS target
	}

	proposals := loop.CheckAGSHealth(agsFunctions)

	if len(proposals) != 1 {
		t.Fatalf("expected 1 RSI proposal for AGS, got %d", len(proposals))
	}

	if proposals[0].TargetFunction != "GoalSynthesizer" {
		t.Fatalf("expected GoalSynthesizer, got %s", proposals[0].TargetFunction)
	}
}

func TestRecursiveLoop_NilReport(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	loop := NewRecursiveLoop(scorer, store, nil)

	goals, err := loop.RunAfterMetacognition(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("should not error on nil report: %v", err)
	}
	if len(goals) != 0 {
		t.Fatal("should return nil goals for nil report")
	}
}
