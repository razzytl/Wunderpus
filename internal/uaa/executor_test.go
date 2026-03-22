package uaa

import (
	"context"
	"os"
	"testing"
)

func TestUAAExecutor_Tier4RejectedByBudget(t *testing.T) {
	os.Setenv("TEST_JWT_SECRET", "test")
	defer os.Unsetenv("TEST_JWT_SECRET")

	trust, _ := NewTrustBudget(t.TempDir()+"/trust.db", 10, 10, nil, nil, "TEST_JWT_SECRET")
	defer trust.Close()

	classifier := NewClassifier(nil)

	mockRunner := func(ctx context.Context, action Action) (*ActionResult, error) {
		return &ActionResult{ActionID: action.ID, Success: true, Output: "done"}, nil
	}

	uaa := NewUAA(classifier, trust, nil, nil, nil, mockRunner)

	// Deduct most trust
	trust.Deduct(9, "setup")

	// Tier 4 action costs 20 — should be rejected (only 1 trust left)
	action := Action{
		ID:          "test-tier4",
		Tool:        "send_file",
		Parameters:  map[string]interface{}{},
		Description: "send file externally",
	}

	_, err := uaa.Execute(context.Background(), action)
	if err == nil {
		t.Fatal("Tier 4 action should be rejected with insufficient budget")
	}

	// Budget should be unchanged (1)
	if trust.Current() != 1 {
		t.Fatalf("expected budget 1 after rejection, got %d", trust.Current())
	}
}

func TestUAAExecutor_Tier1NeverHitsShadow(t *testing.T) {
	os.Setenv("TEST_JWT_SECRET", "test")
	defer os.Unsetenv("TEST_JWT_SECRET")

	trust, _ := NewTrustBudget(t.TempDir()+"/trust.db", 1000, 10, nil, nil, "TEST_JWT_SECRET")
	defer trust.Close()

	classifier := NewClassifier(nil)
	shadowCalled := false

	judgeFn := func(ctx context.Context, summary string) (bool, string, error) {
		shadowCalled = true
		return true, "ok", nil
	}
	shadow := NewShadowSimulator(judgeFn)

	mockRunner := func(ctx context.Context, action Action) (*ActionResult, error) {
		return &ActionResult{ActionID: action.ID, Success: true}, nil
	}

	uaa := NewUAA(classifier, trust, shadow, nil, nil, mockRunner)

	// Tier 1 action — should never hit shadow mode
	action := Action{
		ID:         "test-tier1",
		Tool:       "read_file",
		Parameters: map[string]interface{}{"path": "/tmp/file.txt"},
	}

	result, err := uaa.Execute(context.Background(), action)
	if err != nil {
		t.Fatalf("Tier 1 should execute: %v", err)
	}
	if !result.Success {
		t.Fatal("Tier 1 should succeed")
	}
	if shadowCalled {
		t.Fatal("Shadow mode should NOT be called for Tier 1 actions")
	}
}

func TestUAAExecutor_Tier3ShadowApproves(t *testing.T) {
	os.Setenv("TEST_JWT_SECRET", "test")
	defer os.Unsetenv("TEST_JWT_SECRET")

	trust, _ := NewTrustBudget(t.TempDir()+"/trust.db", 1000, 10, nil, nil, "TEST_JWT_SECRET")
	defer trust.Close()

	classifier := NewClassifier(nil)

	judgeFn := func(ctx context.Context, summary string) (bool, string, error) {
		return true, "safe write", nil
	}
	shadow := NewShadowSimulator(judgeFn)

	mockRunner := func(ctx context.Context, action Action) (*ActionResult, error) {
		return &ActionResult{ActionID: action.ID, Success: true, Output: "written"}, nil
	}

	uaa := NewUAA(classifier, trust, shadow, nil, nil, mockRunner)

	// Tier 3 action — should go through shadow mode
	action := Action{
		ID:         "test-tier3",
		Tool:       "write_file",
		Parameters: map[string]interface{}{"path": "/home/user/file.txt", "content": "data"},
	}

	initialTrust := trust.Current()
	result, err := uaa.Execute(context.Background(), action)
	if err != nil {
		t.Fatalf("Tier 3 should execute after shadow approval: %v", err)
	}
	if !result.Success {
		t.Fatal("Tier 3 should succeed")
	}

	// Trust should be deducted (Tier 3 = 5 cost)
	if trust.Current() >= initialTrust {
		t.Fatal("trust should have been deducted")
	}
}
