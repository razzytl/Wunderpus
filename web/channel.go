package web

import (
	"context"
	"log/slog"
)

// Channel wraps the web Server to implement the channel.Channel interface.
// This allows the web UI to be started/stopped as part of the
// standard channel lifecycle in app.Bootstrap.
type Channel struct {
	server *Server
}

// NewChannel creates a new web UI channel.
func NewChannel(server *Server) *Channel {
	return &Channel{server: server}
}

// Name returns the channel identifier.
func (c *Channel) Name() string {
	return "web-ui"
}

// Start launches the web UI server.
func (c *Channel) Start(ctx context.Context) error {
	slog.Info("starting web UI channel")
	return c.server.Start()
}

// Stop gracefully shuts down the web UI server.
func (c *Channel) Stop() error {
	slog.Info("stopping web UI channel")
	return c.server.Stop()
}

// GetServer returns the underlying web server for direct access
// (e.g., to push tool execution events).
func (c *Channel) GetServer() *Server {
	return c.server
}
