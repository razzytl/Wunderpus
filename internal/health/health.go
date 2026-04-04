package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Status represents the health of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// ComponentStatus holds the health of a single component.
type ComponentStatus struct {
	Name    Status `json:"status"`
	Details string `json:"details,omitempty"`
}

// Aggregator collects health status from multiple subsystems.
type Aggregator struct {
	mu         sync.RWMutex
	components map[string]func() ComponentStatus
}

// NewAggregator creates a new health aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{
		components: make(map[string]func() ComponentStatus),
	}
}

// Register adds a health check function for a named component.
func (a *Aggregator) Register(name string, check func() ComponentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.components[name] = check
}

// Collect runs all registered checks and returns the aggregated status.
func (a *Aggregator) Collect() map[string]ComponentStatus {
	a.mu.RLock()
	checks := make(map[string]func() ComponentStatus, len(a.components))
	for k, v := range a.components {
		checks[k] = v
	}
	a.mu.RUnlock()

	results := make(map[string]ComponentStatus)
	for name, check := range checks {
		results[name] = check()
	}
	return results
}

// OverallStatus returns the worst status across all components.
func OverallStatus(results map[string]ComponentStatus) Status {
	worst := StatusHealthy
	for _, cs := range results {
		switch cs.Name {
		case StatusUnhealthy:
			return StatusUnhealthy
		case StatusDegraded:
			worst = StatusDegraded
		}
	}
	return worst
}

// RegisterDBCheck adds a database health check (Ping).
func RegisterDBCheck(agg *Aggregator, name string, db *sql.DB) {
	agg.Register(name, func() ComponentStatus {
		if db == nil {
			return ComponentStatus{Name: StatusUnhealthy, Details: "nil connection"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			return ComponentStatus{Name: StatusUnhealthy, Details: err.Error()}
		}
		return ComponentStatus{Name: StatusHealthy}
	})
}

// RegisterProviderCheck adds a provider health check.
func RegisterProviderCheck(agg *Aggregator, name string, getHealthy func() bool) {
	agg.Register(name, func() ComponentStatus {
		if getHealthy() {
			return ComponentStatus{Name: StatusHealthy}
		}
		return ComponentStatus{Name: StatusDegraded, Details: "provider not healthy"}
	})
}

// RegisterChannelCheck adds a channel health check.
func RegisterChannelCheck(agg *Aggregator, name string, isConnected func() bool) {
	agg.Register(name, func() ComponentStatus {
		if isConnected() {
			return ComponentStatus{Name: StatusHealthy}
		}
		return ComponentStatus{Name: StatusDegraded, Details: "channel disconnected"}
	})
}

// RegisterMemoryCheck adds a memory/vector store health check.
func RegisterMemoryCheck(agg *Aggregator, name string, getRecordCount func() int) {
	agg.Register(name, func() ComponentStatus {
		count := getRecordCount()
		if count < 0 {
			return ComponentStatus{Name: StatusUnhealthy, Details: "store error"}
		}
		return ComponentStatus{Name: StatusHealthy, Details: fmt.Sprintf("%d records", count)}
	})
}

// Server provides a simple HTTP health check endpoint.
type Server struct {
	server     *http.Server
	StartTime  time.Time
	aggregator *Aggregator
}

// NewServer creates a new health check server with an aggregator.
func NewServer(port int, agg *Aggregator) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		StartTime:  time.Now(),
		aggregator: agg,
	}

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/live", s.handleLiveness)
	mux.HandleFunc("/ready", s.handleReadiness)

	return s
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	results := s.aggregator.Collect()
	overall := OverallStatus(results)

	statusCode := http.StatusOK
	if overall == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	resp := map[string]any{
		"status":     overall,
		"uptime":     time.Since(s.StartTime).String(),
		"components": results,
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	results := s.aggregator.Collect()
	overall := OverallStatus(results)

	if overall == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// Start runs the health check server in the background.
func (s *Server) Start() {
	go func() {
		slog.Info("health server started", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server error", "error", err)
		}
	}()
}

// Shutdown gracefully stops the health server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
