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

	resp, err := a.doRequest(ctx, body, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
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
	for _, c := range result.Content {
		content += c.Text
	}

	return &CompletionResponse{
		Content:      content,
		Model:        model,
		FinishReason: result.StopReason,
		PromptTokens: result.Usage.InputTokens,
		CompTokens:   result.Usage.OutputTokens,
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

func toAnthropicMessages(msgs []Message) []map[string]string {
	out := make([]map[string]string, len(msgs))
	for i, m := range msgs {
		out[i] = map[string]string{"role": m.Role, "content": m.Content}
	}
	return out
}
