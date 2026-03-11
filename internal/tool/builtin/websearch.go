package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/wunderpus/wunderpus/internal/tool"
)

const (
	searchTimeout = 10 * time.Second
	userAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
)

var (
	reScript     = regexp.MustCompile(`<script[\s\S]*?</script>`)
	reStyle      = regexp.MustCompile(`<style[\s\S]*?</style>`)
	reTags       = regexp.MustCompile(`<[^>]+>`)
	reWhitespace = regexp.MustCompile(`[^\S\n]+`)
	reBlankLines = regexp.MustCompile(`\n{3,}`)
	reDDGLink    = regexp.MustCompile(`<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	reDDGSnippet = regexp.MustCompile(`<a class="result__snippet[^"]*".*?>([\s\S]*?)</a>`)
)

type SearchProvider interface {
	Search(ctx context.Context, query string, count int) (string, error)
}

type BraveSearchProvider struct {
	apiKey string
	client *http.Client
}

func (p *BraveSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), count)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("brave api error (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResp struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	results := searchResp.Web.Results
	if len(results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s", query))
	for i, item := range results {
		if i >= count {
			break
		}
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, item.Title, item.URL))
		if item.Description != "" {
			lines = append(lines, fmt.Sprintf("   %s", item.Description))
		}
	}

	return strings.Join(lines, "\n"), nil
}

type TavilySearchProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func (p *TavilySearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := p.baseURL
	if searchURL == "" {
		searchURL = "https://api.tavily.com/search"
	}

	payload := map[string]any{
		"api_key":             p.apiKey,
		"query":               query,
		"search_depth":        "advanced",
		"include_answer":      false,
		"include_images":      false,
		"include_raw_content": false,
		"max_results":         count,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tavily api error (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResp struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	results := searchResp.Results
	if len(results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via Tavily)", query))
	for i, item := range results {
		if i >= count {
			break
		}
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, item.Title, item.URL))
		if item.Content != "" {
			lines = append(lines, fmt.Sprintf("   %s", item.Content))
		}
	}

	return strings.Join(lines, "\n"), nil
}

type DuckDuckGoSearchProvider struct {
	client *http.Client
}

func (p *DuckDuckGoSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return p.extractResults(string(body), count, query)
}

func (p *DuckDuckGoSearchProvider) extractResults(html string, count int, query string) (string, error) {
	matches := reDDGLink.FindAllStringSubmatch(html, count+5)

	if len(matches) == 0 {
		return fmt.Sprintf("No results found or extraction failed. Query: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via DuckDuckGo)", query))

	snippetMatches := reDDGSnippet.FindAllStringSubmatch(html, count+5)

	maxItems := min(len(matches), count)

	for i := 0; i < maxItems; i++ {
		urlStr := matches[i][1]
		title := stripTags(matches[i][2])
		title = strings.TrimSpace(title)

		if strings.Contains(urlStr, "uddg=") {
			if u, err := url.QueryUnescape(urlStr); err == nil {
				_, after, ok := strings.Cut(u, "uddg=")
				if ok {
					urlStr = after
				}
			}
		}

		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, title, urlStr))

		if i < len(snippetMatches) {
			snippet := stripTags(snippetMatches[i][1])
			snippet = strings.TrimSpace(snippet)
			if snippet != "" {
				lines = append(lines, fmt.Sprintf("   %s", snippet))
			}
		}
	}

	return strings.Join(lines, "\n"), nil
}

func stripTags(content string) string {
	return reTags.ReplaceAllString(content, "")
}

type WebSearch struct {
	provider   SearchProvider
	maxResults int
}

type WebSearchConfig struct {
	BraveAPIKey   string
	TavilyAPIKey  string
	TavilyBaseURL string
	PerplexityKey string
	Proxy         string
}

func NewWebSearch(cfg WebSearchConfig) (*WebSearch, error) {
	var provider SearchProvider
	maxResults := 5

	client := &http.Client{Timeout: searchTimeout}

	if cfg.BraveAPIKey != "" {
		provider = &BraveSearchProvider{apiKey: cfg.BraveAPIKey, client: client}
	} else if cfg.TavilyAPIKey != "" {
		baseURL := cfg.TavilyBaseURL
		if baseURL == "" {
			baseURL = "https://api.tavily.com/search"
		}
		provider = &TavilySearchProvider{apiKey: cfg.TavilyAPIKey, baseURL: baseURL, client: client}
	} else {
		provider = &DuckDuckGoSearchProvider{client: client}
	}

	return &WebSearch{
		provider:   provider,
		maxResults: maxResults,
	}, nil
}

