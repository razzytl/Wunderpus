# Browser Automation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add native browser automation to Wunderpus using chromedp, enabling the agent to navigate websites, interact with pages, and extract content—all integrated into the tool system as a single generic browser tool.

**Architecture:** Create internal/browser package with BrowserSession management, chromedp engine wrapper, and tool definitions. Wire into existing tool registry and executor pipeline. MVP supports single session, headless mode, basic actions (navigate, click, fill, text, snapshot).

**Tech Stack:** Go 1.25, chromedp v0.14.x, CDP protocol

---

## Phase 1: Core Infrastructure

### Task 1: Add BrowserConfig to Config System

**Files:**
- Modify: `internal/config/config.go:1-260`
- Modify: `config.example.yaml:1-73`

**Step 1: Add BrowserConfig struct to config.go**

Read: `internal/config/config.go`

Add after ToolsConfig:
```go
// BrowserConfig holds browser automation settings.
type BrowserConfig struct {
    Enabled       bool          `yaml:"enabled"`
    Headless      bool          `yaml:"headless"`
    Timeout       int           `yaml:"timeout_seconds"`
    UserAgent     string        `yaml:"user_agent"`
    Viewport      ViewportConfig `yaml:"viewport"`
    Stealth       bool          `yaml:"stealth"`
    MaxInstances  int           `yaml:"max_instances"`
}

// ViewportConfig holds browser viewport settings.
type ViewportConfig struct {
    Width  int `yaml:"width"`
    Height int `yaml:"height"`
}
```

Add to Config struct:
```go
Browser BrowserConfig `yaml:"browser"`
```

**Step 2: Add defaults in applyDefaults()**

Add after tool defaults:
```go
cfg.Browser.Enabled = false
cfg.Browser.Headless = true
cfg.Browser.Timeout = 30
cfg.Browser.UserAgent = ""
cfg.Browser.Viewport.Width = 1280
cfg.Browser.Viewport.Height = 720
cfg.Browser.Stealth = false
cfg.Browser.MaxInstances = 1
```

**Step 3: Add config example section**

Read: `config.example.yaml`

Add at end:
```yaml
# Browser Automation
browser:
  enabled: false             # Enable browser automation
  headless: true            # Run without visible window
  timeout_seconds: 30        # Max time per action
  user_agent: ""            # Custom user agent (default: Chrome)
  viewport:
    width: 1280
    height: 720
  stealth: false           # Enable stealth mode
  max_instances: 1          # Max concurrent browser sessions
```

**Step 4: Commit**

```bash
git add internal/config/config.go config.example.yaml
git commit -m "feat(config): add browser configuration"
```

---

### Task 2: Create Browser Package Core Types

**Files:**
- Create: `internal/browser/browser.go`

**Step 1: Create the browser package directory**

```bash
mkdir -p internal/browser
```

**Step 2: Write browser.go with core types**

