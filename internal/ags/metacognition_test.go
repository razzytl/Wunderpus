package ags

import (
	"testing"
	"time"
)

func TestMetacognition_ReduceImpactWhenOverestimated(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer(nil, nil)

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
}

func TestMetacognition_NoSuddenWeightCollapse(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer(nil, nil)

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

	// Weight delta should be bounded
	if report.WeightDelta.Impact < -0.06 {
		t.Fatalf("impact delta too large: %.4f", report.WeightDelta.Impact)
	}
}
