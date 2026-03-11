package channel

import (
	"context"
	"testing"
)

// mockChannel implements Channel interface for testing
type mockChannel struct {
	name    string
	started bool
	stopped bool
}

func (m *mockChannel) Name() string {
	return m.name
}

func (m *mockChannel) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockChannel) Stop() error {
	m.stopped = true
	return nil
}

func TestChannelInterface(t *testing.T) {
	// Verify mockChannel implements Channel interface
	var _ Channel = &mockChannel{name: "test"}
}

func TestMockChannelName(t *testing.T) {
	ch := &mockChannel{name: "telegram"}
	if ch.Name() != "telegram" {
		t.Errorf("Expected 'telegram', got %s", ch.Name())
	}
}

func TestMockChannelStart(t *testing.T) {
	ch := &mockChannel{name: "test"}
	err := ch.Start(context.Background())
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}
	if !ch.started {
		t.Error("Expected started to be true")
	}
}

func TestMockChannelStop(t *testing.T) {
	ch := &mockChannel{name: "test"}
	err := ch.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	if !ch.stopped {
		t.Error("Expected stopped to be true")
	}
}

func TestUserMessageTypeAlias(t *testing.T) {
	// Test that the type alias works
	msg := UserMessage{
		SessionID: "session-123",
		Content:   "Hello",
		AuthorID:  "user-456",
	}

	if msg.SessionID != "session-123" {
		t.Errorf("Expected session_id 'session-123', got %s", msg.SessionID)
	}
	if msg.Content != "Hello" {
		t.Errorf("Expected content 'Hello', got %s", msg.Content)
	}
	if msg.AuthorID != "user-456" {
		t.Errorf("Expected author_id 'user-456', got %s", msg.AuthorID)
	}
}

func TestAgentResponseTypeAlias(t *testing.T) {
	// Test that the type alias works
	resp := AgentResponse{
		SessionID: "session-123",
		Content:   "Hello!",
	}

	if resp.SessionID != "session-123" {
		t.Errorf("Expected session_id 'session-123', got %s", resp.SessionID)
	}
	if resp.Content != "Hello!" {
		t.Errorf("Expected content 'Hello!', got %s", resp.Content)
	}
}

// Test with empty values
func TestUserMessageEmpty(t *testing.T) {
	msg := UserMessage{}

	if msg.SessionID != "" {
		t.Errorf("Expected empty session_id, got %s", msg.SessionID)
	}
	if msg.Content != "" {
		t.Errorf("Expected empty content, got %s", msg.Content)
	}
}

func TestAgentResponseEmpty(t *testing.T) {
	resp := AgentResponse{}

	if resp.SessionID != "" {
		t.Errorf("Expected empty session_id, got %s", resp.SessionID)
	}
	if resp.Content != "" {
		t.Errorf("Expected empty content, got %s", resp.Content)
	}
}