```go
package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// Browser manages browser sessions.
type Browser struct {
	mu        sync.RWMutex
	sessions  map[string]*BrowserSession
	config    *BrowserConfig
	allocOpts []chromedp.ExecAllocatorOption
}

// BrowserSession represents a single browser context.
type BrowserSession struct {
	ID        string
	ctx       context.Context
	cancel    context.CancelFunc
	allocator chromedp.ExecAllocator
	taskCtx   context.Context
}

// BrowserConfig holds configuration for browser automation.
type BrowserConfig struct {
	Enabled      bool
	Headless     bool
	Timeout      int
	UserAgent    string
	Viewport     ViewportConfig
	Stealth      bool
	MaxInstances int
}

// ViewportConfig holds viewport dimensions.
type ViewportConfig struct {
	Width  int
	Height int
}

// ActionResult represents the result of a browser action.
type ActionResult struct {
	Output   string
	Error    string
	Metadata map[string]interface{}
}

// New creates a new Browser instance.
func New(cfg *BrowserConfig) *Browser {
	opts := chromedp.DefaultExecAllocatorOptions[:]
	
	if cfg.Headless {
		opts = append(opts, chromedp.Headless)
	}
	
	if cfg.UserAgent != "" {
		opts = append(opts, chromedp.UserAgent(cfg.UserAgent))
	}
	
	opts = append(opts, chromedp.WindowSize(cfg.Viewport.Width, cfg.Viewport.Height))
	
	// Basic stealth options
	if cfg.Stealth {
		opts = append(opts,
			chromedp.DisableGPU(),
			chromedp.EnableChromeStack(),
		)
	}
	
	return &Browser{
		sessions:  make(map[string]*BrowserSession),
		config:    cfg,
		allocOpts: opts,
	}
}

// CreateSession creates a new browser session.
func (b *Browser) CreateSession(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if len(b.sessions) >= b.config.MaxInstances {
		return fmt.Errorf("max browser instances (%d) reached", b.config.MaxInstances)
	}
	
	if _, exists := b.sessions[id]; exists {
		return fmt.Errorf("session %s already exists", id)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	allocator := chromedp.NewExecAllocator(ctx, b.allocOpts...)
	taskCtx, _ := chromedp.NewContext(allocator)
	
	session := &BrowserSession{
		ID:        id,
		ctx:       ctx,
		cancel:    cancel,
		allocator: allocator,
		taskCtx:   taskCtx,
	}
	
	b.sessions[id] = session
	return nil
}

// GetSession retrieves a browser session by ID.
func (b *Browser) GetSession(id string) (*BrowserSession, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	session, ok := b.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return session, nil
}

// CloseSession closes and removes a browser session.
func (b *Browser) CloseSession(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	session, ok := b.sessions[id]
	if !ok {
		return nil
	}
	
	session.cancel()
	delete(b.sessions, id)
	return nil
}

// Close closes all browser sessions.
func (b *Browser) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	for _, session := range b.sessions {
		session.cancel()
	}
	b.sessions = make(map[string]*BrowserSession)
	return nil
}
```

**Step 3: Commit**

```bash
git add internal/browser/browser.go
git commit -m "feat(browser): add core Browser and BrowserSession types"
```

---

### Task 3: Create Engine with Action Methods

**Files:**
- Create: `internal/browser/engine.go`

**Step 1: Write engine.go with action methods**

