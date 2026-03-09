package types

import (
	"testing"
	"time"
)

func TestMessage_Structure(t *testing.T) {
	m := Message{
		Role:      "user",
		Content:   "Hello",
		Timestamp: time.Now(),
	}

	if m.Role != "user" {
		t.Errorf("expected user, got %s", m.Role)
	}
	if m.Content != "Hello" {
		t.Errorf("expected Hello, got %s", m.Content)
	}
}

func TestSession_Structure(t *testing.T) {
	s := Session{
		ID:       "session-123",
		Provider: "openai",
		Model:    "gpt-4",
		Messages: []Message{},
	}

	if s.ID != "session-123" {
		t.Errorf("expected session-123, got %s", s.ID)
	}
	if s.Provider != "openai" {
		t.Errorf("expected openai, got %s", s.Provider)
	}
	if s.Model != "gpt-4" {
		t.Errorf("expected gpt-4, got %s", s.Model)
	}
}

func TestSession_WithMessages(t *testing.T) {
	s := Session{
		ID:       "session-123",
		Provider: "openai",
		Model:    "gpt-4",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi!"},
		},
	}

	if len(s.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(s.Messages))
	}
}

func TestUserMessage_Structure(t *testing.T) {
	um := UserMessage{
		SessionID: "session-123",
		Content:   "Hello",
		AuthorID:  "user-456",
	}

	if um.SessionID != "session-123" {
		t.Errorf("expected session-123, got %s", um.SessionID)
	}
	if um.Content != "Hello" {
		t.Errorf("expected Hello, got %s", um.Content)
	}
	if um.AuthorID != "user-456" {
		t.Errorf("expected user-456, got %s", um.AuthorID)
	}
}

func TestAgentResponse_Structure(t *testing.T) {
	ar := AgentResponse{
		SessionID: "session-123",
		Content:   "Hello!",
	}

	if ar.SessionID != "session-123" {
		t.Errorf("expected session-123, got %s", ar.SessionID)
	}
	if ar.Content != "Hello!" {
		t.Errorf("expected Hello!, got %s", ar.Content)
	}
}
