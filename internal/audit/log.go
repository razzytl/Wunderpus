package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// AuditLog is a hash-chained, append-only, SQLite-backed audit log.
// It is the single source of truth for everything the system does.
type AuditLog struct {
	db     *sql.DB
	mu     sync.Mutex
	dbPath string
}

// NewAuditLog opens or creates the audit log database at dbPath.
// The database uses WAL mode for concurrent read performance.
func NewAuditLog(dbPath string) (*AuditLog, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("audit: opening db: %w", err)
	}

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_entries (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp  TEXT    NOT NULL,
			subsystem  TEXT    NOT NULL,
			event_type TEXT    NOT NULL,
			actor_id   TEXT    NOT NULL DEFAULT '',
			payload    TEXT    NOT NULL DEFAULT '{}',
			prev_hash  TEXT    NOT NULL DEFAULT '',
			hash       TEXT    NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_audit_subsystem ON audit_entries(subsystem);
		CREATE INDEX IF NOT EXISTS idx_audit_event_type ON audit_entries(event_type);
		CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_entries(timestamp);
		CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_entries(actor_id);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("audit: creating schema: %w", err)
	}

	return &AuditLog{db: db, dbPath: dbPath}, nil
}

// Write appends a new entry to the audit log with hash chaining.
// The entry's Hash is computed from prevHash + entry data.
// All writes are serialized via mutex.
func (l *AuditLog) Write(entry AuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Get the hash of the last entry
	var prevHash string
	err := l.db.QueryRow(
		`SELECT hash FROM audit_entries ORDER BY id DESC LIMIT 1`,
	).Scan(&prevHash)
	if err != nil {
		if err == sql.ErrNoRows {
			prevHash = "" // Genesis entry
		} else {
			return fmt.Errorf("audit: reading last hash: %w", err)
		}
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	entry.PrevHash = prevHash
	entry.Hash = ComputeHash(prevHash, entry)

	_, err = l.db.Exec(
		`INSERT INTO audit_entries (timestamp, subsystem, event_type, actor_id, payload, prev_hash, hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
		entry.Subsystem,
		string(entry.EventType),
		entry.ActorID,
		string(entry.Payload),
		entry.PrevHash,
		entry.Hash,
	)
	if err != nil {
		return fmt.Errorf("audit: writing entry: %w", err)
	}

	return nil
}

// Verify walks all entries and recomputes hashes to verify chain integrity.
// Returns nil if the chain is intact, or an error describing the first mismatch.
func (l *AuditLog) Verify() error {
	rows, err := l.db.Query(
		`SELECT id, timestamp, subsystem, event_type, actor_id, payload, prev_hash, hash
		 FROM audit_entries ORDER BY id ASC`,
	)
	if err != nil {
		return fmt.Errorf("audit: query for verify: %w", err)
	}
	defer rows.Close()

	var prevHash string
	var rowID int

	for rows.Next() {
		var entry AuditEntry
		var tsStr string
		var payloadStr string

		err := rows.Scan(
			&rowID,
			&tsStr,
			&entry.Subsystem,
			&entry.EventType,
			&entry.ActorID,
			&payloadStr,
			&entry.PrevHash,
			&entry.Hash,
		)
		if err != nil {
			return fmt.Errorf("audit: scanning row %d: %w", rowID, err)
		}

		entry.Payload = json.RawMessage(payloadStr)
		entry.Timestamp, _ = time.Parse(time.RFC3339Nano, tsStr)

		// Check prev hash linkage
		if entry.PrevHash != prevHash {
			return fmt.Errorf("audit: hash chain broken at entry %d: expected prev_hash=%s, got %s",
				rowID, prevHash, entry.PrevHash)
		}

		// Recompute and verify hash
		expectedHash := ComputeHash(prevHash, entry)
		if entry.Hash != expectedHash {
			return fmt.Errorf("audit: hash mismatch at entry %d: expected %s, got %s",
				rowID, expectedHash, entry.Hash)
		}

		prevHash = entry.Hash
	}

	return rows.Err()
}

// Query retrieves entries matching the given filter criteria.
func (l *AuditLog) Query(filter AuditFilter) ([]AuditEntry, error) {
	var conditions []string
	var args []interface{}

	if filter.Subsystem != "" {
		conditions = append(conditions, "subsystem = ?")
		args = append(args, filter.Subsystem)
	}
	if filter.EventType != "" {
		conditions = append(conditions, "event_type = ?")
		args = append(args, string(filter.EventType))
	}
	if filter.ActorID != "" {
		conditions = append(conditions, "actor_id = ?")
		args = append(args, filter.ActorID)
	}
	if filter.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, filter.StartTime.UTC().Format(time.RFC3339Nano))
	}
	if filter.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, filter.EndTime.UTC().Format(time.RFC3339Nano))
	}

	query := `SELECT id, timestamp, subsystem, event_type, actor_id, payload, prev_hash, hash
	          FROM audit_entries`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY id ASC"

	if filter.Limit > 0 {
		query += " LIMIT " + strconv.Itoa(filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET " + strconv.Itoa(filter.Offset)
	}

	rows, err := l.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("audit: query: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var tsStr string
		var payloadStr string

		err := rows.Scan(
			&entry.ID,
			&tsStr,
			&entry.Subsystem,
			&entry.EventType,
			&entry.ActorID,
			&payloadStr,
			&entry.PrevHash,
			&entry.Hash,
		)
		if err != nil {
			return nil, fmt.Errorf("audit: scanning: %w", err)
		}

		entry.Payload = json.RawMessage(payloadStr)
		entry.Timestamp, _ = time.Parse(time.RFC3339Nano, tsStr)
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// Count returns the total number of entries in the audit log.
func (l *AuditLog) Count() (int, error) {
	var count int
	err := l.db.QueryRow(`SELECT COUNT(*) FROM audit_entries`).Scan(&count)
	return count, err
}

// LatestHash returns the hash of the most recent entry, or empty string if no entries.
func (l *AuditLog) LatestHash() (string, error) {
	var hash string
	err := l.db.QueryRow(`SELECT hash FROM audit_entries ORDER BY id DESC LIMIT 1`).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}

// Close closes the database connection.
func (l *AuditLog) Close() error {
	if l.db != nil {
		slog.Debug("audit: closing database")
		return l.db.Close()
	}
	return nil
}