func (s *WebSearch) Name() string { return "web_search" }
func (s *WebSearch) Description() string {
	return "Search the web for current information. Returns titles, URLs, and snippets from search results."
}
func (s *WebSearch) Sensitive() bool        { return false }
func (s *WebSearch) Version() string        { return "1.0.0" }
func (s *WebSearch) Dependencies() []string { return nil }
func (s *WebSearch) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "query", Type: "string", Description: "Search query", Required: true},
		{Name: "count", Type: "number", Description: "Number of results (1-10)", Required: false},
	}
}

func (s *WebSearch) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return &tool.Result{Error: "query is required"}, nil
	}

	count := s.maxResults
	if c, ok := args["count"].(float64); ok {
		if int(c) > 0 && int(c) <= 10 {
			count = int(c)
		}
	}

	result, err := s.provider.Search(ctx, query, count)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("search failed: %v", err)}, nil
	}

	return &tool.Result{Output: result}, nil
}

type WebFetch struct {
	maxChars int
	client   *http.Client
}

func NewWebFetch(proxy string) (*WebFetch, error) {
	client := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			return nil
		},
	}

	return &WebFetch{
		maxChars: 50000,
		client:   client,
	}, nil
}

func (f *WebFetch) Name() string { return "web_fetch" }
func (f *WebFetch) Description() string {
	return "Fetch a URL and extract readable content (HTML to text)."
}
func (f *WebFetch) Sensitive() bool        { return false }
func (f *WebFetch) Version() string        { return "1.0.0" }
func (f *WebFetch) Dependencies() []string { return nil }
func (f *WebFetch) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "url", Type: "string", Description: "URL to fetch", Required: true},
		{Name: "maxChars", Type: "number", Description: "Maximum characters to extract", Required: false},
	}
}

func (f *WebFetch) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	urlStr, _ := args["url"].(string)
	if urlStr == "" {
		return &tool.Result{Error: "url is required"}, nil
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("invalid URL: %v", err)}, nil
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &tool.Result{Error: "only http/https URLs are allowed"}, nil
	}

	if parsedURL.Host == "" {
		return &tool.Result{Error: "missing domain in URL"}, nil
	}

	maxChars := f.maxChars
	if mc, ok := args["maxChars"].(float64); ok {
		if int(mc) > 100 {
			maxChars = int(mc)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("failed to create request: %v", err)}, nil
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("request failed: %v", err)}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("failed to read response: %v", err)}, nil
	}

	contentType := resp.Header.Get("Content-Type")
	var text string

	if strings.Contains(contentType, "application/json") {
		var jsonData any
		if err := json.Unmarshal(body, &jsonData); err == nil {
			formatted, _ := json.MarshalIndent(jsonData, "", "  ")
			text = string(formatted)
		} else {
			text = string(body)
		}
	} else if strings.Contains(contentType, "text/html") || len(body) > 0 &&
		(strings.HasPrefix(string(body), "<!DOCTYPE") || strings.HasPrefix(strings.ToLower(string(body)), "<html")) {
		text = f.extractText(string(body))
	} else {
		text = string(body)
	}

	truncated := len(text) > maxChars
	if truncated {
		text = text[:maxChars]
	}

	result := map[string]any{
		"url":       urlStr,
		"status":    resp.StatusCode,
		"truncated": truncated,
		"length":    len(text),
		"text":      text,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &tool.Result{Output: string(resultJSON)}, nil
}

func (f *WebFetch) extractText(htmlContent string) string {
	result := reScript.ReplaceAllLiteralString(htmlContent, "")
	result = reStyle.ReplaceAllLiteralString(result, "")
	result = reTags.ReplaceAllLiteralString(result, "")
	result = strings.TrimSpace(result)
	result = reWhitespace.ReplaceAllString(result, " ")
	result = reBlankLines.ReplaceAllString(result, "\n\n")

	lines := strings.Split(result, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}
