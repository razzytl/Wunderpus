package ags

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// GoalStore provides SQLite-backed persistence for goals.
type GoalStore struct {
	db *sql.DB
}

// NewGoalStore opens or creates the goals database at dbPath.
func NewGoalStore(dbPath string) (*GoalStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ags store: opening db: %w", err)
	}

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS goals (
			id               TEXT PRIMARY KEY,
			title            TEXT NOT NULL,
			description      TEXT NOT NULL DEFAULT '',
			tier             INTEGER NOT NULL,
			priority         REAL NOT NULL DEFAULT 0.5,
			status           TEXT NOT NULL DEFAULT 'pending',
			parent_id        TEXT NOT NULL DEFAULT '',
			child_ids        TEXT NOT NULL DEFAULT '[]',
			created_at       TEXT NOT NULL,
			updated_at       TEXT NOT NULL,
			evidence         TEXT NOT NULL DEFAULT '[]',
			success_criteria TEXT NOT NULL DEFAULT '[]',
			expected_value   REAL NOT NULL DEFAULT 0.0,
			attempt_count    INTEGER NOT NULL DEFAULT 0,
			last_attempt     TEXT,
			completed_at     TEXT,
			actual_value     REAL
		);
		CREATE INDEX IF NOT EXISTS idx_goals_status ON goals(status);
		CREATE INDEX IF NOT EXISTS idx_goals_tier ON goals(tier);
		CREATE INDEX IF NOT EXISTS idx_goals_priority ON goals(priority DESC);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("ags store: creating schema: %w", err)
	}

	return &GoalStore{db: db}, nil
}

