package uaa

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestUAAExecutor_RejectedByBudget(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	trust, _ := NewTrustBudget(db, 10, 10, nil, nil)
	defer trust.Close()

	mockRunner := func(ctx context.Context, action Action) (*ActionResult, error) {
		return &ActionResult{ActionID: action.ID, Success: true, Output: "done"}, nil
	}

	uaa := NewUAA(trust, nil, nil, mockRunner)

	// Deduct most trust
	trust.Deduct(9, "setup")

	// Action costs 20 — should be rejected (only 1 trust left)
	action := Action{
		ID:         "test-action",
		Tool:       "send_file",
		Parameters: map[string]interface{}{},
	}

	_, err = uaa.Execute(context.Background(), action)
	if err == nil {
		t.Fatal("action should be rejected with insufficient budget")
	}

	// Budget should be unchanged (1)
	if trust.Current() != 1 {
		t.Fatalf("expected budget 1 after rejection, got %d", trust.Current())
	}
}

func TestUAAExecutor_ExecuteSuccess(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	trust, _ := NewTrustBudget(db, 1000, 10, nil, nil)
	defer trust.Close()

	mockRunner := func(ctx context.Context, action Action) (*ActionResult, error) {
		return &ActionResult{ActionID: action.ID, Success: true, Output: "done"}, nil
	}

	uaa := NewUAA(trust, nil, nil, mockRunner)

	action := Action{
		ID:         "test-action",
		Tool:       "read_file",
		Parameters: map[string]interface{}{"path": "/tmp/file.txt"},
	}

	result, err := uaa.Execute(context.Background(), action)
	if err != nil {
		t.Fatalf("action should execute: %v", err)
	}
	if !result.Success {
		t.Fatal("action should succeed")
	}
}

func TestUAAExecutor_TrustDeducted(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	trust, _ := NewTrustBudget(db, 1000, 10, nil, nil)
	defer trust.Close()

	mockRunner := func(ctx context.Context, action Action) (*ActionResult, error) {
		return &ActionResult{ActionID: action.ID, Success: true, Output: "written"}, nil
	}

	uaa := NewUAA(trust, nil, nil, mockRunner)

	action := Action{
		ID:         "test-action",
		Tool:       "write_file",
		Parameters: map[string]interface{}{"path": "/home/user/file.txt", "content": "data"},
	}

	initialTrust := trust.Current()
	result, err := uaa.Execute(context.Background(), action)
	if err != nil {
		t.Fatalf("action should execute: %v", err)
	}
	if !result.Success {
		t.Fatal("action should succeed")
	}

	// Trust should have been deducted
	if trust.Current() >= initialTrust {
		t.Fatal("trust should have been deducted")
	}
}
