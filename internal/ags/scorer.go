package ags

import (
	"math"
	"sync"
	"time"
)

// ScorerWeights holds the configurable weights for the priority scoring formula.
type ScorerWeights struct {
	Urgency     float64 // default 0.25
	Impact      float64 // default 0.30
	Feasibility float64 // default 0.20
	Novelty     float64 // default 0.10
	Alignment   float64 // default 0.15
}

// DefaultScorerWeights returns the initial scoring weights.
func DefaultScorerWeights() ScorerWeights {
	return ScorerWeights{
		Urgency:     0.25,
		Impact:      0.30,
		Feasibility: 0.20,
		Novelty:     0.10,
		Alignment:   0.15,
	}
}

// Validate checks that weights sum to 1.0 ± 0.001.
func (w ScorerWeights) Validate() bool {
	sum := w.Urgency + w.Impact + w.Feasibility + w.Novelty + w.Alignment
	return math.Abs(sum-1.0) < 0.001
}

// Clamp adjusts each weight by at most maxDelta (to prevent sudden collapse).
func (w ScorerWeights) Clamp(target ScorerWeights, maxDelta float64) ScorerWeights {
	clamp := func(current, tgt float64) float64 {
		diff := tgt - current
		if diff > maxDelta {
			return current + maxDelta
		}
		if diff < -maxDelta {
			return current - maxDelta
		}
		return tgt
	}

	return ScorerWeights{
		Urgency:     clamp(w.Urgency, target.Urgency),
		Impact:      clamp(w.Impact, target.Impact),
		Feasibility: clamp(w.Feasibility, target.Feasibility),
		Novelty:     clamp(w.Novelty, target.Novelty),
		Alignment:   clamp(w.Alignment, target.Alignment),
	}
}

// PriorityScorer computes composite priority scores for goals.
type PriorityScorer struct {
	mu      sync.RWMutex
	weights ScorerWeights
}

// NewPriorityScorer creates a scorer with default weights.
func NewPriorityScorer() *PriorityScorer {
	return &PriorityScorer{
		weights: DefaultScorerWeights(),
	}
}

// Weights returns a copy of the current scoring weights.
func (s *PriorityScorer) Weights() ScorerWeights {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.weights
}

// SetWeights updates the scoring weights. Caller must ensure they are valid.
func (s *PriorityScorer) SetWeights(w ScorerWeights) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weights = w
}

// Score computes a composite priority score for the given goal.
func (s *PriorityScorer) Score(g Goal) float64 {
	s.mu.RLock()
	w := s.weights
	s.mu.RUnlock()

	urgency := computeUrgency(g)
	impact := g.ExpectedValue
	feasibility := s.computeFeasibility(g)
	novelty := computeNovelty(g)
	alignment := computeAlignment(g)

	result := (urgency * w.Urgency) +
		(impact * w.Impact) +
		(feasibility * w.Feasibility) +
		(novelty * w.Novelty) +
		(alignment * w.Alignment)

	return result
}

// computeUrgency: base 0.5, +0.3 if deferred often, −0.1 per day since creation.
func computeUrgency(g Goal) float64 {
	urgency := 0.5

	if g.AttemptCount > 2 {
		urgency += 0.3
	}

	daysSinceCreation := time.Since(g.CreatedAt).Hours() / 24.0
	urgency -= daysSinceCreation * 0.1 // Decay over time — old goals lose urgency

	return clamp01(urgency)
}

func (s *PriorityScorer) computeFeasibility(g Goal) float64 {
	return clamp01(0.7) // base default
}

// computeNovelty: 1.0 / (1.0 + AttemptCount)
func computeNovelty(g Goal) float64 {
	return 1.0 / (1.0 + float64(g.AttemptCount))
}

// computeAlignment: how strongly the goal ties to a Tier 0 goal.
// For now, uses a heuristic based on parent. Full implementation uses LLM.
func computeAlignment(g Goal) float64 {
	// If the goal has a Tier 0 parent, it's well-aligned
	if g.ParentID != "" {
		for _, t0 := range AllTier0Goals {
			if g.ParentID == t0.Title {
				return 0.8
			}
		}
	}
	return 0.5
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
