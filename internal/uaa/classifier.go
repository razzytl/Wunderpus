package uaa

import (
	"path/filepath"
	"strings"
)

// ActionTier classifies actions by risk level and required trust cost.
type ActionTier int

const (
	// TierReadOnly — read-only operations with no side effects (cost 0)
	// Examples: web search, read files, API GETs, list directories
	TierReadOnly ActionTier = 1

	// TierEphemeral — operations that create temporary or disposable artifacts (cost 1)
	// Examples: create temp files, run tests, go build, sandbox exec
	TierEphemeral ActionTier = 2

	// TierPersistent — operations that modify persistent state (cost 5)
	// Examples: git commit, write to DB, modify config, write non-temp files
	TierPersistent ActionTier = 3

	// TierExternal — operations with external impact beyond the local system (cost 20)
	// Examples: send HTTP POST to external, deploy, send comms, spend money
	TierExternal ActionTier = 4
)

// Action represents a proposed operation to be classified and gated.
type Action struct {
	ID          string
	Description string
	Tool        string
	Parameters  map[string]interface{}
	Tier        ActionTier
	TrustCost   int
	Reversible  bool
	Scope       ActionScope
}

// ActionScope describes the scope of an action's impact.
type ActionScope string

const (
	ScopeLocal       ActionScope = "local"
	ScopeNetwork     ActionScope = "network"
	ScopeFinancial   ActionScope = "financial"
	ScopeDestructive ActionScope = "destructive"
)

// TrustCostForTier returns the standard trust cost for a given tier.
func TrustCostForTier(tier ActionTier) int {
	switch tier {
	case TierReadOnly:
		return 0
	case TierEphemeral:
		return 1
	case TierPersistent:
		return 5
	case TierExternal:
		return 20
	default:
		return 20 // fail-safe: treat unknown as Tier 4
	}
}

// Classifier assigns ActionTier to actions using rule-based pattern matching.
type Classifier struct {
	allowlist map[string]bool // known-safe external hosts
}

// NewClassifier creates a classifier with the given allowlist of safe external hosts.
func NewClassifier(allowlist []string) *Classifier {
	m := make(map[string]bool, len(allowlist))
	for _, host := range allowlist {
		m[strings.ToLower(host)] = true
	}
	return &Classifier{allowlist: m}
}

// Classify determines the ActionTier for the given action based on tool name
// and parameter inspection. Unknown tools default to Tier 4 (fail-safe).
func (c *Classifier) Classify(a Action) ActionTier {
	tool := strings.ToLower(a.Tool)

	switch {
	// Tier 1: Read-only operations
	case tool == "web_search" || tool == "websearch":
		return TierReadOnly
	case tool == "read_file" || tool == "file_read":
		return TierReadOnly
	case tool == "list_files" || tool == "file_list" || tool == "file_glob":
		return TierReadOnly
	case tool == "http_request" && isGetRequest(a.Parameters):
		return TierReadOnly
	case tool == "search_files":
		return TierReadOnly
	case tool == "system_info":
		return TierReadOnly
	case tool == "calculator":
		return TierReadOnly

	// Tier 2: Ephemeral writes
	case tool == "write_file" && isTempPath(a.Parameters):
		return TierEphemeral
	case tool == "shell_exec" && isSafeShellCommand(a.Parameters):
		return TierEphemeral
	case tool == "sandbox_exec" || tool == "sandbox":
		return TierEphemeral

	// Tier 3: Persistent writes
	case tool == "write_file" || tool == "file_write":
		return TierPersistent
	case tool == "file_edit" || tool == "edit_file":
		return TierPersistent
	case tool == "git_commit" || tool == "git_push":
		return TierPersistent

	// Tier 4: External impact — check allowlist for HTTP
	case tool == "http_request" || tool == "http_post":
		if c.isAllowedHost(a.Parameters) {
			return TierEphemeral
		}
		return TierExternal
	case tool == "send_file":
		return TierExternal
	case tool == "spawn" || tool == "spawn_agent":
		return TierExternal

	// Default: fail-safe to Tier 4
	default:
		return TierExternal
	}
}

// isGetRequest checks if an HTTP request is a GET (read-only).
func isGetRequest(params map[string]interface{}) bool {
	if method, ok := params["method"].(string); ok {
		return strings.ToUpper(method) == "GET"
	}
	return false
}

// isTempPath checks if a file path targets a temporary directory.
func isTempPath(params map[string]interface{}) bool {
	path, ok := params["path"].(string)
	if !ok {
		return false
	}
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	return strings.Contains(cleanPath, "/tmp/") ||
		strings.Contains(cleanPath, "/temp/") ||
		strings.HasPrefix(cleanPath, osTempDir())
}

// osTempDir returns the OS temp directory with forward slashes.
func osTempDir() string {
	return filepath.ToSlash(filepath.Clean(filepath.Join("C:", "Temp")))
}

// isSafeShellCommand checks if a shell command is in the safe whitelist.
func isSafeShellCommand(params map[string]interface{}) bool {
	cmd, ok := params["command"].(string)
	if !ok {
		return false
	}
	safeCmds := []string{"ls", "dir", "cat", "type", "echo", "head", "tail", "pwd", "date", "whoami", "go build", "go test", "go vet"}
	cmdLower := strings.ToLower(strings.TrimSpace(cmd))
	for _, safe := range safeCmds {
		if cmdLower == safe || strings.HasPrefix(cmdLower, safe+" ") {
			return true
		}
	}
	return false
}

// isAllowedHost checks if the HTTP target host is in the allowlist.
func (c *Classifier) isAllowedHost(params map[string]interface{}) bool {
	url, ok := params["url"].(string)
	if !ok {
		// Also check "host" parameter
		host, ok := params["host"].(string)
		if !ok {
			return false
		}
		return c.allowlist[strings.ToLower(host)]
	}

	// Extract host from URL
	host := extractHost(url)
	return c.allowlist[strings.ToLower(host)]
}

// extractHost pulls the hostname from a URL string.
func extractHost(rawURL string) string {
	s := rawURL
	// Remove scheme
	if idx := strings.Index(s, "://"); idx >= 0 {
		s = s[idx+3:]
	}
	// Remove path
	if idx := strings.IndexAny(s, "/?#"); idx >= 0 {
		s = s[:idx]
	}
	// Remove port
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		s = s[:idx]
	}
	return s
}