// Save inserts a goal into the database.
func (s *GoalStore) Save(g Goal) error {
	childIDs, _ := json.Marshal(g.ChildIDs)
	evidence, _ := json.Marshal(g.Evidence)
	criteria, _ := json.Marshal(g.SuccessCriteria)

	var lastAttempt, completedAt *string
	if g.LastAttempt != nil {
		t := g.LastAttempt.Format(time.RFC3339Nano)
		lastAttempt = &t
	}
	if g.CompletedAt != nil {
		t := g.CompletedAt.Format(time.RFC3339Nano)
		completedAt = &t
	}

	_, err := s.db.Exec(`
		INSERT INTO goals (id, title, description, tier, priority, status, parent_id,
			child_ids, created_at, updated_at, evidence, success_criteria,
			expected_value, attempt_count, last_attempt, completed_at, actual_value)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		g.ID, g.Title, g.Description, g.Tier, g.Priority, string(g.Status),
		g.ParentID, string(childIDs),
		g.CreatedAt.Format(time.RFC3339Nano), g.UpdatedAt.Format(time.RFC3339Nano),
		string(evidence), string(criteria),
		g.ExpectedValue, g.AttemptCount, lastAttempt, completedAt, g.ActualValue,
	)
	return err
}

// GetByID retrieves a goal by its UUID.
func (s *GoalStore) GetByID(id string) (Goal, error) {
	return s.scanGoal(s.db.QueryRow(`SELECT * FROM goals WHERE id = ?`, id))
}

// GetByStatus returns all goals with the given status, ordered by priority desc.
func (s *GoalStore) GetByStatus(status GoalStatus) ([]Goal, error) {
	rows, err := s.db.Query(`SELECT * FROM goals WHERE status = ? ORDER BY priority DESC`, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanGoals(rows)
}

// GetByTier returns all goals at the given tier.
func (s *GoalStore) GetByTier(tier int) ([]Goal, error) {
	rows, err := s.db.Query(`SELECT * FROM goals WHERE tier = ? ORDER BY priority DESC`, tier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanGoals(rows)
}

// Update updates an existing goal.
func (s *GoalStore) Update(g Goal) error {
	childIDs, _ := json.Marshal(g.ChildIDs)
	evidence, _ := json.Marshal(g.Evidence)
	criteria, _ := json.Marshal(g.SuccessCriteria)

	var lastAttempt, completedAt *string
	if g.LastAttempt != nil {
		t := g.LastAttempt.Format(time.RFC3339Nano)
		lastAttempt = &t
	}
	if g.CompletedAt != nil {
		t := g.CompletedAt.Format(time.RFC3339Nano)
		completedAt = &t
	}

	g.UpdatedAt = time.Now().UTC()

	_, err := s.db.Exec(`
		UPDATE goals SET title=?, description=?, tier=?, priority=?, status=?,
			parent_id=?, child_ids=?, updated_at=?, evidence=?, success_criteria=?,
			expected_value=?, attempt_count=?, last_attempt=?, completed_at=?, actual_value=?
		WHERE id=?`,
		g.Title, g.Description, g.Tier, g.Priority, string(g.Status),
		g.ParentID, string(childIDs),
		g.UpdatedAt.Format(time.RFC3339Nano), string(evidence), string(criteria),
		g.ExpectedValue, g.AttemptCount, lastAttempt, completedAt, g.ActualValue,
		g.ID,
	)
	return err
}

// History returns completed and abandoned goals, newest first, limited to `limit`.
func (s *GoalStore) History(limit int) ([]Goal, error) {
	rows, err := s.db.Query(`
		SELECT * FROM goals WHERE status IN ('completed', 'abandoned')
		ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanGoals(rows)
}

// RecentCompleted returns goals completed within the given duration.
func (s *GoalStore) RecentCompleted(since time.Duration) ([]Goal, error) {
	sinceTime := time.Now().UTC().Add(-since)
	rows, err := s.db.Query(`
		SELECT * FROM goals WHERE status = 'completed' AND completed_at >= ?
		ORDER BY completed_at DESC`, sinceTime.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanGoals(rows)
}

// RecentAbandoned returns goals abandoned within the given duration.
func (s *GoalStore) RecentAbandoned(since time.Duration) ([]Goal, error) {
	sinceTime := time.Now().UTC().Add(-since)
	rows, err := s.db.Query(`
		SELECT * FROM goals WHERE status = 'abandoned' AND updated_at >= ?
		ORDER BY updated_at DESC`, sinceTime.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanGoals(rows)
}

// SystematicallyPending returns goals with AttemptCount > 2 that are still pending.
func (s *GoalStore) SystematicallyPending() ([]Goal, error) {
	rows, err := s.db.Query(`
		SELECT * FROM goals WHERE status = 'pending' AND attempt_count > 2
		ORDER BY attempt_count DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanGoals(rows)
}

// Count returns the total number of goals with the given status.
func (s *GoalStore) Count(status GoalStatus) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM goals WHERE status = ?`, string(status)).Scan(&count)
	return count, err
}

// Close closes the database connection.
func (s *GoalStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *GoalStore) scanGoal(row *sql.Row) (Goal, error) {
	var g Goal
	var childIDsStr, evidenceStr, criteriaStr string
	var createdAtStr, updatedAtStr string
	var lastAttemptStr, completedAtStr sql.NullString
	var actualValue sql.NullFloat64

	err := row.Scan(
		&g.ID, &g.Title, &g.Description, &g.Tier, &g.Priority, &g.Status,
		&g.ParentID, &childIDsStr, &createdAtStr, &updatedAtStr,
		&evidenceStr, &criteriaStr, &g.ExpectedValue, &g.AttemptCount,
		&lastAttemptStr, &completedAtStr, &actualValue,
	)
	if err != nil {
		return g, fmt.Errorf("ags store: scan goal: %w", err)
	}

	json.Unmarshal([]byte(childIDsStr), &g.ChildIDs)
	json.Unmarshal([]byte(evidenceStr), &g.Evidence)
	json.Unmarshal([]byte(criteriaStr), &g.SuccessCriteria)
	g.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
	g.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)

	if lastAttemptStr.Valid {
		t, _ := time.Parse(time.RFC3339Nano, lastAttemptStr.String)
		g.LastAttempt = &t
	}
	if completedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339Nano, completedAtStr.String)
		g.CompletedAt = &t
	}
	if actualValue.Valid {
		v := actualValue.Float64
		g.ActualValue = &v
	}

	return g, nil
}

func (s *GoalStore) scanGoals(rows *sql.Rows) ([]Goal, error) {
	var goals []Goal
	for rows.Next() {
		var g Goal
		var childIDsStr, evidenceStr, criteriaStr string
		var createdAtStr, updatedAtStr string
		var lastAttemptStr, completedAtStr sql.NullString
		var actualValue sql.NullFloat64

		err := rows.Scan(
			&g.ID, &g.Title, &g.Description, &g.Tier, &g.Priority, &g.Status,
			&g.ParentID, &childIDsStr, &createdAtStr, &updatedAtStr,
			&evidenceStr, &criteriaStr, &g.ExpectedValue, &g.AttemptCount,
			&lastAttemptStr, &completedAtStr, &actualValue,
		)
		if err != nil {
			return nil, fmt.Errorf("ags store: scan goals: %w", err)
		}

		json.Unmarshal([]byte(childIDsStr), &g.ChildIDs)
		json.Unmarshal([]byte(evidenceStr), &g.Evidence)
		json.Unmarshal([]byte(criteriaStr), &g.SuccessCriteria)
		g.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		g.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)

		if lastAttemptStr.Valid {
			t, _ := time.Parse(time.RFC3339Nano, lastAttemptStr.String)
			g.LastAttempt = &t
		}
		if completedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339Nano, completedAtStr.String)
			g.CompletedAt = &t
		}
		if actualValue.Valid {
			v := actualValue.Float64
			g.ActualValue = &v
		}

		goals = append(goals, g)
	}
	return goals, rows.Err()
}
