package builtin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/wonderpus/wonderpus/internal/tool"
)

// HTTPRequest makes HTTP requests with SSRF protection.
type HTTPRequest struct {
	client *http.Client
}

// NewHTTPRequest creates a new HTTP request tool.
func NewHTTPRequest() *HTTPRequest {
	return &HTTPRequest{client: &http.Client{}}
}

func (h *HTTPRequest) Name() string        { return "http_request" }
func (h *HTTPRequest) Description() string  { return "Make an HTTP request to a URL. POST/PUT/DELETE require user approval. Internal network addresses are blocked." }
func (h *HTTPRequest) Sensitive() bool      { return true }
func (h *HTTPRequest) Version() string      { return "1.0.0" }
func (h *HTTPRequest) Dependencies() []string { return nil }
func (h *HTTPRequest) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "url", Type: "string", Description: "The URL to request", Required: true},
		{Name: "method", Type: "string", Description: "HTTP method: GET, POST, PUT, DELETE (default: GET)", Required: false},
		{Name: "body", Type: "string", Description: "Request body for POST/PUT", Required: false},
	}
}

func (h *HTTPRequest) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	rawURL, _ := args["url"].(string)
	method, _ := args["method"].(string)
	body, _ := args["body"].(string)

	if rawURL == "" {
		return &tool.Result{Error: "url is required"}, nil
	}
	if method == "" {
		method = "GET"
	}
	method = strings.ToUpper(method)

	// SSRF protection: block internal IPs
	if blocked, reason := isBlockedURL(rawURL); blocked {
		return &tool.Result{Error: fmt.Sprintf("blocked: %s", reason)}, nil
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("invalid request: %v", err)}, nil
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("request failed: %v", err)}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10000))
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("read error: %v", err)}, nil
	}

	output := fmt.Sprintf("Status: %d %s\n\n%s", resp.StatusCode, resp.Status, string(respBody))

	return &tool.Result{Output: output}, nil
}

// isBlockedURL checks if a URL targets internal/private networks.
func isBlockedURL(rawURL string) (bool, string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true, "invalid URL"
	}

	host := u.Hostname()

	// Block common internal hostnames
	lowerHost := strings.ToLower(host)
	for _, blocked := range []string{"localhost", "metadata.google", "169.254.169.254"} {
		if strings.Contains(lowerHost, blocked) {
			return true, fmt.Sprintf("internal address %q blocked", host)
		}
	}

	// Block private IP ranges
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true, fmt.Sprintf("private/internal IP %s blocked", host)
		}
	}

	return false, ""
}
