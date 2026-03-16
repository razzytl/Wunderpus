package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wunderpus/wunderpus/internal/agent"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local dev
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Server serves the embedded frontend and handles WebSocket connections.
type Server struct {
	port    int
	manager *agent.Manager
	server  *http.Server

	mu    sync.Mutex
	conns map[*websocket.Conn]string // conn → sessionID
}

// NewServer creates a new web UI server.
// distFS: the embedded filesystem containing the built frontend.
// fsRoot: subdirectory inside the embed.FS (e.g., "dist").
// port: HTTP port to listen on.
// manager: the agent manager for processing messages.
func NewServer(distFS fs.FS, fsRoot string, port int, manager *agent.Manager) (*Server, error) {
	sub, err := fs.Sub(distFS, fsRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create sub filesystem: %w", err)
	}

	s := &Server{
		port:    port,
		manager: manager,
		conns:   make(map[*websocket.Conn]string),
	}

	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.handleWebSocket)

	// API endpoints
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/history", s.handleHistory)

	// Static file serving with SPA fallback
	fileServer := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Serve API and WS routes first (already handled above)
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Check if file exists in embedded FS
		f, openErr := sub.Open(path[1:])
		if openErr != nil {
			// SPA fallback — serve index.html
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()

		fileServer.ServeHTTP(w, r)
	})

	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return s, nil
}

// Start begins serving the web UI and WebSocket.
func (s *Server) Start() error {
	slog.Info("[Wunderpus] UI running", "url", fmt.Sprintf("http://localhost:%d", s.port))

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("web server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the server and closes all WebSocket connections.
func (s *Server) Stop() error {
	s.mu.Lock()
	for conn := range s.conns {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"))
		conn.Close()
	}
	s.mu.Unlock()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleWebSocket upgrades HTTP to WebSocket and runs the read/write loop.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	sessionID := "web_" + r.RemoteAddr

	s.mu.Lock()
	s.conns[conn] = sessionID
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()

	slog.Info("web UI client connected", "addr", r.RemoteAddr, "session", sessionID)

	// Send welcome system log
	s.sendMessage(conn, WSMessage{
		Type:      MsgTypeSystemLog,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Payload: SystemLogPayload{
			Level:   "info",
			Message: "Connected to Wunderpus Web UI",
		},
	})

	// Configure connection keepalive
	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	// Ping ticker for keepalive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	// Read loop
	for {
		_, message, readErr := conn.ReadMessage()
		if readErr != nil {
			if websocket.IsUnexpectedCloseError(readErr, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("websocket read error", "error", readErr)
			}
			break
		}

		// Parse envelope
		var envelope WSMessage
		if err := json.Unmarshal(message, &envelope); err != nil {
			s.sendMessage(conn, WSMessage{
				Type:      MsgTypeError,
				Timestamp: time.Now(),
				SessionID: sessionID,
				Payload:   ErrorPayload{Message: "invalid message format"},
			})
			continue
		}

		switch envelope.Type {
		case MsgTypeUserMessage:
			s.handleUserMessage(conn, sessionID, envelope)
		default:
			slog.Warn("unknown message type", "type", envelope.Type)
		}
	}
}

// handleUserMessage processes a user message and streams the response.
func (s *Server) handleUserMessage(conn *websocket.Conn, sessionID string, envelope WSMessage) {
	// Extract payload
	payloadBytes, err := json.Marshal(envelope.Payload)
	if err != nil {
		s.sendError(conn, sessionID, "failed to parse message payload")
		return
	}

	var payload UserMessagePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		s.sendError(conn, sessionID, "invalid user message payload")
		return
	}

	if payload.Content == "" {
		s.sendError(conn, sessionID, "empty message")
		return
	}

	// Use session from payload if provided
	sid := sessionID
	if payload.SessionID != "" {
		sid = payload.SessionID
	}

	// Notify frontend that processing started
	s.sendMessage(conn, WSMessage{
		Type:      MsgTypeSystemLog,
		Timestamp: time.Now(),
		SessionID: sid,
		Payload: SystemLogPayload{
			Level:   "info",
			Message: "Processing your request...",
		},
	})

	// Process in goroutine to not block the read loop
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		response, processErr := s.manager.ProcessMessage(ctx, sid, payload.Content)
		if processErr != nil {
			s.sendError(conn, sid, processErr.Error())
			return
		}

		// Send the complete response
		// (For now, we send the full response. Phase 4+ can add streaming.)
		s.sendMessage(conn, WSMessage{
			Type:      MsgTypeChatToken,
			Timestamp: time.Now(),
			SessionID: sid,
			Payload: ChatTokenPayload{
				Token: response,
				Done:  true,
			},
		})

		s.sendMessage(conn, WSMessage{
			Type:      MsgTypeChatComplete,
			Timestamp: time.Now(),
			SessionID: sid,
			Payload: ChatCompletePayload{
				Content: response,
			},
		})
	}()
}

// sendMessage sends a typed JSON message over the WebSocket.
func (s *Server) sendMessage(conn *websocket.Conn, msg WSMessage) {
	b, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal ws message", "error", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
		slog.Error("websocket write error", "error", err)
	}
}

// sendError sends an error message to the client.
func (s *Server) sendError(conn *websocket.Conn, sessionID, msg string) {
	s.sendMessage(conn, WSMessage{
		Type:      MsgTypeError,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Payload:   ErrorPayload{Message: msg},
	})
}

// Broadcast sends a message to all connected clients.
func (s *Server) Broadcast(msg WSMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for conn := range s.conns {
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			slog.Error("broadcast write error", "error", err)
		}
	}
}
