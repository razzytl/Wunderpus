package rsi

import (
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Deployer applies winning RSI proposals to the live codebase via git.
type Deployer struct {
	repoRoot        string
	firewallEnabled bool
}

// NewDeployer creates a deployer for the given repository root.
func NewDeployer(repoRoot string, firewallEnabled bool) *Deployer {
	return &Deployer{
		repoRoot:        repoRoot,
		firewallEnabled: firewallEnabled,
	}
}

// Deploy applies the winning diff, commits to a new branch, tags the previous
// commit as a rollback target, and rebuilds the binary.
func (d *Deployer) Deploy(proposal Proposal, fitness float64) error {
	// RSI Firewall: scan diff for paths outside internal/
	if d.firewallEnabled {
		if err := d.checkFirewall(proposal.Diff); err != nil {
			return fmt.Errorf("rsi firewall: %w", err)
		}
	}

	// Tag previous commit as rollback target BEFORE any changes
	prevTag := fmt.Sprintf("rsi/rollback-%s", time.Now().Format("20060102150405"))
	tagCmd := exec.Command("git", "tag", prevTag, "HEAD")
	tagCmd.Dir = d.repoRoot
	if tagOutput, tagErr := tagCmd.CombinedOutput(); tagErr != nil {
		slog.Warn("rsi deployer: failed to tag rollback", "error", string(tagOutput))
	}

	// Create a new branch for the improvement BEFORE applying
	branchName := fmt.Sprintf("rsi/auto-%s", time.Now().Format("2006-01-02-150405"))
	branchCmd := exec.Command("git", "checkout", "-b", branchName)
	branchCmd.Dir = d.repoRoot
	if branchOutput, branchErr := branchCmd.CombinedOutput(); branchErr != nil {
		slog.Warn("rsi deployer: branch creation failed (may already be on branch)",
			"error", string(branchOutput))
	}

	// Apply the diff to the live source tree
	applyCmd := exec.Command("git", "apply")
	applyCmd.Dir = d.repoRoot
	applyCmd.Stdin = strings.NewReader(proposal.Diff)
	output, err := applyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsi deployer: patch apply failed: %s", string(output))
	}

	// Stage the changes
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = d.repoRoot
	if addOutput, addErr := addCmd.CombinedOutput(); addErr != nil {
		return fmt.Errorf("rsi deployer: git add failed: %s", string(addOutput))
	}

	// Commit with fitness metrics
	commitMsg := fmt.Sprintf("RSI: improve %s (fitness=%.4f)\n\nTarget: %s\nTemperature: %.1f\nRollback tag: %s",
		proposal.TargetFunction,
		fitness,
		proposal.TargetFunction,
		proposal.Temperature,
		prevTag)

	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = d.repoRoot
	commitOutput, commitErr := commitCmd.CombinedOutput()
	if commitErr != nil {
		return fmt.Errorf("rsi deployer: git commit failed: %s", string(commitOutput))
	}

	// Build the new binary
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = d.repoRoot
	buildOutput, buildErr := buildCmd.CombinedOutput()
	if buildErr != nil {
		// Build failed — rollback
		slog.Error("rsi deployer: build failed after deploy, rolling back",
			"error", string(buildOutput))
		if rbErr := d.Rollback(prevTag); rbErr != nil {
			return fmt.Errorf("rsi deployer: post-deploy build failed AND rollback failed: build=%v, rollback=%v", buildErr, rbErr)
		}
		return fmt.Errorf("rsi deployer: post-deploy build failed, rolled back to %s", prevTag)
	}

	slog.Info("rsi deployer: deployment complete",
		"branch", branchName,
		"fitness", fitness,
		"target", proposal.TargetFunction,
		"rollback", prevTag,
	)

	return nil
}

// MonitorPostDeploy starts a background goroutine that monitors error rates
// for 10 minutes post-deploy. If error rate increases by >20% vs baseline,
// auto-rollback is triggered.
func (d *Deployer) MonitorPostDeploy(profiler *Profiler, baseline SpanStats, rollbackTag string) {
	go func() {
		slog.Info("rsi deployer: starting post-deploy monitoring", "duration", "10m", "tag", rollbackTag)

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		deadline := time.Now().Add(10 * time.Minute)

		for time.Now().Before(deadline) {
			<-ticker.C

			stats, ok := profiler.GetStats(baseline.FunctionName)
			if !ok || stats.CallCount == 0 {
				continue
			}

			currentErrorRate := float64(stats.ErrorCount) / float64(stats.CallCount)
			baselineErrorRate := float64(0)
			if baseline.CallCount > 0 {
				baselineErrorRate = float64(baseline.ErrorCount) / float64(baseline.CallCount)
			}

			// If error rate increased by >20% relative to baseline
			if baselineErrorRate > 0 && currentErrorRate > baselineErrorRate*1.2 {
				slog.Error("rsi deployer: regression detected, auto-rolling back",
					"baseline_error_rate", baselineErrorRate,
					"current_error_rate", currentErrorRate,
					"tag", rollbackTag,
				)
				if err := d.Rollback(rollbackTag); err != nil {
					slog.Error("rsi deployer: auto-rollback failed", "error", err)
				}
				return
			}
		}

		slog.Info("rsi deployer: post-deploy monitoring complete, no regression detected")
	}()
}

// Rollback reverts to the given git tag and rebuilds.
func (d *Deployer) Rollback(tag string) error {
	slog.Info("rsi deployer: rolling back", "tag", tag)

	// Stash any uncommitted changes
	stashCmd := exec.Command("git", "stash", "--include-untracked")
	stashCmd.Dir = d.repoRoot
	_ = stashCmd.Run()

	// Checkout the rollback tag
	checkoutCmd := exec.Command("git", "checkout", tag)
	checkoutCmd.Dir = d.repoRoot
	output, err := checkoutCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsi deployer: rollback checkout failed: %s", string(output))
	}

	// Rebuild
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = d.repoRoot
	buildOutput, buildErr := buildCmd.CombinedOutput()
	if buildErr != nil {
		return fmt.Errorf("rsi deployer: rollback build failed: %s", string(buildOutput))
	}

	slog.Info("rsi deployer: rollback complete", "tag", tag)
	return nil
}

// checkFirewall scans a diff for any path modifications outside internal/.
func (d *Deployer) checkFirewall(diff string) error {
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			path := strings.TrimPrefix(line, "+++ ")
			path = strings.TrimPrefix(path, "--- ")
			path = strings.TrimPrefix(path, "a/")
			path = strings.TrimPrefix(path, "b/")

			// Normalize path separators
			path = filepath.ToSlash(path)

			// Must be under internal/
			if !strings.HasPrefix(path, "internal/") && path != "/dev/null" {
				return fmt.Errorf("diff modifies %s which is outside internal/ — blocked", path)
			}
		}
	}
	return nil
}
