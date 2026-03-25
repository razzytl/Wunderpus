package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "test_audit.db")
}

func TestAuditLog_WriteAndVerify(t *testing.T) {
	dbPath := tempDB(t)
	log, err := NewAuditLog(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	defer log.Close()

	// Write 10 sequential entries
	for i := 0; i < 10; i++ {
		payload, _ := json.Marshal(map[string]interface{}{"i": i})
		err := log.Write(AuditEntry{
			Subsystem: "test",
			EventType: EventActionExecuted,
			ActorID:   "tester",
			Payload:   payload,
		})
		if err != nil {
			t.Fatalf("Write entry %d: %v", i, err)
		}
	}

	// Verify chain integrity
	if err := log.Verify(); err != nil {
		t.Fatalf("Verify failed on clean log: %v", err)
	}

	count, _ := log.Count()
	if count != 10 {
		t.Fatalf("expected 10 entries, got %d", count)
	}
}

func TestAuditLog_ConcurrentWritesAndVerify(t *testing.T) {
	dbPath := tempDB(t)
	log, err := NewAuditLog(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	defer log.Close()

	const numWriters = 10
	const numEntries = 100 // 10 * 100 = 1000 entries total

	var wg sync.WaitGroup
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < numEntries; i++ {
				payload, _ := json.Marshal(map[string]interface{}{
					"worker": workerID,
					"i":      i,
				})
				_ = log.Write(AuditEntry{
					Subsystem: "test",
					EventType: EventActionExecuted,
					ActorID:   fmt.Sprintf("worker-%d", workerID),
					Payload:   payload,
				})
			}
		}(w)
	}
	wg.Wait()

	count, _ := log.Count()
	if count != numWriters*numEntries {
		t.Fatalf("expected %d entries, got %d", numWriters*numEntries, count)
	}

	// Verify chain integrity after concurrent writes
	if err := log.Verify(); err != nil {
		t.Fatalf("Verify failed after concurrent writes: %v", err)
	}
}

func TestAuditLog_CorruptedHash(t *testing.T) {
	dbPath := tempDB(t)
	log, err := NewAuditLog(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}

	// Write 5 entries
	for i := 0; i < 5; i++ {
		payload, _ := json.Marshal(map[string]int{"i": i})
		_ = log.Write(AuditEntry{
			Subsystem: "test",
			EventType: EventActionExecuted,
			ActorID:   "tester",
			Payload:   payload,
		})
	}
	log.Close()

	// Corrupt the hash of entry 3 directly in SQLite

	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	_, err = rawDB.Exec(`UPDATE audit_entries SET hash = 'corrupted_hash' WHERE id = 3`)
	if err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	rawDB.Close()

	// Reopen and verify
	log2, err := NewAuditLog(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	defer log2.Close()

	err = log2.Verify()
	if err == nil {
		t.Fatal("Verify should have returned error for corrupted hash")
	}
	if err.Error() == "" {
		t.Fatal("error message should not be empty")
	}
}

func TestAuditLog_Query(t *testing.T) {
	dbPath := tempDB(t)
	log, err := NewAuditLog(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	defer log.Close()

	// Write entries for different subsystems
	types := []EventType{EventActionExecuted, EventRSICycleStarted, EventGoalCreated}
	for i, et := range types {
		for j := 0; j < 3; j++ {
			payload, _ := json.Marshal(map[string]int{"i": i, "j": j})
			_ = log.Write(AuditEntry{
				Subsystem: fmt.Sprintf("subsys-%d", i),
				EventType: et,
				ActorID:   "tester",
				Payload:   payload,
			})
		}
	}

	// Query by subsystem
	entries, err := log.Query(AuditFilter{Subsystem: "subsys-0"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries for subsystem-0, got %d", len(entries))
	}

	// Query by event type
	entries, err = log.Query(AuditFilter{EventType: EventGoalCreated})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries for goal.created, got %d", len(entries))
	}
}
