package ags

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestGoalSynthesizer_Deduplicate(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()

	// Save an existing active goal
	existing := NewGoal("Improve browser reliability", "desc", 2, GoalBeUseful.Title, nil, nil, 0.8)
	existing.Status = GoalStatusActive
	store.Save(existing)

	scorer := NewPriorityScorer()

	// Mock LLM that proposes the same goal again + a new one
	mockLLM := func(ctx context.Context, system, user string) (string, error) {
		resp := ProposedGoalsResponse{
			ProposedGoals: []ProposedGoal{
				{
					Title:           "Improve browser reliability", // duplicate!
					Description:     "Same goal again",
					Tier:            2,
					Evidence:        []string{"memory"},
					ParentTier0:     GoalBeUseful.Title,
					ExpectedValue:   0.7,
					SuccessCriteria: []string{"works"},
				},
				{
					Title:           "New unique goal",
					Description:     "A genuinely new goal",
					Tier:            2,
					Evidence:        []string{"pattern"},
					ParentTier0:     GoalExpandKnowledge.Title,
					ExpectedValue:   0.6,
					SuccessCriteria: []string{"done"},
				},
			},
		}
		b, _ := json.Marshal(resp)
		return string(b), nil
	}

	synth := NewGoalSynthesizer(mockLLM, store, scorer)

	// Pass memories with repeated errors so synthesizer doesn't bail early
	memories := []MemoryEntry{
		{EventType: "test_fail", Content: "fail", Timestamp: time.Now(), Success: false},
		{EventType: "test_fail", Content: "fail", Timestamp: time.Now(), Success: false},
		{EventType: "test_fail", Content: "fail", Timestamp: time.Now(), Success: false},
	}

	goals, err := synth.Synthesize(context.Background(), memories, nil)
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}

	// Should only return 1 goal (the non-duplicate)
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal after dedup, got %d", len(goals))
	}
	if goals[0].Title != "New unique goal" {
		t.Fatalf("expected 'New unique goal', got '%s'", goals[0].Title)
	}
}

func TestGoalSynthesizer_MaxPerCycle(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	// Mock LLM that proposes 10 goals
	mockLLM := func(ctx context.Context, system, user string) (string, error) {
		var goals []ProposedGoal
		for i := 0; i < 10; i++ {
			goals = append(goals, ProposedGoal{
				Title:           "Goal " + string(rune('A'+i)),
				Description:     "desc",
				Tier:            2,
				Evidence:        []string{"e"},
				ParentTier0:     GoalBeUseful.Title,
				ExpectedValue:   0.5,
				SuccessCriteria: []string{"s"},
			})
		}
		b, _ := json.Marshal(ProposedGoalsResponse{ProposedGoals: goals})
		return string(b), nil
	}

	synth := NewGoalSynthesizer(mockLLM, store, scorer)

	// Pass memories with >= 3 errors so Synthesize doesn't bail early
	memories := make([]MemoryEntry, 5)
	for i := range memories {
		memories[i] = MemoryEntry{EventType: "err", Content: "fail", Timestamp: time.Now(), Success: false}
	}

	goals, _ := synth.Synthesize(context.Background(), memories, nil)

	// Should be capped at 5
	if len(goals) > 5 {
		t.Fatalf("expected max 5 goals per cycle, got %d", len(goals))
	}
	if len(goals) != 5 {
		t.Fatalf("expected exactly 5 goals (10 proposed, 5 cap), got %d", len(goals))
	}
}

func TestGoalSynthesizer_MemoryPatternDetection(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer()

	// Inject 10 identical browser_timeout errors
	memories := make([]MemoryEntry, 15)
	for i := range memories {
		memories[i] = MemoryEntry{
			EventType: "browser_timeout",
			Content:   "timeout waiting for page load",
			Timestamp: time.Now(),
			Duration:  30 * time.Second,
			Success:   false,
		}
	}

	mockLLM := func(ctx context.Context, system, user string) (string, error) {
		// Check that findings mention browser_timeout
		resp := ProposedGoalsResponse{
			ProposedGoals: []ProposedGoal{
				{
					Title:           "Fix browser timeout reliability",
					Description:     "Address repeated browser timeouts",
					Tier:            2,
					Evidence:        []string{"browser_timeout occurred 15 times"},
					ParentTier0:     GoalMaintainContinuity.Title,
					ExpectedValue:   0.8,
					SuccessCriteria: []string{"zero browser timeouts in 24h"},
				},
			},
		}
		b, _ := json.Marshal(resp)
		return string(b), nil
	}

	synth := NewGoalSynthesizer(mockLLM, store, scorer)
	goals, err := synth.Synthesize(context.Background(), memories, nil)
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}

	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}
	if goals[0].Title != "Fix browser timeout reliability" {
		t.Fatalf("expected browser reliability goal, got '%s'", goals[0].Title)
	}
}
