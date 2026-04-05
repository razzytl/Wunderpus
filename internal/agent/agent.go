package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/constants"
	"github.com/wunderpus/wunderpus/internal/logging"
	"github.com/wunderpus/wunderpus/internal/memory"
	"github.com/wunderpus/wunderpus/internal/provider"
	"github.com/wunderpus/wunderpus/internal/security"
	"github.com/wunderpus/wunderpus/internal/skills"
	"github.com/wunderpus/wunderpus/internal/tool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Agent is the core agent that processes user messages.
type Agent struct {
	router        *provider.Router
	sanitizer     *security.Sanitizer
	audit         *security.AuditLogger
	ctx           *ContextManager
	registry      *tool.Registry
	executor      *tool.Executor
	skillsLoader  *skills.SkillsLoader
	sysPrompt     string
	temp          float64
	toolFunc      func(name string, args map[string]any, result string) // optional callback for TUI
	limiter       *security.RateLimiter
	sessionID     string
	encryptionKey []byte
	getSOPsFunc   func(ctx context.Context, task string, topK int) ([]string, error) // For RAG
}

// NewAgent creates a new agent instance.
func NewAgent(
	router *provider.Router,
	sanitizer *security.Sanitizer,
	audit *security.AuditLogger,
	store *memory.Store,
	registry *tool.Registry,
	executor *tool.Executor,
	skillsLoader *skills.SkillsLoader,
	systemPrompt string,
	maxContextTokens int,
	temperature float64,
	sessionID string,
) *Agent {
	// Inject configured core built-in skills if loader is provided
	if skillsLoader != nil {
		coreSkills := []string{"agents", "tools", "identity", "soul", "user"}
		skillsContext := skillsLoader.LoadSkillsForContext(coreSkills)
		if skillsContext != "" {
			systemPrompt = systemPrompt + "\n\n" + skillsContext
		}
	}

	return &Agent{
		router:       router,
		sanitizer:    sanitizer,
		audit:        audit,
		registry:     registry,
		executor:     executor,
		skillsLoader: skillsLoader,
		ctx:          NewContextManager(maxContextTokens, store, sessionID, nil), // defaults to no encryption unless SetEncryptionKey called
		sysPrompt:    systemPrompt,
		temp:         temperature,
		sessionID:    sessionID,
	}
}

// SetRateLimiter sets the rate limiter for the agent.
func (a *Agent) SetRateLimiter(rl *security.RateLimiter) {
	a.limiter = rl
}

// SetToolCallback sets a function to be called when a tool executes (useful for TUI).
func (a *Agent) SetToolCallback(fn func(name string, args map[string]any, result string)) {
	a.toolFunc = fn
}

// SetEncryptionKey sets the optional key for encrypting messages at rest.
// The key should be a base64-encoded 32-byte (256-bit) key.
func (a *Agent) SetEncryptionKey(key string) {
	if key != "" {
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil || len(decoded) != 32 {
			slog.Warn("invalid encryption key - must be 32 bytes (base64 encoded)", "error", err)
			return
		}
		a.encryptionKey = decoded
	}
}

// SetSOPGetter sets the function to retrieve relevant SOPs for RAG.
func (a *Agent) SetSOPGetter(fn func(ctx context.Context, task string, topK int) ([]string, error)) {
	a.getSOPsFunc = fn
}

