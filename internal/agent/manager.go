package agent

import (
	"context"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/config"
	"github.com/wunderpus/wunderpus/internal/cost"
	"github.com/wunderpus/wunderpus/internal/memory"
	"github.com/wunderpus/wunderpus/internal/provider"
	"github.com/wunderpus/wunderpus/internal/security"
	"github.com/wunderpus/wunderpus/internal/skills"
	"github.com/wunderpus/wunderpus/internal/tool"
	"github.com/wunderpus/wunderpus/internal/types"
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
	loader    *skills.SkillsLoader

	mu      sync.RWMutex
	agents  map[string]*Agent
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
	loader *skills.SkillsLoader,
) *Manager {
	m := &Manager{
		cfg:       cfg,
		router:    router,
		sanitizer: sanitizer,
		audit:     audit,
		store:     store,
		registry:  registry,
		executor:  executor,
		loader:    loader,
		agents:    make(map[string]*Agent),
	}

	if cfg.Security.RateLimit.Enabled {
		m.limiter = security.NewRateLimiter(
			time.Duration(cfg.Security.RateLimit.WindowSec)*time.Second,
			cfg.Security.RateLimit.MaxRequests,
		)
		// Start automatic cleanup if configured (default: every 5 minutes)
		if cfg.Security.RateLimit.CleanupIntervalSec > 0 {
			m.limiter.StartAutoCleanup(time.Duration(cfg.Security.RateLimit.CleanupIntervalSec) * time.Second)
		}
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
		m.loader,
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

// GetSkillsLoader returns the skills loader.
func (m *Manager) GetSkillsLoader() *skills.SkillsLoader {
	return m.loader
}

// ProcessRequest processes a full UserMessage request.
func (m *Manager) ProcessRequest(ctx context.Context, req types.UserMessage) (types.AgentResponse, error) {
	resp, err := m.ProcessMessage(ctx, req.SessionID, req.Content)
	return types.AgentResponse{Content: resp, SessionID: req.SessionID}, err
}
