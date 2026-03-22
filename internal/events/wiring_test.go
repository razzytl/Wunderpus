package events

import (
	"sync/atomic"
	"testing"

	"github.com/wunderpus/wunderpus/internal/audit"
)

func TestWireEvents_RSIDeployed_Credits(t *testing.T) {
	bus := NewBus()

	var credited bool
	mockCreditor := &mockCreditor{onCredit: func(amount int, reason string) {
		if amount == 100 {
			credited = true
		}
	}}

	WireEvents(bus, mockCreditor, nil, nil, nil, nil)

	bus.PublishSync(Event{Type: audit.EventRSIDeployed, Source: "test"})

	if !credited {
		t.Fatal("RSI deployed should credit +100 trust")
	}
}

func TestWireEvents_ResourceExhausted_Gates(t *testing.T) {
	bus := NewBus()

	var suspended atomic.Bool
	mockGate := &mockGate{onSuspend: func(s bool) {
		suspended.Store(s)
	}}

	WireEvents(bus, nil, mockGate, nil, nil, nil)

	bus.PublishSync(Event{Type: audit.EventResourceExhausted, Source: "test"})

	if !suspended.Load() {
		t.Fatal("resource exhausted should suspend Tier 4 actions")
	}
}

func TestWireEvents_Lockdown_SuspendsProvisioning(t *testing.T) {
	bus := NewBus()

	var suspended atomic.Bool
	mockRA := &mockRA{onSuspend: func(s bool) {
		suspended.Store(s)
	}}

	WireEvents(bus, nil, nil, nil, nil, mockRA)

	bus.PublishSync(Event{Type: audit.EventTrustLockdown, Source: "test"})

	if !suspended.Load() {
		t.Fatal("lockdown should suspend provisioning")
	}
}

// Mock implementations
type mockCreditor struct {
	onCredit func(int, string)
}

func (m *mockCreditor) Credit(amount int, reason string) {
	if m.onCredit != nil {
		m.onCredit(amount, reason)
	}
}

type mockGate struct {
	onSuspend func(bool)
}

func (m *mockGate) SuspendExternalActions(s bool) {
	if m.onSuspend != nil {
		m.onSuspend(s)
	}
}

type mockRA struct {
	onSuspend func(bool)
}

func (m *mockRA) SuspendProvisioning(s bool) {
	if m.onSuspend != nil {
		m.onSuspend(s)
	}
}
