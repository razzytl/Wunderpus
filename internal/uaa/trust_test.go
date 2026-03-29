package uaa

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func tempTrustDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test_trust.db")
}

func TestTrustBudget_DeductBelowZero_Lockdown(t *testing.T) {
	dbPath := tempTrustDB(t)
	t.Setenv("TEST_JWT_SECRET", "test-secret-key-for-testing")

	tb, err := NewTrustBudget(dbPath, 100, 10, nil, nil, "TEST_JWT_SECRET")
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

func TestTrustBudget_ExpiredJWT_ResetFails(t *testing.T) {
	dbPath := tempTrustDB(t)
	t.Setenv("TEST_JWT_SECRET", "test-secret-key")

	tb, err := NewTrustBudget(dbPath, 100, 10, nil, nil, "TEST_JWT_SECRET")
	if err != nil {
		t.Fatalf("NewTrustBudget: %v", err)
	}
	defer tb.Close()

	// Create an expired JWT
	secret := os.Getenv("TEST_JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "human-operator",
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // expired 1 hour ago
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	err = tb.Reset(tokenString)
	if err == nil {
		t.Fatal("expired JWT reset should have failed")
	}
}

func TestTrustBudget_AgentCannotGenerateJWT(t *testing.T) {
	dbPath := tempTrustDB(t)
	t.Setenv("TEST_JWT_SECRET", "test-secret-key")

	tb, err := NewTrustBudget(dbPath, 100, 10, nil, nil, "TEST_JWT_SECRET")
	if err != nil {
		t.Fatalf("NewTrustBudget: %v", err)
	}
	defer tb.Close()

	// Agent tries to issue a JWT with issuer = wunderpus-agent
	secret := os.Getenv("TEST_JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "wunderpus-agent", // this should be rejected
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))

	err = tb.Reset(tokenString)
	if err == nil {
		t.Fatal("agent-issued JWT should be rejected for reset")
	}
}

func TestTrustBudget_PersistAcrossRestart(t *testing.T) {
	dbPath := tempTrustDB(t)
	t.Setenv("TEST_JWT_SECRET", "test-secret")

	// Create budget, deduct, then close
	tb1, err := NewTrustBudget(dbPath, 100, 10, nil, nil, "TEST_JWT_SECRET")
	if err != nil {
		t.Fatalf("NewTrustBudget: %v", err)
	}
	tb1.Deduct(80, "test")
	if tb1.Current() != 20 {
		t.Fatalf("expected 20, got %d", tb1.Current())
	}
	tb1.Close()

	// Reopen — should load persisted value, not reset to max
	tb2, err := NewTrustBudget(dbPath, 100, 10, nil, nil, "TEST_JWT_SECRET")
	if err != nil {
		t.Fatalf("NewTrustBudget reopen: %v", err)
	}
	defer tb2.Close()

	if tb2.Current() != 20 {
		t.Fatalf("expected persisted value 20, got %d", tb2.Current())
	}
}

func TestTrustBudget_CreditCappedAtMax(t *testing.T) {
	dbPath := tempTrustDB(t)
	tb, _ := NewTrustBudget(dbPath, 100, 10, nil, nil, "TEST_JWT_SECRET")
	defer tb.Close()

	tb.Deduct(10, "test")
	tb.Credit(50, "refund") // 90 + 50 would be 140, should cap at 100

	if tb.Current() != 100 {
		t.Fatalf("expected capped at 100, got %d", tb.Current())
	}
}

func TestTrustBudget_ValidHumanReset(t *testing.T) {
	dbPath := tempTrustDB(t)
	secretVal := "valid-human-secret"
	t.Setenv("TEST_JWT_SECRET", secretVal)

	tb, _ := NewTrustBudget(dbPath, 100, 10, nil, nil, "TEST_JWT_SECRET")
	defer tb.Close()

	// Enter lockdown
	tb.Deduct(100, "drain")
	if tb.Current() != 0 {
		t.Fatal("should be in lockdown")
	}

	// Valid human-issued JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "human-operator",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secretVal))

	err := tb.Reset(tokenString)
	if err != nil {
		t.Fatalf("valid reset should succeed: %v", err)
	}

	if tb.Current() != 100 {
		t.Fatalf("expected 100 after reset, got %d", tb.Current())
	}
}
