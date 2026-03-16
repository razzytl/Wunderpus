package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/config"
)

func TestAPIConfig(t *testing.T) {
	// Setup dummy manager config
	cfg := &config.Config{
		DefaultProvider: "openai",
		Tools: config.ToolsConfig{
			Enabled: true,
		},
	}
	manager := agent.NewManager(cfg, nil, nil, nil, nil, nil, nil, nil)

	// dummy FS
	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("mock html")},
	}

	server, err := NewServer(mockFS, "dist", 0, manager)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()

	server.handleConfig(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %v", res.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if data["default_provider"] != "openai" {
		t.Errorf("Expected default_provider 'openai', got %v", data["default_provider"])
	}
	
	if data["tools_enabled"] != true {
		t.Errorf("Expected tools_enabled true, got %v", data["tools_enabled"])
	}
}

func TestAPIHistoryNoStore(t *testing.T) {
	// Testing handleHistory without a memory store attached
	cfg := &config.Config{}
	manager := agent.NewManager(cfg, nil, nil, nil, nil, nil, nil, nil)
	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("mock html")},
	}

	server, _ := NewServer(mockFS, "dist", 0, manager)
	
	req := httptest.NewRequest("GET", "/api/history", nil)
	w := httptest.NewRecorder()

	server.handleHistory(w, req)
	
	res := w.Result()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for no history store, got %v", res.StatusCode)
	}
}
