package perception

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// BrowserAgent executes goal-driven browser automation.
// It runs a loop: screenshot → LLM plans next action → execute → check goal.
type BrowserAgent struct {
	vision     *Vision
	maxActions int
	timeout    time.Duration
}

// NewBrowserAgent creates a new browser agent.
func NewBrowserAgent(vision *Vision) *BrowserAgent {
	return &BrowserAgent{
		vision:     vision,
		maxActions: 50,
		timeout:    10 * time.Minute,
	}
}

// SetMaxActions overrides the default action limit (50).
func (b *BrowserAgent) SetMaxActions(n int) {
	if n > 0 {
		b.maxActions = n
	}
}

// SetTimeout overrides the default timeout (10 minutes).
func (b *BrowserAgent) SetTimeout(d time.Duration) {
	if d > 0 {
		b.timeout = d
	}
}

// Execute runs a browser automation goal.
// It navigates to the URL, then loops:
//  1. Plan action via vision LLM
//  2. Execute action
//  3. Check if goal is achieved
//  4. Repeat until goal or max actions reached
func (b *BrowserAgent) Execute(ctx context.Context, goal string, url string) (*BrowserResult, error) {
	start := time.Now()
	result := &BrowserResult{}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	slog.Info("perception: browser agent starting", "goal", truncate(goal, 100), "url", url)

	// Navigate to URL
	if url != "" {
		if err := b.vision.capturer.Navigate(url); err != nil {
			result.Error = fmt.Sprintf("navigation failed: %v", err)
			result.Duration = time.Since(start)
			return result, nil
		}
	}

	// Main automation loop
	for i := 0; i < b.maxActions; i++ {
		select {
		case <-ctx.Done():
			result.Error = fmt.Sprintf("timeout after %d actions", i)
			result.ActionsTaken = i
			result.Duration = time.Since(start)
			return result, nil
		default:
		}

		// Plan action
		action, err := b.vision.PlanAction(goal)
		if err != nil {
			slog.Warn("perception: planning failed", "step", i, "error", err)
			result.ActionLog = append(result.ActionLog, ActionLog{
				Action:  BrowserAction{Action: "plan"},
				Success: false,
				Error:   err.Error(),
			})
			continue
		}

		// Handle extract action — means goal is likely achieved
		if action.Action == "extract" {
			pageContent := b.vision.executor.GetPageContent()
			result.Success = true
			result.FinalURL = b.vision.executor.GetURL()
			result.ExtractedData = pageContent
			result.ActionsTaken = i
			result.Duration = time.Since(start)
			result.ActionLog = append(result.ActionLog, ActionLog{
				Action:  *action,
				Success: true,
			})
			slog.Info("perception: goal achieved via extract", "actions", i, "duration", time.Since(start))
			return result, nil
		}

		// Execute with self-healing
		logEntry := ActionLog{Action: *action}
		execStart := time.Now()

		err = b.vision.SelfHeal(action, goal, 2)
		logEntry.Duration = time.Since(execStart)

		if err != nil {
			logEntry.Success = false
			logEntry.Error = err.Error()
			slog.Warn("perception: action failed", "step", i, "action", action.Action, "error", err)
		} else {
			logEntry.Success = true
		}

		result.ActionLog = append(result.ActionLog, logEntry)

		// Capture screenshot for the record
		if screenshot, err := b.vision.capturer.Screenshot(); err == nil {
			result.Screenshots = append(result.Screenshots, string(screenshot))
		}
	}

	result.ActionsTaken = b.maxActions
	result.Duration = time.Since(start)
	result.Error = fmt.Sprintf("max actions (%d) reached without goal completion", b.maxActions)

	slog.Info("perception: browser agent completed", "actions", b.maxActions, "success", result.Success)
	return result, nil
}