```go
package browser

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

// Engine provides low-level browser actions via chromedp.
type Engine struct{}

// NewEngine creates a new browser engine.
func NewEngine() *Engine {
	return &Engine{}
}

// NavigateAction represents a navigation action.
type NavigateAction struct {
	URL string
}

// ClickAction represents a click action.
type ClickAction struct {
	Selector string
}

// FillAction represents a fill input action.
type FillAction struct {
	Selector string
	Value    string
}

// TextAction represents a text extraction action.
type TextAction struct {
	Selector string
}

// SnapshotAction represents a page snapshot action.
type SnapshotAction struct {
	OnlyInteractive bool
}

// ScreenshotAction represents a screenshot action.
type ScreenshotAction struct {
	FullPage bool
}

// EvaluateAction represents a JavaScript evaluation action.
type EvaluateAction struct {
	Script string
}

// Navigate performs navigation.
func (e *Engine) Navigate(session *BrowserSession, action NavigateAction) (*ActionResult, error) {
	ctx, cancel := context.WithTimeout(session.taskCtx, time.Duration(30)*time.Second)
	defer cancel()
	
	var url string
	err := chromedp.Run(ctx,
		chromedp.Location(&url),
		chromedp.Navigate(action.URL),
		chromedp.WaitReady("body"),
	)
	
	if err != nil {
		return &ActionResult{Error: fmt.Sprintf("navigation failed: %v", err)}, nil
	}
	
	return &ActionResult{
		Output:   url,
		Metadata: map[string]interface{}{"url": url},
	}, nil
}

// Click performs a click action.
func (e *Engine) Click(session *BrowserSession, action ClickAction) (*ActionResult, error) {
	ctx, cancel := context.WithTimeout(session.taskCtx, time.Duration(30)*time.Second)
	defer cancel()
	
	err := chromedp.Run(ctx,
		chromedp.Click(action.Selector, chromedp.ByQuery),
	)
	
	if err != nil {
		return &ActionResult{Error: fmt.Sprintf("click failed: %v", err)}, nil
	}
	
	return &ActionResult{Output: "clicked"}, nil
}

// Fill performs a fill input action.
func (e *Engine) Fill(session *BrowserSession, action FillAction) (*ActionResult, error) {
	ctx, cancel := context.WithTimeout(session.taskCtx, time.Duration(30)*time.Second)
	defer cancel()
	
	err := chromedp.Run(ctx,
		chromedp.SetValue(action.Selector, action.Value, chromedp.ByQuery),
	)
	
	if err != nil {
		return &ActionResult{Error: fmt.Sprintf("fill failed: %v", err)}, nil
	}
	
	return &ActionResult{Output: "filled"}, nil
}

// Text extracts text content.
func (e *Engine) Text(session *BrowserSession, action TextAction) (*ActionResult, error) {
	ctx, cancel := context.WithTimeout(session.taskCtx, time.Duration(30)*time.Second)
	defer cancel()
	
	var text string
	var err error
	
	if action.Selector == "" {
		// Get all page text
		err = chromedp.Run(ctx,
			chromedp.Body(&text),
		)
	} else {
		err = chromedp.Run(ctx,
			chromedp.Text(action.Selector, &text, chromedp.ByQuery),
		)
	}
	
	if err != nil {
		return &ActionResult{Error: fmt.Sprintf("text extraction failed: %v", err)}, nil
	}
	
	return &ActionResult{Output: text}, nil
}

// Snapshot gets page structure with interactive elements.
func (e *Engine) Snapshot(session *BrowserSession, action SnapshotAction) (*ActionResult, error) {
	ctx, cancel := context.WithTimeout(session.taskCtx, time.Duration(30)*time.Second)
	defer cancel()
	
	// Get document
	var doc *dom.GetDocumentResult
	err := chromedp.Run(ctx,
		chromedp.Action(func(ctx context.Context) error {
			var err error
			doc, err = dom.GetDocument().Do(ctx)
			return err
		}),
	)
	
	if err != nil {
		return &ActionResult{Error: fmt.Sprintf("snapshot failed: %v", err)}, nil
	}
	
	// For MVP, just return that we got the document
	// Full implementation would extract clickable elements
	result := fmt.Sprintf("document loaded: %d nodes", doc.Root.NodeID)
	
	return &ActionResult{Output: result}, nil
}

// Screenshot takes a screenshot.
func (e *Engine) Screenshot(session *BrowserSession, action ScreenshotAction) (*ActionResult, error) {
	ctx, cancel := context.WithTimeout(session.taskCtx, time.Duration(30)*time.Second)
	defer cancel()
	
	var screenshot []byte
	err := chromedp.Run(ctx,
		chromedp.FullScreenshot(&screenshot, 100),
	)
	
	if err != nil {
		return &ActionResult{Error: fmt.Sprintf("screenshot failed: %v", err)}, nil
	}
	
	encoded := base64.StdEncoding.EncodeToString(screenshot)
	return &ActionResult{
		Output:   encoded,
		Metadata: map[string]interface{}{"format": "base64", "size": len(screenshot)},
	}, nil
}

// Evaluate executes JavaScript.
func (e *Engine) Evaluate(session *BrowserSession, action EvaluateAction) (*ActionResult, error) {
	ctx, cancel := context.WithTimeout(session.taskCtx, time.Duration(30)*time.Second)
	defer cancel()
	
	var result interface{}
	err := chromedp.Run(ctx,
		chromedp.Evaluate(action.Script, &result),
	)
	
	if err != nil {
		return &ActionResult{Error: fmt.Sprintf("evaluation failed: %v", err)}, nil
	}
	
	return &ActionResult{Output: fmt.Sprintf("%v", result)}, nil
}

// ValidateSelector checks if a selector is valid.
func (e *Engine) ValidateSelector(session *BrowserSession, selector string) error {
	ctx, cancel := context.WithTimeout(session.taskCtx, 5*time.Second)
	defer cancel()
	
	var nodeCount int
	err := chromedp.Run(ctx,
		chromedp.Evaluate(fmt.Sprintf(`document.querySelectorAll('%s').length`, selector), &nodeCount),
	)
	
	if err != nil {
		return errors.New("invalid selector")
	}
	
	if nodeCount == 0 {
		return errors.New("no elements match selector")
	}
	
	return nil
}
```

**Step 2: Commit**

```bash
git add internal/browser/engine.go
git commit -m "feat(browser): add Engine with action methods"
```

---

### Task 4: Create Tool Definitions

**Files:**
- Create: `internal/browser/tools.go`

