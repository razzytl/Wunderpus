package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/wunderpus/wunderpus/internal/provider"
	"github.com/wunderpus/wunderpus/internal/tool"
)

// Orchestrator manages the execution of a TaskGraph through concurrent worker arms.
type Orchestrator struct {
	router         *provider.Router
	globalRegistry *tool.Registry
	executor       *tool.Executor
	model          string
}

// NewOrchestrator creates a new Orchestrator instance.
func NewOrchestrator(router *provider.Router, registry *tool.Registry, executor *tool.Executor, defaultModel string) *Orchestrator {
	return &Orchestrator{
		router:         router,
		globalRegistry: registry,
		executor:       executor,
		model:          defaultModel,
	}
}

// Execute sets up the execution environment, launches workers for independent tasks,
// and synthesizes the final output.
func (o *Orchestrator) Execute(ctx context.Context, graph *TaskGraph) (string, error) {
	if len(graph.Subtasks) == 0 {
		return "No subtasks to execute.", nil
	}

	results := make(map[string]string)
	var mu sync.Mutex

	// We'll execute tasks that have their dependencies met.
	// Since tasks can have dependencies, we loop until all tasks are done or an error occurs.
	completed := make(map[string]bool)

	for len(completed) < len(graph.Subtasks) {
		var wg sync.WaitGroup
		var loopErrs []error
		var muErr sync.Mutex

		didWorkThisLoop := false

		// Find tasks that are ready to run (dependencies are in the 'completed' map) and aren't completed yet
		for _, task := range graph.Subtasks {
			if completed[task.ID] {
				continue
			}

			// Check dependencies
			ready := true
			for _, depID := range task.Dependencies {
				if !completed[depID] {
					ready = false
					break
				}
			}

			if ready {
				didWorkThisLoop = true
				wg.Add(1)

				// Clone state to pass into the goroutine safely
				t := task
				go func(subtask Subtask) {
					defer wg.Done()

					// Compile context from completed dependencies
					contextData := ""
					mu.Lock()
					for _, depID := range subtask.Dependencies {
						if res, ok := results[depID]; ok {
							contextData += fmt.Sprintf("\n[Result of %s]:\n %s\n", depID, res)
						}
					}
					mu.Unlock()

					// Spawn a worker arm with the specific role
					worker := NewWorkerArm(subtask.ID, subtask.Type, o.router, o.globalRegistry, o.executor)
					
					res, err := worker.ExecuteSubtask(ctx, subtask, contextData)

					if err != nil {
						muErr.Lock()
						loopErrs = append(loopErrs, fmt.Errorf("task %s failed: %w", subtask.ID, err))
						muErr.Unlock()
						return
					}

					mu.Lock()
					results[subtask.ID] = res
					completed[subtask.ID] = true
					mu.Unlock()

				}(t)
			}
		}

		// Wait for the batch of independent tasks to finish before looping to check dependencies again
		wg.Wait()

		if len(loopErrs) > 0 {
			return "", fmt.Errorf("orchestrator hit subtask failures: %v", loopErrs)
		}

		if !didWorkThisLoop && len(completed) < len(graph.Subtasks) {
			// This means there's an unresolved dependency graph (circular loop or missing ID mapping)
			return "", fmt.Errorf("deadlock detected in TaskGraph. Some tasks cannot be resolved")
		}
	}

	// ----------------------------------------------------
	// Synthesize phase
	// ----------------------------------------------------
	return o.synthesize(ctx, graph.Goal, results)
}

func (o *Orchestrator) synthesize(ctx context.Context, originalGoal string, results map[string]string) (string, error) {
	sysPrompt := "You are the final synthesizer agent. Your job is to take the original user goal and a raw dump of answers from specialized sub-agents, and return a clean, cohesive final response to the user."
	
	compiledContext := fmt.Sprintf("Original Goal: %s\n\nSub-agent reports:\n", originalGoal)
	for id, res := range results {
		compiledContext += fmt.Sprintf("--- Report from %s ---\n%s\n\n", id, res)
	}

	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: sysPrompt},
		{Role: provider.RoleUser, Content: compiledContext},
	}

	req := &provider.CompletionRequest{
		Messages:    messages,
		Model:       o.model,
		Temperature: 0.3,
	}

	prov := o.router.Active()
	resp, err := prov.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("synthesizer failed: %w", err)
	}

	return resp.Content, nil
}
