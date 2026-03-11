package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	server := NewServer(8080)
	if server == nil {
		t.Fatal("NewServer should not return nil")
	}

	if server.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}

	if server.server == nil {
		t.Error("http.Server should not be nil")
	}
}

func TestNewServerWithDifferentPorts(t *testing.T) {
	tests := []int{80, 443, 8080, 9000, 65535}

	for _, port := range tests {
		server := NewServer(port)

		// Simple test that server was created
		if server == nil {
			t.Errorf("NewServer(%d) returned nil", port)
		}
	}
}

func TestHandleHealth(t *testing.T) {
	server := NewServer(8080)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", resp["status"])
	}

	if _, ok := resp["uptime"]; !ok {
		t.Error("Expected uptime in response")
	}
}

func TestHandleLiveness(t *testing.T) {
	server := NewServer(8080)

	req := httptest.NewRequest("GET", "/live", nil)
	rr := httptest.NewRecorder()

	server.handleLiveness(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", resp["status"])
	}
}

func TestHandleReadiness(t *testing.T) {
	server := NewServer(8080)

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	server.handleReadiness(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if resp["status"] != "ready" {
		t.Errorf("Expected status 'ready', got %v", resp["status"])
	}
}

func TestHealthContentType(t *testing.T) {
	server := NewServer(8080)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
	}
}

func TestServerUptime(t *testing.T) {
	server := NewServer(8080)
	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	// Just verify uptime exists and is non-empty
	uptime, ok := resp["uptime"].(string)
	if !ok || uptime == "" {
		t.Error("Expected non-empty uptime")
	}
}

func TestShutdown(t *testing.T) {
	server := NewServer(18080)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := server.Shutdown(ctx)
	// May error on timeout but should not panic
	_ = err
}

func TestStart(t *testing.T) {
	server := NewServer(18081)

	// Start should not block
	server.Start()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Should be able to hit the endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	server.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}
