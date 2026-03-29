package swarm

import "log/slog"

// Config holds configuration for the swarm system.
type Config struct {
	Enabled bool
}

// SwarmSystem holds all swarm components wired together.
type SwarmSystem struct {
	Orchestrator *Orchestrator
}

// InitSwarm initializes the swarm system.
func InitSwarm(cfg Config, executor SpecialistExecutor, synthesizer Synthesizer) (*SwarmSystem, error) {
	if !cfg.Enabled {
		slog.Info("swarm: disabled by config")
		return nil, nil
	}

	slog.Info("swarm: initializing", "specialists", len(SpecialistProfiles))

	orch := NewOrchestrator(executor, synthesizer)

	return &SwarmSystem{
		Orchestrator: orch,
	}, nil
}
