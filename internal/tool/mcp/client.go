package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wunderpus/wunderpus/internal/config"
	"github.com/wunderpus/wunderpus/internal/tool"
)

type ServerConnection struct {
	Name    string
	Client  *mcp.Client
	Session *mcp.ClientSession
	Tools   []*mcp.Tool
}

type Manager struct {
	servers map[string]*ServerConnection
	mu      sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*ServerConnection),
	}
}

func (m *Manager) LoadFromConfig(ctx context.Context, cfg config.MCPConfig, workspacePath string) error {
	if !cfg.Enabled {
		return nil
	}

	if len(cfg.Servers) == 0 {
		return nil
	}

	for name, serverCfg := range cfg.Servers {
		if !serverCfg.Enabled {
			continue
		}

		if err := m.ConnectServer(ctx, name, serverCfg, workspacePath); err != nil {
			return fmt.Errorf("failed to connect to MCP server %s: %w", name, err)
		}
	}

	return nil
}

func (m *Manager) ConnectServer(ctx context.Context, name string, cfg config.MCPServerConfig, workspacePath string) error {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "wunderpus",
		Version: "1.0.0",
	}, nil)

	var transport mcp.Transport

	transportType := cfg.Type
	if transportType == "" {
		if cfg.URL != "" {
			transportType = "sse"
		} else if cfg.Command != "" {
			transportType = "stdio"
		}
	}

	switch transportType {
	case "sse", "http":
		if cfg.URL == "" {
			return fmt.Errorf("URL is required for SSE/HTTP transport")
		}
		transport = &mcp.StreamableClientTransport{Endpoint: cfg.URL}
	case "stdio":
		if cfg.Command == "" {
			return fmt.Errorf("command is required for stdio transport")
		}
		cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)

		env := cmd.Environ()
		for k, v := range cfg.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env

		transport = &mcp.CommandTransport{Command: cmd}
	default:
		return fmt.Errorf("unsupported transport type: %s", transportType)
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	initResult := session.InitializeResult()

	var tools []*mcp.Tool
	if initResult.Capabilities.Tools != nil {
		for tool, err := range session.Tools(ctx, nil) {
			if err != nil {
				continue
			}
			tools = append(tools, tool)
		}
	}

	m.mu.Lock()
	m.servers[name] = &ServerConnection{
		Name:    name,
		Client:  client,
		Session: session,
		Tools:   tools,
	}
	m.mu.Unlock()

	return nil
}

func (m *Manager) GetServers() map[string]*ServerConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ServerConnection, len(m.servers))
	for k, v := range m.servers {
		result[k] = v
	}
	return result
}

func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
	m.mu.RLock()
	conn, ok := m.servers[serverName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	}

	return conn.Session.CallTool(ctx, params)
}

func (m *Manager) GetAllTools() map[string][]*mcp.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]*mcp.Tool)
	for name, conn := range m.servers {
		if len(conn.Tools) > 0 {
			result[name] = conn.Tools
		}
	}
	return result
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.servers {
		conn.Session.Close()
	}
	m.servers = make(map[string]*ServerConnection)
	return nil
}

type MCPToolWrapper struct {
	manager    *Manager
	serverName string
	tool       *mcp.Tool
}

func NewMCPToolWrapper(manager *Manager, serverName string, tool *mcp.Tool) *MCPToolWrapper {
	return &MCPToolWrapper{
		manager:    manager,
		serverName: serverName,
		tool:       tool,
	}
}

func (t *MCPToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", sanitizeIdentifier(t.serverName), sanitizeIdentifier(t.tool.Name))
}

func (t *MCPToolWrapper) Description() string {
	desc := t.tool.Description
	if desc == "" {
		desc = fmt.Sprintf("MCP tool from %s server", t.serverName)
	}
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, desc)
}

func (t *MCPToolWrapper) Sensitive() bool        { return false }
func (t *MCPToolWrapper) Version() string        { return "1.0.0" }
func (t *MCPToolWrapper) Dependencies() []string { return nil }

func (t *MCPToolWrapper) Parameters() []tool.ParameterDef {
	schema := t.tool.InputSchema
	if schema == nil {
		return []tool.ParameterDef{}
	}

	var params []tool.ParameterDef
	if schemaMap, ok := schema.(map[string]any); ok {
		if props, ok := schemaMap["properties"].(map[string]any); ok {
			for name, prop := range props {
				propMap, _ := prop.(map[string]any)
				param := tool.ParameterDef{
					Name:        name,
					Type:        "string",
					Description: "",
					Required:    false,
				}
				if t, ok := propMap["type"].(string); ok {
					param.Type = t
				}
				if d, ok := propMap["description"].(string); ok {
					param.Description = d
				}
				if required, ok := schemaMap["required"].([]any); ok {
					for _, r := range required {
						if r == name {
							param.Required = true
							break
						}
					}
				}
				params = append(params, param)
			}
		}
	}
	return params
}

func (t *MCPToolWrapper) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	result, err := t.manager.CallTool(ctx, t.serverName, t.tool.Name, args)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("MCP tool execution failed: %v", err)}, nil
	}

	if result == nil {
		return &tool.Result{Error: "MCP tool returned nil result"}, nil
	}

	if result.IsError {
		var errMsg strings.Builder
		for _, c := range result.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				errMsg.WriteString(tc.Text)
			}
		}
		return &tool.Result{Error: fmt.Sprintf("MCP tool error: %s", errMsg.String())}, nil
	}

	var output strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			output.WriteString(tc.Text)
		}
	}

	return &tool.Result{Output: output.String()}, nil
}

func sanitizeIdentifier(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}
	return result.String()
}
