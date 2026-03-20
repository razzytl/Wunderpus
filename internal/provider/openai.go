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

// OpenAI implements the Provider interface for OpenAI-compatible APIs.
// This includes OpenAI, OpenRouter, Groq, vLLM, Together, Mistral, etc.
type OpenAI struct {
	apiKey       string
	model        string
	maxTok       int
	baseURL      string // Configurable API base (e.g. "https://api.openai.com/v1")
	providerName string // Custom name for this provider instance
	client       *http.Client
}

// NewOpenAI creates a new OpenAI-compatible provider.
func NewOpenAI(apiKey, model string, maxTokens int) *OpenAI {
	return &OpenAI{
		apiKey:       apiKey,
		model:        model,
		maxTok:       maxTokens,
		baseURL:      "https://api.openai.com/v1",
		providerName: "openai",
		client:       DefaultClient,
	}
}

// NewOpenAICompatible creates a provider for any OpenAI-compatible API endpoint.
func NewOpenAICompatible(apiKey, model string, maxTokens int, baseURL, name string) *OpenAI {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if name == "" {
		name = "openai"
	}
	// Ensure baseURL doesn't end with trailing slash
	baseURL = strings.TrimRight(baseURL, "/")
	return &OpenAI{
		apiKey:       apiKey,
		model:        model,
		maxTok:       maxTokens,
		baseURL:      baseURL,
		providerName: name,
		client:       DefaultClient,
	}
}

func (o *OpenAI) Name() string { return o.providerName }

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
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
		if req.ToolChoice != nil {
			body["tool_choice"] = req.ToolChoice
		}
	}

	httpReq, err := o.createRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	resp, err := RetryDo(ctx, o.client, httpReq, DefaultRetryOptions)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content   string         `json:"content"`
				ToolCalls []ToolCallInfo `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
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
		ToolCalls:    result.Choices[0].Message.ToolCalls,
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
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
		if req.ToolChoice != nil {
			body["tool_choice"] = req.ToolChoice
		}
	}

	httpReq, err := o.createRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	// Response body is closed by the goroutine below (defer resp.Body.Close())
	//nolint:bodyclose
	resp, err := RetryDo(ctx, o.client, httpReq, DefaultRetryOptions) // Stream still uses standard request, but RetryDo might need adjustment for live streams. For now, we only retry the connection.
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

func (o *OpenAI) createRequest(ctx context.Context, body map[string]any) (*http.Request, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, errors.Wrap(errors.InternalError, "marshal openai request", err)
	}

	url := o.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(errors.InternalError, "create openai request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	return httpReq, nil
}

func toOpenAIMessages(msgs []Message) []map[string]any {
	out := make([]map[string]any, len(msgs))
	for i, m := range msgs {
		var content any
		if len(m.MultiContent) > 0 {
			parts := make([]map[string]any, len(m.MultiContent))
			for j, p := range m.MultiContent {
				part := map[string]any{"type": p.Type}
				if p.Type == "text" {
					part["text"] = p.Text
				} else if p.Type == "image_url" && p.ImageURL != nil {
					part["image_url"] = map[string]any{
						"url":    p.ImageURL.URL,
						"detail": p.ImageURL.Detail,
					}
				}
				parts[j] = part
			}
			content = parts
		} else {
			content = m.Content
		}

		msg := map[string]any{"role": m.Role, "content": content}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			msg["tool_calls"] = m.ToolCalls
		}
		out[i] = msg
	}
	return out
}

// EmbeddingModel is the default model for text embeddings.
const EmbeddingModel = "text-embedding-3-small"

// Embed generates embeddings for the given texts using OpenAI's embedding API.
func (o *OpenAI) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body := map[string]any{
		"model": EmbeddingModel,
		"input": texts,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, errors.Wrap(errors.InternalError, "marshal embedding request", err)
	}

	url := o.baseURL + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(errors.InternalError, "create embedding request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := RetryDo(ctx, o.client, httpReq, DefaultRetryOptions)
	if err != nil {
		return nil, errors.Wrap(errors.InternalError, "embedding request failed", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(errors.InternalError, "decode embedding response", err)
	}

	embeddings := make([][]float64, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}

	return embeddings, nil
}

// EmbedSingle generates embedding for a single text.
func (o *OpenAI) EmbedSingle(ctx context.Context, text string) ([]float64, error) {
	embs, err := o.Embed(ctx, []string{text})
	if err != nil || len(embs) == 0 {
		return nil, err
	}
	return embs[0], nil
}

// Dimension returns the embedding dimension for the default model.
func (o *OpenAI) Dimension() int {
	return 1536 // text-embedding-3-small
}
