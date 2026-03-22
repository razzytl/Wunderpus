package events

import (
	"log/slog"

	"github.com/wunderpus/wunderpus/internal/audit"
)

// WireEvents connects all four pillars via the event bus.
// Call this once at application startup after all subsystems are initialized.
func WireEvents(
	bus *Bus,
	trustCreditor interface {
		Credit(amount int, reason string)
	},
	uaaGate interface{ SuspendExternalActions(suspend bool) },
	profilerReset interface{ ResetBaseline(name string) },
	synthReframer interface{ TriggerReframe() },
	raSuspend interface{ SuspendProvisioning(suspend bool) },
) {
	// RSI deployed → Trust budget credits +100
	bus.Subscribe(audit.EventRSIDeployed, func(e Event) {
		if tc, ok := trustCreditor.(interface{ Credit(int, string) }); ok {
			tc.Credit(100, "RSI improvement deployed")
			slog.Info("events wiring: RSI deployed → trust +100")
		}
	})

	// Resource exhausted → UAA gates Tier 4 actions
	bus.Subscribe(audit.EventResourceExhausted, func(e Event) {
		if gate, ok := uaaGate.(interface{ SuspendExternalActions(bool) }); ok {
			gate.SuspendExternalActions(true)
			slog.Info("events wiring: resource exhausted → Tier 4 actions suspended")
		}
	})

	// Goal completed → Profiler resets baseline
	bus.Subscribe(audit.EventGoalCompleted, func(e Event) {
		if pr, ok := profilerReset.(interface{ ResetBaseline(string) }); ok {
			if goalID, ok := e.Payload.(map[string]interface{}); ok {
				if title, ok := goalID["title"].(string); ok {
					pr.ResetBaseline(title)
					slog.Info("events wiring: goal completed → profiler baseline reset", "goal", title)
				}
			}
		}
	})

	// Goal abandoned → Synthesizer reframe
	bus.Subscribe(audit.EventGoalAbandoned, func(e Event) {
		if sr, ok := synthReframer.(interface{ TriggerReframe() }); ok {
			sr.TriggerReframe()
			slog.Info("events wiring: goal abandoned → synthesizer reframing")
		}
	})

	// Lockdown → RA suspends provisioning
	bus.Subscribe(audit.EventTrustLockdown, func(e Event) {
		if ra, ok := raSuspend.(interface{ SuspendProvisioning(bool) }); ok {
			ra.SuspendProvisioning(true)
			slog.Info("events wiring: lockdown engaged → provisioning suspended")
		}
	})

	slog.Info("events wiring: all cross-pillar subscriptions established")
}
