package builtin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/wunderpus/wunderpus/internal/tool"
)

// HTTPRequest makes HTTP requests with SSRF protection.
type HTTPRequest struct {
	client    *http.Client
	blocklist []string
}

// NewHTTPRequest creates a new HTTP request tool with an optional custom blocklist.
func NewHTTPRequest(blocklist []string) *HTTPRequest {
	return &HTTPRequest{
		client:    &http.Client{},
		blocklist: blocklist,
	}
}

func (h *HTTPRequest) Name() string        { return "http_request" }
func (h *HTTPRequest) Description() string  { return "Make an HTTP request to a URL. POST/PUT/DELETE require user approval. Internal network addresses and metadata endpoints are strictly blocked to prevent SSRF." }
func (h *HTTPRequest) Sensitive() bool      { return true }
func (h *HTTPRequest) Version() string      { return "1.1.0" }
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

	// SSRF protection
	if blocked, reason := h.isBlockedURL(rawURL); blocked {
		return &tool.Result{Error: fmt.Sprintf("ssrf protection blocked request: %s", reason)}, nil
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

// isBlockedURL checks if a URL targets internal/private networks or matched the custom blocklist.
func (h *HTTPRequest) isBlockedURL(rawURL string) (bool, string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true, "invalid URL format"
	}

	host := u.Hostname()
	lowerHost := strings.ToLower(host)

	// 1. Check custom blocklist first
	for _, blocked := range h.blocklist {
		if strings.Contains(lowerHost, strings.ToLower(blocked)) {
			return true, fmt.Sprintf("matches custom blocklist entry %q", blocked)
		}
	}

	// 2. Block explicitly defined localhost names
	for _, blocked := range []string{"localhost", "::1"} {
		if lowerHost == blocked {
			return true, fmt.Sprintf("localhost access %q blocked", host)
		}
	}

	// 3. Block known cloud metadata endpoints
	for _, blocked := range []string{"169.254.169.254", "metadata.google.internal"} {
		if lowerHost == blocked || strings.Contains(lowerHost, blocked) {
			return true, fmt.Sprintf("cloud metadata endpoint %q blocked", host)
		}
	}
	
	ips, err := net.LookupIP(host)
	if err != nil {
		// If DNS fails and it's already an IP, we handle it below. If it's a domain that doesn't resolve,
		// it's arguably safe (or just broken). But if it is an IP already, net.ParseIP handles it.
		ip := net.ParseIP(host)
		if ip != nil {
			ips = []net.IP{ip}
		}
	}

	// 4. Block private and loopback IP ranges using CIDR checking for robustness
	privateCIDRs := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"::1/128",        // IPv6 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // Link-local
		"fc00::/7",       // IPv6 Unique Local Addresses
		"fe80::/10",      // IPv6 Link-local
	}

	var parsedCIDRs []*net.IPNet
	for _, cidr := range privateCIDRs {
		_, network, _ := net.ParseCIDR(cidr)
		parsedCIDRs = append(parsedCIDRs, network)
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true, fmt.Sprintf("resolves to internal IP %s", ip.String())
		}
		
		for _, network := range parsedCIDRs {
			if network.Contains(ip) {
				return true, fmt.Sprintf("resolves to blocked private IP range %s", ip.String())
			}
		}
	}

	return false, ""
}
