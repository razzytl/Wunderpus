package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/wunderpus/wunderpus/internal/errors"
)

// Anthropic implements the Provider interface for Anthropic's API.
type Anthropic struct {
	apiKey       string
	model        string
	maxTok       int
	baseURL      string
	providerName string
	client       *http.Client
}

// NewAnthropic creates a new Anthropic provider.
func NewAnthropic(apiKey, model string, maxTokens int) *Anthropic {
	return &Anthropic{
		apiKey:       apiKey,
		model:        model,
		maxTok:       maxTokens,
		baseURL:      "https://api.anthropic.com",
		providerName: "anthropic",
		client:       DefaultClient,
	}
}

// NewAnthropicCompatible creates an Anthropic provider with a custom base URL.
func NewAnthropicCompatible(apiKey, model string, maxTokens int, baseURL, name string) *Anthropic {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	if name == "" {
		name = "anthropic"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Anthropic{
		apiKey:       apiKey,
		model:        model,
		maxTok:       maxTokens,
		baseURL:      baseURL,
		providerName: name,
		client:       DefaultClient,
	}
}

func (a *Anthropic) validate() error {
	if a.apiKey == "" {
		return errors.New(errors.ProviderError, "anthropic: API key is missing")
	}
	// Only enforce sk-ant- prefix for the default Anthropic endpoint
	if a.baseURL == "https://api.anthropic.com" && !strings.HasPrefix(a.apiKey, "sk-ant-") {
		return errors.New(errors.ProviderError, "anthropic: invalid API key format (expected sk-ant-...)")
	}
	return nil
}

func (a *Anthropic) Name() string { return a.providerName }

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

	httpReq, err := a.createRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	resp, err := RetryDo(ctx, a.client, httpReq, DefaultRetryOptions)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Wrap body in size limit
	resp.Body = LimitResponseReader(resp.Body)

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

	httpReq, err := a.createRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	// Response body is closed by the goroutine below (defer resp.Body.Close())
	//nolint:bodyclose
	resp, err := RetryDo(ctx, a.client, httpReq, DefaultRetryOptions)
	if err != nil {
		return nil, err
	}

	// Wrap body in size limit
	resp.Body = LimitResponseReader(resp.Body)

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

func (a *Anthropic) createRequest(ctx context.Context, body map[string]any) (*http.Request, error) {
	if err := a.validate(); err != nil {
		return nil, err
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, errors.Wrap(errors.InternalError, "marshal anthropic request", err)
	}

	url := a.baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(errors.InternalError, "create anthropic request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", a.apiKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	return httpReq, nil
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
						"type":        "tool_result",
						"tool_use_id": m.ToolCallID,
						"content":     m.Content,
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

		var content any
		if len(m.MultiContent) > 0 {
			parts := make([]map[string]any, 0, len(m.MultiContent))
			for _, p := range m.MultiContent {
				if p.Type == "text" {
					parts = append(parts, map[string]any{
						"type": "text",
						"text": p.Text,
					})
				} else if p.Type == "image_url" && p.ImageURL != nil {
					// Anthropic needs base64 and media_type for images
					// Expect URL to be "data:image/jpeg;base64,..." or similar
					if strings.HasPrefix(p.ImageURL.URL, "data:") {
						parts = append(parts, parseAnthropicImage(p.ImageURL.URL))
					}
				}
			}
			content = parts
		} else {
			content = m.Content
		}

		out = append(out, map[string]any{"role": m.Role, "content": content})
	}
	return out
}

func parseAnthropicImage(dataURL string) map[string]any {
	// data:image/png;base64,iVBOR...
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) < 2 {
		return nil
	}
	header := parts[0]
	data := parts[1]

	mediaType := "image/jpeg"
	if strings.Contains(header, "image/png") {
		mediaType = "image/png"
	} else if strings.Contains(header, "image/gif") {
		mediaType = "image/gif"
	} else if strings.Contains(header, "image/webp") {
		mediaType = "image/webp"
	}

	return map[string]any{
		"type": "image",
		"source": map[string]any{
			"type":       "base64",
			"media_type": mediaType,
			"data":       data,
		},
	}
}

func toAnthropicTools(tools []ToolSchema) []map[string]any {
	out := make([]map[string]any, len(tools))
	for i, t := range tools {
		var inputSchema map[string]any
		_ = json.Unmarshal(t.Function.Parameters, &inputSchema)
		out[i] = map[string]any{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": inputSchema,
		}
	}
	return out
}
