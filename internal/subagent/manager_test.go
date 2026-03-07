package subagent

import (
	"testing"
	"time"
)

func TestSubAgent_GetTask(t *testing.T) {
	sub := &SubAgent{
		Task: "Test task",
	}

	task := sub.GetTask()
	if task != "Test task" {
		t.Errorf("expected 'Test task', got %s", task)
	}
}

func TestSubAgent_GetStatus(t *testing.T) {
	sub := &SubAgent{
		Status: StatusRunning,
	}

	status := sub.GetStatus()
	if status != StatusRunning {
		t.Errorf("expected StatusRunning, got %s", status)
	}
}

func TestSubAgent_GetResult(t *testing.T) {
	sub := &SubAgent{
		Result: "Task completed successfully",
	}

	result := sub.GetResult()
	if result != "Task completed successfully" {
		t.Errorf("expected result, got %s", result)
	}
}

func TestSubAgent_GetError(t *testing.T) {
	sub := &SubAgent{
		Error: "Something went wrong",
	}

	err := sub.GetError()
	if err != "Something went wrong" {
		t.Errorf("expected error message, got %s", err)
	}
}

func TestStatus_Constants(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.status)
		}
	}
}

func TestSubAgent_Timestamps(t *testing.T) {
	now := time.Now()
	sub := &SubAgent{
		CreatedAt: now,
	}

	if sub.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestSubAgent_WithStartedAt(t *testing.T) {
	now := time.Now()
	sub := &SubAgent{
		StartedAt: &now,
	}

	if sub.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}
}

func TestSubAgent_WithCompletedAt(t *testing.T) {
	now := time.Now()
	sub := &SubAgent{
		CompletedAt: &now,
	}

	if sub.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestSubAgent_SessionID(t *testing.T) {
	sub := &SubAgent{
		SessionID: "subagent_test123",
	}

	if sub.SessionID != "subagent_test123" {
		t.Errorf("expected session ID, got %s", sub.SessionID)
	}
}

func TestSubAgent_ID(t *testing.T) {
	sub := &SubAgent{
		ID: "abc123def456",
	}

	if sub.ID != "abc123def456" {
		t.Errorf("expected ID, got %s", sub.ID)
	}

	// Test ID prefix access (first 8 chars)
	if sub.ID[:8] != "abc123de" {
		t.Errorf("expected prefix, got %s", sub.ID[:8])
	}
}

// Note: Full integration tests for Manager would require
// mocking agent.Manager and provider.Router, which is complex.
// The above tests cover the SubAgent data structure.
