package tool

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/security"
)

// ApprovalFunc is called before executing sensitive tools.
// Returns true to allow, false to deny.
type ApprovalFunc func(toolName string, args map[string]any) (bool, error)

// Analytics tracks tool usage metrics.
type Analytics struct {
	mu    sync.Mutex
	stats map[string]*ToolStats
}

// ToolStats holds metrics for a single tool.
type ToolStats struct {
	CallCount    int
	ErrorCount   int
	TotalLatency time.Duration
}

// Executor wraps tool execution with sandbox, approval, and audit.
type Executor struct {
	registry   *Registry
	audit      *security.AuditLogger
	approvalFn ApprovalFunc
	timeout    time.Duration
	analytics  *Analytics
}

// NewExecutor creates a new tool executor.
func NewExecutor(
	registry *Registry,
	audit *security.AuditLogger,
	approvalFn ApprovalFunc,
	timeout time.Duration,
) *Executor {
	return &Executor{
		registry:   registry,
		audit:      audit,
		approvalFn: approvalFn,
		timeout:    timeout,
		analytics: &Analytics{
			stats: make(map[string]*ToolStats),
		},
	}
}

// Execute runs a tool call with sandbox, approval, and audit.
func (e *Executor) Execute(ctx context.Context, call ToolCall) *Result {
	start := time.Now()

	// 1. Look up tool
	t, ok := e.registry.Get(call.Name)
	if !ok {
		return &Result{Error: fmt.Sprintf("unknown tool: %s", call.Name)}
	}

	// 2. Check approval for sensitive tools
	if t.Sensitive() && e.approvalFn != nil {
		approved, err := e.approvalFn(call.Name, call.Args)
		if err != nil {
			return &Result{Error: fmt.Sprintf("approval error: %v", err)}
		}
		if !approved {
			e.audit.Log(security.AuditEvent{
				Timestamp:   time.Now(),
				Action:      "tool_denied",
				Input:       fmt.Sprintf("%s(%v)", call.Name, call.Args),
				ThreatLevel: "none",
			})
			return &Result{Error: "tool execution denied by user"}
		}
	}

	// 3. Apply timeout
	execCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// 4. Execute
	slog.Info("tool executing", "tool", call.Name, "args", call.Args)

	result, err := t.Execute(execCtx, call.Args)
	elapsed := time.Since(start)

	if err != nil {
		result = &Result{Error: err.Error()}
	}
	if result == nil {
		result = &Result{Error: "tool returned nil result"}
	}

	// 5. Record analytics
	e.analytics.record(call.Name, elapsed, result.Error != "")

	// 6. Audit log
	auditResult := result.Output
	if result.Error != "" {
		auditResult = "ERROR: " + result.Error
	}

	e.audit.Log(security.AuditEvent{
		Timestamp:   time.Now(),
		Action:      "tool_executed",
		Input:       fmt.Sprintf("%s(%v)", call.Name, call.Args),
		Result:      auditResult,
		ThreatLevel: "none",
	})

	slog.Info("tool completed", "tool", call.Name, "elapsed", elapsed, "hasError", result.Error != "")

	return result
}

// GetStats returns analytics for all tools.
func (e *Executor) GetStats() map[string]*ToolStats {
	e.analytics.mu.Lock()
	defer e.analytics.mu.Unlock()

	out := make(map[string]*ToolStats)
	for k, v := range e.analytics.stats {
		cp := *v
		out[k] = &cp
	}
	return out
}

func (a *Analytics) record(name string, latency time.Duration, isError bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	s, ok := a.stats[name]
	if !ok {
		s = &ToolStats{}
		a.stats[name] = s
	}
	s.CallCount++
	s.TotalLatency += latency
	if isError {
		s.ErrorCount++
	}
}
