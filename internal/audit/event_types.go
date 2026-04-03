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

// Goal events — Autonomous Goal Synthesis
const (
	EventGoalCreated   EventType = "goal.created"
	EventGoalActivated EventType = "goal.activated"
	EventGoalCompleted EventType = "goal.completed"
	EventGoalAbandoned EventType = "goal.abandoned"
	EventGoalDeferred  EventType = "goal.deferred"
)

// Trust events — Unbounded Autonomous Action
const (
	EventTrustDebited  EventType = "trust.debited"
	EventTrustCredited EventType = "trust.credited"
	EventTrustLockdown EventType = "trust.lockdown"
	EventTrustReset    EventType = "trust.reset"
	EventTrustRegen    EventType = "trust.regen"

	// EventLockdownEngaged is an alias for EventTrustLockdown (checklist naming).
	EventLockdownEngaged = EventTrustLockdown
)

// Tool Synthesis events
const (
	EventToolGapDetected EventType = "toolsynth.gap_detected"
	EventToolDesigned    EventType = "toolsynth.designed"
	EventToolCoded       EventType = "toolsynth.coded"
	EventToolTestPassed  EventType = "toolsynth.test_passed"
	EventToolTestFailed  EventType = "toolsynth.test_failed"
	EventToolSynthesized EventType = "toolsynth.synthesized"
	EventToolMarketplace EventType = "toolsynth.marketplace_scan"
)

// World Model events
const (
	EventEntityCreated      EventType = "worldmodel.entity_created"
	EventRelationCreated    EventType = "worldmodel.relation_created"
	EventKnowledgeExtracted EventType = "worldmodel.knowledge_extracted"
	EventWorldModelUpdated  EventType = "worldmodel.updated"
)

// Perception events
const (
	EventBrowserAction   EventType = "perception.browser_action"
	EventBrowserGoalDone EventType = "perception.browser_goal_done"
	EventDesktopAction   EventType = "perception.desktop_action"
)

// Swarm events
const (
	EventSwarmDispatch  EventType = "swarm.dispatch"
	EventSwarmCompleted EventType = "swarm.completed"
	EventSwarmFailed    EventType = "swarm.failed"
)

// System events
const (
	EventSystemStartup  EventType = "system.startup"
	EventSystemShutdown EventType = "system.shutdown"
	EventConfigChanged  EventType = "system.config_changed"
	EventDLQEvent       EventType = "system.dlq"
)