// HandleMessage processes a user message and returns the agent response.
func (a *Agent) HandleMessage(ctx context.Context, input string) (string, error) {
	ctx, span := otel.Tracer("agent").Start(ctx, "agent.handle_message")
	defer span.End()
	span.SetAttributes(
		attribute.String("session_id", a.sessionID),
		attribute.Int("input_length", len(input)),
	)

	// 0. Rate Limit
	if a.limiter != nil && !a.limiter.Allow(a.sessionID) {
		return "⚠️  Rate limit exceeded. Please wait a moment before sending more messages.", nil
	}

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

	// 3. Loop for tool execution (up to MaxIterations iterations)
	for i := 0; i < constants.MaxIterations; i++ {
		loopCtx, loopSpan := otel.Tracer("agent").Start(ctx, "agent.loop_iteration")
		loopSpan.SetAttributes(attribute.Int("iteration.count", i+1))

		messages := a.buildMessages()
		prov := a.router.Active()
		loopSpan.SetAttributes(attribute.String("provider.name", prov.Name()))
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

		// Check cache
		if cached, ok := a.router.Cache.Get(req); ok {
			logging.L(loopCtx).Info("cache hit", "provider", prov.Name())
			resp := cached
			// Process tool calls as usual if cached
			if len(resp.ToolCalls) == 0 {
				a.ctx.AddMessage(provider.RoleAssistant, resp.Content)
				loopSpan.End()
				return resp.Content, nil
			}
			// ... continue to tool execution if tool calls are in cache
		}

		resp, err := a.router.CompleteWithFallback(loopCtx, req)
		if err != nil {
			loopSpan.SetStatus(codes.Error, err.Error())
			loopSpan.SetAttributes(attribute.String("error.message", err.Error()))
			loopSpan.End()
			return "", err
		}

		// Store in cache
		a.router.Cache.Set(req, resp)

		// If no tools were called, this is the final response
		if len(resp.ToolCalls) == 0 {
			a.ctx.AddMessage(provider.RoleAssistant, resp.Content)
			a.audit.Log(security.AuditEvent{
				Timestamp:   time.Now(),
				Action:      "agent_response",
				Result:      resp.Content,
				ThreatLevel: "none",
			})
			logging.MessagesProcessed.WithLabelValues("unknown", prov.Name()).Inc()
			loopSpan.End()
			return resp.Content, nil
		}

		// Add assistant's tool-call request to context
		a.ctx.AddToolCallMessage(resp.Content, resp.ToolCalls)

		// Execute tools in parallel
		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make([]struct {
			id     string
			output string
		}, len(resp.ToolCalls))

		for idx, tc := range resp.ToolCalls {
			wg.Add(1)
			go func(i int, t provider.ToolCallInfo) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						logging.L(loopCtx).Error("PANIC in tool execution", "tool", t.Function.Name, "panic", r)
						mu.Lock()
						results[i] = struct {
							id     string
							output string
						}{id: t.ID, output: "Error: Tool execution panicked"}
						mu.Unlock()
					}
				}()

				args := make(map[string]any)
				if err := json.Unmarshal([]byte(t.Function.Arguments), &args); err != nil {
					mu.Lock()
					results[i] = struct {
						id     string
						output string
					}{id: t.ID, output: fmt.Sprintf("Error: failed to parse tool arguments: %v", err)}
					mu.Unlock()
					return
				}

				toolCall := tool.ToolCall{
					ID:   t.ID,
					Name: t.Function.Name,
					Args: args,
				}

				start := time.Now()
				res := a.executor.Execute(loopCtx, toolCall)
				logging.ToolExecutionTime.WithLabelValues(t.Function.Name).Observe(time.Since(start).Seconds())

				outputStr := res.Output
				if res.Error != "" {
					outputStr = "Error: " + res.Error
				}

				mu.Lock()
				results[i] = struct {
					id     string
					output string
				}{id: t.ID, output: outputStr}
				mu.Unlock()

				// Fire callback for TUI
				if a.toolFunc != nil {
					a.toolFunc(t.Function.Name, args, outputStr)
				}
			}(idx, tc)
		}
		wg.Wait()

		// Add tool results to context in order
		for _, res := range results {
			a.ctx.AddToolResultMessage(res.id, res.output)
		}

		loopSpan.SetAttributes(attribute.Int("tool_call_count", len(resp.ToolCalls)))

		// Check if summarization is needed
		if a.ctx.NeedsSummarization() {
			a.triggerSummarization(loopCtx)
		}

		loopSpan.End()
	}

	return "", fmt.Errorf("agent reached maximum tool call iterations (%d)", constants.MaxIterations)
}

// HandleComplexTask engages Phase 4 Multi-Agent architecture.
// It bypasses the simple agent loop, creates a TaskGraph, and runs it through the Orchestrator.
func (a *Agent) triggerSummarization(ctx context.Context) {
	slog.Info("context pressure detected, triggering summarization")
	messages := a.ctx.GetMessages()
	if len(messages) < 4 {
		return
	}

	prompt := "Please provide a concise summary of the key points in this conversation so far, focusing on facts and decisions made. Keep it under 200 words."
	summaryMsgs := append(messages, provider.Message{Role: provider.RoleUser, Content: prompt})

	req := &provider.CompletionRequest{
		Messages:    summaryMsgs,
		Temperature: 0.3,
	}

	resp, err := a.router.Active().Complete(ctx, req)
	if err != nil {
		slog.Warn("summarization failed", "error", err)
		return
	}

	a.ctx.SummarizeOldest(resp.Content)
}

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
	a.ctx.AddMessage(provider.RoleUser, "Deep Task: "+input)
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
	systemPrompt := a.sysPrompt

	// Inject relevant SOPs if RAG is enabled
	if a.getSOPsFunc != nil {
		ctx := context.Background()
		// Get the last user message to find relevant SOPs
		userMsgs := a.ctx.GetMessages()
		var lastUserInput string
		for i := len(userMsgs) - 1; i >= 0; i-- {
			if userMsgs[i].Role == provider.RoleUser {
				lastUserInput = userMsgs[i].Content
				break
			}
		}

		if lastUserInput != "" {
			sops, err := a.getSOPsFunc(ctx, lastUserInput, 3)
			if err == nil && len(sops) > 0 {
				sopContext := "\n\n## Relevant Past Procedures (SOPs):\n"
				for _, sop := range sops {
					sopContext += sop + "\n---\n"
				}
				sopContext += "Use these past procedures when relevant to the user's request."
				systemPrompt += sopContext
			}
		}
	}

	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: systemPrompt},
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

// SetBranch switches the active conversation branch and reloads context.
func (a *Agent) SetBranch(branchID string) {
	a.ctx.SetBranch(branchID, nil)
	// Reload in-memory messages from the new branch
	if a.ctx.store != nil && a.ctx.sessionID != "" {
		msgs, err := a.ctx.store.LoadSessionBranch(a.ctx.sessionID, branchID, a.encryptionKey)
		if err == nil && len(msgs) > 0 {
			a.ctx.SetMessages(msgs)
		}
	}
}

// MessageCount returns the number of messages in context.
func (a *Agent) MessageCount() int {
	return a.ctx.Count()
}
