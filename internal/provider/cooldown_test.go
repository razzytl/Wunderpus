package provider

import (
	"testing"
	"time"
)

func TestCooldownTracker_Creation(t *testing.T) {
	ct := NewCooldownTracker()
	if ct == nil {
		t.Error("expected non-nil cooldown tracker")
	}
}

func TestCooldownTracker_StartCooldown(t *testing.T) {
	ct := NewCooldownTracker()
	ct.StartCooldown("test-provider", 30*time.Second)

	if !ct.IsInCooldown("test-provider") {
		t.Error("expected provider to be in cooldown")
	}
}

func TestCooldownTracker_IsInCooldown(t *testing.T) {
	ct := NewCooldownTracker()

	if ct.IsInCooldown("nonexistent") {
		t.Error("expected nonexistent provider to not be in cooldown")
	}
}

func TestCooldownTracker_RecordFailure(t *testing.T) {
	ct := NewCooldownTracker()

	// Record multiple failures to trigger cooldown
	for i := 0; i < 6; i++ {
		ct.RecordFailure("test-provider")
	}

	if !ct.IsInCooldown("test-provider") {
		t.Error("expected provider to be in cooldown after 6 failures")
	}
}

func TestCooldownTracker_RecordSuccess(t *testing.T) {
	ct := NewCooldownTracker()

	ct.StartCooldown("test-provider", 30*time.Second)
	ct.RecordSuccess("test-provider")

	if ct.IsInCooldown("test-provider") {
		t.Error("expected provider to not be in cooldown after success")
	}
}

func TestCooldownTracker_GetFailCount(t *testing.T) {
	ct := NewCooldownTracker()

	ct.RecordFailure("test-provider")
	ct.RecordFailure("test-provider")

	if ct.GetFailCount("test-provider") != 2 {
		t.Errorf("expected fail count 2, got %d", ct.GetFailCount("test-provider"))
	}
}

func TestFallbackAttempt_Structure(t *testing.T) {
	fa := FallbackAttempt{
		Provider: "openai",
		Model:    "gpt-4",
		Error:    nil,
		Reason:   FailoverReasonTimeout,
		Duration: 100 * time.Millisecond,
		Skipped:  false,
	}

	if fa.Provider != "openai" {
		t.Errorf("expected openai, got %s", fa.Provider)
	}
	if fa.Duration != 100*time.Millisecond {
		t.Errorf("expected 100ms, got %v", fa.Duration)
	}
}

func TestFallbackResult_Structure(t *testing.T) {
	fr := FallbackResult{
		Response: &CompletionResponse{Content: "test response"},
		Provider: "openai",
		Model:    "gpt-4",
		Attempts: []FallbackAttempt{},
	}

	if fr.Response.Content != "test response" {
		t.Errorf("expected test response, got %s", fr.Response.Content)
	}
	if fr.Provider != "openai" {
		t.Errorf("expected openai, got %s", fr.Provider)
	}
}
