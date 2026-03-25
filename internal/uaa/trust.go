package uaa

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/wunderpus/wunderpus/internal/audit"
	"github.com/wunderpus/wunderpus/internal/events"
)

// TrustBudget manages a points-based autonomy budget.
// Actions cost trust points. Success earns partial refunds. Failures incur penalties.
// When trust reaches zero, the system enters lockdown (only Tier 1 actions allowed).
type TrustBudget struct {
	current      int
	max          int
	regenPerHour int
	mu           sync.Mutex
	db           *sql.DB
	events       *events.Bus
	audit        *audit.AuditLog
	jwtSecretEnv string
	stopCh       chan struct{}
}

// NewTrustBudget creates a trust budget backed by SQLite.
// If no existing state is found, the budget is initialized to max.
func NewTrustBudget(dbPath string, max int, regenPerHour int, bus *events.Bus, auditLog *audit.AuditLog, jwtSecretEnv string) (*TrustBudget, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("trust: opening db: %w", err)
	}

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trust_state (
			id      INTEGER PRIMARY KEY CHECK (id = 1),
			current INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS trust_history (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			action_id TEXT NOT NULL,
			delta     INTEGER NOT NULL,
			reason    TEXT NOT NULL DEFAULT '',
			balance   INTEGER NOT NULL
		);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("trust: creating schema: %w", err)
	}

	// Load persisted state or initialize
	var current int
	err = db.QueryRow(`SELECT current FROM trust_state WHERE id = 1`).Scan(&current)
	if err == sql.ErrNoRows {
		current = max
		_, _ = db.Exec(`INSERT INTO trust_state (id, current) VALUES (1, ?)`, max)
	} else if err != nil {
		db.Close()
		return nil, fmt.Errorf("trust: loading state: %w", err)
	}

	tb := &TrustBudget{
		current:      current,
		max:          max,
		regenPerHour: regenPerHour,
		db:           db,
		events:       bus,
		audit:        auditLog,
		jwtSecretEnv: jwtSecretEnv,
		stopCh:       make(chan struct{}),
	}

	return tb, nil
}

// CanExecute checks whether the current trust budget can cover the given cost.
func (tb *TrustBudget) CanExecute(cost int) (bool, string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.current <= 0 {
		return false, "system in lockdown — trust budget depleted"
	}
	if tb.current < cost {
		return false, fmt.Sprintf("insufficient trust budget: have %d, need %d", tb.current, cost)
	}
	return true, ""
}

// TryDeduct atomically checks if cost can be deducted and deducts if so.
// This eliminates the TOCTOU race between CanExecute and Deduct.
func (tb *TrustBudget) TryDeduct(cost int, actionID string) (bool, string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.current <= 0 {
		return false, "system in lockdown — trust budget depleted"
	}
	if tb.current < cost {
		return false, fmt.Sprintf("insufficient trust budget: have %d, need %d", tb.current, cost)
	}

	tb.current -= cost
	tb.persist()

	if tb.audit != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"action_id":     actionID,
			"cost":          cost,
			"remaining":     tb.current,
			"transaction":   "deduct",
			"trust_balance": tb.current,
		})
		_ = tb.audit.Write(audit.AuditEntry{
			Subsystem: "trust_budget",
			EventType: audit.EventTrustDebited,
			ActorID:   "trust-budget",
			Payload:   payload,
		})
	}

	if tb.current <= 0 {
		tb.current = 0
		tb.persist()
		slog.Warn("trust: LOCKDOWN ENGAGED — all Tier 2+ actions blocked")
		if tb.events != nil {
			tb.events.Publish(events.Event{
				Type:   audit.EventLockdownEngaged,
				Source: "trust_budget",
				Payload: map[string]interface{}{
					"reason": "trust budget depleted",
				},
			})
		}
	}

	return true, ""
}

// Current returns the current trust balance.
func (tb *TrustBudget) Current() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.current
}

// Deduct subtracts cost from the trust budget. If budget drops to zero or below,
// lockdown is engaged automatically.
func (tb *TrustBudget) Deduct(cost int, actionID string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.current -= cost
	if tb.current < 0 {
		tb.current = 0
	}
	tb.persist()
	tb.recordHistory(actionID, -cost, "deduct")

	if tb.current <= 0 {
		tb.enterLockdown()
	}
}

// Credit adds points to the trust budget, capped at max.
func (tb *TrustBudget) Credit(amount int, reason string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	old := tb.current
	tb.current += amount
	if tb.current > tb.max {
		tb.current = tb.max
	}

	actual := tb.current - old
	if actual > 0 {
		tb.persist()
		tb.recordHistory("", actual, reason)

		if tb.audit != nil {
			payload := fmt.Sprintf(`{"amount":%d,"reason":"%s","balance":%d}`, actual, reason, tb.current)
			_ = tb.audit.Write(audit.AuditEntry{
				Subsystem: "uaa",
				EventType: audit.EventTrustCredited,
				ActorID:   "system",
				Payload:   []byte(payload),
			})
		}

		if tb.events != nil {
			tb.events.Publish(events.Event{
				Type:   audit.EventTrustCredited,
				Source: "trust_budget",
				Payload: map[string]interface{}{
					"amount":  actual,
					"reason":  reason,
					"balance": tb.current,
				},
			})
		}
	}
}

