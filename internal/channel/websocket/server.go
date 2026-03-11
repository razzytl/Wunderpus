package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for MVP
	},
}

// Server implements the WebSocket communication channel.
type Server struct {
	port    int
	manager *agent.Manager
	server  *http.Server
	mu      sync.Mutex
	conns   map[*websocket.Conn]bool
}

// NewServer creates a new WebSocket channel server.
func NewServer(port int, manager *agent.Manager) *Server {
	return &Server{
		port:    port,
		manager: manager,
		conns:   make(map[*websocket.Conn]bool),
	}
}

// Name returns the channel name.
func (s *Server) Name() string {
	return "websocket"
}

// Start launches the WebSocket server.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleConnection)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	slog.Info("websocket channel starting", "port", s.port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("websocket server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s.mu.Lock()
		for conn := range s.conns {
			conn.Close()
		}
		s.mu.Unlock()

		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.conns[conn] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()

	slog.Info("new websocket client connected", "addr", r.RemoteAddr)

	// Set read deadline for heartbeat
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Heartbeat ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("websocket read error", "error", err)
			}
			break
		}

		if messageType == websocket.BinaryMessage {
			slog.Info("received binary message, skipping processing for now", "size", len(message))
			continue
		}

		var req types.UserMessage
		if err := json.Unmarshal(message, &req); err != nil {
			slog.Warn("invalid websocket message", "error", err)
			continue
		}

		// If session ID is missing, use a default or some identifier
		if req.SessionID == "" {
			req.SessionID = "ws_session_" + r.RemoteAddr
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			response, err := s.manager.ProcessMessage(ctx, req.SessionID, req.Content)
			if err != nil {
				s.sendError(conn, err)
				return
			}

			s.sendResponse(conn, response)
		}()
	}
}

func (s *Server) sendResponse(conn *websocket.Conn, content string) {
	resp := types.AgentResponse{
		Content: content,
	}
	b, _ := json.Marshal(resp)
	if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
		slog.Error("websocket write error", "error", err)
	}
}

func (s *Server) sendError(conn *websocket.Conn, err error) {
	resp := map[string]string{"error": err.Error()}
	b, _ := json.Marshal(resp)
	_ = conn.WriteMessage(websocket.TextMessage, b)
}
