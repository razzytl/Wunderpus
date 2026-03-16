package builtin

import (
	"context"
	"strings"
	"testing"
)

func TestBrowserToolParameters(t *testing.T) {
	bt := NewBrowserTool()

	if bt.Name() != "browser_action" {
		t.Errorf("Expected name 'browser_action', got %s", bt.Name())
	}

	params := bt.Parameters()
	if len(params) < 5 {
		t.Errorf("Expected at least 5 parameters, got %d", len(params))
	}

	hasAction := false
	for _, p := range params {
		if p.Name == "action" && p.Required {
			hasAction = true
			break
		}
	}

	if !hasAction {
		t.Errorf("Expected a required 'action' parameter")
	}
}

func TestBrowserToolMissingAction(t *testing.T) {
	bt := NewBrowserTool()
	res, err := bt.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if res.Error == "" || !strings.Contains(res.Error, "action is required") {
		t.Errorf("Expected 'action is required' error, got: %s", res.Error)
	}
}

func TestBrowserToolUnknownAction(t *testing.T) {
	bt := NewBrowserTool()
	res, err := bt.Execute(context.Background(), map[string]any{
		"action": "unknown_action_xyz",
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Because ensureBrowser runs before checking action in our implementation,
	// if Playwright is not installed in the CI environment, ensureBrowser might fail first.
	// So we accept either an initialization error or "unknown action".
	if res.Error == "" || (!strings.Contains(res.Error, "unknown action") && !strings.Contains(res.Error, "failed to init browser")) {
		t.Errorf("Expected an unknown action or init error, got: %s", res.Error)
	}
}