**Step 1: Write tools.go with tool definitions**

```go
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolName is the name of the browser tool.
const ToolName = "browser"

// Action represents a browser action with parameters.
type Action struct {
	Name   string
	Params map[string]interface{}
}

// ParseAction parses a string action into an Action struct.
func ParseAction(input string) (*Action, error) {
	parts := strings.Fields(input)
	if len(parts) < 1 {
		return nil, fmt.Errorf("no action specified")
	}
	
	action := &Action{
		Name:   parts[0],
		Params: make(map[string]interface{}),
	}
	
	// Parse key=value pairs
	for _, part := range parts[1:] {
		if idx := strings.Index(part, "="); idx > 0 {
			key := part[:idx]
			value := part[idx+1:]
			action.Params[key] = value
		}
	}
	
	return action, nil
}

// ParseActionJSON parses a JSON-formatted action.
func ParseActionJSON(data string) (*Action, error) {
	var action Action
	if err := json.Unmarshal([]byte(data), &action); err != nil {
		return nil, fmt.Errorf("failed to parse action: %w", err)
	}
	return &action, nil
}

// GetStringParam gets a string parameter with a default.
func GetStringParam(params map[string]interface{}, key, defaultValue string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

// GetBoolParam gets a boolean parameter with a default.
func GetBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// ToolHandler handles browser tool execution.
type ToolHandler struct {
	browser *Browser
	engine  *Engine
}

// NewToolHandler creates a new tool handler.
func NewToolHandler(browser *Browser) *ToolHandler {
	return &ToolHandler{
		browser: browser,
		engine:  NewEngine(),
	}
}

// Handle executes a browser action.
func (h *ToolHandler) Handle(ctx context.Context, actionJSON string) (*ActionResult, error) {
	action, err := ParseActionJSON(actionJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse action: %w", err)
	}
	
	// Use default session if none specified
	sessionID := GetStringParam(action.Params, "session", "default")
	
	session, err := h.browser.GetSession(sessionID)
	if err != nil {
		// Try to create default session
		if sessionID == "default" {
			if createErr := h.browser.CreateSession("default"); createErr != nil {
				return nil, fmt.Errorf("failed to create session: %w", createErr)
			}
			session, _ = h.browser.GetSession("default")
		}
		if session == nil {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
	}
	
	switch action.Name {
	case "navigate":
		url := GetStringParam(action.Params, "url", "")
		if url == "" {
			return nil, fmt.Errorf("url parameter required")
		}
		return h.engine.Navigate(session, NavigateAction{URL: url})
		
	case "click":
		selector := GetStringParam(action.Params, "selector", "")
		if selector == "" {
			return nil, fmt.Errorf("selector parameter required")
		}
		return h.engine.Click(session, ClickAction{Selector: selector})
		
	case "fill":
		selector := GetStringParam(action.Params, "selector", "")
		value := GetStringParam(action.Params, "value", "")
		if selector == "" {
			return nil, fmt.Errorf("selector parameter required")
		}
		return h.engine.Fill(session, FillAction{Selector: selector, Value: value})
		
	case "text":
		selector := GetStringParam(action.Params, "selector", "")
		return h.engine.Text(session, TextAction{Selector: selector})
		
	case "snapshot":
		onlyInteractive := GetBoolParam(action.Params, "interactive", false)
		return h.engine.Snapshot(session, SnapshotAction{OnlyInteractive: onlyInteractive})
		
	case "screenshot":
		fullPage := GetBoolParam(action.Params, "fullpage", true)
		return h.engine.Screenshot(session, ScreenshotAction{FullPage: fullPage})
		
	case "evaluate":
		script := GetStringParam(action.Params, "script", "")
		if script == "" {
			return nil, fmt.Errorf("script parameter required")
		}
		return h.engine.Evaluate(session, EvaluateAction{Script: script})
		
	case "close":
		err := h.browser.CloseSession(sessionID)
		if err != nil {
			return &ActionResult{Output: "already closed"}, nil
		}
		return &ActionResult{Output: "closed"}, nil
		
	default:
		return nil, fmt.Errorf("unknown action: %s", action.Name)
	}
}
```

**Step 2: Commit**

```bash
git add internal/browser/tools.go
git commit -m "feat(browser): add tool definitions and handler"
```

