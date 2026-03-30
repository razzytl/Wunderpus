package perception

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Vision provides the vision-language interface for computer use.
// It captures screenshots, sends them to a vision-capable LLM,
// receives actions, and executes them.
type Vision struct {
	capturer ScreenshotCapturer
	executor ActionExecutor
	llm      VisionLLMCaller
}

// NewVision creates a new vision interface.
func NewVision(capturer ScreenshotCapturer, executor ActionExecutor, llm VisionLLMCaller) *Vision {
	return &Vision{
		capturer: capturer,
		executor: executor,
		llm:      llm,
	}
}

// PlanAction takes a screenshot and asks the LLM to plan the next action
// to achieve the given instruction.
func (v *Vision) PlanAction(instruction string) (*BrowserAction, error) {
	slog.Debug("perception: planning action", "instruction", truncate(instruction, 100))

	// Capture screenshot
	screenshot, err := v.capturer.Screenshot()
	if err != nil {
		return nil, fmt.Errorf("vision: screenshot failed: %w", err)
	}

	// Get current DOM context
	domContext := ""
	var elements []DOMElement
	if elements, err = v.executor.ExtractDOM(); err == nil && len(elements) > 0 {
		domCtx, _ := json.Marshal(elements)
		domContext = string(domCtx)
	}

	// Build prompt
	prompt := buildActionPrompt(instruction, v.executor.GetURL(), domContext)

	// Send to vision LLM
	response, err := v.llm.CompleteWithVision(prompt, screenshot, "image/jpeg")
	if err != nil {
		return nil, fmt.Errorf("vision: LLM call failed: %w", err)
	}

	// Parse action
	action, err := parseAction(response)
	if err != nil {
		return nil, fmt.Errorf("vision: parse action: %w", err)
	}

	slog.Debug("perception: action planned", "action", action.Action, "reasoning", truncate(action.Reasoning, 80))
	return action, nil
}

// ExecuteAction runs a browser action using the executor.
func (v *Vision) ExecuteAction(action *BrowserAction) error {
	slog.Debug("perception: executing action", "type", action.Action)

	start := time.Now()

	switch action.Action {
	case "click":
		if action.Coordinates[0] == 0 && action.Coordinates[1] == 0 {
			return fmt.Errorf("click requires coordinates")
		}
		err := v.executor.Click(action.Coordinates[0], action.Coordinates[1])
		if err != nil {
			return fmt.Errorf("click failed: %w", err)
		}

	case "type":
		if action.Text == "" {
			return fmt.Errorf("type requires text")
		}
		err := v.executor.Types(action.Text)
		if err != nil {
			return fmt.Errorf("type failed: %w", err)
		}

	case "scroll":
		dir := action.Direction
		if dir == "" {
			dir = "down"
		}
		err := v.executor.Scroll(dir)
		if err != nil {
			return fmt.Errorf("scroll failed: %w", err)
		}

	case "navigate":
		if action.Text == "" {
			return fmt.Errorf("navigate requires URL")
		}
		err := v.executor.Navigate(action.Text)
		if err != nil {
			return fmt.Errorf("navigate failed: %w", err)
		}

	case "press":
		if action.Key == "" {
			return fmt.Errorf("press requires key name")
		}
		err := v.executor.Press(action.Key)
		if err != nil {
			return fmt.Errorf("press failed: %w", err)
		}

	case "wait":
		// Simple wait — just pause briefly
		time.Sleep(1 * time.Second)

	case "screenshot":
		// No-op — screenshot was already taken for planning
		return nil

	case "extract":
		// Extraction handled at higher level
		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}

	_ = time.Since(start) // timing for future use
	return nil
}

// SelfHeal attempts an action, and if it fails, takes a new screenshot
// and replans with the failure context.
func (v *Vision) SelfHeal(action *BrowserAction, instruction string, maxRetries int) error {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := v.ExecuteAction(action)
		if err == nil {
			return nil
		}

		slog.Warn("perception: action failed, self-healing",
			"action", action.Action,
			"attempt", attempt+1,
			"error", err)

		if attempt >= maxRetries {
			return fmt.Errorf("action failed after %d retries: %w", maxRetries, err)
		}

		// Replan with failure context
		newInstruction := fmt.Sprintf("%s (Previous action '%s' failed: %s. Try a different approach.)",
			instruction, action.Action, err.Error())

		newAction, planErr := v.PlanAction(newInstruction)
		if planErr != nil {
			return fmt.Errorf("replan failed: %w", planErr)
		}
		action = newAction
	}

	return nil
}

// buildActionPrompt creates the prompt for the LLM.
func buildActionPrompt(instruction, currentURL, domContext string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Current URL: %s\n\n", currentURL))
	sb.WriteString(fmt.Sprintf("Instruction: %s\n\n", instruction))

	if domContext != "" {
		sb.WriteString("Interactive elements on the page:\n")
		sb.WriteString(domContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Look at the screenshot and determine the next action to accomplish the instruction.\n")
	sb.WriteString("Return ONLY the JSON action object.\n")

	return sb.String()
}

// parseAction extracts a BrowserAction from LLM response.
func parseAction(response string) (*BrowserAction, error) {
	response = strings.TrimSpace(response)

	// Strip markdown fences
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) >= 2 {
			endIdx := len(lines) - 1
			for i := len(lines) - 1; i > 0; i-- {
				if strings.TrimSpace(lines[i]) == "```" {
					endIdx = i
					break
				}
			}
			response = strings.Join(lines[1:endIdx], "\n")
		}
	}
	response = strings.TrimSpace(response)

	var action BrowserAction
	if err := json.Unmarshal([]byte(response), &action); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if action.Action == "" {
		return nil, fmt.Errorf("action field is required")
	}

	return &action, nil
}

func truncate(s string, maxVal int) string {
	if len(s) <= maxVal {
		return s
	}
	return s[:maxVal] + "..."
}
