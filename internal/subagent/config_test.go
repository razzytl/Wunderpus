package subagent

import (
	"testing"
)

func TestSubAgent(t *testing.T) {
	sub := &SubAgent{
		ID:     "agent-1",
		Status: StatusRunning,
	}

	if sub.ID != "agent-1" {
		t.Errorf("expected ID 'agent-1', got %q", sub.ID)
	}

	if sub.Status != StatusRunning {
		t.Errorf("expected Status StatusRunning, got %v", sub.Status)
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusCancelled, "canceled"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.status)
			}
		})
	}
}
