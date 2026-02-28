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
	startTime time.Time
}

// NewServer creates a new health check server.
func NewServer(port int) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		startTime: time.Now(),
	}

	mux.HandleFunc("/health", s.handleHealth)

	return s
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := map[string]string{
		"status": "ok",
		"uptime": time.Since(s.startTime).Round(time.Second).String(),
	}

	json.NewEncoder(w).Encode(resp)
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
