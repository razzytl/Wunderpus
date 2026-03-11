package types

import (
	"time"
)

// Message represents a single message in a conversation.
type Message struct {
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
	TokenCount int       `json:"token_count,omitempty"`
}

// Session represents a chat session.
type Session struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserMessage is the structure for incoming messages from any channel.
type UserMessage struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	AuthorID  string `json:"author_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

// AgentResponse is the structure for outgoing messages to any channel.
type AgentResponse struct {
	SessionID string `json:"session_id,omitempty"`
	Content   string `json:"content"`
}
