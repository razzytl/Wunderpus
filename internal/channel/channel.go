package channel

import (
	"context"
)

// Channel defines the interface for communication platforms (Telegram, Discord, etc.)
type Channel interface {
	// Start begins the channel's message loop or server.
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down the channel.
	Stop() error
	
	// Name returns the identifier for this channel.
	Name() string
}

// UserMessage represents a message received from a channel.
type UserMessage struct {
	SessionID string
	Content   string
	Channel   string
}

// AgentResponse represents a response to be sent back to a channel.
type AgentResponse struct {
	Content string
}
