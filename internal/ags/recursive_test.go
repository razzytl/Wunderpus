package ags

import (
	"context"
	"testing"
)

func TestRecursiveLoop_GeneratesRSIGoalsFromWeakness(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer(nil, nil)
	synth := NewGoalSynthesizer(nil, store, scorer, nil)

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
}

func TestRecursiveLoop_CheckAGSHealth(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer(nil, nil)

	loop := NewRecursiveLoop(scorer, store, nil)

	agsFunctions := []WeaknessCandidate{
		{FunctionName: "GoalSynthesizer", Score: 0.8, PrimaryReason: "latency"},
		{FunctionName: "PriorityScorer", Score: 0.3, PrimaryReason: "complexity"}, // below threshold
	}

	proposals := loop.CheckAGSHealth(agsFunctions)

	if len(proposals) != 1 {
		t.Fatalf("expected 1 RSI proposal for AGS, got %d", len(proposals))
	}
}