---

### Task 5: Register Browser Tools in Main

**Files:**
- Modify: `cmd/wonderpus/main.go:1-172`

**Step 1: Import browser package and initialize**

Read: `cmd/wunderpus/main.go`

Add import:
```go
"github.com/wonderpus/wonderpus/internal/browser"
```

Add after tool initialization:
```go
// 5b. Init browser if enabled
var browserHandler *browser.ToolHandler
if cfg.Browser.Enabled {
    b := browser.NewBrowser(&browser.BrowserConfig{
        Enabled:      cfg.Browser.Enabled,
        Headless:     cfg.Browser.Headless,
        Timeout:      cfg.Browser.Timeout,
        UserAgent:    cfg.Browser.UserAgent,
        Viewport:     browser.ViewportConfig{
            Width:  cfg.Browser.Viewport.Width,
            Height: cfg.Browser.Viewport.Height,
        },
        Stealth:      cfg.Browser.Stealth,
        MaxInstances: cfg.Browser.MaxInstances,
    })
    browserHandler = browser.NewToolHandler(b)
    
    // Register browser tool
    registry.Register(browser.NewBrowserTool(browserHandler))
    
    // Create default session
    if err := b.CreateSession("default"); err != nil {
        slog.Warn("failed to create default browser session", "error", err)
    }
    
    slog.Info("browser automation enabled")
}
```

**Step 2: Add NewBrowserTool function**

Create `internal/browser/tool_registration.go`:
```go
package browser

import (
	"context"
	"encoding/json"
	
	"github.com/wonderpus/wonderpus/internal/tool"
)

// browserTool implements tool.Tool for browser actions.
type browserTool struct {
	handler *ToolHandler
}

func NewBrowserTool(handler *ToolHandler) tool.Tool {
	return &browserTool{handler: handler}
}

func (t *browserTool) Name() string {
	return ToolName
}

func (t *browserTool) Description() string {
	return `Browser automation tool. Actions: navigate url=<url>, click selector=<css>, fill selector=<css> value=<text>, text [selector=<css>], snapshot, screenshot, evaluate script=<js>, close`
}

func (t *browserTool) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "action", Type: "string", Description: "Action to perform: navigate, click, fill, text, snapshot, screenshot, evaluate, close", Required: true},
		{Name: "params", Type: "object", Description: "Action parameters as JSON", Required: false},
	}
}

func (t *browserTool) Sensitive() bool {
	return false
}

func (t *browserTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	// Build action JSON
	action, ok := args["action"].(string)
	if !ok {
		return &tool.Result{Error: "action parameter required"}, nil
	}
	
	params, _ := args["params"].(map[string]any)
	if params == nil {
		params = make(map[string]any)
	}
	
	// Add action to params for JSON serialization
	params["_action"] = action
	
	actionJSON, err := json.Marshal(params)
	if err != nil {
		return &tool.Result{Error: "failed to marshal params"}, nil
	}
	
	result, err := t.handler.Handle(ctx, string(actionJSON))
	if err != nil {
		return &tool.Result{Error: err.Error()}, nil
	}
	
	if result.Error != "" {
		return &tool.Result{Error: result.Error}, nil
	}
	
	return &tool.Result{Output: result.Output}, nil
}
```

**Step 3: Commit**

```bash
git add cmd/wonderpus/main.go internal/browser/tool_registration.go
git commit -m "feat: integrate browser tool into main"
```

---

### Task 6: Add go.mod Dependency

**Files:**
- Modify: `go.mod:1-70`
- Modify: `go.sum` (auto-generated)

**Step 1: Add chromedp dependency**

```bash
go get github.com/chromedp/chromedp@latest
go get github.com/chromedp/cdproto@latest
```

**Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add chromedp for browser automation"
```

---

## Phase 2: Testing & Polish

### Task 7: Add Browser Tool Tests

**Files:**
- Create: `internal/browser/browser_test.go`

**Step 1: Write unit tests**

```go
package browser

import (
	"testing"
)

