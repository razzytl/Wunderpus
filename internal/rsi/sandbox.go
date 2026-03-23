package rsi

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SandboxReport contains the results of running a proposal in the sandbox.
type SandboxReport struct {
	PatchApplied  bool
	BuildPassed   bool
	TestsPassed   bool
	RaceClean     bool
	BenchmarkNsOp map[string]float64
	TestOutput    string
	BuildOutput   string
	Duration      time.Duration
	Error         string
}

// Sandbox executes code proposals in an isolated environment.
// It copies the repo, applies the diff, builds, tests, and reports results.
type Sandbox struct {
	repoRoot   string
	workDir    string
	timeoutSec int
	UseDocker  bool
}

// NewSandbox creates a sandbox for the given repository root.
func NewSandbox(repoRoot string) *Sandbox {
	_, err := exec.LookPath("docker")
	return &Sandbox{
		repoRoot:   repoRoot,
		workDir:    os.TempDir(),
		timeoutSec: 60,
		UseDocker:  err == nil, // Auto-detect Docker
	}
}

// Run applies a proposal's diff to a copy of the repo and runs the test suite.
func (s *Sandbox) Run(proposal Proposal, baseRepoPath string) (*SandboxReport, error) {
	start := time.Now()
	report := &SandboxReport{
		BenchmarkNsOp: make(map[string]float64),
	}

	// Create sandbox directory
	sandboxDir := filepath.Join(s.workDir, fmt.Sprintf("wunderpus-sandbox-%s", uuid.New().String()))
	defer func() {
		if err := os.RemoveAll(sandboxDir); err != nil {
			slog.Warn("rsi sandbox: cleanup failed", "dir", sandboxDir, "error", err)
		}
	}()

	// Copy repo to sandbox
	slog.Debug("rsi sandbox: copying repo", "from", baseRepoPath, "to", sandboxDir)
	if err := copyDir(baseRepoPath, sandboxDir); err != nil {
		report.Error = fmt.Sprintf("copy failed: %v", err)
		report.Duration = time.Since(start)
		return report, nil
	}

	// Apply the diff
	patchCmd := exec.Command("git", "apply", "--check")
	patchCmd.Dir = sandboxDir
	patchCmd.Stdin = strings.NewReader(proposal.Diff)

	checkOutput, checkErr := patchCmd.CombinedOutput()
	if checkErr != nil {
		report.PatchApplied = false
		report.Error = fmt.Sprintf("patch check failed: %s", string(checkOutput))
		report.Duration = time.Since(start)
		return report, nil
	}

	// Actually apply the patch
	applyCmd := exec.Command("git", "apply")
	applyCmd.Dir = sandboxDir
	applyCmd.Stdin = strings.NewReader(proposal.Diff)
	applyOutput, applyErr := applyCmd.CombinedOutput()
	if applyErr != nil {
		report.PatchApplied = false
		report.Error = fmt.Sprintf("patch apply failed: %s", string(applyOutput))
		report.Duration = time.Since(start)
		return report, nil
	}
	report.PatchApplied = true

	timeout := time.Duration(s.timeoutSec) * time.Second

	if s.UseDocker {
		return s.runInDocker(sandboxDir, timeout, start, report)
	}
	return s.runLocally(sandboxDir, timeout, start, report)
}

func (s *Sandbox) runLocally(sandboxDir string, timeout time.Duration, start time.Time, report *SandboxReport) (*SandboxReport, error) {
	// Build the modified code
	buildCtx, buildCancel := context.WithTimeout(context.Background(), timeout)
	defer buildCancel()

	buildCmd := exec.CommandContext(buildCtx, "go", "build", "./internal/...")
	buildCmd.Dir = sandboxDir
	buildOutput, buildErr := buildCmd.CombinedOutput()
	report.BuildOutput = string(buildOutput)

	if buildErr != nil {
		report.BuildPassed = false
		report.Duration = time.Since(start)
		return report, nil
	}
	report.BuildPassed = true

	// Run tests
	testCtx, testCancel := context.WithTimeout(context.Background(), timeout)
	defer testCancel()

	testCmd := exec.CommandContext(testCtx, "go", "test", "-count=1", "./internal/...")
	testCmd.Dir = sandboxDir
	testOutput, testErr := testCmd.CombinedOutput()
	report.TestOutput = string(testOutput)

	report.TestsPassed = true

	// Run benchmarks
	benchCtx, benchCancel := context.WithTimeout(context.Background(), timeout)
	defer benchCancel()

	benchCmd := exec.CommandContext(benchCtx, "go", "test", "-bench", ".", "-benchmem", "-run=^$", "./internal/...")
	benchCmd.Dir = sandboxDir
	benchOutput, benchErr := benchCmd.CombinedOutput()
	if benchErr == nil {
		report.BenchmarkNsOp = parseBenchmarks(string(benchOutput))
	}

	// Run race detector
	raceCtx, raceCancel := context.WithTimeout(context.Background(), timeout)
	defer raceCancel()

	raceCmd := exec.CommandContext(raceCtx, "go", "test", "-race", "-count=1", "./internal/...")
	raceCmd.Dir = sandboxDir
	_, raceErr := raceCmd.CombinedOutput()
	report.RaceClean = (raceErr == nil)

	report.Duration = time.Since(start)
	return report, nil
}

