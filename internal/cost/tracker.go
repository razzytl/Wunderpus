package cost

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/logging"
	_ "modernc.org/sqlite"
)

// Tracker handles asynchronous cost tracking and budgeting.
type Tracker struct {
	db     *sql.DB
	mu     sync.RWMutex
	usage  map[string]float64 // sessionID -> total cost
	budget float64
	prices map[string]ModelPrice
}

// ModelPrice defines pricing per 1M tokens.
type ModelPrice struct {
	InputPrice  float64
	OutputPrice float64
}

// NewTracker creates a new cost tracker.
func NewTracker(dbPath string, budget float64) (*Tracker, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("cost: opening db: %w", err)
	}

	// Optimize SQLite
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cost_log (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp    TEXT NOT NULL,
			session_id   TEXT NOT NULL,
			model        string NOT NULL,
			input_tokens INTEGER NOT NULL,
			output_tokens INTEGER NOT NULL,
			cost         REAL NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_cost_session ON cost_log(session_id);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("cost: creating table: %w", err)
	}

	return &Tracker{
		db:     db,
		usage:  make(map[string]float64),
		budget: budget,
		prices: map[string]ModelPrice{
			"gpt-4o":            {InputPrice: 2.50, OutputPrice: 10.00},
			"claude-3-5-sonnet": {InputPrice: 3.00, OutputPrice: 15.00},
			"gemini-2.0-flash":  {InputPrice: 0.10, OutputPrice: 0.40},
		},
	}, nil
}

// Track logs used tokens and updates session cost.
func (t *Tracker) Track(sessionID, model string, input, output int) error {
	price, ok := t.prices[model]
	if !ok {
		// Default very low price if unknown
		price = ModelPrice{InputPrice: 0.1, OutputPrice: 0.1}
	}

	cost := (float64(input)/1000000.0)*price.InputPrice + (float64(output)/1000000.0)*price.OutputPrice

	t.mu.Lock()
	t.usage[sessionID] += cost
	t.mu.Unlock()

	// Update Prometheus metrics
	logging.TokenUsage.WithLabelValues(model, "input").Add(float64(input))
	logging.TokenUsage.WithLabelValues(model, "output").Add(float64(output))
	logging.ProviderCost.WithLabelValues(model, sessionID).Add(cost)

	_, err := t.db.Exec(
		`INSERT INTO cost_log (timestamp, session_id, model, input_tokens, output_tokens, cost)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		time.Now().Format(time.RFC3339),
		sessionID,
		model,
		input,
		output,
		cost,
	)
	return err
}

// IsOverBudget checks if a session (or global) is over budget.
func (t *Tracker) IsOverBudget(sessionID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.usage[sessionID] >= t.budget && t.budget > 0
}

// GetSessionCost returns accumulated cost for a session.
func (t *Tracker) GetSessionCost(sessionID string) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.usage[sessionID]
}

// Close closes the database connection.
func (t *Tracker) Close() error {
	return t.db.Close()
}
