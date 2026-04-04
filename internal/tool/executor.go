package tool

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/security"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// ApprovalFunc is called before executing sensitive tools.
// Returns true to allow, false to deny.
type ApprovalFunc func(toolName string, args map[string]any) (bool, error)

// Analytics tracks tool usage metrics.
type Analytics struct {
	mu    sync.Mutex
	stats map[string]*ToolStats
}

// ProfilerFn wraps a function with telemetry.
type ProfilerFn func(name string, fn func() error) error

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
	profiler   ProfilerFn // optional: wraps tool calls with telemetry
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

// SetProfiler sets the profiler function that wraps tool calls with telemetry.
func (e *Executor) SetProfiler(fn ProfilerFn) {
	e.profiler = fn
}

// Execute runs a tool call with sandbox, approval, and audit.
// Approval is now policy-based via the tool's ApprovalLevel.
func (e *Executor) Execute(ctx context.Context, call ToolCall) *Result {
	ctx, span := otel.Tracer("tool").Start(ctx, "tool.execute")
	defer span.End()
	span.SetAttributes(
		attribute.String("tool.name", call.Name),
	)

	start := time.Now()

	// 1. Look up tool
	t, ok := e.registry.Get(call.Name)
	if !ok {
		span.SetStatus(codes.Error, "unknown tool")
		return &Result{Error: fmt.Sprintf("unknown tool: %s", call.Name)}
	}

	// 2. Policy-based approval check
	level := t.ApprovalLevel()
	switch level {
	case Blocked:
		span.SetStatus(codes.Error, "blocked by policy")
		span.SetAttributes(attribute.String("error.message", "blocked by policy"))
		e.audit.Log(security.AuditEvent{
			Timestamp:   time.Now(),
			Action:      "tool_blocked",
			Input:       fmt.Sprintf("%s(%v)", call.Name, call.Args),
			ThreatLevel: "high",
		})
		return &Result{Error: fmt.Sprintf("tool %q is blocked by policy", call.Name)}

	case RequiresApproval:
		if e.approvalFn != nil {
			approved, err := e.approvalFn(call.Name, call.Args)
			if err != nil {
				span.SetStatus(codes.Error, "approval error")
				span.SetAttributes(attribute.String("error.message", err.Error()))
				return &Result{Error: fmt.Sprintf("approval error: %v", err)}
			}
			if !approved {
				span.SetStatus(codes.Error, "denied by user")
				e.audit.Log(security.AuditEvent{
					Timestamp:   time.Now(),
					Action:      "tool_denied",
					Input:       fmt.Sprintf("%s(%v)", call.Name, call.Args),
					ThreatLevel: "none",
				})
				return &Result{Error: "tool execution denied by user"}
			}
		}

	case NotifyOnly:
		slog.Info("tool executing (notify-only)", "tool", call.Name, "args", call.Args)

	case AutoExecute:
		// Run immediately, no approval needed
	}

	// 3. Apply timeout
	execCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// 4. Execute (with optional profiler wrapping)
	slog.Info("tool executing", "tool", call.Name, "args", call.Args, "level", level)

	var result *Result
	var execErr error

	if e.profiler != nil {
		_ = e.profiler(call.Name, func() error {
			result, execErr = t.Execute(execCtx, call.Args)
			return execErr
		})
	} else {
		result, execErr = t.Execute(execCtx, call.Args)
	}
	err := execErr
	elapsed := time.Since(start)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.message", err.Error()))
		result = &Result{Error: err.Error()}
	}
	if result == nil {
		result = &Result{Error: "tool returned nil result"}
	}

	span.SetAttributes(
		attribute.Float64("duration_ms", float64(elapsed.Milliseconds())),
		attribute.Bool("has_error", result.Error != ""),
	)

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
