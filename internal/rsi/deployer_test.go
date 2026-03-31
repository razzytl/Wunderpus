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

func setupTestRepo(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "test")
	cmd.Dir = repoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	_ = cmd.Run()
	// Prevent CRLF conversion on Windows — patches must match exactly
	cmd = exec.Command("git", "config", "core.autocrlf", "false")
	cmd.Dir = repoDir
	_ = cmd.Run()

	// Need a go.mod
	err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module testmod\n\ngo 1.21\n"), 0o644)
	if err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Make an internal/testpkg/test.go
	err = os.MkdirAll(filepath.Join(repoDir, "internal", "testpkg"), 0o755)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	testGo := `package testpkg
	
func Add(a, b int) int {
	return a + b
}
`
	err = os.WriteFile(filepath.Join(repoDir, "internal", "testpkg", "test.go"), []byte(testGo), 0o644)
	if err != nil {
		t.Fatalf("write test.go: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	_ = cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = repoDir
	_ = cmd.Run()

	return repoDir
}

func TestDeployer_DeployValid(t *testing.T) {
	repoRoot := setupTestRepo(t)

	dbPath := filepath.Join(t.TempDir(), "audit.db")
	auditLog, err := audit.NewAuditLog(dbPath)
	if err != nil {
		t.Fatalf("failed to open audit log: %v", err)
	}
	defer auditLog.Close()

	d := NewDeployer(repoRoot, true, auditLog)

	diff := `--- a/internal/testpkg/test.go
+++ b/internal/testpkg/test.go
@@ -2,3 +2,4 @@
        
 func Add(a, b int) int {
+       // Now it is better!
        return a + b
`
	proposal := Proposal{
		TargetFunction: "testpkg.Add",
		Diff:           diff,
		Temperature:    0.5,
	}

	err = d.Deploy(proposal, 0.15)
	if err != nil {
		t.Fatalf("Deploy() failed: %v", err)
	}

	// Verify branch changed
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoRoot
	out, _ := cmd.CombinedOutput()
	if !strings.HasPrefix(string(out), "rsi/auto-") {
		t.Errorf("Expected branch rsi/auto-xxx, got %s", string(out))
	}

	// Verify tag created
	cmd = exec.Command("git", "tag")
	cmd.Dir = repoRoot
	out, _ = cmd.CombinedOutput()
	if !strings.Contains(string(out), "rsi/rollback-") {
		t.Errorf("Expected tag rsi/rollback-xxx, got %s", string(out))
	}

	// Verify file changed
	content, _ := os.ReadFile(filepath.Join(repoRoot, "internal", "testpkg", "test.go"))
	if !strings.Contains(string(content), "Now it is better!") {
		t.Errorf("File was not modified. Content:\n%s", string(content))
	}

	// Verify audit log has event
	entries, _ := auditLog.Query(audit.AuditFilter{EventType: audit.EventRSIDeployed})
	if len(entries) != 1 {
		t.Errorf("Expected 1 audit entry, got %d", len(entries))
	}
}

func TestDeployer_Rollback(t *testing.T) {
	repoRoot := setupTestRepo(t)
	d := NewDeployer(repoRoot, true, nil)

	// Create a tag
	cmd := exec.Command("git", "tag", "mytag")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	// Make a change
	err := os.WriteFile(filepath.Join(repoRoot, "internal", "testpkg", "test.go"), []byte("package testpkg\n\nfunc Add(a, b int) int { return a + b + 1 }\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("git", "add", "internal/testpkg/test.go")
	cmd.Dir = repoRoot
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "bad commit")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	// Roll back
	err = d.Rollback("mytag")
	if err != nil {
		t.Fatalf("Rollback() failed: %v", err)
	}

	// Check syntax is gone
	content, _ := os.ReadFile(filepath.Join(repoRoot, "internal", "testpkg", "test.go"))
	if strings.Contains(string(content), "return a + b + 1") {
		t.Errorf("Rollback did not revert file")
	}
}

func TestDeployer_MonitorPostDeploy(t *testing.T) {
	profiler, _ := NewProfiler("")

	// Track once to establish a baseline error count of 0 out of 1
	profiler.Track("target_func", func() error { return nil })

	baseline, ok := profiler.GetStats("target_func")
	if !ok {
		t.Fatalf("failed to get baseline")
	}

	repoRoot := setupTestRepo(t)

	dbPath := filepath.Join(t.TempDir(), "audit.db")
	auditLog, _ := audit.NewAuditLog(dbPath)
	defer auditLog.Close()

	d := NewDeployer(repoRoot, true, auditLog)

	d.monitorInterval = 50 * time.Millisecond
	d.monitorDuration = 500 * time.Millisecond

	// Create a tag to rollback to
	cmd := exec.Command("git", "tag", "rollback-target")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	// Create a bad change and commit it
	os.WriteFile(filepath.Join(repoRoot, "internal", "testpkg", "test.go"), []byte("package testpkg\n\nfunc Add(a, b int) int { return a + b + 1 }\n"), 0o644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoRoot
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "bad code")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	// Start monitor
	d.MonitorPostDeploy(profiler, baseline, "rollback-target")

	// Inject errors quickly so current error rate jumps to 1.0
	for i := 0; i < 5; i++ {
		profiler.Track("target_func", func() error { return os.ErrPermission })
	}

	// Wait for monitor to detect and rollback
	// Rollback takes time (git stash, checkout, go build)
	var entries []audit.AuditEntry
	for i := 0; i < 50; i++ {
		entries, _ = auditLog.Query(audit.AuditFilter{EventType: audit.EventRSIRolledBack})
		if len(entries) == 1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Verify rollback happened
	content, _ := os.ReadFile(filepath.Join(repoRoot, "internal", "testpkg", "test.go"))
	if strings.Contains(string(content), "return a + b + 1") {
		t.Errorf("Rollback was not triggered by MonitorPostDeploy! File content: %s", string(content))
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 audit entry for rollback, got %d", len(entries))
	}
}
