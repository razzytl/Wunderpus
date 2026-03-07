package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Ollama implements the Provider interface for local Ollama inference.
type Ollama struct {
	host         string
	model        string
	maxTok       int
	providerName string
	client       *http.Client
}

// NewOllama creates a new Ollama provider.
func NewOllama(host, model string, maxTokens int) *Ollama {
	return &Ollama{
		host:         host,
		model:        model,
		maxTok:       maxTokens,
		providerName: "ollama",
		client:       &http.Client{},
	}
}

// NewOllamaCompatible creates an Ollama provider with a custom name.
func NewOllamaCompatible(host, model string, maxTokens int, name string) *Ollama {
	if name == "" {
		name = "ollama"
	}
	return &Ollama{
		host:         host,
		model:        model,
		maxTok:       maxTokens,
		providerName: name,
		client:       &http.Client{},
	}
}

func (o *Ollama) Name() string { return o.providerName }

func (o *Ollama) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}

	body := map[string]any{
		"model":    model,
		"messages": toOllamaMessages(req.Messages),
		"stream":   false,
		"options": map[string]any{
			"num_predict": o.maxTok,
		},
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools // Ollama supports OpenAI-compatible tool schemas directly
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.host+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama: API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		DoneReason string `json:"done_reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama: decode response: %w", err)
	}

	var toolCalls []ToolCallInfo
	for i, tc := range result.Message.ToolCalls {
		argsJson, _ := json.Marshal(tc.Function.Arguments)
		toolCalls = append(toolCalls, ToolCallInfo{
			ID:   fmt.Sprintf("call_%d", i), // Ollama doesn't always provide IDs
			Type: "function",
			Function: ToolCallFunc{
				Name:      tc.Function.Name,
				Arguments: string(argsJson),
			},
		})
	}

	return &CompletionResponse{
		Content:      result.Message.Content,
		Model:        model,
		FinishReason: result.DoneReason,
		ToolCalls:    toolCalls,
	}, nil
}

func (o *Ollama) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}

	body := map[string]any{
		"model":    model,
		"messages": toOllamaMessages(req.Messages),
		"stream":   true,
		"options": map[string]any{
			"num_predict": o.maxTok,
		},
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.host+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama: API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			var chunk struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Done bool `json:"done"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
				continue
			}
			if chunk.Message.Content != "" {
				ch <- StreamChunk{Content: chunk.Message.Content}
			}
			if chunk.Done {
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

func toOllamaMessages(msgs []Message) []map[string]any {
	out := make([]map[string]any, len(msgs))
	for i, m := range msgs {
		msg := map[string]any{"role": m.Role, "content": m.Content}
		
		if len(m.ToolCalls) > 0 {
			var ollamaToolCalls []map[string]any
			for _, tc := range m.ToolCalls {
				var arguments map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &arguments)
				
				ollamaToolCalls = append(ollamaToolCalls, map[string]any{
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": arguments,
					},
				})
			}
			msg["tool_calls"] = ollamaToolCalls
		}
		
		out[i] = msg
	}
	return out
}
