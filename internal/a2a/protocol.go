package a2a

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Handler processes incoming A2A tasks.
type Handler func(task Task) (*TaskResult, error)

// Server implements the A2A server — accepts tasks via HTTP and routes them.
type Server struct {
	card       AgentCard
	handler    Handler
	httpServer *http.Server
	registry   *Registry
	mu         sync.RWMutex
}

// NewServer creates an A2A server with the given agent card and task handler.
func NewServer(card AgentCard, handler Handler) *Server {
	s := &Server{
		card:     card,
		handler:  handler,
		registry: NewRegistry(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/a2a/card", s.handleCard)
	mux.HandleFunc("/a2a/task", s.handleTask)
	mux.HandleFunc("/a2a/discover", s.handleDiscover)

	s.httpServer = &http.Server{
		Addr:    card.Endpoint,
		Handler: mux,
	}

	return s
}

// Start begins listening for A2A requests.
func (s *Server) Start() error {
	slog.Info("a2a: server starting", "endpoint", s.card.Endpoint, "agent", s.card.Name)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	return s.httpServer.Close()
}

// Card returns the agent card.
func (s *Server) Card() AgentCard {
	return s.card
}

// handleCard serves the agent card for discovery.
func (s *Server) handleCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.card)
}

// handleTask accepts a task assignment.
func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if task.ID == "" {
		task.ID = uuid.New().String()
	}

	slog.Info("a2a: task received", "task", task.ID, "description", truncate(task.Description, 80))

	// Execute task
	start := time.Now()
	result, err := s.handler(task)
	duration := time.Since(start)

	taskResult := &TaskResult{
		TaskID:   task.ID,
		Duration: duration,
		AgentID:  s.card.Name,
	}

	if err != nil {
		taskResult.Status = TaskFailed
		taskResult.Error = err.Error()
	} else if result != nil {
		taskResult.Status = result.Status
		taskResult.Output = result.Output
		taskResult.Cost = result.Cost
		if result.Error != "" {
			taskResult.Error = result.Error
		}
	} else {
		taskResult.Status = TaskCompleted
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(taskResult)
}

// handleDiscover returns all known agents.
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	agents := s.registry.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

// Registry tracks known A2A agents for discovery.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]AgentCard
}

// NewRegistry creates an empty agent registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]AgentCard),
	}
}

// Register adds an agent to the registry.
func (r *Registry) Register(card AgentCard) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[card.Name] = card
	slog.Info("a2a: agent registered", "name", card.Name, "capabilities", card.Capabilities)
}

// Unregister removes an agent from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, name)
}

// Get retrieves an agent by name.
func (r *Registry) Get(name string) (AgentCard, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	card, ok := r.agents[name]
	return card, ok
}

// List returns all registered agents.
func (r *Registry) List() []AgentCard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cards := make([]AgentCard, 0, len(r.agents))
	for _, c := range r.agents {
		cards = append(cards, c)
	}
	return cards
}

// FindByCapability returns agents that have the given capability.
func (r *Registry) FindByCapability(capability string) []AgentCard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matches []AgentCard
	for _, c := range r.agents {
		for _, cap := range c.Capabilities {
			if cap == capability {
				matches = append(matches, c)
				break
			}
		}
	}
	return matches
}

// Client connects to remote A2A servers.
type Client struct {
	httpClient *http.Client
}

// NewClient creates an A2A client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// DiscoverCard fetches an agent card from an endpoint.
func (c *Client) DiscoverCard(endpoint string) (*AgentCard, error) {
	resp, err := c.httpClient.Get(endpoint + "/a2a/card")
	if err != nil {
		return nil, fmt.Errorf("a2a client: discover: %w", err)
	}
	defer resp.Body.Close()

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("a2a client: decode card: %w", err)
	}
	return &card, nil
}

// AssignTask sends a task to a remote agent and returns the result.
func (c *Client) AssignTask(endpoint string, task Task) (*TaskResult, error) {
	body, _ := json.Marshal(task)

	resp, err := c.httpClient.Post(endpoint+"/a2a/task", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("a2a client: assign task: %w", err)
	}
	defer resp.Body.Close()

	var result TaskResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("a2a client: decode result: %w", err)
	}
	return &result, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
