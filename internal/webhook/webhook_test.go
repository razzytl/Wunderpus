package webhook

import (
	"testing"

	"github.com/wunderpus/wunderpus/internal/events"
)

func TestManager_NoConfig(t *testing.T) {
	bus := events.NewBus()
	m := NewManager(bus, nil)
	if m == nil {
		t.Fatal("NewManager should not return nil")
	}
}

func TestManager_EmptyConfig(t *testing.T) {
	bus := events.NewBus()
	m := NewManager(bus, []Config{})
	if m == nil {
		t.Fatal("NewManager should not return nil")
	}

	deliveries := m.GetDeliveries()
	if len(deliveries) != 0 {
		t.Errorf("expected 0 deliveries, got %d", len(deliveries))
	}
}

func TestManager_GetFailedDeliveries(t *testing.T) {
	bus := events.NewBus()
	m := NewManager(bus, nil)

	// Manually record some deliveries
	m.recordDelivery(Delivery{ID: "1", Status: 200})
	m.recordDelivery(Delivery{ID: "2", Status: 500})
	m.recordDelivery(Delivery{ID: "3", Status: 200})
	m.recordDelivery(Delivery{ID: "4", Status: 0})

	failed := m.GetFailedDeliveries()
	if len(failed) != 2 {
		t.Errorf("expected 2 failed deliveries, got %d", len(failed))
	}

	// Verify the failed ones are the right IDs
	failedIDs := make(map[string]bool)
	for _, d := range failed {
		failedIDs[d.ID] = true
	}
	if !failedIDs["2"] || !failedIDs["4"] {
		t.Errorf("expected failed IDs 2 and 4, got %v", failedIDs)
	}
}

func TestConfig_Events(t *testing.T) {
	cfg := Config{
		Name:   "test",
		URL:    "http://example.com/hook",
		Events: []string{"tool.failed", "goal.completed"},
	}

	if cfg.Name != "test" {
		t.Errorf("expected name 'test', got %s", cfg.Name)
	}
	if len(cfg.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(cfg.Events))
	}
	if cfg.Events[0] != "tool.failed" {
		t.Errorf("expected first event 'tool.failed', got %s", cfg.Events[0])
	}
}

func TestDelivery_Struct(t *testing.T) {
	d := Delivery{
		ID:      "wh-123",
		Webhook: "slack",
		Event:   "tool.failed",
		Payload: `{"test": true}`,
		Status:  200,
	}

	if d.ID != "wh-123" {
		t.Errorf("expected ID 'wh-123', got %s", d.ID)
	}
	if d.Status != 200 {
		t.Errorf("expected status 200, got %d", d.Status)
	}
}
