package memory

import (
	"fmt"
	"strings"
)

// Migration represents a single database schema change.
type Migration struct {
	Version int
	SQL     string
}

var migrations = []Migration{
	{
		Version: 1,
		SQL: `
			CREATE TABLE IF NOT EXISTS schema_version (
				version INTEGER PRIMARY KEY
			);
			INSERT INTO schema_version (version) VALUES (0);
		`,
	},
	{
		Version: 2,
		SQL: `
			ALTER TABLE messages ADD COLUMN encrypted INTEGER DEFAULT 0;
		`,
	},
	{
		Version: 3,
		SQL: `
			CREATE TABLE IF NOT EXISTS sop_store (
				id TEXT PRIMARY KEY,
				title TEXT NOT NULL,
				content TEXT NOT NULL,
				created_at TEXT NOT NULL,
				success_count INTEGER DEFAULT 1
			);
			CREATE INDEX IF NOT EXISTS idx_sop_title ON sop_store(title);
		`,
	},
}

// Migrate runs all pending migrations.
func (s *Store) Migrate() error {
	var currentVersion int
	err := s.db.QueryRow("SELECT version FROM schema_version").Scan(&currentVersion)
	if err != nil {
		// If table doesn't exist, we start from 0
		currentVersion = 0
	}

	for _, m := range migrations {
		if m.Version > currentVersion {
			tx, err := s.db.Begin()
			if err != nil {
				return err
			}

			if _, err := tx.Exec(m.SQL); err != nil {
				tx.Rollback()
				// Ignore "duplicate column" errors for ALTER TABLE ADD COLUMN
				// This handles the case where new installs have the column in CREATE TABLE
				// and upgrades already have the column from a previous migration run
				if m.Version == 2 && strings.Contains(err.Error(), "duplicate column name") {
					// Column already exists, skip this migration
					currentVersion = m.Version
					continue
				}
				return fmt.Errorf("migration version %d failed: %w", m.Version, err)
			}

			if _, err := tx.Exec("UPDATE schema_version SET version = ?", m.Version); err != nil {
				tx.Rollback()
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			currentVersion = m.Version
		}
	}
	return nil
}
