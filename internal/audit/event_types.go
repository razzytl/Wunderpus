package audit

// EventType is a typed string representing the category of an audited event.
type EventType string

// Action events — UAA subsystem
const (
	EventActionExecuted  EventType = "action.executed"
	EventActionRejected  EventType = "action.rejected"
	EventActionFailed    EventType = "action.failed"
	EventActionSimulated EventType = "action.simulated"
)

// RSI events — Recursive Self-Improvement
const (
	EventRSICycleStarted      EventType = "rsi.cycle_started"
	EventRSIProposalGenerated EventType = "rsi.proposal_generated"
	EventRSIDeployed          EventType = "rsi.deployed"
	EventRSIRolledBack        EventType = "rsi.rolled_back"
	EventRSISandboxPassed     EventType = "rsi.sandbox_passed"
	EventRSISandboxFailed     EventType = "rsi.sandbox_failed"
	EventRSIFitnessEvaluated  EventType = "rsi.fitness_evaluated"
)

// Goal events — Autonomous Goal Synthesis
const (
	EventGoalCreated   EventType = "goal.created"
	EventGoalActivated EventType = "goal.activated"
	EventGoalCompleted EventType = "goal.completed"
	EventGoalAbandoned EventType = "goal.abandoned"
	EventGoalDeferred  EventType = "goal.deferred"
)

// Resource events — Resource Acquisition
const (
	EventResourceAcquired    EventType = "resource.acquired"
	EventResourceReleased    EventType = "resource.released"
	EventResourceExhausted   EventType = "resource.exhausted"
	EventResourceProvisioned EventType = "resource.provisioned"
)

// Trust events — Unbounded Autonomous Action
const (
	EventTrustDebited  EventType = "trust.debited"
	EventTrustCredited EventType = "trust.credited"
	EventTrustLockdown EventType = "trust.lockdown"
	EventTrustReset    EventType = "trust.reset"
	EventTrustRegen    EventType = "trust.regen"
)

// System events
const (
	EventSystemStartup  EventType = "system.startup"
	EventSystemShutdown EventType = "system.shutdown"
	EventConfigChanged  EventType = "system.config_changed"
	EventDLQEvent       EventType = "system.dlq"
)
