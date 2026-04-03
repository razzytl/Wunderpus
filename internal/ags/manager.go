package ags

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/wunderpus/wunderpus/internal/events"
	"github.com/wunderpus/wunderpus/internal/provider"
	"github.com/wunderpus/wunderpus/internal/uaa"
)

// AGSManager centralizes all AGS components and orchestrates their loops.
type AGSManager struct {
	Store         *GoalStore
	Scorer        *PriorityScorer
	Synthesizer   *GoalSynthesizer
	Executor      *GoalExecutor
	Metacognition *MetacognitionLoop
	Bus           *events.Bus

	stopFns []func()
}

// NewAGSManager initializes all AGS components and wires them together.
func NewAGSManager(
	db *sql.DB,
	p provider.Provider,
	trust *uaa.TrustBudget,
	bus *events.Bus,
	taskExec TaskExecutorFn,
	successJudge SuccessJudgeFn,
) (*AGSManager, error) {
	store, err := NewGoalStore(db)
	if err != nil {
		return nil, err
	}

	scorer := NewPriorityScorer(trust)
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

// Start initiates all background loops.
func (m *AGSManager) Start(ctx context.Context) {
	slog.Info("ags manager: starting background loops")

	// 1. Synthesizer loop (1 hour)
	// Note: We provide default getters for memories and weaknesses for now.
	// In production, these should be wired to actual Profiler/AuditLog sources.
	stopSyn := m.Synthesizer.StartScheduler(ctx, func() []MemoryEntry { return nil }, func() []WeaknessSnapshot { return nil })
	m.stopFns = append(m.stopFns, stopSyn)

	// 2. Executor loop (5 minutes)
	stopExec := m.Executor.StartExecutionLoop(ctx, 5*time.Minute)
	m.stopFns = append(m.stopFns, stopExec)

	// 3. Metacognition loop (7 days)
	stopMeta := m.Metacognition.StartScheduler(ctx)
	m.stopFns = append(m.stopFns, stopMeta)
}

// Stop gracefully shuts down all loops.
func (m *AGSManager) Stop() {
	slog.Info("ags manager: stopping background loops")
	for _, stop := range m.stopFns {
		stop()
	}
}

// SynthesizeImmediately triggers a synthesis cycle off-schedule (e.g., on operator request).
func (m *AGSManager) SynthesizeImmediately(ctx context.Context, memories []MemoryEntry, weaknesses []WeaknessSnapshot) ([]Goal, error) {
	return m.Synthesizer.Synthesize(ctx, memories, weaknesses)
}

// RunMetacognitionImmediately triggers a metacognition cycle off-schedule.
func (m *AGSManager) RunMetacognitionImmediately() (*MetacognitionReport, error) {
	return m.Metacognition.Run()
}
