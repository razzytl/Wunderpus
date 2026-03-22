package ags

import (
	"path/filepath"
	"testing"
	"time"
)

func tempGoalDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test_goals.db")
}

func TestGoalStore_SaveAndRetrieve(t *testing.T) {
	store, err := NewGoalStore(tempGoalDB(t))
	if err != nil {
		t.Fatalf("NewGoalStore: %v", err)
	}
	defer store.Close()

	// Save 50 goals with various statuses and tiers
	statuses := []GoalStatus{GoalStatusPending, GoalStatusActive, GoalStatusCompleted, GoalStatusAbandoned}
	for i := 0; i < 50; i++ {
		g := NewGoal(
			"Goal "+string(rune('A'+i%26)),
			"Description for goal",
			(i%3)+1,
			GoalBeUseful.Title,
			[]string{"evidence1"},
			[]string{"criterion1"},
			0.5+float64(i%5)*0.1,
		)
		g.Status = statuses[i%len(statuses)]
		if err := store.Save(g); err != nil {
			t.Fatalf("Save goal %d: %v", i, err)
		}
	}

	// Query by status
	pending, _ := store.GetByStatus(GoalStatusPending)
	active, _ := store.GetByStatus(GoalStatusActive)
	completed, _ := store.GetByStatus(GoalStatusCompleted)
	abandoned, _ := store.GetByStatus(GoalStatusAbandoned)

	// Each status should have 50/4 = 12 or 13 goals
	total := len(pending) + len(active) + len(completed) + len(abandoned)
	if total != 50 {
		t.Fatalf("expected 50 total goals, got %d (p=%d a=%d c=%d ab=%d)",
			total, len(pending), len(active), len(completed), len(abandoned))
	}

	// Query by tier
	tier1, _ := store.GetByTier(1)
	tier2, _ := store.GetByTier(2)
	tier3, _ := store.GetByTier(3)
	if len(tier1)+len(tier2)+len(tier3) != 50 {
		t.Fatalf("tier query mismatch: %d + %d + %d != 50", len(tier1), len(tier2), len(tier3))
	}

	// GetByID
	if len(pending) > 0 {
		got, err := store.GetByID(pending[0].ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ID != pending[0].ID {
			t.Fatalf("ID mismatch")
		}
	}
}

func TestGoalStore_Update(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()

	g := NewGoal("Test Goal", "desc", 2, GoalBeUseful.Title, nil, nil, 0.8)
	store.Save(g)

	// Update
	g.Status = GoalStatusActive
	g.Priority = 0.9
	g.AttemptCount = 1
	now := time.Now().UTC()
	g.LastAttempt = &now
	store.Update(g)

	got, _ := store.GetByID(g.ID)
	if got.Status != GoalStatusActive {
		t.Fatalf("expected active, got %s", got.Status)
	}
	if got.AttemptCount != 1 {
		t.Fatalf("expected attempt count 1, got %d", got.AttemptCount)
	}
	if got.Priority != 0.9 {
		t.Fatalf("expected priority 0.9, got %f", got.Priority)
	}
}

func TestGoalStore_History(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()

	// Insert 3 completed goals
	for i := 0; i < 3; i++ {
		g := NewGoal("Completed", "desc", 1, GoalBeUseful.Title, nil, nil, 0.5)
		g.Status = GoalStatusCompleted
		now := time.Now().UTC()
		g.CompletedAt = &now
		v := float64(i) * 0.2
		g.ActualValue = &v
		store.Save(g)
	}

	// Insert 2 abandoned goals
	for i := 0; i < 2; i++ {
		g := NewGoal("Abandoned", "desc", 1, GoalBeUseful.Title, nil, nil, 0.5)
		g.Status = GoalStatusAbandoned
		store.Save(g)
	}

	history, _ := store.History(10)
	// History should return both completed AND abandoned
	if len(history) != 5 {
		t.Fatalf("expected 5 history entries (3 completed + 2 abandoned), got %d", len(history))
	}
}

func TestGoalStore_Count(t *testing.T) {
	store, _ := NewGoalStore(tempGoalDB(t))
	defer store.Close()

	for i := 0; i < 10; i++ {
		g := NewGoal("Goal", "desc", 1, GoalBeUseful.Title, nil, nil, 0.5)
		g.Status = GoalStatusPending
		store.Save(g)
	}

	count, _ := store.Count(GoalStatusPending)
	if count != 10 {
		t.Fatalf("expected 10 pending, got %d", count)
	}
}
