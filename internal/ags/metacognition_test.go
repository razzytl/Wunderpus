package ags

import (
	"testing"
	"time"
)

func TestMetacognition_ReduceImpactWhenOverestimated(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	// Inject 20 completed goals where ExpectedValue=0.9 but ActualValue avg=0.4
	for i := 0; i < 20; i++ {
		g := NewGoal("Overestimated goal", "desc", 2, GoalBeUseful.Title,
			nil, []string{"done"}, 0.9)
		g.Status = GoalStatusCompleted
		now := time.Now().UTC()
		g.CompletedAt = &now
		v := 0.4
		g.ActualValue = &v
		store.Save(g)
	}

	meta := NewMetacognitionLoop(store, scorer)
	report, err := meta.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Impact weight should have decreased (values are overestimated)
	if report.NewWeights.Impact >= report.OldWeights.Impact {
		t.Fatalf("expected impact weight to decrease: old=%.4f new=%.4f",
			report.OldWeights.Impact, report.NewWeights.Impact)
	}

	// Weights should still sum to ~1.0
	if !report.NewWeights.Validate() {
		t.Fatalf("new weights should sum to 1.0, got %.4f",
			report.NewWeights.Urgency+report.NewWeights.Impact+report.NewWeights.Feasibility+
				report.NewWeights.Novelty+report.NewWeights.Alignment)
	}

	t.Logf("Impact: %.4f → %.4f (delta=%.4f)",
		report.OldWeights.Impact, report.NewWeights.Impact, report.WeightDelta.Impact)
}

func TestMetacognition_NoSuddenWeightCollapse(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	// Extreme case: all goals way overestimated
	for i := 0; i < 50; i++ {
		g := NewGoal("Way off", "desc", 2, GoalBeUseful.Title, nil, []string{"done"}, 0.99)
		g.Status = GoalStatusCompleted
		now := time.Now().UTC()
		g.CompletedAt = &now
		v := 0.01
		g.ActualValue = &v
		store.Save(g)
	}

	meta := NewMetacognitionLoop(store, scorer)
	report, _ := meta.Run()

	// Weight delta should be bounded by maxDelta (0.05)
	if report.WeightDelta.Impact < -0.06 {
		t.Fatalf("impact delta too large: %.4f (expected >= -0.05)", report.WeightDelta.Impact)
	}
	if report.WeightDelta.Feasibility > 0.06 {
		t.Fatalf("feasibility delta too large: %.4f (expected <= 0.05)", report.WeightDelta.Feasibility)
	}
}

func TestMetacognition_HighCompletionRate(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	// 95% completion rate — goals might be too easy
	for i := 0; i < 19; i++ {
		g := NewGoal("Easy goal", "desc", 1, GoalBeUseful.Title, nil, []string{"done"}, 0.5)
		g.Status = GoalStatusCompleted
		now := time.Now().UTC()
		g.CompletedAt = &now
		v := 0.5
		g.ActualValue = &v
		store.Save(g)
	}
	// 1 abandoned
	g := NewGoal("Failed goal", "desc", 1, GoalBeUseful.Title, nil, []string{"done"}, 0.5)
	g.Status = GoalStatusAbandoned
	store.Save(g)

	meta := NewMetacognitionLoop(store, scorer)
	report, _ := meta.Run()

	// With 95% completion, impact/novelty should increase
	if report.CompletionRate < 0.9 {
		t.Fatalf("expected completion rate > 0.9, got %.2f", report.CompletionRate)
	}
}
