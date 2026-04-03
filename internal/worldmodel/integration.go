package worldmodel

import (
	"database/sql"
	"log/slog"
	"time"
)

// Config holds configuration for the world model system.
type Config struct {
	Enabled         bool
	ScanIntervalH   int  // hours between web scans (default 24)
	ConfidenceDecay bool // enable confidence decay (default true)
}

// WorldModelSystem holds all world model components wired together.
type WorldModelSystem struct {
	Store     *Store
	Extractor *Extractor
	Query     *QueryInterface
	Updater   *Updater
}

// InitWorldModel initializes the complete world model system using the shared core DB connection.
func InitWorldModel(llm LLMCaller, searcher WebSearcher, db *sql.DB) (*WorldModelSystem, error) {
	slog.Info("worldmodel: initializing")

	// 1. Store — SQLite knowledge graph
	store, err := NewStore(db)
	if err != nil {
		return nil, err
	}

	// 2. Extractor — LLM-based knowledge extraction
	extractor := NewExtractor(store, llm)

	// 3. Query interface — natural language graph queries
	queryIntf := NewQueryInterface(store, llm)

	// 4. Updater — self-updating via web scan and events
	updater := NewUpdater(store, extractor, searcher)
	updater.SetScanInterval(24 * time.Hour)

	return &WorldModelSystem{
		Store:     store,
		Extractor: extractor,
		Query:     queryIntf,
		Updater:   updater,
	}, nil
}
