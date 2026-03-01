package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/wonderpus/wonderpus/internal/memory"
	"github.com/wonderpus/wonderpus/internal/provider"
	"github.com/wonderpus/wonderpus/internal/security"
	"github.com/wonderpus/wonderpus/internal/tool"
)

// Agent is the core agent that processes user messages.
type Agent struct {
	router    *provider.Router
	sanitizer *security.Sanitizer
	audit     *security.AuditLogger
	ctx       *ContextManager
	registry  *tool.Registry
	executor  *tool.Executor
	sysPrompt string
	temp      float64
	toolFunc  func(name string, args map[string]any, result string) // optional callback for TUI
}

// NewAgent creates a new agent instance.
func NewAgent(
	router *provider.Router,
	sanitizer *security.Sanitizer,
	audit *security.AuditLogger,
	store *memory.Store,
	registry *tool.Registry,
	executor *tool.Executor,
	systemPrompt string,
	maxContextTokens int,
	temperature float64,
	sessionID string,
) *Agent {
	return &Agent{
		router:    router,
		sanitizer: sanitizer,
		audit:     audit,
		registry:  registry,
		executor:  executor,
		ctx:       NewContextManager(maxContextTokens, store, sessionID),
		sysPrompt: systemPrompt,
		temp:      temperature,
	}
}

// SetToolCallback sets a function to be called when a tool executes (useful for TUI).
func (a *Agent) SetToolCallback(fn func(name string, args map[string]any, result string)) {
	a.toolFunc = fn
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

	// 3. Loop for tool execution (up to 5 iterations)
	maxIterations := 5
	for i := 0; i < maxIterations; i++ {
		messages := a.buildMessages()
		prov := a.router.Active()
		req := &provider.CompletionRequest{
			Messages:    messages,
			Temperature: a.temp,
		}

		if a.registry != nil && a.registry.Count() > 0 {
			toolSchemas := a.registry.Schemas()
			req.Tools = make([]provider.ToolSchema, len(toolSchemas))
			for j, ts := range toolSchemas {
				req.Tools[j] = provider.ToolSchema{
					Type: ts.Type,
					Function: provider.FunctionSchema{
						Name:        ts.Function.Name,
						Description: ts.Function.Description,
						Parameters:  ts.Function.Parameters,
					},
				}
			}
		}

		resp, err := prov.Complete(ctx, req)
		if err != nil {
			return "", fmt.Errorf("provider %s: %w", prov.Name(), err)
		}

		// If no tools were called, this is the final response
		if len(resp.ToolCalls) == 0 {
			a.ctx.AddMessage(provider.RoleAssistant, resp.Content)
			a.audit.Log(security.AuditEvent{
				Timestamp:   time.Now(),
				Action:      "agent_response",
				Result:      resp.Content,
				ThreatLevel: "none",
			})
			return resp.Content, nil
		}

		// Add assistant's tool-call request to context
		a.ctx.AddToolCallMessage(resp.Content, resp.ToolCalls)

		// Execute tools
		for _, tc := range resp.ToolCalls {
			var args map[string]any
			// fallback if args aren't unmarshalable, though provider mostly handles this
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args) 

			toolCall := tool.ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: args,
			}

			// execute tool
			res := a.executor.Execute(ctx, toolCall)

			outputStr := res.Output
			if res.Error != "" {
				outputStr = "Error: " + res.Error
			}

			// Fire callback for TUI
			if a.toolFunc != nil {
				a.toolFunc(tc.Function.Name, args, outputStr)
			}

			// Add tool result to context
			a.ctx.AddToolResultMessage(tc.ID, outputStr)
		}
	}

	return "", fmt.Errorf("agent reached maximum tool call iterations (%d)", maxIterations)
}

// HandleComplexTask engages Phase 4 Multi-Agent architecture.
// It bypasses the simple agent loop, creates a TaskGraph, and runs it through the Orchestrator.
func (a *Agent) HandleComplexTask(ctx context.Context, input string) (string, error) {
	a.audit.Log(security.AuditEvent{
		Timestamp:   time.Now(),
		Action:      "complex_task_start",
		Input:       input,
		ThreatLevel: "none",
	})

	// 1. Planner decomposes the task
	planner := NewTaskPlanner(a.router.Active(), a.router.ActiveName()) // Use current model
	
	// Print a little callback letting TUI know we are planning
	if a.toolFunc != nil {
		a.toolFunc("system:planner", map[string]any{"goal": input}, "Decomposing task into dependency graph...")
	}
	
	graph, err := planner.Decompose(ctx, input)
	if err != nil {
		return "", fmt.Errorf("planner failed: %w", err)
	}

	if a.toolFunc != nil {
		a.toolFunc("system:orchestrator", map[string]any{"subtasks": len(graph.Subtasks)}, "Graph resolved. Launching worker arms...")
	}

	// 2. Orchestrator executes graph
	orchestator := NewOrchestrator(a.router, a.registry, a.executor, a.router.ActiveName())
	
	finalRes, err := orchestator.Execute(ctx, graph)
	if err != nil {
		return "", fmt.Errorf("orchestrator failed: %w", err)
	}

	// Add final result to main agent context so it remembers the synthesis
	a.ctx.AddMessage(provider.RoleUser, "Deep Task: " + input)
	a.ctx.AddMessage(provider.RoleAssistant, finalRes)

	a.audit.Log(security.AuditEvent{
		Timestamp:   time.Now(),
		Action:      "complex_task_end",
		Result:      finalRes,
		ThreatLevel: "none",
	})

	return finalRes, nil
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
	// Currently, if the agent decides to use a tool, many models don't stream the tool call nicely,
	// or the logic fundamentally switches from streaming back to the user vs executing.
	// For Phase 2, if tools are enabled, we do a synchronous Complete loop until the final text response, THEN stream?
	// Actually, doing full multi-step streaming is complex. M37: For simplicity, if streaming, we don't expose tools
	// OR we just execute tools synchronously and then start streaming. 
	// For Phase 2 let's enable tools but if it returns a tool call during stream, we abort streaming, execute, and stream the next step.
	// We'll leave `StreamMessage` for simple text for now, or adapt it inside.
	
	prov := a.router.Active()
	req := &provider.CompletionRequest{
		Messages:    messages,
		Temperature: a.temp,
	}

	if a.registry != nil && a.registry.Count() > 0 {
		toolSchemas := a.registry.Schemas()
		req.Tools = make([]provider.ToolSchema, len(toolSchemas))
		for j, ts := range toolSchemas {
			req.Tools[j] = provider.ToolSchema{
				Type: ts.Type,
				Function: provider.FunctionSchema{
					Name:        ts.Function.Name,
					Description: ts.Function.Description,
					Parameters:  ts.Function.Parameters,
				},
			}
		}
	}

	streamCh, err := prov.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", prov.Name(), err)
	}

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
				// Note: streaming tool calls isn't fully implemented in this MVP StreamMessage.
				// If the model tried to stream a tool call, the content might be empty or partial JSON.
				// Production implementation requires intercepting chunks, buffering JSON, executing, and recursing.
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
