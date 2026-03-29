package swarm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Goal represents a task to be dispatched to specialists.
type Goal struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Input       string `json:"input"`
}

// SpecialistExecutor runs a goal with a specific specialist config.
type SpecialistExecutor func(ctx context.Context, goal Goal, config AgentConfig) (*SpecialistResult, error)

// SpecialistResult is the output from a single specialist.
type SpecialistResult struct {
	Specialist string        `json:"specialist"`
	GoalID     string        `json:"goal_id"`
	Output     string        `json:"output"`
	Success    bool          `json:"success"`
	Duration   time.Duration `json:"duration"`
	Error      string        `json:"error,omitempty"`
}

// SwarmResult is the aggregated output from all specialists.
type SwarmResult struct {
	GoalID        string             `json:"goal_id"`
	Specialists   []string           `json:"specialists"`
	Results       []SpecialistResult `json:"results"`
	Synthesis     string             `json:"synthesis"` // unified output from LLM synthesis
	Success       bool               `json:"success"`
	TotalCost     float64            `json:"total_cost"`
	TotalDuration time.Duration      `json:"total_duration"`
}

// Synthesizer aggregates multiple specialist results into a unified output.
type Synthesizer func(ctx context.Context, goal Goal, results []SpecialistResult) (string, error)

// Orchestrator dispatches goals to specialist agents.
// It analyzes the goal, identifies required specialists,
// runs them in parallel, and synthesizes results.
type Orchestrator struct {
	executor    SpecialistExecutor
	synthesizer Synthesizer
}

// NewOrchestrator creates a new swarm orchestrator.
func NewOrchestrator(executor SpecialistExecutor, synthesizer Synthesizer) *Orchestrator {
	return &Orchestrator{
		executor:    executor,
		synthesizer: synthesizer,
	}
}

// Dispatch analyzes a goal, identifies required specialists,
// executes them (in parallel when possible), and synthesizes results.
func (o *Orchestrator) Dispatch(ctx context.Context, goal Goal) (*SwarmResult, error) {
	start := time.Now()

	slog.Info("swarm: dispatching goal", "goal", goal.Title)

	// Match specialists to the goal
	specialists := MatchAllSpecialists(goal.Description)
	if len(specialists) == 0 {
		// Default to operator if no match
		if op, ok := GetSpecialist("operator"); ok {
			specialists = []AgentConfig{op}
		} else {
			return nil, fmt.Errorf("swarm: no specialist matched for goal: %s", goal.Title)
		}
	}

	slog.Info("swarm: specialists matched",
		"goal", goal.Title,
		"specialists", len(specialists),
		"names", specialistNames(specialists))

	// Execute specialists in parallel
	results := o.executeParallel(ctx, goal, specialists)

	// Synthesize results
	var synthesis string
	if o.synthesizer != nil {
		synth, err := o.synthesizer(ctx, goal, results)
		if err != nil {
			slog.Warn("swarm: synthesis failed", "error", err)
			synthesis = o.fallbackSynthesis(results)
		} else {
			synthesis = synth
		}
	} else {
		synthesis = o.fallbackSynthesis(results)
	}

	// Determine overall success
	allSucceeded := true
	for _, r := range results {
		if !r.Success {
			allSucceeded = false
			break
		}
	}

	swarmResult := &SwarmResult{
		GoalID:        goal.ID,
		Specialists:   specialistNames(specialists),
		Results:       results,
		Synthesis:     synthesis,
		Success:       allSucceeded,
		TotalDuration: time.Since(start),
	}

	slog.Info("swarm: dispatch complete",
		"goal", goal.Title,
		"success", allSucceeded,
		"duration", swarmResult.TotalDuration)

	return swarmResult, nil
}

// executeParallel runs specialists concurrently and collects results.
func (o *Orchestrator) executeParallel(ctx context.Context, goal Goal, specialists []AgentConfig) []SpecialistResult {
	var wg sync.WaitGroup
	results := make([]SpecialistResult, len(specialists))
	var mu sync.Mutex

	for i, specialist := range specialists {
		wg.Add(1)
		go func(idx int, config AgentConfig) {
			defer wg.Done()

			slog.Debug("swarm: specialist starting", "specialist", config.Name, "goal", goal.Title)

			start := time.Now()
			result, err := o.executor(ctx, goal, config)
			duration := time.Since(start)

			sr := SpecialistResult{
				Specialist: config.Name,
				GoalID:     goal.ID,
				Duration:   duration,
			}

			if err != nil {
				sr.Success = false
				sr.Error = err.Error()
				slog.Warn("swarm: specialist failed", "specialist", config.Name, "error", err)
			} else if result != nil {
				sr.Output = result.Output
				sr.Success = result.Success
				sr.Error = result.Error
			} else {
				sr.Success = true
			}

			mu.Lock()
			results[idx] = sr
			mu.Unlock()
		}(i, specialist)
	}

	wg.Wait()
	return results
}

// fallbackSynthesis combines specialist outputs without an LLM.
func (o *Orchestrator) fallbackSynthesis(results []SpecialistResult) string {
	var output string
	for _, r := range results {
		if r.Output != "" {
			output += fmt.Sprintf("[%s]: %s\n\n", r.Specialist, r.Output)
		}
	}
	return output
}

// specialistNames extracts names from a list of configs.
func specialistNames(configs []AgentConfig) []string {
	names := make([]string, len(configs))
	for i, c := range configs {
		names[i] = c.Name
	}
	return names
}
