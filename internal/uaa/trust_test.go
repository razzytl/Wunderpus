package uaa

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newTrustDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestTrustBudget_DeductBelowZero_Lockdown(t *testing.T) {
	db := newTrustDB(t)

	tb, err := NewTrustBudget(db, 100, 10, nil, nil)
	if err != nil {
		t.Fatalf("NewTrustBudget: %v", err)
	}
	defer tb.Close()

	// Deduct to near zero
	tb.Deduct(95, "test-action-1")
	if tb.Current() != 5 {
		t.Fatalf("expected 5, got %d", tb.Current())
	}

	// Deduct below zero → should trigger lockdown
	tb.Deduct(10, "test-action-2")

	// Lockdown: budget should be 0
	if tb.Current() != 0 {
		t.Fatalf("expected 0 after lockdown, got %d", tb.Current())
	}

	// CanExecute should fail for any action
	ok, reason := tb.CanExecute(0)
	if ok {
		t.Fatal("should not be able to execute in lockdown (even cost 0)")
	}
	if reason == "" {
		t.Fatal("reason should not be empty in lockdown")
	}
}

func TestTrustBudget_Reset(t *testing.T) {
	db := newTrustDB(t)

	tb, err := NewTrustBudget(db, 100, 10, nil, nil)
	if err != nil {
		t.Fatalf("NewTrustBudget: %v", err)
	}
	defer tb.Close()

	// Enter lockdown
	tb.Deduct(100, "drain")
	if tb.Current() != 0 {
		t.Fatal("should be in lockdown")
	}

	// Reset should work without JWT
	err = tb.Reset()
	if err != nil {
		t.Fatalf("reset should succeed: %v", err)
	}

	if tb.Current() != 100 {
		t.Fatalf("expected 100 after reset, got %d", tb.Current())
	}
}

func TestTrustBudget_PersistAcrossRestart(t *testing.T) {
	// Use file-based DB for persistence test
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test_trust_persist.db"

	db1, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	tb1, err := NewTrustBudget(db1, 100, 10, nil, nil)
	if err != nil {
		t.Fatalf("NewTrustBudget: %v", err)
	}
	tb1.Deduct(80, "test")
	if tb1.Current() != 20 {
		t.Fatalf("expected 20, got %d", tb1.Current())
	}
	tb1.Close()
	db1.Close()

	// Reopen — should load persisted value, not reset to max
	db2, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db2.Close()

	tb2, err := NewTrustBudget(db2, 100, 10, nil, nil)
	if err != nil {
		t.Fatalf("NewTrustBudget reopen: %v", err)
	}
	defer tb2.Close()

	if tb2.Current() != 20 {
		t.Fatalf("expected persisted value 20, got %d", tb2.Current())
	}
}

func TestTrustBudget_CreditCappedAtMax(t *testing.T) {
	db := newTrustDB(t)
	tb, _ := NewTrustBudget(db, 100, 10, nil, nil)
	defer tb.Close()

	tb.Deduct(10, "test")
	tb.Credit(50, "refund") // 90 + 50 would be 140, should cap at 100

	if tb.Current() != 100 {
		t.Fatalf("expected capped at 100, got %d", tb.Current())
	}
}
