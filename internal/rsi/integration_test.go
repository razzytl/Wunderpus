package rsi

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wunderpus/wunderpus/internal/audit"
)

func TestFullRSICycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	repoRoot := setupTestRepo(t)

	// We need a dummy code file to profile and optimize
	dummyCode := `package testpkg

import "time"

func SlowFunction() {
	time.Sleep(50 * time.Millisecond)
}
`
	err := os.WriteFile(filepath.Join(repoRoot, "internal", "testpkg", "slow.go"), []byte(dummyCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoRoot
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "add slow func")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	// 1. Audit Log & Profiler
	dbPath := filepath.Join(t.TempDir(), "audit.db")
	auditLog, _ := audit.NewAuditLog(dbPath)
	defer auditLog.Close()

	profiler, _ := NewProfiler("")
	profiler.Track("testpkg.SlowFunction", func() error {
		time.Sleep(50 * time.Millisecond) // Simulated initial latency
		return nil
	})

	// 2. CodeMapper
	mapper := NewCodeMapper(false)
	codeMap, err := mapper.Build(repoRoot)
	if err != nil {
		t.Fatalf("CodeMapper failed: %v", err)
	}

	// 3. WeaknessReporter
	reporter := NewWeaknessReporter(profiler, mapper)
	report := reporter.Generate(codeMap)

	if len(report.TopCandidates) == 0 {
		t.Fatal("Expected at least one weakness entry")
	}
	target := report.TopCandidates[0]

	// 4. Proposer (Mocked via API override to ensure it returns a valid faster function)
	// We'll mock the LLM client behavior or just inject a Proposal directly since 
	// ProposalEngine needs an API key and network call.
	
	// Create a dummy proposal
	diff := `--- a/internal/testpkg/slow.go
+++ b/internal/testpkg/slow.go
@@ -4,3 +4,3 @@
 func SlowFunction() {
-	time.Sleep(50 * time.Millisecond)
+	time.Sleep(1 * time.Millisecond)
 }
`
	proposals := []Proposal{
		{
			ID:             "prop-test-1",
			TargetFunction: target.FunctionNode.QualifiedName,
			Diff:           diff,
			Temperature:    0.2,
		},
	}

	// 5. FitnessEvaluator (using mock SandboxReport for integration test)
	// The sandbox unit tests cover real execution; here we test component wiring.
	_ = NewSandbox(repoRoot) // verify it constructs correctly
	mockReport := &SandboxReport{
		PatchApplied: true,
		BuildPassed:  true,
		TestsPassed:  true,
		RaceClean:    true,
		BenchmarkNsOp: map[string]float64{
			target.FunctionNode.QualifiedName: 1e6, // 1ms
		},
	}

	evaluator := NewFitnessEvaluator(nil, auditLog)

	beforeStats, _ := profiler.GetStats(target.FunctionNode.QualifiedName)
	t.Logf("Before stats: name=%s, P99=%d", beforeStats.FunctionName, beforeStats.P99LatencyNs)

	fitness := evaluator.Score(beforeStats, *mockReport)
	if fitness <= 0.05 {
		t.Fatalf("Expected fitness > 0.05, got %v", fitness)
	}

	// 6. Deployer
	deployer := NewDeployer(repoRoot, true, auditLog)
	
	err = deployer.Deploy(proposals[0], fitness)
	if err != nil {
		t.Fatalf("Deployer failed: %v", err)
	}

	// Verify the file was changed in the repo
	content, _ := os.ReadFile(filepath.Join(repoRoot, "internal", "testpkg", "slow.go"))
	if !strings.Contains(string(content), "time.Sleep(1 * time.Millisecond)") {
		t.Errorf("File was not updated by deployer:\n%s", string(content))
	}

	// Check branch
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoRoot
	out, _ := cmd.CombinedOutput()
	if !strings.HasPrefix(string(out), "rsi/auto-") {
		t.Errorf("Expected branch to be changed to rsi/auto-, got %s", string(out))
	}
	
	// Check audit log
	entries, _ := auditLog.Query(audit.AuditFilter{EventType: audit.EventRSIDeployed})
	if len(entries) == 0 {
		t.Errorf("Expected deploy audit event")
	}

	// Phase 1 integration test complete!
}
