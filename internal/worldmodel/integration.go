package worldmodel

import (
	"log/slog"
	"path/filepath"
	"time"
)

// Config holds configuration for the world model system.
type Config struct {
	Enabled         bool
	DBPath          string
	ScanIntervalH   int  // hours between web scans (default 24)
	ConfidenceDecay bool // enable confidence decay (default true)
}

// DefaultConfig returns sensible defaults for the world model.
func DefaultConfig(homeDir string) Config {
	return Config{
		Enabled:         false,
		DBPath:          filepath.Join(homeDir, "wunderpus_worldmodel.db"),
		ScanIntervalH:   24,
		ConfidenceDecay: true,
	}
}

// WorldModelSystem holds all world model components wired together.
type WorldModelSystem struct {
	Store     *Store
	Extractor *Extractor
	Query     *QueryInterface
	Updater   *Updater
}

// InitWorldModel initializes the complete world model system.
func InitWorldModel(cfg Config, llm LLMCaller, searcher WebSearcher) (*WorldModelSystem, error) {
	if !cfg.Enabled {
		slog.Info("worldmodel: disabled by config")
		return nil, nil
	}

	slog.Info("worldmodel: initializing", "db", cfg.DBPath)

	// 1. Store — SQLite knowledge graph
	store, err := NewStore(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	// 2. Extractor — LLM-based knowledge extraction
	extractor := NewExtractor(store, llm)

	// 3. Query interface — natural language graph queries
	queryIntf := NewQueryInterface(store, llm)

	// 4. Updater — self-updating via web scan and events
	updater := NewUpdater(store, extractor, searcher)
	if cfg.ScanIntervalH > 0 {
		updater.SetScanInterval(time.Duration(cfg.ScanIntervalH) * time.Hour)
	}

	return &WorldModelSystem{
		Store:     store,
		Extractor: extractor,
		Query:     queryIntf,
		Updater:   updater,
	}, nil
}
