package perception

import (
	"fmt"
	"log/slog"
	"strings"
)

// DOMAgent provides a fast, vision-free way to interact with web pages.
// Before resorting to vision (slow, expensive), it parses the DOM:
// 1. Extracts interactive elements (buttons, inputs, links, forms)
// 2. Converts to semantic description
// 3. Sends to LLM for action planning (no image = 5x cheaper)
type DOMAgent struct {
	executor ActionExecutor
	llm      VisionLLMCaller
}

// NewDOMAgent creates a new DOM-first agent.
func NewDOMAgent(executor ActionExecutor, llm VisionLLMCaller) *DOMAgent {
	return &DOMAgent{
		executor: executor,
		llm:      llm,
	}
}

// PlanAction plans an action using DOM elements instead of a screenshot.
// This is ~5x cheaper than vision-based planning.
func (d *DOMAgent) PlanAction(instruction string) (*BrowserAction, error) {
	slog.Debug("perception: DOM agent planning", "instruction", truncate(instruction, 100))

	// Extract DOM elements
	elements, err := d.executor.ExtractDOM()
	if err != nil {
		return nil, fmt.Errorf("dom: extract failed: %w", err)
	}

	if len(elements) == 0 {
		return nil, fmt.Errorf("dom: no interactive elements found")
	}

	// Build semantic description
	description := d.buildDescription(elements)

	// Send to LLM (no image needed)
	prompt := buildDOMPrompt(instruction, d.executor.GetURL(), description)

	response, err := d.llm.CompleteText(prompt)
	if err != nil {
		return nil, fmt.Errorf("dom: LLM call failed: %w", err)
	}

	// Parse action — DOM agent prefers selector-based actions
	action, err := parseAction(response)
	if err != nil {
		return nil, fmt.Errorf("dom: parse action: %w", err)
	}

	return action, nil
}

// buildDescription creates a human-readable description of DOM elements.
func (d *DOMAgent) buildDescription(elements []DOMElement) string {
	var sb strings.Builder

	for i, el := range elements {
		if !el.IsVisible {
			continue
		}

		desc := fmt.Sprintf("[%d] %s", i, el.Tag)
		if el.Text != "" {
			desc += fmt.Sprintf(" '%s'", el.Text)
		}
		if el.Position != "" {
			desc += fmt.Sprintf(" at %s", el.Position)
		}
		if el.Selector != "" {
			desc += fmt.Sprintf(" (selector: %s)", el.Selector)
		}
		sb.WriteString(desc)
		sb.WriteString("\n")
	}

	return sb.String()
}

// CanHandle returns true if the page can be handled without vision
// (i.e., has interactive DOM elements).
func (d *DOMAgent) CanHandle() bool {
	elements, err := d.executor.ExtractDOM()
	if err != nil {
		return false
	}

	// Page is DOM-handleable if it has visible interactive elements
	for _, el := range elements {
		if el.IsVisible && (el.Tag == "button" || el.Tag == "input" ||
			el.Tag == "a" || el.Tag == "select" || el.Tag == "form") {
			return true
		}
	}
	return false
}

func buildDOMPrompt(instruction, currentURL, domDescription string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Current URL: %s\n\n", currentURL))
	sb.WriteString(fmt.Sprintf("Instruction: %s\n\n", instruction))
	sb.WriteString("Interactive elements:\n")
	sb.WriteString(domDescription)
	sb.WriteString("\nDetermine the next action. Return ONLY the JSON action object.\n")

	return sb.String()
}
