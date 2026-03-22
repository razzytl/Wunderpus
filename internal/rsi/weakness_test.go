package rsi

import (
	"testing"
)

func TestWeaknessReporter_Generate(t *testing.T) {
	profiler, _ := NewProfiler("")

	// Inject known SpanStats with a high-error function
	profiler.TrackDuration("pkg.HighError", 1_000_000, true) // 1ms, error
	profiler.TrackDuration("pkg.HighError", 1_000_000, true)
	profiler.TrackDuration("pkg.HighError", 1_000_000, true)
	profiler.TrackDuration("pkg.HighError", 1_000_000, false) // 75% error rate
	profiler.TrackDuration("pkg.LowError", 5_000_000, false)  // 5ms, no errors
	profiler.TrackDuration("pkg.LowError", 5_000_000, false)
	profiler.TrackDuration("pkg.LowError", 5_000_000, false)
	profiler.TrackDuration("pkg.LowError", 5_000_000, false)

	// Build a code map with these functions
	codeMap := &CodeMap{
		Functions: map[string]*FunctionNode{
			"pkg.HighError": {
				Name:           "HighError",
				QualifiedName:  "pkg.HighError",
				CyclomaticComp: 15,
			},
			"pkg.LowError": {
				Name:           "LowError",
				QualifiedName:  "pkg.LowError",
				CyclomaticComp: 2,
			},
		},
	}

	mapper := NewCodeMapper(false)
	reporter := NewWeaknessReporter(profiler, mapper)

	report := reporter.Generate(codeMap)

	if report.TotalFunctionsAnalyzed != 2 {
		t.Fatalf("expected 2 functions analyzed, got %d", report.TotalFunctionsAnalyzed)
	}

	if len(report.TopCandidates) == 0 {
		t.Fatal("expected at least 1 candidate")
	}

	// HighError should be #1 (highest weakness score)
	top := report.TopCandidates[0]
	if top.FunctionNode.QualifiedName != "pkg.HighError" {
		t.Fatalf("expected HighError at #1, got %s (score=%.4f)",
			top.FunctionNode.QualifiedName, top.WeaknessScore)
	}

	t.Logf("Top candidate: %s (score=%.4f, reason=%s)",
		top.FunctionNode.QualifiedName, top.WeaknessScore, top.PrimaryReason)
}
