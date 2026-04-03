// Package db provides shared database connection management.
// All subsystems receive *sql.DB connections from here rather than opening
// their own SQLite files.  Two databases are maintained:
//
//	Core DB  (wunderpus.db)      — memory, world-model, cost, heartbeat, AGS
//	Audit DB (wunderpus-audit.db) — append-only hash-chained audit log
package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Manager holds the two canonical database connections.
type Manager struct {
	CoreDB  *sql.DB // sessions, messages, SOPs, world model, cost, heartbeat
	AuditDB *sql.DB // append-only audit log (kept separate for integrity)
}

// Open creates both databases at the given home directory.
func Open(homeDir string) (*Manager, error) {
	corePath := filepath.Join(homeDir, "wunderpus.db")
	auditPath := filepath.Join(homeDir, "wunderpus-audit.db")

	coreDB, err := openSQLite(corePath)
	if err != nil {
		return nil, fmt.Errorf("db: opening core DB: %w", err)
	}

	auditDB, err := openSQLite(auditPath)
	if err != nil {
		coreDB.Close()
		return nil, fmt.Errorf("db: opening audit DB: %w", err)
	}

	slog.Info("db: connections opened",
		"core", corePath,
		"audit", auditPath,
	)

	return &Manager{
		CoreDB:  coreDB,
		AuditDB: auditDB,
	}, nil
}

// openSQLite opens a SQLite file with sensible defaults.
func openSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")
	_, _ = db.Exec("PRAGMA foreign_keys=ON;")
	return db, nil
}

// Close closes both database connections.
func (m *Manager) Close() error {
	var firstErr error
	if m.CoreDB != nil {
		firstErr = m.CoreDB.Close()
	}
	if m.AuditDB != nil {
		if err := m.AuditDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
