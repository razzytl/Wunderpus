package agent

import (
	"context"
	"sync"

	"github.com/wonderpus/wonderpus/internal/config"
	"github.com/wonderpus/wonderpus/internal/cost"
	"github.com/wonderpus/wonderpus/internal/memory"
	"github.com/wonderpus/wonderpus/internal/provider"
	"github.com/wonderpus/wonderpus/internal/security"
	"github.com/wonderpus/wonderpus/internal/tool"
)

// Manager handles multiple agent instances, one per session.
type Manager struct {
	cfg       *config.Config
	router    *provider.Router
	sanitizer *security.Sanitizer
	audit     *security.AuditLogger
	store     *memory.Store
	registry  *tool.Registry
	executor  *tool.Executor

	mu     sync.RWMutex
	agents map[string]*Agent
	limiter *security.RateLimiter
	tracker *cost.Tracker
}

// NewManager creates a new Agent Manager.
func NewManager(
	cfg *config.Config,
	router *provider.Router,
	sanitizer *security.Sanitizer,
	audit *security.AuditLogger,
	store *memory.Store,
	registry *tool.Registry,
	executor *tool.Executor,
) *Manager {
	return &Manager{
		cfg:       cfg,
		router:    router,
		sanitizer: sanitizer,
		audit:     audit,
		store:     store,
		registry:  registry,
		executor:  executor,
		agents:    make(map[string]*Agent),
	}

	if cfg.Security.RateLimit.Enabled {
		m.limiter = security.NewRateLimiter(
			time.Duration(cfg.Security.RateLimit.WindowSec)*time.Second,
			cfg.Security.RateLimit.MaxRequests,
		)
	}

	tr, err := cost.NewTracker(cfg.Agent.CostDBPath, cfg.Agent.Budget)
	if err == nil {
		m.tracker = tr
	}

	return m
}

// GetAgent retrieves or creates an agent for the given session ID.
func (m *Manager) GetAgent(sessionID string) *Agent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ag, ok := m.agents[sessionID]; ok {
		return ag
	}

	// Create a new agent for this session
	ag := NewAgent(
		m.router,
		m.sanitizer,
		m.audit,
		m.store,
		m.registry,
		m.executor,
		m.cfg.Agent.SystemPrompt,
		m.cfg.Agent.MaxContextTokens,
		m.cfg.Agent.Temperature,
		sessionID,
	)

	if m.limiter != nil {
		ag.SetRateLimiter(m.limiter)
	}

	m.agents[sessionID] = ag
	return ag
}

// ProcessMessage routes a message for a specific session and returns the response.
func (m *Manager) ProcessMessage(ctx context.Context, sessionID, input string) (string, error) {
	if m.tracker != nil && m.tracker.IsOverBudget(sessionID) {
		return "⚠️  This session has exceeded its budget. Please contact the administrator.", nil
	}

	ag := m.GetAgent(sessionID)
	return ag.HandleMessage(ctx, input)
}
