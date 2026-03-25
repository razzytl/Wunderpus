package rsi

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func createWasmTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testrepo\n\ngo 1.21\n"), 0644)

	pkgDir := filepath.Join(dir, "internal", "testpkg")
	os.MkdirAll(pkgDir, 0755)
	os.WriteFile(filepath.Join(pkgDir, "test.go"), []byte(`package testpkg

func Add(a, b int) int {
	return a + b
}
`), 0644)
	os.WriteFile(filepath.Join(pkgDir, "test_test.go"), []byte(`package testpkg

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("Add failed")
	}
}
`), 0644)

	// Init git
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	for _, c := range [][]string{
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"config", "core.autocrlf", "false"},
		{"add", "-A"},
		{"commit", "-m", "init"},
	} {
		cmd = exec.Command("git", c...)
		cmd.Dir = dir
		_ = cmd.Run()
	}

	return dir
}

func TestWasmSandbox_FallbackToDocker(t *testing.T) {
	repoDir := createWasmTestRepo(t)

	// useWasm=true but TinyGo is likely not installed, so it should fall back
	sandbox := NewWasmSandbox(repoDir, true)
	sandbox.timeoutSec = 30

	validDiff := `--- a/internal/testpkg/test.go
+++ b/internal/testpkg/test.go
@@ -1,5 +1,5 @@
 package testpkg
 
 func Add(a, b int) int {
-	return a + b
+	return a + b // unchanged
 }
 `
	proposal := Proposal{ID: "wasm-fallback-test", Diff: validDiff}
	report, err := sandbox.Run(proposal, repoDir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should succeed via fallback (either WASM or Docker/local)
	if !report.PatchApplied {
		t.Fatalf("expected patch applied, got error: %s", report.Error)
	}
	if !report.BuildPassed {
		t.Fatalf("expected build passed: %s", report.BuildOutput)
	}
	if !report.TestsPassed {
		t.Fatalf("expected tests passed: %s", report.TestOutput)
	}
}

func TestWasmSandbox_UseDockerDirectly(t *testing.T) {
	repoDir := createWasmTestRepo(t)

	// useWasm=false — should go directly to Docker/local sandbox
	sandbox := NewWasmSandbox(repoDir, false)
	sandbox.timeoutSec = 30

	validDiff := `--- a/internal/testpkg/test.go
+++ b/internal/testpkg/test.go
@@ -1,5 +1,5 @@
 package testpkg
 
 func Add(a, b int) int {
-	return a + b
+	return a + b // unchanged
 }
 `
	proposal := Proposal{ID: "wasm-docker-direct", Diff: validDiff}
	report, err := sandbox.Run(proposal, repoDir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !report.PatchApplied {
		t.Fatalf("expected patch applied, got error: %s", report.Error)
	}
	if !report.BuildPassed {
		t.Fatalf("expected build passed: %s", report.BuildOutput)
	}
}

func TestWasmSandbox_BadDiffFailsGracefully(t *testing.T) {
	repoDir := createWasmTestRepo(t)

	sandbox := NewWasmSandbox(repoDir, false)
	sandbox.timeoutSec = 30

	badDiff := `--- a/internal/testpkg/test.go
+++ b/internal/testpkg/test.go
@@ -1,5 +1,5 @@
 package testpkg
 
 func Add(a, b int) int {
-	return a + b
+	return a + b {
 `
	proposal := Proposal{ID: "wasm-bad-diff", Diff: badDiff}
	report, err := sandbox.Run(proposal, repoDir)
	if err != nil {
		t.Fatalf("Run should not return error: %v", err)
	}

	// Either patch fails or build fails — both are acceptable
	if report.PatchApplied && report.BuildPassed {
		t.Fatal("expected either patch or build to fail with bad diff")
	}
}

func TestWasmSandbox_Cleanup(t *testing.T) {
	repoDir := createWasmTestRepo(t)
	sandbox := NewWasmSandbox(repoDir, false)
	sandbox.timeoutSec = 30

	tmpDir := os.TempDir()

	// Snapshot wasm sandbox dirs before test
	beforeEntries, _ := os.ReadDir(tmpDir)
	beforeSet := make(map[string]bool)
	for _, e := range beforeEntries {
		if filepath.Base(e.Name()) != "" {
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
	proposal := Proposal{ID: "wasm-cleanup", Diff: validDiff}
	sandbox.Run(proposal, repoDir)

	// Check no wunderpus-wasm-* directories remain (new ones only)
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "wunderpus-wasm-") && !beforeSet[e.Name()] {
			t.Fatalf("wasm sandbox directory not cleaned up: %s", e.Name())
		}
	}
}
