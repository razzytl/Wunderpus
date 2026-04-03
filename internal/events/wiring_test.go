package events

import (
	"sync/atomic"
	"testing"

	"github.com/wunderpus/wunderpus/internal/audit"
)

func TestWireEvents_GoalCompleted_ResetsProfiler(t *testing.T) {
	bus := NewBus()

	var resetGoal atomic.Value
	mockProfiler := &mockProfiler{onReset: func(name string) {
		resetGoal.Store(name)
	}}

	WireEvents(bus, mockProfiler, nil)

	bus.PublishSync(Event{
		Type:   audit.EventGoalCompleted,
		Source: "test",
		Payload: map[string]interface{}{
			"title": "test-goal",
		},
	})

	if got, _ := resetGoal.Load().(string); got != "test-goal" {
		t.Fatal("goal completed should reset profiler baseline")
	}
}

func TestWireEvents_GoalAbandoned_RefamesSynth(t *testing.T) {
	bus := NewBus()

	var reframed atomic.Bool
	mockSynth := &mockSynth{onReframe: func() {
		reframed.Store(true)
	}}

	WireEvents(bus, nil, mockSynth)

	bus.PublishSync(Event{Type: audit.EventGoalAbandoned, Source: "test"})

	if !reframed.Load() {
		t.Fatal("goal abandoned should trigger synthesizer reframe")
	}
}

// Mock implementations
type mockProfiler struct {
	onReset func(string)
}

func (m *mockProfiler) ResetBaseline(name string) {
	if m.onReset != nil {
		m.onReset(name)
	}
}

type mockSynth struct {
	onReframe func()
}

func (m *mockSynth) TriggerReframe() {
	if m.onReframe != nil {
		m.onReframe()
	}
}
