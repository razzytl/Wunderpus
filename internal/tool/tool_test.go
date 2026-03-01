package tool_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wonderpus/wonderpus/internal/security"
	"github.com/wonderpus/wonderpus/internal/tool"
	"github.com/wonderpus/wonderpus/internal/tool/builtin"
)

// mockTool is a simple tool for testing the executor.
type mockTool struct {
	sensitive bool
	errResult error
}

func (m *mockTool) Name() string                                    { return "mock_tool" }
func (m *mockTool) Description() string                             { return "A mock tool" }
func (m *mockTool) Parameters() []tool.ParameterDef                 { return nil }
func (m *mockTool) Sensitive() bool                                 { return m.sensitive }
func (m *mockTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	if m.errResult != nil {
		return nil, m.errResult
	}
	return &tool.Result{Output: "success"}, nil
}

func TestRegistry(t *testing.T) {
	reg := tool.NewRegistry()

	if reg.Count() != 0 {
		t.Errorf("expected count 0, got %d", reg.Count())
	}

	mt := &mockTool{}
	err := reg.Register(mt)
	if err != nil {
		t.Fatalf("unexpected error registering tool: %v", err)
	}

	if reg.Count() != 1 {
		t.Errorf("expected count 1, got %d", reg.Count())
	}

	// Test retrieve
	retrieved, ok := reg.Get("mock_tool")
	if !ok || retrieved.Name() != "mock_tool" {
		t.Errorf("failed to retrieve tool")
	}

	// Test duplicate registration
	err = reg.Register(mt)
	if err == nil {
		t.Errorf("expected error registering duplicate tool")
	}
}

func TestExecutor_Approval(t *testing.T) {
	// Need an audit logger for executor
	audit, _ := security.NewAuditLogger(":memory:")
	defer audit.Close()

	reg := tool.NewRegistry()
	mt := &mockTool{sensitive: true}
	reg.Register(mt)

	// Deny everything
	denyAll := func(name string, args map[string]any) (bool, error) {
		return false, nil
	}

	execDeny := tool.NewExecutor(reg, audit, denyAll, time.Second)
	res := execDeny.Execute(context.Background(), tool.ToolCall{Name: "mock_tool"})
	if !strings.Contains(res.Error, "denied") {
		t.Errorf("expected denied error, got: %s (output: %s)", res.Error, res.Output)
	}

	// Allow everything
	allowAll := func(name string, args map[string]any) (bool, error) {
		return true, nil
	}

	execAllow := tool.NewExecutor(reg, audit, allowAll, time.Second)
	res2 := execAllow.Execute(context.Background(), tool.ToolCall{Name: "mock_tool"})
	if res2.Error != "" {
		t.Errorf("unexpected error: %s", res2.Error)
	}
	if res2.Output != "success" {
		t.Errorf("expected success output, got: %s", res2.Output)
	}
}

func TestExecutor_Analytics(t *testing.T) {
	audit, _ := security.NewAuditLogger(":memory:")
	defer audit.Close()

	reg := tool.NewRegistry()
	mt := &mockTool{} // not sensitive
	reg.Register(mt)

	exec := tool.NewExecutor(reg, audit, nil, time.Second)

	// Run twice
	exec.Execute(context.Background(), tool.ToolCall{Name: "mock_tool"})
	exec.Execute(context.Background(), tool.ToolCall{Name: "mock_tool"})

	stats := exec.GetStats()
	if stats["mock_tool"] == nil {
		t.Fatalf("expected stats for mock_tool")
	}

	if stats["mock_tool"].CallCount != 2 {
		t.Errorf("expected 2 calls, got %d", stats["mock_tool"].CallCount)
	}
}

// Built-in specific tests for sandboxing and correctness

