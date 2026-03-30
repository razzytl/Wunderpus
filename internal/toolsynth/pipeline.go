package toolsynth

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Pipeline orchestrates the full tool synthesis cycle:
// Detect → Design → Code → Test → Register
type Pipeline struct {
	detector  *Detector
	designer  *Designer
	coder     *Coder
	tester    *Tester
	registrar *Registrar
	improver  *ImprovementLoop
}

// PipelineResult captures the outcome of a full synthesis cycle.
type PipelineResult struct {
	GapsDetected    int           `json:"gaps_detected"`
	ToolsDesigned   int           `json:"tools_designed"`
	ToolsCoded      int           `json:"tools_coded"`
	ToolsTested     int           `json:"tools_tested"`
	ToolsRegistered int           `json:"tools_registered"`
	Errors          []string      `json:"errors"`
	Duration        time.Duration `json:"duration"`
	Details         []ToolResult  `json:"details"`
}

// ToolResult captures per-tool pipeline progress.
type ToolResult struct {
	Name       string          `json:"name"`
	Gap        ToolGap         `json:"gap"`
	Spec       *ToolSpec       `json:"spec,omitempty"`
	Candidates int             `json:"candidates"`
	TestResult *ToolTestResult `json:"test_result,omitempty"`
	Registered bool            `json:"registered"`
	Error      string          `json:"error,omitempty"`
}

// NewPipeline creates a pipeline with all components.
func NewPipeline(
	detector *Detector,
	designer *Designer,
	coder *Coder,
	tester *Tester,
	registrar *Registrar,
) *Pipeline {
	return &Pipeline{
		detector:  detector,
		designer:  designer,
		coder:     coder,
		tester:    tester,
		registrar: registrar,
	}
}

// SetImprover wires the improvement loop for usage tracking.
func (p *Pipeline) SetImprover(improver *ImprovementLoop) {
	p.improver = improver
}

// Run executes the full tool synthesis pipeline:
// 1. Scan episodic memory for tool gaps
// 2. Design a tool spec for each gap (via LLM)
// 3. Generate Go source candidates (via LLM)
// 4. Test the best candidate
// 5. Register if tests pass
func (p *Pipeline) Run(ctx context.Context) (*PipelineResult, error) {
	start := time.Now()
	result := &PipelineResult{}

	slog.Info("pipeline: starting tool synthesis cycle")

	// Step 1: Detect gaps
	gaps, err := p.detector.Scan()
	if err != nil {
		return nil, fmt.Errorf("pipeline: detection failed: %w", err)
	}
	result.GapsDetected = len(gaps)

	if len(gaps) == 0 {
		slog.Info("pipeline: no gaps detected, nothing to do")
		result.Duration = time.Since(start)
		return result, nil
	}

	slog.Info("pipeline: gaps detected", "count", len(gaps))

	// Process each gap
	for _, gap := range gaps {
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, "pipeline canceled")
			result.Duration = time.Since(start)
			return result, ctx.Err()
		default:
		}

		toolResult := p.processGap(gap)
		result.Details = append(result.Details, toolResult)

		if toolResult.Error != "" {
			result.Errors = append(result.Errors, toolResult.Error)
		}
		if toolResult.Spec != nil {
			result.ToolsDesigned++
		}
		if toolResult.Candidates > 0 {
			result.ToolsCoded++
		}
		if toolResult.TestResult != nil {
			result.ToolsTested++
		}
		if toolResult.Registered {
			result.ToolsRegistered++
		}
	}

	result.Duration = time.Since(start)

	slog.Info("pipeline: synthesis cycle complete",
		"detected", result.GapsDetected,
		"designed", result.ToolsDesigned,
		"coded", result.ToolsCoded,
		"tested", result.ToolsTested,
		"registered", result.ToolsRegistered,
		"errors", len(result.Errors),
		"duration", result.Duration)

	return result, nil
}

// processGap runs one tool through the full synthesis pipeline.
func (p *Pipeline) processGap(gap ToolGap) ToolResult {
	tr := ToolResult{
		Name: gap.Name,
		Gap:  gap,
	}

	// Design
	spec, err := p.designer.Design(gap)
	if err != nil {
		tr.Error = fmt.Sprintf("design failed: %v", err)
		slog.Warn("pipeline: design failed", "gap", gap.Name, "error", err)
		return tr
	}
	tr.Spec = spec

	// Code
	candidates, err := p.coder.Generate(*spec)
	if err != nil {
		tr.Error = fmt.Sprintf("coding failed: %v", err)
		slog.Warn("pipeline: coding failed", "tool", spec.Name, "error", err)
		return tr
	}
	tr.Candidates = len(candidates)

	// Find best valid candidate
	bestSource := ""
	for _, c := range candidates {
		if c.BuildError == "" {
			bestSource = c.Source
			break
		}
	}
	if bestSource == "" {
		tr.Error = "no valid candidate produced"
		slog.Warn("pipeline: no valid candidate", "tool", spec.Name)
		return tr
	}

	// Test
	testResult, err := p.tester.Test(*spec, bestSource)
	if err != nil {
		tr.Error = fmt.Sprintf("testing failed: %v", err)
		slog.Warn("pipeline: testing failed", "tool", spec.Name, "error", err)
		return tr
	}
	tr.TestResult = testResult

	// Register if tests passed
	if testResult.AllPassed {
		// Write source to output directory
		filePath, err := p.coder.WriteValidSource(*spec, candidates)
		if err != nil {
			tr.Error = fmt.Sprintf("write source failed: %v", err)
			return tr
		}

		// Register
		if err := p.registrar.Register(*spec, bestSource, *testResult); err != nil {
			tr.Error = fmt.Sprintf("registration failed: %v", err)
			slog.Warn("pipeline: registration failed", "tool", spec.Name, "error", err)
			return tr
		}
		tr.Registered = true

		slog.Info("pipeline: tool synthesized and registered",
			"name", spec.Name,
			"passRate", testResult.PassRate,
			"path", filePath)
	} else {
		tr.Error = fmt.Sprintf("tests failed: pass rate %.1f%% < %.1f%%",
			testResult.PassRate*100, p.tester.MinPassRate()*100)
		slog.Warn("pipeline: tool rejected by tests",
			"tool", spec.Name,
			"passRate", testResult.PassRate)
	}

	return tr
}
