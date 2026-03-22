package rsi

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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
}

// NewSandbox creates a sandbox for the given repository root.
func NewSandbox(repoRoot string) *Sandbox {
	return &Sandbox{
		repoRoot:   repoRoot,
		workDir:    os.TempDir(),
		timeoutSec: 60,
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

	if testErr != nil {
		report.TestsPassed = false
		report.Duration = time.Since(start)
		return report, nil
	}
	report.TestsPassed = true

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
