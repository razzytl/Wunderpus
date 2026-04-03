package uaa

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/wunderpus/wunderpus/internal/audit"
	"github.com/wunderpus/wunderpus/internal/events"
)

// ActionResult contains the output of an executed action.
type ActionResult struct {
	ActionID string
	Success  bool
	Output   string
	Error    string
}

// Action represents a tool execution request.
type Action struct {
	ID         string
	Tool       string
	Parameters map[string]interface{}
	Tier       int
	TrustCost  int
}

// ToolRunnerFn executes the actual tool operation.
type ToolRunnerFn func(ctx context.Context, action Action) (*ActionResult, error)

// Profiler is the interface for telemetry tracking.
type Profiler interface {
	Track(name string, fn func() error) error
}

// UAA is the Unbounded Autonomous Action executor.
// Actions are gated through trust budget before execution.
type UAA struct {
	trust      *TrustBudget
	audit      *audit.AuditLog
	events     *events.Bus
	toolRunner ToolRunnerFn
	profiler   Profiler // optional — tracks tool execution telemetry
}

// NewUAA creates a new UAA executor.
func NewUAA(
	trust *TrustBudget,
	auditLog *audit.AuditLog,
	bus *events.Bus,
	toolRunner ToolRunnerFn,
) *UAA {
	return &UAA{
		trust:      trust,
		audit:      auditLog,
		events:     bus,
		toolRunner: toolRunner,
	}
}

// SetProfiler attaches a profiler for telemetry tracking.
// When set, every tool execution is wrapped with profiler.Track().
func (u *UAA) SetProfiler(p Profiler) {
	u.profiler = p
}

// Execute gates and executes an action through the UAA pipeline:
// trust check → deduct → execute → record outcome.
func (u *UAA) Execute(ctx context.Context, action Action) (*ActionResult, error) {
	// 1. Check trust budget (non-binding pre-check; atomic TryDeduct happens below)
	ok, reason := u.trust.CanExecute(action.TrustCost)
	if !ok {
		u.writeAudit(audit.EventActionRejected, action, reason)
		u.publishEvent(audit.EventActionRejected, action, reason)
		return nil, fmt.Errorf("uaa: action rejected — %s", reason)
	}

	// 2. Atomically check + deduct trust (eliminates TOCTOU race)
	deducted, deductReason := u.trust.TryDeduct(action.TrustCost, action.ID)
	if !deducted {
		u.writeAudit(audit.EventActionRejected, action, deductReason)
		u.publishEvent(audit.EventActionRejected, action, deductReason)
		return nil, fmt.Errorf("uaa: trust deduction failed — %s", deductReason)
	}

	// 3. Execute — optionally wrapped with profiler telemetry
	var result *ActionResult
	var err error
	if u.profiler != nil {
		trackErr := u.profiler.Track(action.Tool, func() error {
			result, err = u.toolRunner(ctx, action)
			return err
		})
		if trackErr != nil && err == nil {
			err = trackErr
		}
	} else {
		result, err = u.toolRunner(ctx, action)
	}

	// 4. Record outcome
	success := err == nil
	u.trust.RecordOutcome(action.ID, action.TrustCost, success)

	if success && result != nil {
		u.writeAudit(audit.EventActionExecuted, action, result.Output)
		u.publishEvent(audit.EventActionExecuted, action, result.Output)
	} else if success && result == nil {
		u.writeAudit(audit.EventActionExecuted, action, "nil result")
		u.publishEvent(audit.EventActionExecuted, action, "nil result")
	} else {
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		u.writeAudit(audit.EventActionFailed, action, errStr)
		u.publishEvent(audit.EventActionFailed, action, errStr)
	}

	return result, err
}

func (u *UAA) writeAudit(eventType audit.EventType, action Action, detail string) {
	if u.audit == nil {
		return
	}
	payloadMap := map[string]interface{}{
		"action_id": action.ID,
		"tool":      action.Tool,
		"tier":      int(action.Tier),
		"cost":      action.TrustCost,
		"detail":    detail,
	}
	payload, _ := json.Marshal(payloadMap)
	_ = u.audit.Write(audit.AuditEntry{
		Subsystem: "uaa",
		EventType: eventType,
		ActorID:   action.ID,
		Payload:   payload,
	})
}

func (u *UAA) publishEvent(eventType audit.EventType, action Action, detail string) {
	if u.events == nil {
		return
	}
	u.events.Publish(events.Event{
		Type:   eventType,
		Source: "uaa",
		Payload: map[string]interface{}{
			"action_id": action.ID,
			"tool":      action.Tool,
			"tier":      int(action.Tier),
			"detail":    detail,
		},
	})
}

// Shutdown logs a shutdown event.
func (u *UAA) Shutdown() {
	slog.Info("uaa: shutting down")
	if u.trust != nil {
		u.trust.StopRegen()
	}
}
