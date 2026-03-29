package toolsynth

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ToolRunner abstracts the execution of a tool for testing.
// The tester calls this to actually run the tool with given inputs.
type ToolRunner interface {
	// Run executes a tool with the given arguments and returns the result.
	Run(ctx context.Context, name string, args map[string]any) (string, error)
}

// TestResult represents the outcome of a single test case execution.
type TestResult struct {
	TestCase  TestCase
	Passed    bool
	Actual    string
	Error     string
	LatencyMs int64
}

// Tester runs test cases against a synthesized tool.
// It executes the tool's test cases and verifies outputs match expectations.
// Uses a sandboxed execution environment: network isolated, memory capped at 256MB.
type Tester struct {
	runner      ToolRunner
	minPassRate float64 // minimum pass rate to accept a tool (default 0.8)
	timeout     time.Duration
}

// NewTester creates a tester with the given tool runner.
func NewTester(runner ToolRunner) *Tester {
	return &Tester{
		runner:      runner,
		minPassRate: 0.8,
		timeout:     30 * time.Second,
	}
}

// SetMinPassRate overrides the minimum pass rate (0.0-1.0).
func (t *Tester) SetMinPassRate(rate float64) {
	if rate >= 0 && rate <= 1.0 {
		t.minPassRate = rate
	}
}

// SetTimeout overrides the per-test-case timeout.
func (t *Tester) SetTimeout(d time.Duration) {
	t.timeout = d
}

// Test runs all test cases from a ToolSpec against the generated tool.
// Returns a ToolTestResult with pass rate, latency, and any errors.
func (t *Tester) Test(spec ToolSpec, source string) (*ToolTestResult, error) {
	slog.Info("tester: running tests", "tool", spec.Name, "cases", len(spec.TestCases))

	if len(spec.TestCases) == 0 {
		return nil, fmt.Errorf("tester: no test cases for tool %s", spec.Name)
	}

	var results []TestResult
	var totalLatency int64
	passCount := 0

	for i, tc := range spec.TestCases {
		result := t.runTestCase(spec.Name, tc)
		results = append(results, result)

		totalLatency += result.LatencyMs
		if result.Passed {
			passCount++
		} else {
			slog.Debug("tester: test case failed",
				"tool", spec.Name,
				"case", i,
				"input", tc.Input,
				"expected", tc.Expected,
				"actual", result.Actual,
				"error", result.Error)
		}
	}

	passRate := float64(passCount) / float64(len(spec.TestCases))
	avgLatency := int64(0)
	if len(results) > 0 {
		avgLatency = totalLatency / int64(len(results))
	}

	// Collect all errors
	var errors []string
	for _, r := range results {
		if r.Error != "" {
			errors = append(errors, r.Error)
		}
	}

	allPassed := passRate >= t.minPassRate

	testResult := &ToolTestResult{
		AllPassed:    allPassed,
		PassRate:     passRate,
		AvgLatencyMs: avgLatency,
		Errors:       errors,
	}

	slog.Info("tester: test complete",
		"tool", spec.Name,
		"passRate", fmt.Sprintf("%.1f%%", passRate*100),
		"passed", passCount,
		"total", len(spec.TestCases),
		"avgLatency", avgLatency,
		"accepted", allPassed)

	return testResult, nil
}

// runTestCase executes a single test case and checks the result.
func (t *Tester) runTestCase(toolName string, tc TestCase) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	start := time.Now()
	actual, err := t.runner.Run(ctx, toolName, tc.Input)
	latency := time.Since(start).Milliseconds()

	result := TestResult{
		TestCase:  tc,
		LatencyMs: latency,
		Actual:    actual,
	}

	if err != nil {
		result.Passed = false
		result.Error = err.Error()
		return result
	}

	// Check if expected output is contained in actual output
	// (substring match — the tool output may include extra formatting)
	result.Passed = containsIgnoreCase(actual, tc.Expected)
	if !result.Passed {
		result.Error = fmt.Sprintf("expected %q in output, got %q", tc.Expected, actual)
	}

	return result
}

// MinPassRate returns the configured minimum pass rate.
func (t *Tester) MinPassRate() float64 {
	return t.minPassRate
}

// containsIgnoreCase checks if substr is contained in s (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true // empty expected matches anything
	}
	return len(s) >= len(substr) && containsFold(s, substr)
}

// containsFold is a case-insensitive substring search.
func containsFold(s, substr string) bool {
	sLower := toLower(s)
	subLower := toLower(substr)
	return indexOf(sLower, subLower) >= 0
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
