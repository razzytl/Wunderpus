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

// Gemini implements the Provider interface for Google's Gemini API.
type Gemini struct {
	apiKey string
	model  string
	maxTok int
	client *http.Client
}

// NewGemini creates a new Gemini provider.
func NewGemini(apiKey, model string, maxTokens int) *Gemini {
	return &Gemini{
		apiKey: apiKey,
		model:  model,
		maxTok: maxTokens,
		client: &http.Client{},
	}
}

func (g *Gemini) Name() string { return "gemini" }

func (g *Gemini) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = g.model
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, g.apiKey,
	)

	body := buildGeminiBody(req.Messages, g.maxTok)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gemini: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini: API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("gemini: decode response: %w", err)
	}

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("gemini: no candidates returned")
	}

	var content string
	for _, p := range result.Candidates[0].Content.Parts {
		content += p.Text
	}

	return &CompletionResponse{
		Content:      content,
		Model:        model,
		FinishReason: result.Candidates[0].FinishReason,
		PromptTokens: result.UsageMetadata.PromptTokenCount,
		CompTokens:   result.UsageMetadata.CandidatesTokenCount,
	}, nil
}

func (g *Gemini) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = g.model
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s",
		model, g.apiKey,
	)

	body := buildGeminiBody(req.Messages, g.maxTok)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gemini: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("gemini: API error %d: %s", resp.StatusCode, string(bodyBytes))
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

			var chunk struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Candidates) > 0 {
				for _, p := range chunk.Candidates[0].Content.Parts {
					if p.Text != "" {
						ch <- StreamChunk{Content: p.Text}
					}
				}
			}
		}
		ch <- StreamChunk{Done: true}
		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

func buildGeminiBody(msgs []Message, maxTokens int) map[string]any {
	var contents []map[string]any
	var systemInstruction string

	for _, m := range msgs {
		if m.Role == RoleSystem {
			systemInstruction = m.Content
			continue
		}
		role := "user"
		if m.Role == RoleAssistant {
			role = "model"
		}
		contents = append(contents, map[string]any{
			"role": role,
			"parts": []map[string]string{
				{"text": m.Content},
			},
		})
	}

	body := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"maxOutputTokens": maxTokens,
		},
	}

	if systemInstruction != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{"text": systemInstruction},
			},
		}
	}

	return body
}
