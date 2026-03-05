package memory

import (
	"fmt"
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
