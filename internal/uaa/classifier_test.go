package uaa

import (
	"testing"
)

func TestClassifier_Classify(t *testing.T) {
	c := NewClassifier([]string{"api.openai.com", "safe-host.example.com"})

	tests := []struct {
		name     string
		action   Action
		expected ActionTier
	}{
		// Tier 1: Read-only
		{
			name:     "web_search",
			action:   Action{Tool: "web_search", Parameters: map[string]interface{}{}},
			expected: TierReadOnly,
		},
		{
			name:     "read_file",
			action:   Action{Tool: "read_file", Parameters: map[string]interface{}{"path": "/some/file.go"}},
			expected: TierReadOnly,
		},
		{
			name:     "file_read",
			action:   Action{Tool: "file_read", Parameters: map[string]interface{}{}},
			expected: TierReadOnly,
		},
		{
			name:     "list_files",
			action:   Action{Tool: "list_files", Parameters: map[string]interface{}{}},
			expected: TierReadOnly,
		},
		{
			name:     "search_files",
			action:   Action{Tool: "search_files", Parameters: map[string]interface{}{}},
			expected: TierReadOnly,
		},
		{
			name:     "http_request GET",
			action:   Action{Tool: "http_request", Parameters: map[string]interface{}{"method": "GET", "url": "https://example.com"}},
			expected: TierReadOnly,
		},
		{
			name:     "system_info",
			action:   Action{Tool: "system_info", Parameters: map[string]interface{}{}},
			expected: TierReadOnly,
		},
		{
			name:     "calculator",
			action:   Action{Tool: "calculator", Parameters: map[string]interface{}{}},
			expected: TierReadOnly,
		},

		// Tier 2: Ephemeral writes
		{
			name:     "write_file to /tmp/",
			action:   Action{Tool: "write_file", Parameters: map[string]interface{}{"path": "/tmp/test.txt"}},
			expected: TierEphemeral,
		},
		{
			name:     "safe shell command",
			action:   Action{Tool: "shell_exec", Parameters: map[string]interface{}{"command": "ls -la"}},
			expected: TierEphemeral,
		},
		{
			name:     "sandbox_exec",
			action:   Action{Tool: "sandbox_exec", Parameters: map[string]interface{}{}},
			expected: TierEphemeral,
		},
		{
			name:     "HTTP to allowed host",
			action:   Action{Tool: "http_request", Parameters: map[string]interface{}{"method": "POST", "url": "https://api.openai.com/v1/chat"}},
			expected: TierEphemeral,
		},

		// Tier 3: Persistent writes
		{
			name:     "write_file to non-temp",
			action:   Action{Tool: "write_file", Parameters: map[string]interface{}{"path": "/home/user/file.go"}},
			expected: TierPersistent,
		},
		{
			name:     "file_write",
			action:   Action{Tool: "file_write", Parameters: map[string]interface{}{"path": "/data/output.json"}},
			expected: TierPersistent,
		},
		{
			name:     "file_edit",
			action:   Action{Tool: "file_edit", Parameters: map[string]interface{}{}},
			expected: TierPersistent,
		},
		{
			name:     "git_commit",
			action:   Action{Tool: "git_commit", Parameters: map[string]interface{}{}},
			expected: TierPersistent,
		},

		// Tier 4: External impact
		{
			name:     "HTTP POST to unknown host",
			action:   Action{Tool: "http_request", Parameters: map[string]interface{}{"method": "POST", "url": "https://random-api.com/endpoint"}},
			expected: TierExternal,
		},
		{
			name:     "send_file",
			action:   Action{Tool: "send_file", Parameters: map[string]interface{}{}},
			expected: TierExternal,
		},
		{
			name:     "spawn_agent",
			action:   Action{Tool: "spawn_agent", Parameters: map[string]interface{}{}},
			expected: TierExternal,
		},
		{
			name:     "unknown tool → Tier 4 (fail-safe)",
			action:   Action{Tool: "some_random_tool", Parameters: map[string]interface{}{}},
			expected: TierExternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := c.Classify(tc.action)
			if got != tc.expected {
				t.Errorf("tool=%s: expected tier %d, got %d", tc.action.Tool, tc.expected, got)
			}
		})
	}
}

func TestClassifier_TrustCostForTier(t *testing.T) {
	tests := []struct {
		tier     ActionTier
		expected int
	}{
		{TierReadOnly, 0},
		{TierEphemeral, 1},
		{TierPersistent, 5},
		{TierExternal, 20},
		{ActionTier(99), 20}, // unknown → fail-safe
	}

	for _, tc := range tests {
		got := TrustCostForTier(tc.tier)
		if got != tc.expected {
			t.Errorf("tier %d: expected cost %d, got %d", tc.tier, tc.expected, got)
		}
	}
}
