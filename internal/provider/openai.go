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

// OpenAI implements the Provider interface for OpenAI's API.
type OpenAI struct {
	apiKey string
	model  string
	maxTok int
	client *http.Client
}

// NewOpenAI creates a new OpenAI provider.
func NewOpenAI(apiKey, model string, maxTokens int) *OpenAI {
	return &OpenAI{
		apiKey: apiKey,
		model:  model,
		maxTok: maxTokens,
		client: &http.Client{},
	}
}

func (o *OpenAI) Name() string { return "openai" }

func (o *OpenAI) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}
	maxTok := req.MaxTokens
	if maxTok == 0 {
		maxTok = o.maxTok
	}

	body := map[string]any{
		"model":       model,
		"messages":    toOpenAIMessages(req.Messages),
		"max_tokens":  maxTok,
		"temperature": req.Temperature,
	}

	resp, err := o.doRequest(ctx, body, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message      struct{ Content string } `json:"message"`
			FinishReason string                   `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openai: decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("openai: empty response")
	}

	return &CompletionResponse{
		Content:      result.Choices[0].Message.Content,
		Model:        model,
		FinishReason: result.Choices[0].FinishReason,
		PromptTokens: result.Usage.PromptTokens,
		CompTokens:   result.Usage.CompletionTokens,
	}, nil
}

func (o *OpenAI) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}
	maxTok := req.MaxTokens
	if maxTok == 0 {
		maxTok = o.maxTok
	}

	body := map[string]any{
		"model":       model,
		"messages":    toOpenAIMessages(req.Messages),
		"max_tokens":  maxTok,
		"temperature": req.Temperature,
		"stream":      true,
	}

	resp, err := o.doRequest(ctx, body, true)
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
			if data == "[DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- StreamChunk{Content: chunk.Choices[0].Delta.Content}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

func (o *OpenAI) doRequest(ctx context.Context, body map[string]any, stream bool) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai: API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

func toOpenAIMessages(msgs []Message) []map[string]string {
	out := make([]map[string]string, len(msgs))
	for i, m := range msgs {
		out[i] = map[string]string{"role": m.Role, "content": m.Content}
	}
	return out
}
