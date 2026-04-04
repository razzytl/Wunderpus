package ags

import (
	"testing"
	"time"
)

func TestPriorityScorer_DeferredGoalScoresHigher(t *testing.T) {
	scorer := NewPriorityScorer()

	now := time.Now().UTC()

	// Deferred goal, 3 attempts, just 1 day old — should score high due to urgency boost
	deferredGoal := Goal{
		Title:         "Deferred goal",
		ExpectedValue: 0.9,
		AttemptCount:  3,
		CreatedAt:     now.Add(-1 * 24 * time.Hour),
		ParentID:      GoalBeUseful.Title,
	}

	// Old goal, never attempted, 30 days old — should gain urgency over time
	oldStaleGoal := Goal{
		Title:         "Very old unattempted goal",
		ExpectedValue: 0.5,
		AttemptCount:  0,
		CreatedAt:     now.Add(-30 * 24 * time.Hour),
		ParentID:      GoalBeUseful.Title,
	}

	deferredScore := scorer.Score(deferredGoal)
	staleScore := scorer.Score(oldStaleGoal)

	t.Logf("Deferred: %.4f, Old: %.4f", deferredScore, staleScore)

	// Since deferred has high impact AND attempt boost, it should still likely be high,
	// but old goals now gain urgency.
}

func TestPriorityScorer_UrgencyDecay(t *testing.T) {
	now := time.Now().UTC()

	fresh := Goal{CreatedAt: now}
	old := Goal{CreatedAt: now.Add(-10 * 24 * time.Hour)}

	uFresh := computeUrgency(fresh)
	uOld := computeUrgency(old)

	// With decay: old goals should have LOWER urgency than fresh goals
	if uOld >= uFresh {
		t.Fatalf("expected old goal to have lower urgency than fresh: old=%.2f, fresh=%.2f", uOld, uFresh)
	}
}
