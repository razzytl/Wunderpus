package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/config"
)

func TestStaticFileServing(t *testing.T) {
	cfg := &config.Config{}
	manager := agent.NewManager(cfg, nil, nil, nil, nil, nil, nil, nil)

	mockFS := fstest.MapFS{
		"dist/index.html":       &fstest.MapFile{Data: []byte("index")},
		"dist/assets/style.css": &fstest.MapFile{Data: []byte("body {}")},
	}

	server, err := NewServer(mockFS, "dist", 8080, manager)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create a test server using the mux directly
	// Note: We access internal mux through s.server.Handler
	ts := httptest.NewServer(server.server.Handler)
	defer ts.Close()

	// Test static file hit
	res, err := http.Get(ts.URL + "/assets/style.css")
	if err != nil {
		t.Fatalf("Failed to GET static file: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for static file, got %v", res.StatusCode)
	}

	// Test SPA fallback
	res2, err := http.Get(ts.URL + "/non-existent-route")
	if err != nil {
		t.Fatalf("Failed to GET route: %v", err)
	}
	if res2.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for SPA fallback, got %v", res2.StatusCode)
	}
}
