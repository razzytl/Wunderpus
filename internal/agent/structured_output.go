package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/wunderpus/wunderpus/internal/provider"
)

// OutputFormat describes the expected output format for a tool or agent response.
type OutputFormat struct {
	Type        string `json:"type"`                  // "text" or "json"
	JSONSchema  string `json:"json_schema,omitempty"` // JSON Schema string when Type == "json"
	Description string `json:"description,omitempty"` // Human-readable description of expected output
}

// StructuredOutputEnforcer validates and retries LLM responses to ensure
// they conform to the expected output format.
type StructuredOutputEnforcer struct {
	maxRetries int
}

// NewStructuredOutputEnforcer creates a new enforcer with the given max retry count.
func NewStructuredOutputEnforcer(maxRetries int) *StructuredOutputEnforcer {
	if maxRetries <= 0 {
		maxRetries = 2 // default: 2 retries
	}
	return &StructuredOutputEnforcer{maxRetries: maxRetries}
}

// ValidateAndFix checks if the response conforms to the expected format.
// If Type is "json", it validates against json.Valid and optionally a schema.
// Returns the validated content and whether it passed.
func (e *StructuredOutputEnforcer) ValidateAndFix(content string, format OutputFormat) (string, bool) {
	if format.Type == "" || format.Type == "text" {
		return content, true // text format always passes
	}

	if format.Type == "json" {
		// Basic JSON validity check
		if !json.Valid([]byte(content)) {
			return content, false
		}
		return content, true
	}

	return content, true
}

// CorrectionPrompt generates a system message asking the LLM to fix its output.
func (e *StructuredOutputEnforcer) CorrectionPrompt(originalResponse string, format OutputFormat) string {
	if format.JSONSchema != "" {
		return fmt.Sprintf(
			"Your previous response was not valid JSON matching the expected schema. "+
				"Please regenerate your response as valid JSON that conforms to this schema:\n\n%s\n\n"+
				"Your previous response was:\n%s",
			format.JSONSchema, originalResponse,
		)
	}
	return fmt.Sprintf(
		"Your previous response was not valid JSON. Please regenerate your response as valid JSON.\n\n"+
			"Your previous response was:\n%s",
		originalResponse,
	)
}

// ExecuteWithValidation sends a completion request and validates the response.
// If validation fails, it retries up to maxRetries times with correction prompts.
// Each retry counts against the caller's iteration budget.
func (e *StructuredOutputEnforcer) ExecuteWithValidation(
	ctx context.Context,
	completeFn func(ctx context.Context, messages []provider.Message) (*provider.CompletionResponse, error),
	messages []provider.Message,
	format OutputFormat,
) (*provider.CompletionResponse, int, error) {
	resp, err := completeFn(ctx, messages)
	if err != nil {
		return nil, 0, err
	}

	validated, valid := e.ValidateAndFix(resp.Content, format)
	if valid {
		resp.Content = validated
		return resp, 0, nil
	}

	// Retry with correction prompts
	for attempt := 0; attempt < e.maxRetries; attempt++ {
		slog.Warn("structured output: validation failed, retrying",
			"attempt", attempt+1, "maxRetries", e.maxRetries)

		correctionMsg := e.CorrectionPrompt(resp.Content, format)
		messages = append(messages, provider.Message{
			Role:    provider.RoleAssistant,
			Content: resp.Content,
		}, provider.Message{
			Role:    provider.RoleUser,
			Content: correctionMsg,
		})

		resp, err = completeFn(ctx, messages)
		if err != nil {
			return nil, attempt + 1, err
		}

		validated, valid = e.ValidateAndFix(resp.Content, format)
		if valid {
			resp.Content = validated
			return resp, attempt + 1, nil
		}
	}

	// All retries exhausted — return the last response with a warning
	slog.Warn("structured output: all retries exhausted, returning unvalidated response")
	return resp, e.maxRetries, nil
}
