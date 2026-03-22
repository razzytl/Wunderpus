package uaa

import (
	"context"
	"testing"
	"time"
)

func TestShadowSimulator_DangerousFileRejected(t *testing.T) {
	// Mock LLM judge that rejects dangerous file writes
	judgeFn := func(ctx context.Context, summary string) (bool, string, error) {
		if contains(summary, "/etc/passwd") {
			return false, "modifying /etc/passwd is dangerous", nil
		}
		return true, "looks safe", nil
	}

	shadow := NewShadowSimulator(judgeFn)

	action := Action{
		ID:          "test-danger",
		Description: "Write to passwd",
		Tool:        "write_file",
		Parameters:  map[string]interface{}{"path": "/etc/passwd", "content": "hacked"},
		Tier:        TierPersistent,
		TrustCost:   5,
		Scope:       ScopeDestructive,
	}

	result, err := shadow.Simulate(context.Background(), action)
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}

	if result.Approved {
		t.Fatal("should have rejected /etc/passwd write")
	}
}

func TestShadowSimulator_SafeActionApproved(t *testing.T) {
	judgeFn := func(ctx context.Context, summary string) (bool, string, error) {
		return true, "safe action", nil
	}

	shadow := NewShadowSimulator(judgeFn)

	action := Action{
		ID:          "test-safe",
		Description: "Write temp file",
		Tool:        "write_file",
		Parameters:  map[string]interface{}{"path": "/tmp/test.txt", "content": "hello"},
		Tier:        TierEphemeral,
		TrustCost:   1,
		Scope:       ScopeLocal,
	}

	result, err := shadow.Simulate(context.Background(), action)
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}

	if !result.Approved {
		t.Fatalf("should have approved safe action, got: %s", result.Reason)
	}
}

func TestShadowSimulator_Caching(t *testing.T) {
	callCount := 0
	judgeFn := func(ctx context.Context, summary string) (bool, string, error) {
		callCount++
		return true, "ok", nil
	}

	shadow := NewShadowSimulator(judgeFn)

	action := Action{
		ID:         "test-cache",
		Tool:       "read_file",
		Parameters: map[string]interface{}{"path": "/tmp/file.txt"},
		Tier:       TierPersistent,
		TrustCost:  5,
	}

	// First call
	shadow.Simulate(context.Background(), action)
	// Second call (same action = same cache key)
	shadow.Simulate(context.Background(), action)

	// Judge should only be called once due to caching
	if callCount != 1 {
		t.Fatalf("expected 1 judge call (cached second), got %d", callCount)
	}
}

func TestShadowSimulator_Timeout(t *testing.T) {
	judgeFn := func(ctx context.Context, summary string) (bool, string, error) {
		// Respect context cancellation
		select {
		case <-ctx.Done():
			return false, "timeout", ctx.Err()
		case <-time.After(35 * time.Second):
			return true, "ok", nil
		}
	}

	shadow := NewShadowSimulator(judgeFn)
	shadow.timeout = 100 * time.Millisecond

	action := Action{
		ID:         "test-timeout",
		Tool:       "write_file",
		Parameters: map[string]interface{}{"path": "/tmp/file.txt"},
		Tier:       TierPersistent,
		TrustCost:  5,
	}

	result, err := shadow.Simulate(context.Background(), action)
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}

	// Should auto-reject on timeout
	if result.Approved {
		t.Fatal("should have rejected due to timeout")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
