package rsi

import (
	"testing"

	"github.com/wunderpus/wunderpus/internal/config"
)

func TestFitnessEvaluator_Score(t *testing.T) {
	fe := NewFitnessEvaluator(&config.Config{Genesis: config.GenesisConfig{RSIFitnessThreshold: 0.05}}, nil)

	// Case 1: P99 improved from 1000ms to 800ms, no errors, tests pass
	before := SpanStats{
		FunctionName: "TargetFunc",
		P99LatencyNs: 1_000_000_000, // 1000ms
		ErrorCount:   0,
	}
	report := SandboxReport{
		TestsPassed: true,
		RaceClean:   true,
		BenchmarkNsOp: map[string]float64{
			"TargetFunc": 800_000_000, // 800ms
		},
		ErrorCount: 0,
	}

	score := fe.Score(before, report)
	// Expected: (200ms/1000ms * 0.6) + (0 * 0.4) = 0.12
	expectedScore := 0.12
	if score < expectedScore-0.01 || score > expectedScore+0.01 {
		t.Fatalf("expected score ~%.2f, got %.4f", expectedScore, score)
	}
	if score < 0.05 {
		t.Fatalf("score %.4f should be above 0.05 threshold", score)
	}
}

func TestFitnessEvaluator_TestFailure(t *testing.T) {
	fe := NewFitnessEvaluator(&config.Config{Genesis: config.GenesisConfig{RSIFitnessThreshold: 0.05}}, nil)

	before := SpanStats{FunctionName: "TargetFunc", P99LatencyNs: 1_000_000_000}
	report := SandboxReport{
		TestsPassed: false, // tests failed
		RaceClean:   true,
		BenchmarkNsOp: map[string]float64{
			"TargetFunc": 500_000_000,
		},
	}

	score := fe.Score(before, report)
	if score != -1.0 {
		t.Fatalf("test failure should give -1.0, got %f", score)
	}
}

func TestFitnessEvaluator_RaceDetected(t *testing.T) {
	fe := NewFitnessEvaluator(&config.Config{Genesis: config.GenesisConfig{RSIFitnessThreshold: 0.05}}, nil)

	before := SpanStats{FunctionName: "TargetFunc", P99LatencyNs: 1_000_000_000}
	report := SandboxReport{
		TestsPassed: true,
		RaceClean:   false, // race detected
		BenchmarkNsOp: map[string]float64{
			"TargetFunc": 500_000_000,
		},
	}

	score := fe.Score(before, report)
	if score != -1.0 {
		t.Fatalf("race should give -1.0, got %f", score)
	}
}

func TestFitnessEvaluator_ErrorImprovement(t *testing.T) {
	fe := NewFitnessEvaluator(&config.Config{Genesis: config.GenesisConfig{RSIFitnessThreshold: 0.05}}, nil)

	before := SpanStats{
		FunctionName: "TargetFunc",
		P99LatencyNs: 1_000_000_000,
		ErrorCount:   10,
	}
	report := SandboxReport{
		TestsPassed: true,
		RaceClean:   true,
		BenchmarkNsOp: map[string]float64{
			"TargetFunc": 1_000_000_000, // no latency change
		},
		ErrorCount: 5, // 50% error reduction
	}

	score := fe.Score(before, report)
	// Expected: (0 * 0.6) + (5/10 * 0.4) = 0.2
	expectedScore := 0.2
	if score < expectedScore-0.01 || score > expectedScore+0.01 {
		t.Fatalf("expected score ~%.2f, got %.4f", expectedScore, score)
	}
}

func TestFitnessEvaluator_ErrorPenalty(t *testing.T) {
	fe := NewFitnessEvaluator(&config.Config{Genesis: config.GenesisConfig{RSIFitnessThreshold: 0.05}}, nil)

	// Case: latency improves by 50% but introduced 5 new errors
	before := SpanStats{
		FunctionName: "TargetFunc",
		P99LatencyNs: 1_000_000_000,
		ErrorCount:   0,
	}
	report := SandboxReport{
		TestsPassed: true,
		RaceClean:   true,
		BenchmarkNsOp: map[string]float64{
			"TargetFunc": 500_000_000, // 50% improvement
		},
		ErrorCount: 5, // 5 new errors
	}

	score := fe.Score(before, report)
	// Expected: (0.5 * 0.6) + (-1.0 * 0.4) = 0.3 - 0.4 = -0.1
	expectedScore := -0.1
	if score < expectedScore-0.01 || score > expectedScore+0.01 {
		t.Fatalf("expected score ~%.2f, got %.4f", expectedScore, score)
	}
}

func TestFitnessEvaluator_SelectWinner(t *testing.T) {
	fe := NewFitnessEvaluator(&config.Config{Genesis: config.GenesisConfig{RSIFitnessThreshold: 0.05}}, nil)

	before := SpanStats{FunctionName: "TargetFunc", P99LatencyNs: 1_000_000_000}

	proposals := []Proposal{
		{ID: "p1", Diff: "diff1"},
		{ID: "p2", Diff: "diff2"},
		{ID: "p3", Diff: ""}, // empty = failed generation
	}

	reports := []SandboxReport{
		{
			TestsPassed: true,
			RaceClean:   true,
			BenchmarkNsOp: map[string]float64{
				"TargetFunc": 800_000_000,
			},
		},
		{
			TestsPassed: false,
			RaceClean:   true,
			BenchmarkNsOp: map[string]float64{
				"TargetFunc": 500_000_000,
			},
		},
		{
			TestsPassed: true,
			RaceClean:   true,
			BenchmarkNsOp: map[string]float64{
				"TargetFunc": 900_000_000,
			},
		},
	}

	winner, score := fe.SelectWinner(proposals, reports, before)

	if winner == nil {
		t.Fatal("expected a winner")
	}
	if winner.ID != "p1" {
		t.Fatalf("expected p1 as winner, got %s", winner.ID)
	}
	if score < 0.05 {
		t.Fatalf("winner score %.4f should be above threshold 0.05", score)
	}
}

func TestFitnessEvaluator_NoWinnerBelowThreshold(t *testing.T) {
	fe := NewFitnessEvaluator(&config.Config{Genesis: config.GenesisConfig{RSIFitnessThreshold: 0.5}}, nil) // high threshold

	before := SpanStats{FunctionName: "TargetFunc", P99LatencyNs: 1_000_000_000}

	proposals := []Proposal{{ID: "p1", Diff: "diff"}}
	reports := []SandboxReport{
		{
			TestsPassed: true,
			RaceClean:   true,
			BenchmarkNsOp: map[string]float64{
				"TargetFunc": 990_000_000, // only 1% improvement
			},
		},
	}

	winner, _ := fe.SelectWinner(proposals, reports, before)

	if winner != nil {
		t.Fatal("should not have selected a winner with high threshold")
	}
}
