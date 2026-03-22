package ags

import (
	"testing"
	"time"
)

func TestPriorityScorer_DeferredGoalScoresHigher(t *testing.T) {
	scorer := NewPriorityScorer()

	now := time.Now().UTC()

	// Recent deferred goal, 3 attempts, just 1 day old — should score high due to urgency boost
	deferredGoal := Goal{
		Title:         "Deferred goal",
		ExpectedValue: 0.9,
		AttemptCount:  3,
		CreatedAt:     now.Add(-1 * 24 * time.Hour),
		ParentID:      GoalBeUseful.Title,
	}

	// Old goal, never attempted, 30 days old — stale, low urgency
	oldStaleGoal := Goal{
		Title:         "Very old unattempted goal",
		ExpectedValue: 0.9,
		AttemptCount:  0,
		CreatedAt:     now.Add(-30 * 24 * time.Hour),
		ParentID:      GoalBeUseful.Title,
	}

	deferredScore := scorer.Score(deferredGoal)
	staleScore := scorer.Score(oldStaleGoal)

	t.Logf("Deferred: %.4f, Stale: %.4f", deferredScore, staleScore)

	if deferredScore <= staleScore {
		t.Fatalf("deferred goal (score=%.4f) should score higher than stale goal (score=%.4f)",
			deferredScore, staleScore)
	}
}

func TestPriorityScorer_NoveltyDecreases(t *testing.T) {
	now := time.Now()

	// Verify novelty component decreases with attempts
	gFresh := Goal{Title: "Fresh", ExpectedValue: 0.8, AttemptCount: 0, CreatedAt: now, ParentID: GoalBeUseful.Title}
	gTried := Goal{Title: "Tried", ExpectedValue: 0.8, AttemptCount: 5, CreatedAt: now, ParentID: GoalBeUseful.Title}

	nFresh := computeNovelty(gFresh)
	nTried := computeNovelty(gTried)

	if nFresh <= nTried {
		t.Fatalf("novelty should decrease with attempts: fresh=%.4f, tried=%.4f", nFresh, nTried)
	}
	if nFresh != 1.0 {
		t.Fatalf("expected novelty 1.0 for 0 attempts, got %.4f", nFresh)
	}
	if nTried >= 0.3 {
		t.Fatalf("expected novelty < 0.3 for 5 attempts, got %.4f", nTried)
	}
	t.Logf("Novelty: fresh=%.4f, tried=%.4f ✓", nFresh, nTried)
}

func TestPriorityScorer_Weights(t *testing.T) {
	scorer := NewPriorityScorer()
	w := scorer.Weights()

	if !w.Validate() {
		t.Fatalf("default weights should sum to 1.0, got %.4f",
			w.Urgency+w.Impact+w.Feasibility+w.Novelty+w.Alignment)
	}
}

func TestScorerWeights_Clamp(t *testing.T) {
	current := DefaultScorerWeights()
	target := ScorerWeights{
		Urgency:     0.50,
		Impact:      0.10,
		Feasibility: 0.20,
		Novelty:     0.10,
		Alignment:   0.10,
	}

	clamped := current.Clamp(target, 0.05)

	// Urgency should have moved +0.05 (from 0.25 toward 0.50)
	if clamped.Urgency != 0.30 {
		t.Fatalf("expected urgency 0.30, got %.2f", clamped.Urgency)
	}

	// Impact should have moved -0.05 (from 0.30 toward 0.10)
	if clamped.Impact != 0.25 {
		t.Fatalf("expected impact 0.25, got %.2f", clamped.Impact)
	}
}
