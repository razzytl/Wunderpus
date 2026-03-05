package mcp

import (
	"context"

	"github.com/wonderpus/wonderpus/internal/tool"
)

// Client connects to external MCP servers.
type Client struct {
	endpoint string
}

// NewClient creates a new MCP client.
func NewClient(endpoint string) *Client {
	return &Client{endpoint: endpoint}
}

// Minimal placeholder for registration logic
func (c *Client) FetchTools(ctx context.Context) ([]tool.Tool, error) {
	// Real implementation would use JSON-RPC over HTTP/SSE
	return nil, nil 
}

// RemoteTool is a tool managed by a remote MCP server.
type RemoteTool struct {
	name string
	desc string
	params []tool.ParameterDef
}

func (r *RemoteTool) Name() string { return r.name }
func (r *RemoteTool) Description() string { return r.desc }
func (r *RemoteTool) Parameters() []tool.ParameterDef { return r.params }
func (r *RemoteTool) Sensitive() bool { return false }
func (r *RemoteTool) Version() string { return "remote" }
func (r *RemoteTool) Dependencies() []string { return nil }
func (r *RemoteTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	return &tool.Result{Error: "remote execution not yet implemented"}, nil
}
