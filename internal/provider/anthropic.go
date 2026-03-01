package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Anthropic implements the Provider interface for Anthropic's API.
type Anthropic struct {
	apiKey string
	model  string
	maxTok int
	client *http.Client
}

// NewAnthropic creates a new Anthropic provider.
func NewAnthropic(apiKey, model string, maxTokens int) *Anthropic {
	return &Anthropic{
		apiKey: apiKey,
		model:  model,
		maxTok: maxTokens,
		client: &http.Client{},
	}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = a.model
	}
	maxTok := req.MaxTokens
	if maxTok == 0 {
		maxTok = a.maxTok
	}

	systemMsg, userMsgs := extractSystem(req.Messages)

	body := map[string]any{
		"model":      model,
		"max_tokens": maxTok,
		"messages":   toAnthropicMessages(userMsgs),
	}
	if systemMsg != "" {
		body["system"] = systemMsg
	}
	if len(req.Tools) > 0 {
		body["tools"] = toAnthropicTools(req.Tools)
	}

	resp, err := a.doRequest(ctx, body, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Input any    `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("anthropic: decode response: %w", err)
	}

	var content string
	var toolCalls []ToolCallInfo
	for _, c := range result.Content {
		if c.Type == "text" {
			content += c.Text
		} else if c.Type == "tool_use" {
			inputParams, _ := json.Marshal(c.Input)
			toolCalls = append(toolCalls, ToolCallInfo{
				ID:   c.ID,
				Type: "function",
				Function: ToolCallFunc{
					Name:      c.Name,
					Arguments: string(inputParams),
				},
			})
		}
	}

	return &CompletionResponse{
		Content:      content,
		Model:        model,
		FinishReason: result.StopReason,
		PromptTokens: result.Usage.InputTokens,
		CompTokens:   result.Usage.OutputTokens,
		ToolCalls:    toolCalls,
	}, nil
}

func (a *Anthropic) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = a.model
	}
	maxTok := req.MaxTokens
	if maxTok == 0 {
		maxTok = a.maxTok
	}

	systemMsg, userMsgs := extractSystem(req.Messages)

	body := map[string]any{
		"model":      model,
		"max_tokens": maxTok,
		"messages":   toAnthropicMessages(userMsgs),
		"stream":     true,
	}
	if systemMsg != "" {
		body["system"] = systemMsg
	}
	if len(req.Tools) > 0 {
		body["tools"] = toAnthropicTools(req.Tools)
	}

	resp, err := a.doRequest(ctx, body, true)
	if err != nil {
		return nil, err
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Text != "" {
					ch <- StreamChunk{Content: event.Delta.Text}
				}
			case "message_stop":
				ch <- StreamChunk{Done: true}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

func (a *Anthropic) doRequest(ctx context.Context, body map[string]any, stream bool) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("anthropic: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic: API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// extractSystem separates the system message from the rest.
func extractSystem(msgs []Message) (string, []Message) {
	var system string
	var rest []Message
	for _, m := range msgs {
		if m.Role == RoleSystem {
			system = m.Content
		} else {
			rest = append(rest, m)
		}
	}
	return system, rest
}

func toAnthropicMessages(msgs []Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleTool || m.ToolCallID != "" {
			// Anthropic uses role "user" with content type "tool_result" for tool responses
			out = append(out, map[string]any{
				"role": "user",
				"content": []map[string]any{
					{
						"type":         "tool_result",
						"tool_use_id":  m.ToolCallID,
						"content":      m.Content,
					},
				},
			})
			continue
		}

		if len(m.ToolCalls) > 0 {
			// Anthropic assistant message with tool calls
			contentBlocks := make([]map[string]any, 0)
			if m.Content != "" {
				contentBlocks = append(contentBlocks, map[string]any{
					"type": "text",
					"text": m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				var input map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
				contentBlocks = append(contentBlocks, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": input,
				})
			}
			out = append(out, map[string]any{
				"role":    "assistant",
				"content": contentBlocks,
			})
			continue
		}

		out = append(out, map[string]any{"role": m.Role, "content": m.Content})
	}
	return out
}

func toAnthropicTools(tools []ToolSchema) []map[string]any {
	out := make([]map[string]any, len(tools))
	for i, t := range tools {
		var inputSchema map[string]any
		_ = json.Unmarshal(t.Function.Parameters, &inputSchema)
		out[i] = map[string]any{
			"name":        t.Function.Name,
			"description": t.Function.Description,
			"input_schema": inputSchema,
		}
	}
	return out
}
