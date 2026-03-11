package heartbeat

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/agent"
)

// TaskExecutor executes heartbeat tasks
type TaskExecutor interface {
	ExecuteQuickTask(ctx context.Context, task HeartbeatTask) (string, error)
	ExecuteLongTask(ctx context.Context, task HeartbeatTask) (string, error)
}

// Scheduler handles periodic heartbeat task execution
type Scheduler struct {
	cfg       *HeartbeatConfig
	parser    *Parser
	executor  TaskExecutor
	agentMgr  *agent.Manager
	workspace string

	mu         sync.RWMutex
	lastParsed *ParseResult
	lastCheck  time.Time
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewScheduler creates a new heartbeat scheduler
func NewScheduler(cfg *HeartbeatConfig, parser *Parser, executor TaskExecutor, agentMgr *agent.Manager, workspace string) *Scheduler {
	return &Scheduler{
		cfg:       cfg,
		parser:    parser,
		executor:  executor,
		agentMgr:  agentMgr,
		workspace: workspace,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the heartbeat scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		slog.Info("heartbeat scheduler disabled")
		return nil
	}

	// Find HEARTBEAT.md
	heartbeatPath, err := FindHeartbeatFile(s.workspace)
	if err != nil {
		slog.Warn("heartbeat file not found, skipping", "workspace", s.workspace)
		return nil
	}

	slog.Info("heartbeat scheduler starting", "path", heartbeatPath, "interval", s.cfg.Interval)

	// Initial parse
	parsed, err := s.parser.Parse(heartbeatPath)
	if err != nil {
		return fmt.Errorf("failed to parse HEARTBEAT.md: %w", err)
	}
	s.mu.Lock()
	s.lastParsed = parsed
	s.lastCheck = time.Now()
	s.mu.Unlock()

	// Run initial tasks
	go s.runTasks(ctx)

	// Schedule periodic runs
	interval := time.Duration(s.cfg.Interval) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ticker.C:
				s.runTasks(ctx)
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop stops the heartbeat scheduler
func (s *Scheduler) Stop() {
	slog.Info("stopping heartbeat scheduler")
	close(s.stopCh)
	s.wg.Wait()
}

// runTasks executes the heartbeat tasks
func (s *Scheduler) runTasks(ctx context.Context) {
	// Re-parse the file to check for changes
	heartbeatPath, err := FindHeartbeatFile(s.workspace)
	if err != nil {
		slog.Debug("heartbeat file not found during run", "error", err)
		return
	}

	parsed, err := s.parser.Parse(heartbeatPath)
	if err != nil {
		slog.Error("failed to parse HEARTBEAT.md", "error", err)
		return
	}

	s.mu.Lock()
	wasFirstRun := s.lastParsed == nil
	s.lastParsed = parsed
	s.lastCheck = time.Now()
	s.mu.Unlock()

	// Execute quick tasks immediately
	for _, task := range parsed.QuickTasks {
		s.executeQuickTask(ctx, task)
	}

	// Execute long tasks via spawn (if spawn system available)
	for _, task := range parsed.LongTasks {
		s.executeLongTask(ctx, task)
	}

	if wasFirstRun || len(parsed.QuickTasks) > 0 || len(parsed.LongTasks) > 0 {
		slog.Info("heartbeat tasks executed",
			"quick", len(parsed.QuickTasks),
			"long", len(parsed.LongTasks))
	}
}

// executeQuickTask executes a quick task immediately
func (s *Scheduler) executeQuickTask(ctx context.Context, task HeartbeatTask) {
	result, err := s.executor.ExecuteQuickTask(ctx, task)
	if err != nil {
		slog.Error("quick task failed", "task", task.Content, "error", err)
		return
	}
	slog.Info("quick task completed", "task", task.Content, "result", result)
}

// executeLongTask executes a long task (spawns a subagent)
func (s *Scheduler) executeLongTask(ctx context.Context, task HeartbeatTask) {
	result, err := s.executor.ExecuteLongTask(ctx, task)
	if err != nil {
		slog.Error("long task failed", "task", task.Content, "error", err)
		return
	}
	slog.Info("long task completed", "task", task.Content, "result", result)
}

// GetStatus returns the current heartbeat status
func (s *Scheduler) GetStatus() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]any{
		"enabled":    s.cfg.Enabled,
		"interval":   s.cfg.Interval,
		"last_check": s.lastCheck,
		"workspace":  s.workspace,
	}

	if s.lastParsed != nil {
		status["quick_tasks"] = len(s.lastParsed.QuickTasks)
		status["long_tasks"] = len(s.lastParsed.LongTasks)
	}

	return status
}