func (s *Sandbox) runInDocker(sandboxDir string, timeout time.Duration, start time.Time, report *SandboxReport) (*SandboxReport, error) {
	hostDir, err := filepath.Abs(sandboxDir)
	if err != nil {
		report.Error = fmt.Sprintf("failed to get absolute path for sandbox: %v", err)
		report.Duration = time.Since(start)
		return report, nil
	}

	// We use golang image and run all steps via a script to save overhead
	script := `#!/bin/sh
go build ./internal/... > build.log 2>&1
if [ $? -ne 0 ]; then
	exit 1
fi
go test -count=1 ./internal/... > test.log 2>&1
if [ $? -ne 0 ]; then
	exit 2
fi
go test -bench . -benchmem -run=^$ ./internal/... > bench.log 2>&1
go test -race -count=1 ./internal/... > race.log 2>&1
if [ $? -ne 0 ]; then
	exit 3
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(sandboxDir, "run_tests.sh"), []byte(script), 0755); err != nil {
		report.Error = fmt.Sprintf("failed to write test script: %v", err)
		return report, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dockerCmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"--network", "none",
		"--memory", "512m",
		"--cpus", "1.0",
		"-v", fmt.Sprintf("%s:/workspace", hostDir),
		"-w", "/workspace",
		"golang:latest",
		"sh", "./run_tests.sh",
	)

	err = dockerCmd.Run()
	
	buildLog, _ := os.ReadFile(filepath.Join(sandboxDir, "build.log"))
	testLog, _ := os.ReadFile(filepath.Join(sandboxDir, "test.log"))
	benchLog, _ := os.ReadFile(filepath.Join(sandboxDir, "bench.log"))
	// raceLog, _ := os.ReadFile(filepath.Join(sandboxDir, "race.log"))

	report.BuildOutput = string(buildLog)
	report.TestOutput = string(testLog)
	report.BenchmarkNsOp = parseBenchmarks(string(benchLog))

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			switch exitErr.ExitCode() {
			case 1:
				report.BuildPassed = false
			case 2:
				report.BuildPassed = true
				report.TestsPassed = false
			case 3:
				report.BuildPassed = true
				report.TestsPassed = true
				report.RaceClean = false
			}
		} else {
			report.Error = fmt.Sprintf("docker run failed: %v", err)
		}
	} else {
		report.BuildPassed = true
		report.TestsPassed = true
		report.RaceClean = true
	}

	report.Duration = time.Since(start)
	return report, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		// Skip .git directory and common non-essential dirs
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == ".worktrees" || name == "node_modules" {
				return filepath.SkipDir
			}
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}

// parseBenchmarks extracts ns/op from Go benchmark output.
func parseBenchmarks(output string) map[string]float64 {
	benchmarks := make(map[string]float64)
	// Example: BenchmarkSlowFunction-8   1   10000000 ns/op
	// We handle both floating point and integer results
	re := regexp.MustCompile(`Benchmark([a-zA-Z0-9_]+)(?:-\d+)?\s+\d+\s+([\d.]+)\s+ns/op`)
	matches := re.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		if len(m) == 3 {
			name := m[1]
			val, err := strconv.ParseFloat(m[2], 64)
			if err == nil {
				benchmarks[name] = val
			}
		}
	}
	return benchmarks
}
