package heartbeat

import (
	"testing"
)

func TestHeartbeatConfig(t *testing.T) {
	cfg := &HeartbeatConfig{
		Enabled:   true,
		Interval:  30,
		Workspace: "/tmp/test",
	}

	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}

	if cfg.Interval != 30 {
		t.Errorf("expected Interval 30, got %d", cfg.Interval)
	}

	if cfg.Workspace != "/tmp/test" {
		t.Errorf("expected Workspace '/tmp/test', got %q", cfg.Workspace)
	}
}

func TestSchedulerStatus(t *testing.T) {
	scheduler := &Scheduler{
		cfg: &HeartbeatConfig{
			Enabled:  true,
			Interval: 15,
		},
	}

	status := scheduler.GetStatus()

	if status["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", status["enabled"])
	}

	if status["interval"] != 15 {
		t.Errorf("expected interval=15, got %v", status["interval"])
	}
}