func TestParseAction(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantURL  string
	}{
		{"navigate url=http://example.com", "navigate", "http://example.com"},
		{"click selector=#btn", "click", ""},
		{"text selector=.content", "text", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			action, err := ParseAction(tt.input)
			if err != nil {
				t.Fatalf("ParseAction error: %v", err)
			}
			
			if action.Name != tt.wantName {
				t.Errorf("got name %s, want %s", action.Name, tt.wantName)
			}
			
			if tt.wantURL != "" && action.Params["url"] != tt.wantURL {
				t.Errorf("got url %s, want %s", action.Params["url"], tt.wantURL)
			}
		})
	}
}

func TestGetStringParam(t *testing.T) {
	params := map[string]interface{}{
		"key": "value",
	}
	
	if got := GetStringParam(params, "key", "default"); got != "value" {
		t.Errorf("got %s, want value", got)
	}
	
	if got := GetStringParam(params, "missing", "default"); got != "default" {
		t.Errorf("got %s, want default", got)
	}
}

func TestGetBoolParam(t *testing.T) {
	params := map[string]interface{}{
		"bool_true":  true,
		"bool_false": false,
	}
	
	if !GetBoolParam(params, "bool_true", false) {
		t.Error("expected true")
	}
	
	if GetBoolParam(params, "bool_false", true) {
		t.Error("expected false")
	}
	
	if GetBoolParam(params, "missing", true) != true {
		t.Error("expected default true")
	}
}
```

**Step 2: Run tests**

```bash
go test -v ./internal/browser/...
```

**Step 3: Commit**

```bash
git add internal/browser/browser_test.go
git commit -m "test(browser): add unit tests for browser tools"
```

---

### Task 8: Integration Test (Manual)

**Step 1: Create test script**

```bash
cat > test_browser.sh << 'EOF'
#!/bin/bash
# Manual browser test

# Start wonderpus with browser enabled
./wonderpus --config config.yaml

# In another terminal, test via TUI:
# /tool browser navigate url=https://example.com
# /tool browser text
# /tool browser close
EOF
chmod +x test_browser.sh
```

**Step 2: Document test steps**

Add to `TESTING.md`:
```markdown
## Browser Automation Tests

### Manual Testing

1. Enable browser in config.yaml:
   ```yaml
   browser:
     enabled: true
     headless: true
   ```

2. Start wonderpus:
   ```bash
   ./wunderpus
   ```

3. Test via TUI:
   - `/tool browser navigate url=https://example.com`
   - `/tool browser text`
   - `/tool browser close`
```

**Step 3: Commit**

```bash
git add test_browser.sh TESTING.md
git commit -m "test: add browser integration test docs"
```

---

### Task 9: Update Documentation

**Files:**
- Modify: `README.md` (create if not exists)
- Create: `docs/browser.md`

**Step 1: Add to README**

```markdown
## Browser Automation

Wunderpus can control a headless Chrome browser to interact with websites.

### Configuration

```yaml
browser:
  enabled: true
  headless: true
  timeout_seconds: 30
```

### Usage

In TUI, use the browser tool:

```
/tool browser navigate url=https://example.com
/tool browser click selector=#submit-button
/tool browser fill selector=#email value=user@example.com
/tool browser text selector=h1
/tool browser snapshot
/tool browser screenshot
/tool browser close
```

### Actions

| Action | Description | Parameters |
|--------|-------------|------------|
| navigate | Go to URL | url |
| click | Click element | selector |
| fill | Fill input field | selector, value |
| text | Get text content | selector (optional) |
| snapshot | Get page structure | interactive (bool) |
| screenshot | Take screenshot | fullpage (bool) |
| evaluate | Run JavaScript | script |
| close | Close session | - |
```

**Step 2: Commit**

```bash
git add README.md docs/browser.md
git commit -m "docs: add browser automation documentation"
```

---

## Summary

| Task | Description | Files Modified |
|------|-------------|----------------|
| 1 | Config for browser | config.go, config.example.yaml |
| 2 | Core browser types | browser.go (new) |
| 3 | Engine with actions | engine.go (new) |
| 4 | Tool definitions | tools.go (new) |
| 5 | Register in main | main.go, tool_registration.go |
| 6 | Add dependencies | go.mod, go.sum |
| 7 | Unit tests | browser_test.go |
| 8 | Integration test | test_browser.sh, TESTING.md |
| 9 | Documentation | README.md, docs/browser.md |

---

**Plan complete and saved to `docs/plans/2026-03-04-browser-automation-plan.md`.**

Two execution options:

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
