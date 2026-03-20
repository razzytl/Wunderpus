package events

import "github.com/wunderpus/wunderpus/internal/audit"

// EventType re-exports the audit EventType for convenience.
// The canonical type definition lives in audit/event_types.go.
// This alias allows the events package to use the same type system
// without duplicating constants.
type EventType = audit.EventType
