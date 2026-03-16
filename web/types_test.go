package web

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWSMessageJSON(t *testing.T) {
	ts := time.Now().UTC()
	msg := WSMessage{
		Type:      MsgTypeSystemLog,
		Timestamp: ts,
		SessionID: "test_session",
		Payload: SystemLogPayload{
			Level:   "info",
			Message: "Test log",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal WSMessage: %v", err)
	}

	var parsed WSMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal WSMessage: %v", err)
	}

	if parsed.Type != MsgTypeSystemLog {
		t.Errorf("Expected type %s, got %s", MsgTypeSystemLog, parsed.Type)
	}
	if parsed.SessionID != "test_session" {
		t.Errorf("Expected session test_session, got %s", parsed.SessionID)
	}
}
