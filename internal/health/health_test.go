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
	agg := NewAggregator()
	server := NewServer(8080, agg)
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
	agg := NewAggregator()
	tests := []int{80, 443, 8080, 9000, 65535}

	for _, port := range tests {
		server := NewServer(port, agg)

		// Simple test that server was created
		if server == nil {
			t.Errorf("NewServer(%d) returned nil", port)
		}
	}
}

func TestHandleHealth(t *testing.T) {
	agg := NewAggregator()
	server := NewServer(8080, agg)

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

	if resp["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", resp["status"])
	}

	if _, ok := resp["uptime"]; !ok {
		t.Error("Expected uptime in response")
	}
}

func TestHandleLiveness(t *testing.T) {
	agg := NewAggregator()
	server := NewServer(8080, agg)

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
	agg := NewAggregator()
	server := NewServer(8080, agg)

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
	agg := NewAggregator()
	server := NewServer(8080, agg)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
	}
}

func TestServerUptime(t *testing.T) {
	agg := NewAggregator()
	server := NewServer(8080, agg)
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
	agg := NewAggregator()
	server := NewServer(18080, agg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := server.Shutdown(ctx)
	// May error on timeout but should not panic
	_ = err
}

func TestStart(t *testing.T) {
	agg := NewAggregator()
	server := NewServer(18081, agg)

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

func TestAggregator_RegisterAndCollect(t *testing.T) {
	agg := NewAggregator()
	agg.Register("test", func() ComponentStatus {
		return ComponentStatus{Name: StatusHealthy, Details: "all good"}
	})

	results := agg.Collect()
	if len(results) != 1 {
		t.Fatalf("Expected 1 component, got %d", len(results))
	}

	cs, ok := results["test"]
	if !ok {
		t.Fatal("Expected 'test' component")
	}
	if cs.Name != StatusHealthy {
		t.Errorf("Expected healthy, got %s", cs.Name)
	}
	if cs.Details != "all good" {
		t.Errorf("Expected 'all good', got %s", cs.Details)
	}
}

func TestOverallStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]ComponentStatus
		expected Status
	}{
		{
			name:     "empty",
			input:    map[string]ComponentStatus{},
			expected: StatusHealthy,
		},
		{
			name: "all healthy",
			input: map[string]ComponentStatus{
				"db": {Name: StatusHealthy},
			},
			expected: StatusHealthy,
		},
		{
			name: "one degraded",
			input: map[string]ComponentStatus{
				"db":       {Name: StatusHealthy},
				"provider": {Name: StatusDegraded},
			},
			expected: StatusDegraded,
		},
		{
			name: "one unhealthy",
			input: map[string]ComponentStatus{
				"db":       {Name: StatusHealthy},
				"provider": {Name: StatusUnhealthy},
			},
			expected: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := OverallStatus(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHealthEndpointWithComponents(t *testing.T) {
	agg := NewAggregator()
	agg.Register("db", func() ComponentStatus {
		return ComponentStatus{Name: StatusHealthy}
	})
	agg.Register("provider", func() ComponentStatus {
		return ComponentStatus{Name: StatusDegraded, Details: "slow response"}
	})

	server := NewServer(8080, agg)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["status"] != "degraded" {
		t.Errorf("Expected status 'degraded', got %v", resp["status"])
	}

	components, ok := resp["components"].(map[string]any)
	if !ok {
		t.Fatal("Expected components map")
	}

	if _, ok := components["db"]; !ok {
		t.Error("Expected 'db' component")
	}
	if _, ok := components["provider"]; !ok {
		t.Error("Expected 'provider' component")
	}
}

func TestHealthEndpointUnhealthy(t *testing.T) {
	agg := NewAggregator()
	agg.Register("db", func() ComponentStatus {
		return ComponentStatus{Name: StatusUnhealthy, Details: "connection refused"}
	})

	server := NewServer(8080, agg)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["status"] != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got %v", resp["status"])
	}
}
