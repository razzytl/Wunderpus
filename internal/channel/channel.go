package channel

import (
	"context"

	"github.com/wunderpus/wunderpus/internal/types"
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

// Reusing types from Internal package
type UserMessage = types.UserMessage
type AgentResponse = types.AgentResponse
