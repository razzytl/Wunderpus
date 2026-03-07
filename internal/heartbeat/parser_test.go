package heartbeat

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParser_Parse(t *testing.T) {
	// Create a temporary HEARTBEAT.md file
	tmpDir := t.TempDir()
	heartbeatFile := filepath.Join(tmpDir, "HEARTBEAT.md")

	content := `# Heartbeat Test

## Quick Tasks (respond directly)
- Report the current time
- Summarize the conversation

## Long Tasks (spawn subagents)
- Search for tech news
- Check for updates
`

	err := os.WriteFile(heartbeatFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.Parse(heartbeatFile)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Verify quick tasks
	if len(result.QuickTasks) != 2 {
		t.Errorf("expected 2 quick tasks, got %d", len(result.QuickTasks))
	}

	if result.QuickTasks[0].Type != "quick" {
		t.Errorf("expected quick task type, got %s", result.QuickTasks[0].Type)
	}

	if result.QuickTasks[0].Content != "Report the current time" {
		t.Errorf("unexpected task content: %s", result.QuickTasks[0].Content)
	}

	// Verify long tasks
	if len(result.LongTasks) != 2 {
		t.Errorf("expected 2 long tasks, got %d", len(result.LongTasks))
	}

	if result.LongTasks[0].Type != "long" {
		t.Errorf("expected long task type, got %s", result.LongTasks[0].Type)
	}
}

func TestParser_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	heartbeatFile := filepath.Join(tmpDir, "HEARTBEAT.md")

	// Empty file
	err := os.WriteFile(heartbeatFile, []byte("# Empty"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.Parse(heartbeatFile)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(result.QuickTasks) != 0 {
		t.Errorf("expected 0 quick tasks, got %d", len(result.QuickTasks))
	}

	if len(result.LongTasks) != 0 {
		t.Errorf("expected 0 long tasks, got %d", len(result.LongTasks))
	}
}

func TestParser_NoTasks(t *testing.T) {
	tmpDir := t.TempDir()
	heartbeatFile := filepath.Join(tmpDir, "HEARTBEAT.md")

	// File with only headers, no tasks
	content := `# Heartbeat

## Quick Tasks

## Long Tasks
`

	err := os.WriteFile(heartbeatFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.Parse(heartbeatFile)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(result.QuickTasks) != 0 {
		t.Errorf("expected 0 quick tasks, got %d", len(result.QuickTasks))
	}

	if len(result.LongTasks) != 0 {
		t.Errorf("expected 0 long tasks, got %d", len(result.LongTasks))
	}
}

func TestParser_MalformedTasks(t *testing.T) {
	tmpDir := t.TempDir()
	heartbeatFile := filepath.Join(tmpDir, "HEARTBEAT.md")

	// File with malformed task items (no leading dash)
	content := `# Heartbeat

## Quick Tasks
This is not a task

## Long Tasks
- Valid task
`

	err := os.WriteFile(heartbeatFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.Parse(heartbeatFile)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Should only parse the valid task
	if len(result.LongTasks) != 1 {
		t.Errorf("expected 1 long task, got %d", len(result.LongTasks))
	}

	if result.LongTasks[0].Content != "Valid task" {
		t.Errorf("unexpected task content: %s", result.LongTasks[0].Content)
	}
}

func TestFindHeartbeatFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create HEARTBEAT.md in workspace root
	heartbeatFile := filepath.Join(tmpDir, "HEARTBEAT.md")
	err := os.WriteFile(heartbeatFile, []byte("# Test"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	path, err := FindHeartbeatFile(tmpDir)
	if err != nil {
		t.Fatalf("FindHeartbeatFile failed: %v", err)
	}

	if path != heartbeatFile {
		t.Errorf("expected %s, got %s", heartbeatFile, path)
	}
}

func TestFindHeartbeatFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindHeartbeatFile(tmpDir)
	if err == nil {
		t.Error("expected error when HEARTBEAT.md not found")
	}
}

func TestFindHeartbeatFile_InWunderpusDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create in .wunderpus subdirectory
	wunderpusDir := filepath.Join(tmpDir, ".wunderpus")
	err := os.MkdirAll(wunderpusDir, 0755)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	heartbeatFile := filepath.Join(wunderpusDir, "HEARTBEAT.md")
	err = os.WriteFile(heartbeatFile, []byte("# Test"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	path, err := FindHeartbeatFile(tmpDir)
	if err != nil {
		t.Fatalf("FindHeartbeatFile failed: %v", err)
	}

	if path != heartbeatFile {
		t.Errorf("expected %s, got %s", heartbeatFile, path)
	}
}

func TestHeartbeatConfig_Defaults(t *testing.T) {
	cfg := &HeartbeatConfig{
		Enabled:  true,
		Interval: 30,
	}

	if !cfg.Enabled {
		t.Error("expected heartbeat to be enabled")
	}

	if cfg.Interval != 30 {
		t.Errorf("expected interval 30, got %d", cfg.Interval)
	}
}

func TestHeartbeatTask_Structure(t *testing.T) {
	task := HeartbeatTask{
		Type:    "quick",
		Content: "Test task",
		Line:    10,
	}

	if task.Type != "quick" {
		t.Errorf("expected type 'quick', got %s", task.Type)
	}

	if task.Content != "Test task" {
		t.Errorf("expected content 'Test task', got %s", task.Content)
	}

	if task.Line != 10 {
		t.Errorf("expected line 10, got %d", task.Line)
	}
}

func TestParseResult_LastModified(t *testing.T) {
	now := time.Now()
	result := &ParseResult{
		QuickTasks:   []HeartbeatTask{},
		LongTasks:    []HeartbeatTask{},
		LastModified: now,
	}

	if result.LastModified.IsZero() {
		t.Error("expected LastModified to be set")
	}
}
