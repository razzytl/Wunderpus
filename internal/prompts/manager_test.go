package prompts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	// Create a prompt file
	err := os.WriteFile(filepath.Join(dir, "system_v1.md"), []byte("# System Prompt v1"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	m := NewManager(dir, "v1")
	if m == nil {
		t.Fatal("NewManager should not return nil")
	}

	if m.ActiveVersion() != "v1" {
		t.Errorf("expected active version 'v1', got %s", m.ActiveVersion())
	}
}

func TestManager_Load(t *testing.T) {
	dir := t.TempDir()
	content := "# System Prompt v1\n\nYou are a helpful assistant."
	err := os.WriteFile(filepath.Join(dir, "system_v1.md"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	m := NewManager(dir, "v1")

	loaded, err := m.Load("v1")
	if err != nil {
		t.Fatalf("failed to load prompt: %v", err)
	}
	if loaded != content {
		t.Errorf("expected %q, got %q", content, loaded)
	}

	// Second load should come from cache
	loaded2, err := m.Load("v1")
	if err != nil {
		t.Fatalf("failed to load from cache: %v", err)
	}
	if loaded2 != content {
		t.Errorf("expected cached content %q, got %q", content, loaded2)
	}
}

func TestManager_LoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir, "v1")

	_, err := m.Load("v2")
	if err == nil {
		t.Error("expected error for non-existent version")
	}
}

func TestManager_SetActive(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "system_v1.md"), []byte("# v1"), 0644)
	os.WriteFile(filepath.Join(dir, "system_v2.md"), []byte("# v2"), 0644)

	m := NewManager(dir, "v1")

	err := m.SetActive("v2")
	if err != nil {
		t.Fatalf("failed to set active version: %v", err)
	}
	if m.ActiveVersion() != "v2" {
		t.Errorf("expected active version 'v2', got %s", m.ActiveVersion())
	}
}

func TestManager_SetActiveNonExistent(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "system_v1.md"), []byte("# v1"), 0644)

	m := NewManager(dir, "v1")

	err := m.SetActive("v99")
	if err == nil {
		t.Error("expected error for non-existent version")
	}
}

func TestManager_ListVersions(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "system_v1.md"), []byte("# v1"), 0644)
	os.WriteFile(filepath.Join(dir, "system_v2.md"), []byte("# v2"), 0644)
	os.WriteFile(filepath.Join(dir, "system_v3.md"), []byte("# v3"), 0644)
	// Non-matching files should be ignored
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("readme"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("notes"), 0644)

	m := NewManager(dir, "v1")

	versions, err := m.ListVersions()
	if err != nil {
		t.Fatalf("failed to list versions: %v", err)
	}

	if len(versions) != 3 {
		t.Errorf("expected 3 versions, got %d: %v", len(versions), versions)
	}

	// Check all expected versions are present
	expected := map[string]bool{"v1": true, "v2": true, "v3": true}
	for _, v := range versions {
		if !expected[v] {
			t.Errorf("unexpected version: %s", v)
		}
	}
}

func TestManager_ListVersionsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir, "v1")

	versions, err := m.ListVersions()
	if err != nil {
		t.Fatalf("failed to list versions: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}
}

func TestManager_GetActive(t *testing.T) {
	dir := t.TempDir()
	content := "# Active Prompt"
	os.WriteFile(filepath.Join(dir, "system_current.md"), []byte(content), 0644)

	m := NewManager(dir, "current")

	active, err := m.GetActive()
	if err != nil {
		t.Fatalf("failed to get active: %v", err)
	}
	if active != content {
		t.Errorf("expected %q, got %q", content, active)
	}
}
