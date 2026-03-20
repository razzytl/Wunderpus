package events

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBus_SubscribeAndPublish(t *testing.T) {
	bus := NewBus()
	var counter int64

	// Subscribe 10 handlers to the same event
	for i := 0; i < 10; i++ {
		bus.Subscribe("test.event", func(e Event) {
			atomic.AddInt64(&counter, 1)
		})
	}

	if bus.SubscriberCount("test.event") != 10 {
		t.Fatalf("expected 10 subscribers, got %d", bus.SubscriberCount("test.event"))
	}

	// Publish async
	bus.Publish(Event{Type: "test.event", Source: "test"})

	// Wait for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&counter) != 10 {
		t.Fatalf("expected 10 handler calls, got %d", atomic.LoadInt64(&counter))
	}
}

func TestBus_PanickingHandler_DoesNotCrash(t *testing.T) {
	bus := NewBus()
	var safeHandlerFired atomic.Bool

	// Panicking handler
	bus.Subscribe("test.panic", func(e Event) {
		panic("intentional panic")
	})

	// Safe handler should still fire
	bus.Subscribe("test.panic", func(e Event) {
		safeHandlerFired.Store(true)
	})

	bus.Publish(Event{Type: "test.panic", Source: "test"})
	time.Sleep(100 * time.Millisecond)

	if !safeHandlerFired.Load() {
		t.Fatal("safe handler should have fired despite panicking handler")
	}

	// DLQ should have the event
	if bus.DLQCount() != 1 {
		t.Fatalf("expected 1 DLQ event, got %d", bus.DLQCount())
	}
}

func TestBus_PublishSync_BlocksUntilComplete(t *testing.T) {
	bus := NewBus()
	var results []int
	var mu sync.Mutex

	bus.Subscribe("test.sync", func(e Event) {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		results = append(results, 1)
		mu.Unlock()
	})

	bus.Subscribe("test.sync", func(e Event) {
		time.Sleep(30 * time.Millisecond)
		mu.Lock()
		results = append(results, 2)
		mu.Unlock()
	})

	// PublishSync should block until both handlers complete
	bus.PublishSync(Event{Type: "test.sync", Source: "test"})

	mu.Lock()
	defer mu.Unlock()
	if len(results) != 2 {
		t.Fatalf("expected 2 results after PublishSync, got %d", len(results))
	}
}

func TestBus_MultipleEventTypes(t *testing.T) {
	bus := NewBus()
	var typeA, typeB atomic.Int64

	bus.Subscribe("type.a", func(e Event) {
		typeA.Add(1)
	})
	bus.Subscribe("type.b", func(e Event) {
		typeB.Add(1)
	})

	bus.Publish(Event{Type: "type.a", Source: "test"})
	bus.Publish(Event{Type: "type.b", Source: "test"})
	bus.Publish(Event{Type: "type.a", Source: "test"})

	time.Sleep(100 * time.Millisecond)

	if typeA.Load() != 2 {
		t.Fatalf("type.a expected 2, got %d", typeA.Load())
	}
	if typeB.Load() != 1 {
		t.Fatalf("type.b expected 1, got %d", typeB.Load())
	}
}

func TestBus_DLQClear(t *testing.T) {
	bus := NewBus()

	bus.Subscribe("panic.event", func(e Event) {
		panic("boom")
	})

	bus.Publish(Event{Type: "panic.event", Source: "test"})
	time.Sleep(100 * time.Millisecond)

	if bus.DLQCount() != 1 {
		t.Fatalf("expected 1 DLQ, got %d", bus.DLQCount())
	}

	bus.ClearDLQ()
	if bus.DLQCount() != 0 {
		t.Fatalf("expected 0 DLQ after clear, got %d", bus.DLQCount())
	}
}
