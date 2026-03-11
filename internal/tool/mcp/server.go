package mcp

import (
	"encoding/json"
	"net/http"

	"github.com/wunderpus/wunderpus/internal/tool"
)

// Server exposes local tools via MCP protocol.
type Server struct {
	registry *tool.Registry
}

// NewServer creates a new MCP server.
func NewServer(registry *tool.Registry) *Server {
	return &Server{registry: registry}
}

func (s *Server) ListToolsHandler(w http.ResponseWriter, r *http.Request) {
	tools := s.registry.List()
	var response []map[string]any
	for _, t := range tools {
		response = append(response, map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"version":     t.Version(),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
