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
	db  *sql.DB
	key []byte // optional encryption key
}

// NewAuditLogger opens (or creates) the SQLite audit database.
func NewAuditLogger(dbPath string, encryptionKey []byte) (*AuditLogger, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("audit: opening db: %w", err)
	}

	// Optimize SQLite
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")

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

	return &AuditLogger{db: db, key: encryptionKey}, nil
}

// Log records an audit event.
func (a *AuditLogger) Log(event AuditEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	input := truncate(event.Input, 2000)
	result := truncate(event.Result, 2000)

	// Optional encryption
	if len(a.key) > 0 {
		if encInput, err := Encrypt(input, a.key); err == nil {
			input = "enc:" + encInput
		}
		if encResult, err := Encrypt(result, a.key); err == nil {
			result = "enc:" + encResult
		}
	}

	_, err := a.db.Exec(
		`INSERT INTO audit_log (timestamp, action, user, input, result, threat_level)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.Timestamp.Format(time.RFC3339),
		event.Action,
		event.User,
		input,
		result,
		event.ThreatLevel,
	)
	if err != nil {
		slog.Error("audit: failed to log event", "error", err, "action", event.Action)
	}
}

// Rotate deletes oldest entries if the log exceeds maxRows.
func (a *AuditLogger) Rotate(maxRows int) error {
	_, err := a.db.Exec(`
		DELETE FROM audit_log 
		WHERE id IN (
			SELECT id FROM audit_log 
			ORDER BY timestamp ASC 
			LIMIT (SELECT MAX(0, COUNT(*) - ?) FROM audit_log)
		)
	`, maxRows)
	return err
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
