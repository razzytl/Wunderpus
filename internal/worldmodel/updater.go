package worldmodel

import (
	"fmt"
	"log/slog"
	"time"
)

// EventBus abstracts the event bus for the world model.
type EventBus interface {
	Subscribe(eventType string, handler func(Event))
}

// Event is a minimal event type for the world model.
type Event struct {
	Type      string
	Payload   interface{}
	Timestamp time.Time
	Source    string
}

// WebSearcher abstracts web search for the updater.
type WebSearcher interface {
	Search(query string) ([]SearchResult, error)
}

// SearchResult is a web search result.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Updater manages self-updating of the world model through:
// 1. Periodic web scans for dynamic entities (prices, roles, status)
// 2. Event subscriptions (goal completed → extract, tool synthesized → add entity)
type Updater struct {
	store        *Store
	extractor    *Extractor
	searcher     WebSearcher
	eventBus     EventBus
	scanInterval time.Duration
	stopCh       chan struct{}
}

// NewUpdater creates a new world model updater.
func NewUpdater(store *Store, extractor *Extractor, searcher WebSearcher) *Updater {
	return &Updater{
		store:        store,
		extractor:    extractor,
		searcher:     searcher,
		scanInterval: 24 * time.Hour, // daily by default
		stopCh:       make(chan struct{}),
	}
}

// SetEventBus configures the event bus and subscribes to relevant events.
func (u *Updater) SetEventBus(bus EventBus) {
	u.eventBus = bus

	// Subscribe to goal completed events
	bus.Subscribe("goal.completed", func(e Event) {
		if payload, ok := e.Payload.(map[string]interface{}); ok {
			title, _ := payload["title"].(string)
			output, _ := payload["output"].(string)
			id, _ := payload["id"].(string)
			if title != "" && output != "" {
				u.HandleGoalCompleted(title, output, id)
			}
		}
	})

	// Subscribe to tool synthesized events
	bus.Subscribe("toolsynth.synthesized", func(e Event) {
		if payload, ok := e.Payload.(map[string]interface{}); ok {
			name, _ := payload["tool_name"].(string)
			desc, _ := payload["description"].(string)
			if name != "" {
				u.HandleToolSynthesized(name, desc)
			}
		}
	})

	slog.Info("worldmodel: event bus wired", "subscriptions", []string{"goal.completed", "toolsynth.synthesized"})
}

// SetScanInterval overrides the default scan interval.
func (u *Updater) SetScanInterval(d time.Duration) {
	if d > 0 {
		u.scanInterval = d
	}
}

// Start begins the periodic web scan in a background goroutine.
func (u *Updater) Start() {
	go u.scanLoop()
	slog.Info("worldmodel: updater started", "interval", u.scanInterval)
}

// Stop stops the background scan loop.
func (u *Updater) Stop() {
	close(u.stopCh)
	slog.Info("worldmodel: updater stopped")
}

// scanLoop runs the periodic update cycle.
func (u *Updater) scanLoop() {
	ticker := time.NewTicker(u.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.runUpdateCycle()
		case <-u.stopCh:
			return
		}
	}
}

// runUpdateCycle performs one update cycle:
// 1. Apply confidence decay
// 2. Scan dynamic entities for updates
func (u *Updater) runUpdateCycle() {
	slog.Info("worldmodel: running update cycle")

	// 1. Apply confidence decay
	decayed, err := u.store.ApplyConfidenceDecay()
	if err != nil {
		slog.Error("worldmodel: confidence decay failed", "error", err)
	}

	// 2. Scan dynamic entities for web updates
	updated := 0
	if u.searcher != nil {
		updated = u.updateDynamicEntities()
	}

	slog.Info("worldmodel: update cycle complete", "decayed", decayed, "updated", updated)
}

// updateDynamicEntities refreshes properties for entities marked as dynamic.
func (u *Updater) updateDynamicEntities() int {
	// Get all dynamic entities
	rows, err := u.store.db.Query(`
		SELECT id, type, name, properties, created_at, updated_at, confidence, source, is_dynamic
		FROM entities WHERE is_dynamic = 1 AND confidence > 0.1`)
	if err != nil {
		return 0
	}
	defer rows.Close()

	entities, _ := u.store.scanEntities(rows)
	updated := 0

	for _, entity := range entities {
		// Search for updated information
		query := fmt.Sprintf("%s %s latest", entity.Type, entity.Name)
		results, err := u.searcher.Search(query)
		if err != nil || len(results) == 0 {
			continue
		}

		// Use the top result for extraction
		topResult := results[0]
		fact, err := u.extractor.Extract(
			fmt.Sprintf("%s\n%s", topResult.Title, topResult.Snippet),
			"api", // web search is treated as API data
			fmt.Sprintf("web-refresh-%s", entity.ID),
		)
		if err != nil {
			continue
		}

		// Check if any extracted entity matches ours
		for _, extracted := range fact.Entities {
			if extracted.Name == entity.Name {
				updated++
				slog.Debug("worldmodel: entity refreshed from web",
					"name", entity.Name, "source", topResult.URL)
				break
			}
		}
	}

	return updated
}

// HandleGoalCompleted is called when EventGoalCompleted is received.
// It extracts knowledge from the completed goal's output.
func (u *Updater) HandleGoalCompleted(goalTitle, goalOutput, goalID string) {
	if u.extractor == nil {
		return
	}

	slog.Info("worldmodel: extracting from completed goal", "goal", goalTitle)

	_, err := u.extractor.ExtractFromTask(goalTitle, goalOutput, goalID)
	if err != nil {
		slog.Warn("worldmodel: goal extraction failed", "goal", goalTitle, "error", err)
	}
}

// HandleToolSynthesized is called when EventToolSynthesized is received.
// It adds the new tool as an entity in the world model.
func (u *Updater) HandleToolSynthesized(toolName, toolDescription string) {
	if u.store == nil {
		return
	}

	slog.Info("worldmodel: adding synthesized tool as entity", "tool", toolName)

	_, err := u.store.UpsertEntity(
		EntityInput{
			Name: toolName,
			Type: EntityTool,
			Properties: map[string]interface{}{
				"description": toolDescription,
				"origin":      "synthesized",
			},
		},
		0.9,
		"toolsynth",
	)
	if err != nil {
		slog.Warn("worldmodel: failed to add tool entity", "tool", toolName, "error", err)
	}
}
