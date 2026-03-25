package rsi

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/wunderpus/wunderpus/internal/audit"
	"github.com/wunderpus/wunderpus/internal/config"
)

// FitnessEvaluator computes fitness scores for RSI proposals based on
// before/after metrics and sandbox test results.
type FitnessEvaluator struct {
	threshold float64 // minimum fitness to consider
	audit     *audit.AuditLog
}

// NewFitnessEvaluator creates a fitness evaluator using config threshold and audit log.
func NewFitnessEvaluator(cfg *config.Config, auditLog *audit.AuditLog) *FitnessEvaluator {
	threshold := 0.05
	if cfg != nil && cfg.Genesis.RSIFitnessThreshold > 0 {
		threshold = cfg.Genesis.RSIFitnessThreshold
	}
	return &FitnessEvaluator{
		threshold: threshold,
		audit:     auditLog,
	}
}

// Score computes the fitness of a proposal relative to the original function.
// Returns -1.0 if tests failed or races were detected.
// Latency improvement is derived from benchmarks in the sandbox report.
func (f *FitnessEvaluator) Score(before SpanStats, report SandboxReport) float64 {
	// Hard gate: tests must pass and no races
	if !report.TestsPassed || !report.RaceClean {
		return -1.0
	}

	// Latency improvement: how much did P99 improve?
	// We use the benchmark ns/op for the target function as the "after" latency.
	var latencyDelta float64
	if before.P99LatencyNs > 0 {
		afterP99 := int64(report.BenchmarkNsOp[before.FunctionName])
		if afterP99 > 0 {
			latencyDelta = float64(before.P99LatencyNs-afterP99) / float64(before.P99LatencyNs)
		}
	}

	// Error improvement: how much did error count decrease?
	var errorDelta float64
	if before.ErrorCount > 0 {
		// Existing errors: calculate percentage reduction
		errorDelta = float64(before.ErrorCount-report.ErrorCount) / float64(before.ErrorCount)
	} else if report.ErrorCount > 0 {
		// New errors introduced: penalize hard
		errorDelta = -1.0
	}

	score := (latencyDelta * 0.6) + (errorDelta * 0.4)
	return score
}

// Threshold returns the minimum fitness score required for deployment.
func (f *FitnessEvaluator) Threshold() float64 {
	return f.threshold
}

// SelectWinner returns the highest-scoring proposal above the threshold,
// or nil if no proposal qualifies.
func (f *FitnessEvaluator) SelectWinner(
	proposals []Proposal,
	reports []SandboxReport,
	before SpanStats,
) (*Proposal, float64) {

	if len(proposals) != len(reports) {
		return nil, -1.0
	}

	var bestProposal *Proposal
	bestScore := -math.MaxFloat64

	for i := range proposals {
		if proposals[i].Diff == "" {
			continue // skip empty proposals (failed generation)
		}

		score := f.Score(before, reports[i])

		if f.audit != nil {
			status := "REJECTED"
			if score >= f.threshold {
				status = "QUALIFIED"
			}
			afterP99 := int64(reports[i].BenchmarkNsOp[before.FunctionName])
			payloadBytes, _ := json.Marshal(map[string]interface{}{
				"proposal_id":   proposals[i].ID,
				"target_func":   before.FunctionName,
				"score":         score,
				"status":        status,
				"tests_passed":  reports[i].TestsPassed,
				"race_clean":    reports[i].RaceClean,
				"latency_delta": before.P99LatencyNs - afterP99,
				"error_delta":   before.ErrorCount - reports[i].ErrorCount,
			})

			f.audit.Write(audit.AuditEntry{
				Subsystem: "rsi",
				EventType: audit.EventRSIFitnessEvaluated,
				ActorID:   "wunderpus",
				Payload:   payloadBytes,
			})
		}

		if score > bestScore {
			bestScore = score
			bestProposal = &proposals[i]
		}
	}

	if bestScore < f.threshold {
		return nil, bestScore
	}

	return bestProposal, bestScore
}

// FitnessReport is a log-friendly summary of fitness evaluation.
type FitnessReport struct {
	ProposalID    string
	TargetFunc    string
	Score         float64
	Passed        bool
	LatencyBefore int64
	LatencyAfter  int64
	ErrorsBefore  int64
	ErrorsAfter   int64
	TestsPassed   bool
	RaceClean     bool
}

// String returns a human-readable fitness report.
func (fr FitnessReport) String() string {
	status := "REJECTED"
	if fr.Passed {
		status = "ACCEPTED"
	}
	return fmt.Sprintf("Fitness [%s] %s → score=%.4f (p99: %d→%d ns, errors: %d→%d, tests=%v, race=%v)",
		status, fr.TargetFunc, fr.Score,
		fr.LatencyBefore, fr.LatencyAfter,
		fr.ErrorsBefore, fr.ErrorsAfter,
		fr.TestsPassed, fr.RaceClean)
}
