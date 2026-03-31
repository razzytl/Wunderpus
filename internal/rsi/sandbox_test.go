package rsi

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func createSandboxTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create a minimal Go module
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testrepo\n\ngo 1.25\n"), 0o644)

	// Create internal/testpkg/ with a valid Go file
	pkgDir := filepath.Join(dir, "internal", "testpkg")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "test.go"), []byte(`package testpkg

func Add(a, b int) int {
	return a + b
}
`), 0o644)

	// Create internal/testpkg/ with a test file
	os.WriteFile(filepath.Join(pkgDir, "test_test.go"), []byte(`package testpkg

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("Add failed")
	}
}
`), 0o644)

	// Initialize git repo (required for git apply)
	gitInit := execCommand(t, dir, "git", "init")
	if gitInit != nil {
		t.Skipf("git not available: %v", gitInit)
	}
	execCommand(t, dir, "git", "config", "user.email", "test@test.com")
	execCommand(t, dir, "git", "config", "user.name", "Test")
	execCommand(t, dir, "git", "config", "core.autocrlf", "false") // prevent CRLF on Windows
	execCommand(t, dir, "git", "add", "-A")
	execCommand(t, dir, "git", "commit", "-m", "init")

	return dir
}

func execCommand(t *testing.T, dir string, name string, args ...string) error {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

// TestSandbox_RunKnownGoodDiff applies a valid diff and expects all green.
func TestSandbox_RunKnownGoodDiff(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping sandbox test")
	}

	repoDir := createSandboxTestRepo(t)

	sandbox := NewSandbox(repoDir)

	// Valid diff: change return value (still valid Go, tests still pass)
	validDiff := `--- a/internal/testpkg/test.go
+++ b/internal/testpkg/test.go
@@ -1,5 +1,5 @@
 package testpkg
 
 func Add(a, b int) int {
-	return a + b
+	return a + b // unchanged
 }
`
	proposal := Proposal{
		ID:             "test-good",
		Diff:           validDiff,
		TargetFunction: "testpkg.Add",
	}

	report, err := sandbox.Run(proposal, repoDir)
	if err != nil {
		t.Fatalf("Sandbox.Run: %v", err)
	}

	if !report.PatchApplied {
		t.Fatalf("expected patch applied, got error: %s", report.Error)
	}
	if !report.BuildPassed {
		t.Fatalf("expected build passed, got: %s", report.BuildOutput)
	}
	if !report.TestsPassed {
		t.Fatalf("expected tests passed, got: %s", report.TestOutput)
	}
	t.Logf("Duration: %v", report.Duration)
}

// TestSandbox_RunSyntaxErrorDiff applies a diff that introduces a syntax error.
func TestSandbox_RunSyntaxErrorDiff(t *testing.T) {
	repoDir := createSandboxTestRepo(t)

	sandbox := NewSandbox(repoDir)

	// Invalid diff: introduces syntax error (unclosed brace)
	badDiff := `--- a/internal/testpkg/test.go
+++ b/internal/testpkg/test.go
@@ -1,5 +1,5 @@
 package testpkg
 
 func Add(a, b int) int {
-	return a + b
+	return a + b {
`
	proposal := Proposal{
		ID:   "test-syntax-error",
		Diff: badDiff,
	}

	report, err := sandbox.Run(proposal, repoDir)
	if err != nil {
		t.Fatalf("Sandbox.Run: %v", err)
	}

	if !report.PatchApplied {
		// Some patches may fail at apply time, which is also acceptable
		t.Logf("Patch not applied (acceptable): %s", report.Error)
		return
	}

	if report.BuildPassed {
		t.Fatal("expected build to fail with syntax error")
	}
	t.Logf("Build correctly failed: %s", report.BuildOutput)
}

// TestSandbox_Cleanup verifies temp directory is removed after Run.
func TestSandbox_Cleanup(t *testing.T) {
	repoDir := createSandboxTestRepo(t)
	sandbox := NewSandbox(repoDir)
	sandbox.UseDocker = false // force local — Docker volume mounts can prevent cleanup on Windows

	tmpDir := os.TempDir()

	// Snapshot sandbox directories before test
	beforeEntries, _ := os.ReadDir(tmpDir)
	beforeSet := make(map[string]bool)
	for _, e := range beforeEntries {
		if strings.HasPrefix(e.Name(), "wunderpus-sandbox-") {
			beforeSet[e.Name()] = true
		}
	}

	validDiff := `--- a/internal/testpkg/test.go
+++ b/internal/testpkg/test.go
@@ -1,5 +1,5 @@
 package testpkg
 
 func Add(a, b int) int {
-	return a + b
+	return a + b // unchanged
 }
 `
	proposal := Proposal{ID: "cleanup-test", Diff: validDiff}
	sandbox.Run(proposal, repoDir)

	// Check only NEW sandbox directories were cleaned up
	afterEntries, _ := os.ReadDir(tmpDir)
	for _, e := range afterEntries {
		if strings.HasPrefix(e.Name(), "wunderpus-sandbox-") && !beforeSet[e.Name()] {
			t.Fatalf("sandbox directory not cleaned up: %s", e.Name())
		}
	}
}
