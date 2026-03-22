package ags

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// MetacognitionReport summarizes a metacognition cycle.
type MetacognitionReport struct {
	Timestamp      time.Time
	CompletionRate float64
	ValueAccuracy  float64
	DeferredCount  int
	OldWeights     ScorerWeights
	NewWeights     ScorerWeights
	WeightDelta    ScorerWeights
}

// MetacognitionLoop periodically evaluates goal-setting quality and adjusts
// the priority scorer's weights based on historical outcomes.
type MetacognitionLoop struct {
	store    *GoalStore
	scorer   *PriorityScorer
	maxDelta float64 // max weight change per cycle (default 0.05)
}

// NewMetacognitionLoop creates a new metacognition loop.
func NewMetacognitionLoop(store *GoalStore, scorer *PriorityScorer) *MetacognitionLoop {
	return &MetacognitionLoop{
		store:    store,
		scorer:   scorer,
		maxDelta: 0.05,
	}
}

// Run executes a metacognition cycle. Should be called weekly.
func (m *MetacognitionLoop) Run() (*MetacognitionReport, error) {
	// 1. Pull recent goals (last 7 days)
	completed, err := m.store.RecentCompleted(7 * 24 * time.Hour)
	if err != nil {
		return nil, fmt.Errorf("ags metacognition: fetching completed: %w", err)
	}

	abandoned, err := m.store.RecentAbandoned(7 * 24 * time.Hour)
	if err != nil {
		return nil, fmt.Errorf("ags metacognition: fetching abandoned: %w", err)
	}

	deferred, err := m.store.SystematicallyPending()
	if err != nil {
		return nil, fmt.Errorf("ags metacognition: fetching deferred: %w", err)
	}

	// 2. Compute completion rate
	total := len(completed) + len(abandoned)
	var completionRate float64
	if total > 0 {
		completionRate = float64(len(completed)) / float64(total)
	}

	// 3. Compute value accuracy (how well did expected values match actual)
	var totalAccuracy float64
	accuracyCount := 0
	for _, g := range completed {
		if g.ActualValue != nil && g.ExpectedValue > 0 {
			accuracy := *g.ActualValue / g.ExpectedValue
			totalAccuracy += accuracy
			accuracyCount++
		}
	}
	var avgAccuracy float64
	if accuracyCount > 0 {
		avgAccuracy = totalAccuracy / float64(accuracyCount)
	} else {
		avgAccuracy = 1.0 // no data = assume neutral
	}

	// 4. Compute new weights based on analysis
	oldWeights := m.scorer.Weights()
	newWeights := m.adjustWeights(oldWeights, completionRate, avgAccuracy, len(deferred))

	// 5. Clamp changes to maxDelta
	clamped := oldWeights.Clamp(newWeights, m.maxDelta)

	// 6. Validate
	if !clamped.Validate() {
		slog.Warn("ags metacognition: computed weights invalid, keeping old weights",
			"old", oldWeights, "attempted", clamped)
		clamped = oldWeights
	}

	// 7. Apply
	m.scorer.SetWeights(clamped)

	slog.Info("ags metacognition: weights updated",
		"completion_rate", fmt.Sprintf("%.2f%%", completionRate*100),
		"value_accuracy", fmt.Sprintf("%.2f", avgAccuracy),
		"deferred", len(deferred),
		"old_weights", oldWeights,
		"new_weights", clamped,
	)

	return &MetacognitionReport{
		Timestamp:      time.Now().UTC(),
		CompletionRate: completionRate,
		ValueAccuracy:  avgAccuracy,
		DeferredCount:  len(deferred),
		OldWeights:     oldWeights,
		NewWeights:     clamped,
		WeightDelta: ScorerWeights{
			Urgency:     clamped.Urgency - oldWeights.Urgency,
			Impact:      clamped.Impact - oldWeights.Impact,
			Feasibility: clamped.Feasibility - oldWeights.Feasibility,
			Novelty:     clamped.Novelty - oldWeights.Novelty,
			Alignment:   clamped.Alignment - oldWeights.Alignment,
		},
	}, nil
}

// adjustWeights computes target weights based on performance metrics.
func (m *MetacognitionLoop) adjustWeights(current ScorerWeights, completionRate, valueAccuracy float64, deferredCount int) ScorerWeights {
	w := current

	// If value accuracy is low (ActualValue < 0.7 * ExpectedValue), reduce impact weight
	if valueAccuracy < 0.7 {
		w.Impact -= 0.03
		w.Feasibility += 0.02
		w.Urgency += 0.01
	}

	// If many goals are deferred, increase feasibility weight
	if deferredCount > 3 {
		w.Feasibility += 0.03
		w.Impact -= 0.02
		w.Novelty -= 0.01
	}

	// If completion rate is very high, goals might be too easy — increase impact/novelty
	if completionRate > 0.9 {
		w.Impact += 0.02
		w.Novelty += 0.02
		w.Feasibility -= 0.02
		w.Urgency -= 0.02
	}

	// If completion rate is very low, goals might be too hard — increase feasibility
	if completionRate < 0.3 {
		w.Feasibility += 0.03
		w.Impact -= 0.03
	}

	// Normalize to ensure no negative weights and sum ~1.0
	if w.Urgency < 0.01 {
		w.Urgency = 0.01
	}
	if w.Impact < 0.01 {
		w.Impact = 0.01
	}
	if w.Feasibility < 0.01 {
		w.Feasibility = 0.01
	}
	if w.Novelty < 0.01 {
		w.Novelty = 0.01
	}
	if w.Alignment < 0.01 {
		w.Alignment = 0.01
	}

	sum := w.Urgency + w.Impact + w.Feasibility + w.Novelty + w.Alignment
	if sum > 0 {
		w.Urgency /= sum
		w.Impact /= sum
		w.Feasibility /= sum
		w.Novelty /= sum
		w.Alignment /= sum
	}

	// Round to 4 decimal places
	round := func(v float64) float64 { return math.Round(v*10000) / 10000 }
	w.Urgency = round(w.Urgency)
	w.Impact = round(w.Impact)
	w.Feasibility = round(w.Feasibility)
	w.Novelty = round(w.Novelty)
	w.Alignment = round(w.Alignment)

	return w
}

// StartScheduler runs Run() on a 7-day cycle.
func (m *MetacognitionLoop) StartScheduler(ctx context.Context) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(7 * 24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				report, err := m.Run()
				if err != nil {
					slog.Warn("ags metacognition: scheduled run failed", "error", err)
				} else {
					slog.Info("ags metacognition: scheduled run complete",
						"completion_rate", report.CompletionRate,
						"value_accuracy", report.ValueAccuracy,
					)
				}
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { close(stop) }
}
