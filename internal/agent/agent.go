package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wonderpus/wonderpus/internal/provider"
	"github.com/wonderpus/wonderpus/internal/security"
)

// Agent is the core agent that processes user messages.
type Agent struct {
	router    *provider.Router
	sanitizer *security.Sanitizer
	audit     *security.AuditLogger
	ctx       *ContextManager
	sysPrompt string
	temp      float64
}

// NewAgent creates a new agent instance.
func NewAgent(
	router *provider.Router,
	sanitizer *security.Sanitizer,
	audit *security.AuditLogger,
	systemPrompt string,
	maxContextTokens int,
	temperature float64,
) *Agent {
	return &Agent{
		router:    router,
		sanitizer: sanitizer,
		audit:     audit,
		ctx:       NewContextManager(maxContextTokens),
		sysPrompt: systemPrompt,
		temp:      temperature,
	}
}

// HandleMessage processes a user message and returns the agent response.
func (a *Agent) HandleMessage(ctx context.Context, input string) (string, error) {
	// 1. Sanitize
	cleaned, threats := a.sanitizer.Sanitize(input)

	threatLevel := "none"
	if len(threats) > 0 {
		threatLevel = threats[0].Severity
	}

	// Audit the input
	a.audit.Log(security.AuditEvent{
		Timestamp:   time.Now(),
		Action:      "user_message",
		Input:       input,
		ThreatLevel: threatLevel,
	})

	// Block high severity injections
	if security.HasHighSeverity(threats) {
		slog.Warn("prompt injection blocked",
			"threats", len(threats),
			"first", threats[0].Description,
		)
		a.audit.Log(security.AuditEvent{
			Timestamp:   time.Now(),
			Action:      "injection_blocked",
			Input:       input,
			ThreatLevel: "high",
			Result:      threats[0].Description,
		})
		return "⚠️  Your message was flagged for potential prompt injection and has been blocked. Please rephrase your request.", nil
	}

	// 2. Add user message to context
	a.ctx.AddMessage(provider.RoleUser, cleaned)

	// 3. Build messages
	messages := a.buildMessages()

	// 4. Call provider
	prov := a.router.Active()
	req := &provider.CompletionRequest{
		Messages:    messages,
		Temperature: a.temp,
	}

	resp, err := prov.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("provider %s: %w", prov.Name(), err)
	}

	// 5. Add assistant response to context
	a.ctx.AddMessage(provider.RoleAssistant, resp.Content)

	// 6. Audit
	a.audit.Log(security.AuditEvent{
		Timestamp:   time.Now(),
		Action:      "agent_response",
		Result:      resp.Content,
		ThreatLevel: "none",
	})

	return resp.Content, nil
}

// StreamMessage processes a user message and streams the response.
func (a *Agent) StreamMessage(ctx context.Context, input string) (<-chan provider.StreamChunk, error) {
	// 1. Sanitize
	cleaned, threats := a.sanitizer.Sanitize(input)

	threatLevel := "none"
	if len(threats) > 0 {
		threatLevel = threats[0].Severity
	}

	a.audit.Log(security.AuditEvent{
		Timestamp:   time.Now(),
		Action:      "user_message",
		Input:       input,
		ThreatLevel: threatLevel,
	})

	if security.HasHighSeverity(threats) {
		slog.Warn("prompt injection blocked", "threats", len(threats))
		a.audit.Log(security.AuditEvent{
			Timestamp:   time.Now(),
			Action:      "injection_blocked",
			Input:       input,
			ThreatLevel: "high",
		})
		ch := make(chan provider.StreamChunk, 1)
		ch <- provider.StreamChunk{
			Content: "⚠️  Your message was flagged for potential prompt injection and has been blocked.",
			Done:    true,
		}
		close(ch)
		return ch, nil
	}

	// 2. Add to context
	a.ctx.AddMessage(provider.RoleUser, cleaned)

	// 3. Build messages
	messages := a.buildMessages()

	// 4. Stream from provider
	prov := a.router.Active()
	req := &provider.CompletionRequest{
		Messages:    messages,
		Temperature: a.temp,
	}

	streamCh, err := prov.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", prov.Name(), err)
	}

	// Wrap to capture full response for context
	outCh := make(chan provider.StreamChunk, 64)
	go func() {
		defer close(outCh)
		var full string
		for chunk := range streamCh {
			if chunk.Error != nil {
				outCh <- chunk
				return
			}
			full += chunk.Content
			outCh <- chunk
			if chunk.Done {
				a.ctx.AddMessage(provider.RoleAssistant, full)
				a.audit.Log(security.AuditEvent{
					Timestamp:   time.Now(),
					Action:      "agent_response_stream",
					Result:      full,
					ThreatLevel: "none",
				})
			}
		}
	}()

	return outCh, nil
}

func (a *Agent) buildMessages() []provider.Message {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: a.sysPrompt},
	}
	msgs = append(msgs, a.ctx.GetMessages()...)
	return msgs
}

// Router returns the provider router for external access.
func (a *Agent) Router() *provider.Router {
	return a.router
}

// ClearContext resets the conversation history.
func (a *Agent) ClearContext() {
	a.ctx.Clear()
}

// MessageCount returns the number of messages in context.
func (a *Agent) MessageCount() int {
	return a.ctx.Count()
}