// RecordOutcome adjusts trust based on action success or failure.
// On success: partial refund (cost / 2). On failure: penalty (cost * 3).
func (tb *TrustBudget) RecordOutcome(actionID string, cost int, success bool) {
	if success {
		refund := cost / 2
		if refund > 0 {
			tb.Credit(refund, fmt.Sprintf("success refund for %s", actionID))
		}
		return
	}

	// Failure path — hold lock through the entire operation to prevent TOCTOU race
	tb.mu.Lock()
	penalty := cost * 3
	tb.current -= penalty
	if tb.current < 0 {
		tb.current = 0
	}
	depleted := tb.current <= 0
	tb.persist()
	tb.recordHistory(actionID, -penalty, "failure penalty")

	// If depleted, enter lockdown UNDER THE SAME LOCK to prevent Credit from slipping in
	if depleted {
		tb.current = 0
		tb.persist()
		tb.recordHistory("", 0, "lockdown engaged")

		slog.Warn("trust: LOCKDOWN ENGAGED — all Tier 2+ actions blocked")

		if tb.audit != nil {
			_ = tb.audit.Write(audit.AuditEntry{
				Subsystem: "uaa",
				EventType: audit.EventTrustLockdown,
				ActorID:   "system",
				Payload:   []byte(`{"reason":"trust budget depleted"}`),
			})
		}

		if tb.events != nil {
			tb.events.Publish(events.Event{
				Type:   audit.EventTrustLockdown,
				Source: "trust_budget",
				Payload: map[string]interface{}{
					"reason": "trust budget depleted",
				},
			})
		}
	}
	tb.mu.Unlock()
}

// EnterLockdown forces the system into lockdown. Only Tier 1 actions are permitted.
func (tb *TrustBudget) EnterLockdown() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.enterLockdown()
}

// enterLockdown is the internal lockdown implementation (must hold lock).
func (tb *TrustBudget) enterLockdown() {
	tb.current = 0
	tb.persist()
	tb.recordHistory("", 0, "lockdown engaged")

	slog.Warn("trust: LOCKDOWN ENGAGED — all Tier 2+ actions blocked")

	if tb.audit != nil {
		_ = tb.audit.Write(audit.AuditEntry{
			Subsystem: "uaa",
			EventType: audit.EventTrustLockdown,
			ActorID:   "system",
			Payload:   []byte(`{"reason":"trust budget depleted"}`),
		})
	}

	if tb.events != nil {
		tb.events.Publish(events.Event{
			Type:   audit.EventTrustLockdown,
			Source: "trust_budget",
			Payload: map[string]interface{}{
				"reason": "trust budget depleted",
			},
		})
	}
}

// Reset restores the trust budget to max after validating a JWT token.
// The JWT must be signed with the secret from the configured env var.
// Tokens expire after 1 hour and cannot be self-issued by the agent.
func (tb *TrustBudget) Reset(tokenString string) error {
	secret := os.Getenv(tb.jwtSecretEnv)
	if secret == "" {
		return fmt.Errorf("trust: JWT secret not set in env var %s", tb.jwtSecretEnv)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithExpirationRequired())

	if err != nil {
		return fmt.Errorf("trust: invalid JWT: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("trust: JWT token is not valid")
	}

	// Verify issuer is not the agent itself
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("trust: failed to parse JWT claims")
	}
	if iss, ok := claims["iss"].(string); ok && iss == "wunderpus-agent" {
		return fmt.Errorf("trust: agent cannot self-issue reset tokens")
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.current = tb.max
	tb.persist()
	tb.recordHistory("", 0, "human reset via JWT")

	slog.Info("trust: budget RESET to max via JWT", "max", tb.max)

	if tb.audit != nil {
		_ = tb.audit.Write(audit.AuditEntry{
			Subsystem: "uaa",
			EventType: audit.EventTrustReset,
			ActorID:   "human",
			Payload:   []byte(fmt.Sprintf(`{"reset_to":%d}`, tb.max)),
		})
	}

	if tb.events != nil {
		tb.events.Publish(events.Event{
			Type:   audit.EventTrustReset,
			Source: "trust_budget",
			Payload: map[string]interface{}{
				"reset_to": tb.max,
			},
		})
	}

	return nil
}

// StartRegen begins a background goroutine that passively regenerates trust.
func (tb *TrustBudget) StartRegen() {
	if tb.regenPerHour <= 0 {
		return // no regen configured
	}
	go func() {
		// Regen per second = regenPerHour / 3600
		// For integer math, accumulate and add once per interval
		interval := time.Hour / time.Duration(tb.regenPerHour)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				tb.Credit(1, "passive regen")
			case <-tb.stopCh:
				return
			}
		}
	}()
}

// StopRegen stops the passive regeneration goroutine.
func (tb *TrustBudget) StopRegen() {
	close(tb.stopCh)
}

// persist writes the current trust value to SQLite.
func (tb *TrustBudget) persist() {
	if tb.db == nil {
		return
	}
	_, _ = tb.db.Exec(`UPDATE trust_state SET current = ? WHERE id = 1`, tb.current)
}

// recordHistory appends an entry to the trust history table.
func (tb *TrustBudget) recordHistory(actionID string, delta int, reason string) {
	if tb.db == nil {
		return
	}
	_, _ = tb.db.Exec(
		`INSERT INTO trust_history (timestamp, action_id, delta, reason, balance)
		 VALUES (?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano),
		actionID,
		delta,
		reason,
		tb.current,
	)
}

// Close shuts down the trust budget and its database connection.
func (tb *TrustBudget) Close() error {
	if tb.db != nil {
		return tb.db.Close()
	}
	return nil
}
