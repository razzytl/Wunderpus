package ags

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/wunderpus/wunderpus/internal/events"
	"github.com/wunderpus/wunderpus/internal/provider"
)

// AGSManager centralizes all AGS components and orchestrates their loops.
type AGSManager struct {
	Store         *GoalStore
	Scorer        *PriorityScorer
	Synthesizer   *GoalSynthesizer
	Executor      *GoalExecutor
	Metacognition *MetacognitionLoop
	Bus           *events.Bus
}

// NewAGSManager initializes all AGS components and wires them together.
func NewAGSManager(
	db *sql.DB,
	p provider.Provider,
	bus *events.Bus,
	taskExec TaskExecutorFn,
	successJudge SuccessJudgeFn,
) (*AGSManager, error) {
	store, err := NewGoalStore(db)
	if err != nil {
		return nil, err
	}

	scorer := NewPriorityScorer()
	synthesizer := NewGoalSynthesizer(p, store, scorer, bus)
	executor := NewGoalExecutor(store, scorer, p, bus, taskExec, successJudge)
	meta := NewMetacognitionLoop(store, scorer)

	return &AGSManager{
		Store:         store,
		Scorer:        scorer,
		Synthesizer:   synthesizer,
		Executor:      executor,
		Metacognition: meta,
		Bus:           bus,
	}, nil
}

// Start is a no-op — AGS autonomous loops have been paused.
// Use SynthesizeImmediately() and Executor.RunOnce() for explicit invocation.
func (m *AGSManager) Start(ctx context.Context) {
	slog.Info("ags manager: autonomous loops paused — use explicit API/CLI commands")
}

// Stop is a no-op since no background loops are running.
func (m *AGSManager) Stop() {
	slog.Info("ags manager: stop (no-op — loops are paused)")
}

// SynthesizeImmediately triggers a synthesis cycle off-schedule (e.g., on operator request).
func (m *AGSManager) SynthesizeImmediately(ctx context.Context, memories []MemoryEntry, weaknesses []WeaknessSnapshot) ([]Goal, error) {
	return m.Synthesizer.Synthesize(ctx, memories, weaknesses)
}

// RunMetacognitionImmediately triggers a metacognition cycle off-schedule.
func (m *AGSManager) RunMetacognitionImmediately() (*MetacognitionReport, error) {
	return m.Metacognition.Run()
}
