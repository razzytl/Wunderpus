package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server provides a simple HTTP health check endpoint.
type Server struct {
	server    *http.Server
	StartTime time.Time
}

// NewServer creates a new health check server.
func NewServer(port int) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		StartTime: time.Now(),
	}

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/live", s.handleLiveness)
	mux.HandleFunc("/ready", s.handleReadiness)

	return s
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{
		"status": "ok",
		"uptime": time.Since(s.StartTime).String(),
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// For now, simple readiness.
	w.Header().Set("Content-Type", "application/json")
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
