package agent

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func tempCheckpointDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "test_checkpoint.db")
}

func TestCheckpointStore_SaveAndGet(t *testing.T) {
	dbPath := tempCheckpointDB(t)
	db, err := openTestDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	store, err := NewCheckpointStore(db)
	if err != nil {
		t.Fatalf("failed to create checkpoint store: %v", err)
	}

	snapshot := CheckpointSnapshot{
		SessionID: "test-session",
		BranchID:  "main",
		StepDesc:  "step 1 completed",
		Messages:  nil,
	}

	err = store.SaveCheckpoint("task-1", 1, snapshot)
	if err != nil {
		t.Fatalf("failed to save checkpoint: %v", err)
	}

	// Verify it can be retrieved
	retrieved, stepNum, err := store.GetLatestCheckpoint("task-1")
	if err != nil {
		t.Fatalf("failed to get checkpoint: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected checkpoint, got nil")
	}
	if stepNum != 1 {
		t.Errorf("expected step 1, got %d", stepNum)
	}
	if retrieved.SessionID != "test-session" {
		t.Errorf("expected session test-session, got %s", retrieved.SessionID)
	}
}

func TestCheckpointStore_CompleteAndFail(t *testing.T) {
	dbPath := tempCheckpointDB(t)
	db, err := openTestDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	store, err := NewCheckpointStore(db)
	if err != nil {
		t.Fatalf("failed to create checkpoint store: %v", err)
	}

	snapshot := CheckpointSnapshot{SessionID: "test", BranchID: "main"}
	store.SaveCheckpoint("task-1", 1, snapshot)

	// Complete it
	err = store.CompleteCheckpoint("task-1", 1)
	if err != nil {
		t.Fatalf("failed to complete checkpoint: %v", err)
	}

	// Running checkpoints should not include completed ones
	running, err := store.GetRunningCheckpoints()
	if err != nil {
		t.Fatalf("failed to get running checkpoints: %v", err)
	}
	for _, cp := range running {
		if cp.TaskID == "task-1" {
			t.Error("completed checkpoint should not appear in running list")
		}
	}
}

func TestCheckpointStore_GetRunningCheckpoints(t *testing.T) {
	dbPath := tempCheckpointDB(t)
	db, err := openTestDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	store, err := NewCheckpointStore(db)
	if err != nil {
		t.Fatalf("failed to create checkpoint store: %v", err)
	}

	// Save multiple running checkpoints
	for i := 1; i <= 3; i++ {
		snapshot := CheckpointSnapshot{SessionID: "session", BranchID: "main"}
		store.SaveCheckpoint("task-1", i, snapshot)
	}

	// Save a completed checkpoint for a different task
	store.SaveCheckpoint("task-2", 1, CheckpointSnapshot{SessionID: "session"})
	store.CompleteCheckpoint("task-2", 1)

	running, err := store.GetRunningCheckpoints()
	if err != nil {
		t.Fatalf("failed to get running checkpoints: %v", err)
	}
	runningCount := 0
	for _, cp := range running {
		if cp.TaskID == "task-1" {
			runningCount++
		}
		if cp.TaskID == "task-2" {
			t.Error("task-2 is completed, should not appear in running")
		}
	}
	if runningCount != 3 {
		t.Errorf("expected 3 running checkpoints for task-1, got %d", runningCount)
	}
}

func TestCheckpointStore_ScanRunningCheckpoints(t *testing.T) {
	dbPath := tempCheckpointDB(t)
	db, err := openTestDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	store, err := NewCheckpointStore(db)
	if err != nil {
		t.Fatalf("failed to create checkpoint store: %v", err)
	}

	store.SaveCheckpoint("task-a", 1, CheckpointSnapshot{SessionID: "session-a", BranchID: "main"})
	store.SaveCheckpoint("task-b", 1, CheckpointSnapshot{SessionID: "session-b", BranchID: "feature"})

	resumeMap := store.ScanRunningCheckpoints()
	if len(resumeMap) != 2 {
		t.Errorf("expected 2 tasks to resume, got %d", len(resumeMap))
	}
	if _, ok := resumeMap["task-a"]; !ok {
		t.Error("expected task-a in resume map")
	}
	if _, ok := resumeMap["task-b"]; !ok {
		t.Error("expected task-b in resume map")
	}
}

func TestCheckpointStore_NoCheckpoints(t *testing.T) {
	dbPath := tempCheckpointDB(t)
	db, err := openTestDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	store, err := NewCheckpointStore(db)
	if err != nil {
		t.Fatalf("failed to create checkpoint store: %v", err)
	}

	latest, step, err := store.GetLatestCheckpoint("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest != nil {
		t.Error("expected nil for nonexistent checkpoint")
	}
	if step != 0 {
		t.Errorf("expected step 0, got %d", step)
	}

	running, err := store.GetRunningCheckpoints()
	if err != nil {
		t.Fatalf("failed to get running checkpoints: %v", err)
	}
	if len(running) != 0 {
		t.Errorf("expected 0 running checkpoints, got %d", len(running))
	}
}

func openTestDB(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}
