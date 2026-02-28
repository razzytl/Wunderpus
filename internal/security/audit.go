package security

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

// AuditEvent represents a single auditable action.
type AuditEvent struct {
	Timestamp   time.Time
	Action      string
	User        string
	Input       string
	Result      string
	ThreatLevel string // none, low, medium, high
}

// AuditLogger writes audit events to SQLite.
type AuditLogger struct {
	db *sql.DB
}

// NewAuditLogger opens (or creates) the SQLite audit database.
func NewAuditLogger(dbPath string) (*AuditLogger, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("audit: opening db: %w", err)
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_log (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp    TEXT NOT NULL,
			action       TEXT NOT NULL,
			user         TEXT NOT NULL DEFAULT '',
			input        TEXT NOT NULL DEFAULT '',
			result       TEXT NOT NULL DEFAULT '',
			threat_level TEXT NOT NULL DEFAULT 'none'
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("audit: creating table: %w", err)
	}

	return &AuditLogger{db: db}, nil
}

// Log records an audit event.
func (a *AuditLogger) Log(event AuditEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	_, err := a.db.Exec(
		`INSERT INTO audit_log (timestamp, action, user, input, result, threat_level)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.Timestamp.Format(time.RFC3339),
		event.Action,
		event.User,
		truncate(event.Input, 2000),
		truncate(event.Result, 2000),
		event.ThreatLevel,
	)
	if err != nil {
		slog.Error("audit: failed to log event", "error", err, "action", event.Action)
	}
}

// Close closes the audit database.
func (a *AuditLogger) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
