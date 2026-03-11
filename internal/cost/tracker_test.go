package cost

import (
	"os"
	"testing"
)

func TestNewTracker(t *testing.T) {
	tmpFile := t.TempDir() + "/test_cost.db"
	defer os.Remove(tmpFile)

	tracker, err := NewTracker(tmpFile, 10.0)
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	defer tracker.Close()

	if tracker == nil {
		t.Fatal("Tracker should not be nil")
	}

	if tracker.budget != 10.0 {
		t.Errorf("Expected budget 10.0, got %f", tracker.budget)
	}

	if tracker.prices == nil {
		t.Fatal("Prices map should not be nil")
	}
}

func TestTrack(t *testing.T) {
	tmpFile := t.TempDir() + "/test_track.db"
	defer os.Remove(tmpFile)

	tracker, err := NewTracker(tmpFile, 10.0)
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	defer tracker.Close()

	// Track some tokens
	err = tracker.Track("session-1", "gpt-4o", 1000, 500)
	if err != nil {
		t.Errorf("Track failed: %v", err)
	}

	cost := tracker.GetSessionCost("session-1")
	if cost <= 0 {
		t.Errorf("Expected positive cost, got %f", cost)
	}
}

func TestTrackUnknownModel(t *testing.T) {
	tmpFile := t.TempDir() + "/test_unknown.db"
	defer os.Remove(tmpFile)

	tracker, err := NewTracker(tmpFile, 10.0)
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	defer tracker.Close()

	// Track unknown model - should use default low price
	err = tracker.Track("session-1", "unknown-model-xyz", 1000, 500)
	if err != nil {
		t.Errorf("Track failed: %v", err)
	}

	cost := tracker.GetSessionCost("session-1")
	if cost <= 0 {
		t.Errorf("Expected positive cost, got %f", cost)
	}
}

func TestTrackMultipleSessions(t *testing.T) {
	tmpFile := t.TempDir() + "/test_multi.db"
	defer os.Remove(tmpFile)

	tracker, err := NewTracker(tmpFile, 10.0)
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	defer tracker.Close()

	// Track for multiple sessions - session-1 gets more tokens
	tracker.Track("session-1", "gpt-4o", 1000, 500)
	tracker.Track("session-2", "gpt-4o", 500, 250)  // Less than session-1
	tracker.Track("session-1", "gpt-4o", 1000, 500) // Second track for session-1

	cost1 := tracker.GetSessionCost("session-1")
	cost2 := tracker.GetSessionCost("session-2")

	if cost1 <= 0 || cost2 <= 0 {
		t.Errorf("Expected positive costs, got session-1: %f, session-2: %f", cost1, cost2)
	}

	// session-1 should have accumulated more cost (tracked twice)
	if cost1 <= cost2 {
		t.Errorf("session-1 should have more cost than session-2, got session-1: %f, session-2: %f", cost1, cost2)
	}
}

func TestIsOverBudget(t *testing.T) {
	tmpFile := t.TempDir() + "/test_budget.db"
	defer os.Remove(tmpFile)

	tracker, err := NewTracker(tmpFile, 1.0) // Very low budget
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	defer tracker.Close()

	// Initially not over budget
	if tracker.IsOverBudget("session-new") {
		t.Error("New session should not be over budget")
	}

	// Add cost to exceed budget
	tracker.Track("session-over", "gpt-4o", 1000000, 1000000) // This should exceed $1

	if !tracker.IsOverBudget("session-over") {
		t.Error("Session should be over budget after large track")
	}
}

func TestIsOverBudgetZeroBudget(t *testing.T) {
	tmpFile := t.TempDir() + "/test_zero_budget.db"
	defer os.Remove(tmpFile)

	tracker, err := NewTracker(tmpFile, 0) // Zero budget = unlimited
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	defer tracker.Close()

	// Even with cost, should never be over budget when budget is 0
	tracker.Track("session-1", "gpt-4o", 1000000, 1000000)
	if tracker.IsOverBudget("session-1") {
		t.Error("With zero budget, should never be over budget")
	}
}

func TestGetSessionCostNonExistent(t *testing.T) {
	tmpFile := t.TempDir() + "/test_nonexistent.db"
	defer os.Remove(tmpFile)

	tracker, err := NewTracker(tmpFile, 10.0)
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	defer tracker.Close()

	cost := tracker.GetSessionCost("non-existent-session")
	if cost != 0 {
		t.Errorf("Expected 0 cost for non-existent session, got %f", cost)
	}
}

func TestKnownModelPrices(t *testing.T) {
	tmpFile := t.TempDir() + "/test_prices.db"
	defer os.Remove(tmpFile)

	tracker, err := NewTracker(tmpFile, 100.0)
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	defer tracker.Close()

	// Check known model prices exist
	tests := []struct {
		model   string
		wantMin float64
		wantMax float64
	}{
		{"gpt-4o", 1.0, 20.0},
		{"claude-3-5-sonnet", 1.0, 30.0},
		{"gemini-2.0-flash", 0.01, 1.0},
	}

	for _, tt := range tests {
		price, ok := tracker.prices[tt.model]
		if !ok {
			t.Errorf("Expected price for model %s", tt.model)
			continue
		}
		if price.InputPrice < tt.wantMin || price.InputPrice > tt.wantMax {
			t.Errorf("InputPrice for %s = %f, expected between %f and %f", tt.model, price.InputPrice, tt.wantMin, tt.wantMax)
		}
		if price.OutputPrice < tt.wantMin || price.OutputPrice > tt.wantMax {
			t.Errorf("OutputPrice for %s = %f, expected between %f and %f", tt.model, price.OutputPrice, tt.wantMin, tt.wantMax)
		}
	}
}

func TestClose(t *testing.T) {
	tmpFile := t.TempDir() + "/test_close.db"

	tracker, err := NewTracker(tmpFile, 10.0)
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}

	// Close should not error
	err = tracker.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Double close should not panic
	err = tracker.Close()
	if err != nil {
		t.Errorf("Double close failed: %v", err)
	}
}
