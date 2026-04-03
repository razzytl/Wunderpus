package ags

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/wunderpus/wunderpus/internal/provider"
)

func TestGoalSynthesizer_Deduplicate(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()

	// Save an existing active goal
	existing := NewGoal("Improve browser reliability", "desc", 2, GoalBeUseful.Title, nil, nil, 0.8)
	existing.Status = GoalStatusActive
	store.Save(existing)

	scorer := NewPriorityScorer(nil)

	p := &mockProvider{
		completeFn: func(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
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
			return &provider.CompletionResponse{Content: string(b)}, nil
		},
	}

	synth := NewGoalSynthesizer(p, store, scorer, nil)

	// Pass memories with repeated errors
	memories := []MemoryEntry{
		{EventType: "test_fail", Content: "fail", Timestamp: time.Now(), Success: false},
		{EventType: "test_fail", Content: "fail", Timestamp: time.Now(), Success: false},
		{EventType: "test_fail", Content: "fail", Timestamp: time.Now(), Success: false},
	}

	goals, err := synth.Synthesize(context.Background(), memories, nil)
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}

	if len(goals) != 1 {
		t.Fatalf("expected 1 goal after dedup, got %d", len(goals))
	}
}

func TestGoalSynthesizer_MaxPerCycle(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()
	scorer := NewPriorityScorer(nil)

	p := &mockProvider{
		completeFn: func(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
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
			return &provider.CompletionResponse{Content: string(b)}, nil
		},
	}

	synth := NewGoalSynthesizer(p, store, scorer, nil)

	memories := make([]MemoryEntry, 5)
	for i := range memories {
		memories[i] = MemoryEntry{EventType: "err", Content: "fail", Timestamp: time.Now(), Success: false}
	}

	goals, _ := synth.Synthesize(context.Background(), memories, nil)

	if len(goals) != 5 {
		t.Fatalf("expected exactly 5 goals (cap), got %d", len(goals))
	}
}
