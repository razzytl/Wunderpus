package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/wunderpus/wunderpus/internal/provider"
)

// CheckpointStore manages task checkpoints for crash resilience.
type CheckpointStore struct {
	db *sql.DB
}

// NewCheckpointStore creates a new checkpoint store using the shared core DB.
func NewCheckpointStore(db *sql.DB) (*CheckpointStore, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS task_checkpoints (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id         TEXT NOT NULL,
			step_number     INTEGER NOT NULL,
			context_snapshot TEXT NOT NULL,
			status          TEXT NOT NULL DEFAULT 'running',
			created_at      TEXT NOT NULL,
			updated_at      TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_checkpoint_task ON task_checkpoints(task_id);
		CREATE INDEX IF NOT EXISTS idx_checkpoint_status ON task_checkpoints(status);
	`)
	if err != nil {
		return nil, fmt.Errorf("checkpoint: creating table: %w", err)
	}

	return &CheckpointStore{db: db}, nil
}

// CheckpointSnapshot captures the agent's context at a point in time.
type CheckpointSnapshot struct {
	Messages  []provider.Message `json:"messages"`
	BranchID  string             `json:"branch_id"`
	SessionID string             `json:"session_id"`
	StepDesc  string             `json:"step_description"`
}

// SaveCheckpoint persists a checkpoint after a successful tool execution step.
func (s *CheckpointStore) SaveCheckpoint(taskID string, stepNumber int, snapshot CheckpointSnapshot) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("checkpoint: marshaling snapshot: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	_, err = s.db.Exec(`
		INSERT INTO task_checkpoints (task_id, step_number, context_snapshot, status, created_at, updated_at)
		VALUES (?, ?, ?, 'running', ?, ?)`,
		taskID, stepNumber, string(data), now, now,
	)
	if err != nil {
		return fmt.Errorf("checkpoint: saving: %w", err)
	}

	slog.Debug("checkpoint saved", "task_id", taskID, "step", stepNumber)
	return nil
}

// CompleteCheckpoint marks a checkpoint as completed.
func (s *CheckpointStore) CompleteCheckpoint(taskID string, stepNumber int) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`
		UPDATE task_checkpoints SET status = 'completed', updated_at = ?
		WHERE task_id = ? AND step_number = ?`,
		now, taskID, stepNumber,
	)
	return err
}

// FailCheckpoint marks a checkpoint as failed.
func (s *CheckpointStore) FailCheckpoint(taskID string, stepNumber int) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`
		UPDATE task_checkpoints SET status = 'failed', updated_at = ?
		WHERE task_id = ? AND step_number = ?`,
		now, taskID, stepNumber,
	)
	return err
}

// GetRunningCheckpoints returns all checkpoints with status = 'running'.
func (s *CheckpointStore) GetRunningCheckpoints() ([]struct {
	TaskID   string `json:"task_id"`
	StepNum  int    `json:"step_number"`
	Snapshot CheckpointSnapshot
}, error) {
	rows, err := s.db.Query(`
		SELECT task_id, step_number, context_snapshot
		FROM task_checkpoints WHERE status = 'running'
		ORDER BY task_id, step_number DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		TaskID   string `json:"task_id"`
		StepNum  int    `json:"step_number"`
		Snapshot CheckpointSnapshot
	}

	for rows.Next() {
		var taskID string
		var stepNum int
		var snapshotJSON string
		if err := rows.Scan(&taskID, &stepNum, &snapshotJSON); err != nil {
			continue
		}

		var snap CheckpointSnapshot
		if err := json.Unmarshal([]byte(snapshotJSON), &snap); err != nil {
			slog.Warn("checkpoint: failed to unmarshal snapshot", "task_id", taskID, "step", stepNum)
			continue
		}

		results = append(results, struct {
			TaskID   string `json:"task_id"`
			StepNum  int    `json:"step_number"`
			Snapshot CheckpointSnapshot
		}{TaskID: taskID, StepNum: stepNum, Snapshot: snap})
	}

	return results, rows.Err()
}

// GetLatestCheckpoint returns the most recent checkpoint for a task.
func (s *CheckpointStore) GetLatestCheckpoint(taskID string) (*CheckpointSnapshot, int, error) {
	var stepNum int
	var snapshotJSON string
	err := s.db.QueryRow(`
		SELECT step_number, context_snapshot
		FROM task_checkpoints WHERE task_id = ?
		ORDER BY step_number DESC LIMIT 1`,
		taskID).Scan(&stepNum, &snapshotJSON)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}

	var snap CheckpointSnapshot
	if err := json.Unmarshal([]byte(snapshotJSON), &snap); err != nil {
		return nil, 0, fmt.Errorf("checkpoint: unmarshaling snapshot: %w", err)
	}

	return &snap, stepNum, nil
}

// ScanRunningCheckpoints finds all tasks with 'running' status for resume on startup.
func (s *CheckpointStore) ScanRunningCheckpoints() map[string]*CheckpointSnapshot {
	checkpoints, err := s.GetRunningCheckpoints()
	if err != nil {
		slog.Error("checkpoint: failed to scan running checkpoints", "error", err)
		return nil
	}

	resumeMap := make(map[string]*CheckpointSnapshot)
	for _, cp := range checkpoints {
		// Keep only the latest step per task
		if existing, ok := resumeMap[cp.TaskID]; !ok || cp.StepNum > getStepNum(existing) {
			resumeMap[cp.TaskID] = &cp.Snapshot
		}
	}

	if len(resumeMap) > 0 {
		slog.Info("checkpoint: found tasks to resume", "count", len(resumeMap))
	}
	return resumeMap
}

func getStepNum(snap *CheckpointSnapshot) int {
	// Use the step description as a rough indicator; actual step tracking is external
	return 0
}

// ResumeTask rehydrates an agent's context from a checkpoint snapshot.
func ResumeTask(ctx context.Context, agent *Agent, snapshot *CheckpointSnapshot) error {
	if snapshot == nil {
		return nil
	}

	// Restore the agent's branch
	if snapshot.BranchID != "" {
		agent.SetBranch(snapshot.BranchID)
	}

	// Restore messages into context
	if len(snapshot.Messages) > 0 {
		agent.ctx.SetMessages(snapshot.Messages)
		slog.Info("checkpoint: resumed task context",
			"session", snapshot.SessionID,
			"messages", len(snapshot.Messages),
			"step", snapshot.StepDesc)
	}

	return nil
}
