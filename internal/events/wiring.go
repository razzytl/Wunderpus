package events

import (
	"log/slog"

	"github.com/wunderpus/wunderpus/internal/audit"
)

// WireEvents connects subsystems via the event bus.
// Call this once at application startup after all subsystems are initialized.
func WireEvents(
	bus *Bus,
	profilerReset interface{ ResetBaseline(name string) },
	synthReframer interface{ TriggerReframe() },
) {
	// Goal completed → Profiler resets baseline
	bus.Subscribe(audit.EventGoalCompleted, func(e Event) {
		if goalID, ok := e.Payload.(map[string]interface{}); ok {
			if title, ok := goalID["title"].(string); ok {
				profilerReset.ResetBaseline(title)
				slog.Info("events wiring: goal completed → profiler baseline reset", "goal", title)
			}
		}
	})

	// Goal abandoned → Synthesizer reframe
	bus.Subscribe(audit.EventGoalAbandoned, func(e Event) {
		synthReframer.TriggerReframe()
		slog.Info("events wiring: goal abandoned → synthesizer reframing")
	})

	slog.Info("events wiring: all cross-subsystem subscriptions established")
}
