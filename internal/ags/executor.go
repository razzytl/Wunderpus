package ags

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// TaskExecutorFn is a function that executes a single task and returns the result.
type TaskExecutorFn func(ctx context.Context, task TaskBlueprint) (string, error)

// TaskBlueprint is a concrete task derived from a goal.
type TaskBlueprint struct {
	StepNum         int                    `json:"step_num"`
	Description     string                 `json:"description"`
	Tool            string                 `json:"tool"`
	Parameters      map[string]interface{} `json:"parameters"`
	ExpectedOutcome string                 `json:"expected_outcome"`
	DependsOn       []int                  `json:"depends_on"`
}

// SuccessJudgeFn evaluates whether outcomes satisfy success criteria.
type SuccessJudgeFn func(ctx context.Context, criteria []string, outcomes []string) (bool, float64)

// GoalExecutor selects, decomposes, and executes goals.
type GoalExecutor struct {
	store        *GoalStore
	scorer       *PriorityScorer
	llmFn        LLMFn
	taskExec     TaskExecutorFn
	successJudge SuccessJudgeFn
	maxAttempts  int
}

// NewGoalExecutor creates a new goal executor.
func NewGoalExecutor(
	store *GoalStore,
	scorer *PriorityScorer,
	llmFn LLMFn,
	taskExec TaskExecutorFn,
	successJudge SuccessJudgeFn,
) *GoalExecutor {
	return &GoalExecutor{
		store:        store,
		scorer:       scorer,
		llmFn:        llmFn,
		taskExec:     taskExec,
		successJudge: successJudge,
		maxAttempts:  3,
	}
}

// SelectNext fetches all pending goals, rescores them, and returns the highest-priority one.
func (e *GoalExecutor) SelectNext() (*Goal, error) {
	pending, err := e.store.GetByStatus(GoalStatusPending)
	if err != nil {
		return nil, fmt.Errorf("ags executor: fetching pending goals: %w", err)
	}

	if len(pending) == 0 {
		return nil, nil
	}

	// Rescore all pending goals
	var best *Goal
	bestScore := -1.0

	for i := range pending {
		pending[i].Priority = e.scorer.Score(pending[i])
		_ = e.store.Update(pending[i])

		if pending[i].Priority > bestScore {
			bestScore = pending[i].Priority
			best = &pending[i]
		}
	}

	return best, nil
}

// Decompose breaks a goal into concrete task blueprints via LLM.
func (e *GoalExecutor) Decompose(ctx context.Context, g Goal) ([]TaskBlueprint, error) {
	systemPrompt := `You are a task decomposition engine. Given a goal, break it into an ordered list of concrete tasks.

Each task must specify:
- step_num: integer starting from 1
- description: what to do
- tool: which tool to use (file_read, file_write, shell_exec, web_search, http_request, etc.)
- parameters: JSON object with tool-specific parameters
- expected_outcome: what success looks like
- depends_on: array of step_nums this depends on (empty array if independent)

Output ONLY a JSON array of task objects, no explanation.`

	criteriaJSON, _ := json.Marshal(g.SuccessCriteria)
	userPrompt := fmt.Sprintf(`GOAL: %s
DESCRIPTION: %s
TIER: %d
SUCCESS CRITERIA: %s

Decompose this goal into concrete tasks.`, g.Title, g.Description, g.Tier, string(criteriaJSON))

	response, err := e.llmFn(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("ags executor: decomposition LLM call failed: %w", err)
	}

	var tasks []TaskBlueprint
	if err := json.Unmarshal([]byte(response), &tasks); err != nil {
		return nil, fmt.Errorf("ags executor: failed to parse task blueprint: %w", err)
	}

	return tasks, nil
}

// Execute runs the full goal lifecycle: activate → execute tasks → evaluate → complete/abandon.
func (e *GoalExecutor) Execute(ctx context.Context, g Goal, tasks []TaskBlueprint) error {
	// Mark active
	now := time.Now().UTC()
	g.Status = GoalStatusActive
	g.AttemptCount++
	g.LastAttempt = &now
	g.UpdatedAt = now
	if err := e.store.Update(g); err != nil {
		return fmt.Errorf("ags executor: failed to mark goal active: %w", err)
	}

	slog.Info("ags executor: executing goal", "title", g.Title, "attempt", g.AttemptCount)

	// Execute each task
	var outcomes []string
	for _, task := range tasks {
		outcome, err := e.taskExec(ctx, task)
		if err != nil {
			outcomes = append(outcomes, fmt.Sprintf("Task %d FAILED: %v", task.StepNum, err))
			slog.Warn("ags executor: task failed", "goal", g.Title, "step", task.StepNum, "error", err)
			continue
		}
		outcomes = append(outcomes, fmt.Sprintf("Task %d OK: %s", task.StepNum, outcome))
	}

	// Evaluate success criteria
	success, actualValue := e.successJudge(ctx, g.SuccessCriteria, outcomes)

	if success {
		// Goal completed
		g.Status = GoalStatusCompleted
		g.ActualValue = &actualValue
		now := time.Now().UTC()
		g.CompletedAt = &now
		g.UpdatedAt = now
		_ = e.store.Update(g)
		slog.Info("ags executor: goal COMPLETED", "title", g.Title, "actual_value", actualValue)
		return nil
	}

	// Goal failed this attempt
	if g.AttemptCount >= e.maxAttempts {
		// Abandon after max attempts
		g.Status = GoalStatusAbandoned
		g.UpdatedAt = time.Now().UTC()
		_ = e.store.Update(g)
		slog.Warn("ags executor: goal ABANDONED after max attempts", "title", g.Title, "attempts", g.AttemptCount)
		return nil
	}

	// Return to pending for retry
	g.Status = GoalStatusPending
	g.UpdatedAt = time.Now().UTC()
	_ = e.store.Update(g)
	slog.Info("ags executor: goal failed, returning to pending", "title", g.Title, "attempt", g.AttemptCount)
	return nil
}

// StartExecutionLoop runs SelectNext → Decompose → Execute on a background goroutine.
// Cycles every interval (default 5 minutes between goal executions).
func (e *GoalExecutor) StartExecutionLoop(ctx context.Context, interval time.Duration) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				goal, err := e.SelectNext()
				if err != nil {
					slog.Warn("ags executor: SelectNext failed", "error", err)
					continue
				}
				if goal == nil {
					continue // no pending goals
				}

				slog.Info("ags executor: executing goal", "title", goal.Title, "tier", goal.Tier)

				tasks, err := e.Decompose(ctx, *goal)
				if err != nil {
					slog.Warn("ags executor: Decompose failed", "goal", goal.Title, "error", err)
					// Reset to pending
					goal.Status = GoalStatusPending
					_ = e.store.Update(*goal)
					continue
				}

				if err := e.Execute(ctx, *goal, tasks); err != nil {
					slog.Warn("ags executor: Execute failed", "goal", goal.Title, "error", err)
				}
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { close(stop) }
}
