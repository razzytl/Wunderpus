package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// AuditEntry represents a single immutable entry in the hash-chained audit log.
type AuditEntry struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Subsystem string          `json:"subsystem"`
	EventType EventType       `json:"event_type"`
	ActorID   string          `json:"actor_id"`
	Payload   json.RawMessage `json:"payload"`
	PrevHash  string          `json:"prev_hash"`
	Hash      string          `json:"hash"`
}

// ComputeHash calculates the SHA-256 hash for this entry based on chained data.
// Hash = SHA256(prevHash + timestamp + subsystem + eventType + actorID + payload)
func ComputeHash(prevHash string, entry AuditEntry) string {
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write([]byte(entry.Timestamp.UTC().Format(time.RFC3339Nano)))
	h.Write([]byte(entry.Subsystem))
	h.Write([]byte(string(entry.EventType)))
	h.Write([]byte(entry.ActorID))
	h.Write(entry.Payload)
	return hex.EncodeToString(h.Sum(nil))
}

// AuditFilter defines filtering criteria for querying the audit log.
type AuditFilter struct {
	Subsystem string
	EventType EventType
	ActorID   string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}
