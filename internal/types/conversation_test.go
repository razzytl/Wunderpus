package types

import (
	"testing"
	"time"
)

func TestMessage_Role(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello",
	}

	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}
	if msg.Content != "Hello" {
		t.Errorf("expected content 'Hello', got %q", msg.Content)
	}
}

func TestMessage_Content(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "Hello there!",
	}

	if msg.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", msg.Role)
	}
	if msg.Content != "Hello there!" {
		t.Errorf("expected content 'Hello there!', got %q", msg.Content)
	}
}

func TestMessage_Timestamp(t *testing.T) {
	now := time.Now()
	msg := Message{
		Role:      "user",
		Content:   "Hello",
		Timestamp: now,
	}

	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}
	if msg.Content != "Hello" {
		t.Errorf("expected content 'Hello', got %q", msg.Content)
	}
	if !msg.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, msg.Timestamp)
	}
}

func TestSession(t *testing.T) {
	session := Session{
		ID:       "session-1",
		Provider: "openai",
		Model:    "gpt-4",
	}

	if session.ID != "session-1" {
		t.Errorf("expected ID 'session-1', got %q", session.ID)
	}

	if session.Provider != "openai" {
		t.Errorf("expected Provider 'openai', got %q", session.Provider)
	}

	if session.Model != "gpt-4" {
		t.Errorf("expected Model 'gpt-4', got %q", session.Model)
	}
	if session.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestSession_AddMessage(t *testing.T) {
	session := &Session{
		ID:       "session-1",
		Messages: []Message{},
	}

	if session.ID != "session-1" {
		t.Errorf("expected ID 'session-1', got %q", session.ID)
	}

	msg := Message{
		Role:    "user",
		Content: "Hello",
	}

	session.Messages = append(session.Messages, msg)

	if len(session.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(session.Messages))
	}

	if session.Messages[0].Content != "Hello" {
		t.Errorf("expected message content 'Hello', got %q", session.Messages[0].Content)
	}
}

func TestUserMessage(t *testing.T) {
	msg := UserMessage{
		SessionID: "session-1",
		Content:   "Hello world",
		AuthorID:  "user-123",
		ChannelID: "channel-456",
	}

	if msg.SessionID != "session-1" {
		t.Errorf("expected SessionID 'session-1', got %q", msg.SessionID)
	}

	if msg.Content != "Hello world" {
		t.Errorf("expected Content 'Hello world', got %q", msg.Content)
	}

	if msg.AuthorID != "user-123" {
		t.Errorf("expected AuthorID 'user-123', got %q", msg.AuthorID)
	}
	if msg.ChannelID != "channel-456" {
		t.Errorf("expected ChannelID 'channel-456', got %q", msg.ChannelID)
	}
}

func TestAgentResponse(t *testing.T) {
	resp := AgentResponse{
		SessionID: "session-1",
		Content:   "Hello!",
	}

	if resp.SessionID != "session-1" {
		t.Errorf("expected SessionID 'session-1', got %q", resp.SessionID)
	}

	if resp.Content != "Hello!" {
		t.Errorf("expected Content 'Hello!', got %q", resp.Content)
	}
}
