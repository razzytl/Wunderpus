package heartbeat

import (
	"testing"
	"time"
)

// TestHeartbeatTask_Fields tests HeartbeatTask fields
func TestHeartbeatTask_Fields(t *testing.T) {
	task := HeartbeatTask{
		Type:     "quick",
		Content:  "test content",
		Line:     10,
		CronExpr: "0 * * * *",
	}

	if task.Type != "quick" {
		t.Errorf("expected Type 'quick', got %q", task.Type)
	}
	if task.Content != "test content" {
		t.Errorf("expected Content 'test content', got %q", task.Content)
	}
	if task.Line != 10 {
		t.Errorf("expected Line 10, got %d", task.Line)
	}
	if task.CronExpr != "0 * * * *" {
		t.Errorf("expected CronExpr '0 * * * *', got %q", task.CronExpr)
	}
}

// TestTaskExecutor_Interface tests interface compliance
func TestTaskExecutor_Interface(t *testing.T) {
	// Verify HeartbeatExecutor implements TaskExecutor
	var _ TaskExecutor = (*HeartbeatExecutor)(nil)
}

// TestParseResult_Structure tests ParseResult structure
func TestParseResult_Structure(t *testing.T) {
	result := &ParseResult{
		QuickTasks:   []HeartbeatTask{{Type: "quick", Content: "quick1"}},
		LongTasks:    []HeartbeatTask{{Type: "long", Content: "long1"}},
		LastModified: time.Now(),
	}

	if len(result.QuickTasks) != 1 {
		t.Errorf("expected 1 quick task, got %d", len(result.QuickTasks))
	}
	if len(result.LongTasks) != 1 {
		t.Errorf("expected 1 long task, got %d", len(result.LongTasks))
	}
	if result.LastModified.IsZero() {
		t.Error("expected non-zero LastModified")
	}
}

// TestScheduler_GetStatus tests getting scheduler status
func TestScheduler_GetStatus(t *testing.T) {
	cfg := &HeartbeatConfig{
		Enabled:   true,
		Interval:  60,
		Workspace: "/tmp/test",
	}
	parser := NewParser()

	scheduler := &Scheduler{
		cfg:       cfg,
		parser:    parser,
		workspace: "/tmp/test",
		lastCheck: time.Now(),
	}

	status := scheduler.GetStatus()

	if status["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", status["enabled"])
	}
	if status["workspace"] != "/tmp/test" {
		t.Errorf("expected workspace='/tmp/test', got %v", status["workspace"])
	}
}

// TestDefaultIntervals_Map tests default interval mappings
func TestDefaultIntervals_Map(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"hourly", "0 * * * *"},
		{"daily", "0 0 * * *"},
		{"weekly", "0 0 * * 0"},
		{"monthly", "0 0 1 * *"},
		{"yearly", "0 0 1 1 *"},
	}

	for _, tt := range tests {
		result, ok := DefaultIntervals[tt.key]
		if !ok {
			t.Errorf("expected key %q to exist", tt.key)
		}
		if result != tt.expected {
			t.Errorf("DefaultIntervals[%q] = %q, want %q", tt.key, result, tt.expected)
		}
	}
}

// TestFindHeartbeatFile_NonExistent tests finding heartbeat file in non-existent directory
func TestFindHeartbeatFile_NonExistent(t *testing.T) {
	_, err := FindHeartbeatFile("/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}