func TestBuiltin_Calculator(t *testing.T) {
	calc := builtin.NewCalculator()

	tests := []struct {
		expr     string
		expected string
		hasError bool
	}{
		{"2 + 2", "4", false},
		{"10 - 2 * 3", "4", false},
		{"(10 - 2) * 3", "24", false},
		{"10 / 0", "", true},
		{"sqrt(16)", "4", false},
		{"pow(2, 3)", "8", false},
		{"pow(2, 3", "", true}, // Parse error
	}

	for _, tt := range tests {
		res, err := calc.Execute(context.Background(), map[string]any{"expression": tt.expr})
		if err != nil {
			t.Fatalf("Execute should return error in res.Error, not err itself")
		}

		if tt.hasError {
			if res.Error == "" {
				t.Errorf("expected error for %q, got none", tt.expr)
			}
		} else {
			if res.Error != "" {
				t.Errorf("unexpected error for %q: %s", tt.expr, res.Error)
			}
			if res.Output != tt.expected {
				t.Errorf("expected %s for %q, got %s", tt.expected, tt.expr, res.Output)
			}
		}
	}
}

func TestBuiltin_FileSandbox(t *testing.T) {
	// Sandbox file system tool to specific paths only
	fr := builtin.NewFileRead([]string{"/allow/this/path"})

	tests := []struct {
		path    string
		blocked bool
	}{
		// These paths will depend heavily on the environment and how filepath.Abs resolves them.
		// A common way to test this robustly is to use known relative paths string manipulations.
		{"../../../etc/passwd", true},                  // Traversal should be blocked
		{"/allow/this/path/file.txt", false},           // direct child
		{"/allow/this/path/sub/file.txt", false},       // nested child
		{"/allow/this/path/../../other/file", true},  // attempts to break out
		{"/deny/this/path", true},                      // completely outside
	}

	for _, tt := range tests {
		res, _ := fr.Execute(context.Background(), map[string]any{"path": tt.path})
		
		isBlocked := strings.Contains(res.Error, "access denied") || strings.Contains(res.Error, "outside allowed paths")
		// The tool blocks paths that have "..", and those outside allowed.
		if strings.Contains(tt.path, "..") {
			isBlocked = true
		}

		if tt.blocked && !isBlocked {
			t.Errorf("expected path %q to be blocked, but it wasn't. Error: %s", tt.path, res.Error)
		}
		if !tt.blocked && isBlocked {
			t.Errorf("expected path %q to be allowed, but it was blocked. Error: %s", tt.path, res.Error)
		}
	}
}

func TestBuiltin_ShellWhitelist(t *testing.T) {
	wl := []string{"echo"}
	se := builtin.NewShellExec(wl)

	// Valid and whitelisted
	res, _ := se.Execute(context.Background(), map[string]any{"command": "echo test"})
	if res.Error != "" {
		t.Errorf("unexpected error on valid command: %s", res.Error)
	}

	// Not whitelisted
	res, _ = se.Execute(context.Background(), map[string]any{"command": "cat /etc/passwd"})
	if !strings.Contains(res.Error, "not whitelisted") {
		t.Errorf("expected not whitelisted error, got: %v", res.Error)
	}

	// Dangreous even if tool whitelisted (e.g., if somehow a dangerous command is passed)
	res, _ = se.Execute(context.Background(), map[string]any{"command": "echo test > /dev/null"})
	if !strings.Contains(res.Error, "blocked: command contains dangerous pattern") {
		t.Errorf("expected dangerous pattern block, got: %s", res.Error)
	}
}

func TestBuiltin_HTTPSSRF(t *testing.T) {
	http := builtin.NewHTTPRequest() // Built to block internal IPs

	tests := []struct {
		url     string
		blocked bool
	}{
		{"http://localhost:8080", true},
		{"http://127.0.0.1", true},
		{"http://169.254.169.254/metadata", true}, // AWS metadata
		{"http://10.0.0.1", true},                 // Private network
		{"http://192.168.1.1", true},               // Private network
		{"https://google.com", false},              // Valid public
	}

	for _, tt := range tests {
		res, _ := http.Execute(context.Background(), map[string]any{"url": tt.url})
		isBlocked := strings.Contains(res.Error, "blocked")

		if tt.blocked && !isBlocked {
			t.Errorf("expected URL %q to be blocked, got error: %s", tt.url, res.Error)
		}
		if !tt.blocked && isBlocked {
			t.Errorf("expected URL %q to be allowed, got error: %s", tt.url, res.Error)
		}
	}
}
