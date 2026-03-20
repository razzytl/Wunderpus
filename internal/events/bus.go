package events

import (
	"log/slog"
	"sync"
	"time"
)

// Event represents a single event in the system's pub/sub bus.
type Event struct {
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	Source    string      `json:"source"`
}

// HandlerFunc is a callback that handles an event.
type HandlerFunc func(Event)

// Bus is a typed pub/sub event bus that all four pillars communicate through.
// Publish is non-blocking (goroutine per handler). PublishSync blocks.
type Bus struct {
	subscribers map[EventType][]HandlerFunc
	mu          sync.RWMutex
	dlq         []Event // dead-letter queue for panicking handlers
	dlqMu       sync.Mutex
}

// NewBus creates and returns a new event bus.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[EventType][]HandlerFunc),
	}
}

// Subscribe registers a handler for the given event type.
// Multiple handlers can be registered for the same type.
func (b *Bus) Subscribe(t EventType, h HandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[t] = append(b.subscribers[t], h)
}

// Publish sends an event to all registered handlers non-blocking.
// Each handler runs in its own goroutine. Panicking handlers are caught
// and the event is routed to the dead-letter queue.
func (b *Bus) Publish(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	b.mu.RLock()
	handlers := make([]HandlerFunc, len(b.subscribers[e.Type]))
	copy(handlers, b.subscribers[e.Type])
	b.mu.RUnlock()

	for _, h := range handlers {
		fn := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event handler panic",
						"event_type", e.Type,
						"source", e.Source,
						"panic", r,
					)
					b.sendToDLQ(e)
				}
			}()
			fn(e)
		}()
	}
}

// PublishSync sends an event to all registered handlers and blocks until all complete.
// Useful for testing and deterministic event ordering.
func (b *Bus) PublishSync(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	b.mu.RLock()
	handlers := make([]HandlerFunc, len(b.subscribers[e.Type]))
	copy(handlers, b.subscribers[e.Type])
	b.mu.RUnlock()

	var wg sync.WaitGroup
	for _, h := range handlers {
		fn := h
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event handler panic (sync)",
						"event_type", e.Type,
						"source", e.Source,
						"panic", r,
					)
					b.sendToDLQ(e)
				}
			}()
			fn(e)
		}()
	}
	wg.Wait()
}

// DLQ returns a copy of the dead-letter queue (events whose handlers panicked).
func (b *Bus) DLQ() []Event {
	b.dlqMu.Lock()
	defer b.dlqMu.Unlock()
	cp := make([]Event, len(b.dlq))
	copy(cp, b.dlq)
	return cp
}

// DLQCount returns the number of events in the dead-letter queue.
func (b *Bus) DLQCount() int {
	b.dlqMu.Lock()
	defer b.dlqMu.Unlock()
	return len(b.dlq)
}

// ClearDLQ empties the dead-letter queue.
func (b *Bus) ClearDLQ() {
	b.dlqMu.Lock()
	defer b.dlqMu.Unlock()
	b.dlq = nil
}

func (b *Bus) sendToDLQ(e Event) {
	b.dlqMu.Lock()
	defer b.dlqMu.Unlock()
	b.dlq = append(b.dlq, e)
}

// SubscriberCount returns the number of handlers registered for a given event type.
func (b *Bus) SubscriberCount(t EventType) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[t])
}
