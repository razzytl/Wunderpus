# Browser Automation Engine — Design Document

**Date:** 2026-03-04  
**Topic:** Integrate browser automation into Wunderpus  
**Status:** Approved

---

## 1. Overview

Add browser automation capabilities to Wunderpus using a native Go implementation (chromedp). This enables the agent to browse websites, interact with JavaScript-heavy pages, fill forms, and extract content—directly integrated into the tool system without external dependencies.

---

## 2. Architecture

### 2.1 Package Structure

```
internal/
  browser/
    browser.go        # Core Browser type, session management
    engine.go          # chromedp wrapper, CDP connections
    pool.go            # Browser instance pool (future: multi-instance)
    tools.go           # Tool definitions for registry
    config.go          # Browser-specific config
```

### 2.2 Core Components

| Component | Responsibility |
|-----------|----------------|
| `Browser` | Main orchestrator; manages sessions, routing |
| `BrowserSession` | Single browser context (headless/headed) |
| `Engine` | Low-level chromedp wrapper |
| `Pool` | Manages multiple isolated browser instances |
| `tools` | Tool definitions for Wunderpus registry |

### 2.3 Tool API (Single Generic Tool)

```
browser <action> <params>

Actions:
  - navigate <url>           # Go to URL
  - click <selector>         # Click element by CSS/XPath
  - fill <selector> <text>   # Fill input field
  - text [selector]          # Get text content (page or element)
  - snapshot                # Get page structure (interactive elements)
  - screenshot              # Take screenshot (base64)
  - evaluate <js>           # Execute JavaScript
  - close                   # Close current session
```

---

## 3. Data Flow

```
User Message
    │
    ▼
Agent Loop (existing)
    │
    ▼
Tool Executor ──► Browser Tool
    │                   │
    │                   ▼
    │              Browser Engine (chromedp)
    │                   │
    │                   ▼
    │              Chrome (headless/headed)
    │
    ▼
Response to User
```

---

## 4. Configuration

### 4.1 Config Additions (config.go)

```yaml
browser:
  enabled: true
  headless: true           # Run without visible window
  timeout_seconds: 30       # Max time per action
  user_agent: ""           # Custom UA (default: Chrome)
  viewport:                # Browser viewport
    width: 1280
    height: 720
  stealth: false           # Enable stealth mode (randomized)
  max_instances: 5         # Max concurrent browser sessions
```

### 4.2 Session Management

- Sessions identified by session ID (string)
- Each session = one browser context
- Sessions persist in memory (future: persist to disk)
- Auto-cleanup on timeout or explicit close

---

## 5. Error Handling

| Error Type | Handling |
|------------|----------|
| Navigation timeout | Return error, allow retry |
| Element not found | Return "element not found" error |
| JavaScript error | Return JS error message |
| Browser crash | Auto-restart session, log incident |
| Resource exhaustion | Limit max sessions, return error |

---

## 6. Security Considerations

- **No external network in sandbox** — Browser can access network (needed for web)
- **Same-origin only by default** — Prevent cross-site actions (configurable)
- **Sensitive data in memory only** — No persistent storage of browser data
- **Timeout enforcement** — Prevents runaway browser processes

---

## 7. Testing Strategy

### 7.1 Unit Tests
- Config loading/validation
- Tool parameter parsing
- Error handling paths

### 7.2 Integration Tests
- Simple navigation flow
- Click and fill sequence
- Text extraction
- Session lifecycle

### 7.3 Manual Testing
- Headless mode
- Headed mode (visible window)
- Multi-instance scenarios

---

## 8. Observability

### 8.1 Metrics (Prometheus)
- `wunderpus_browser_sessions_active` — Current active sessions
- `wunderpus_browser_action_duration_seconds` — Action latency histogram
- `wunderpus_browser_actions_total` — Counter by action type and status

### 8.2 Logging
- Structured JSON logs with correlation IDs
- Log levels: DEBUG ( CDP messages), INFO (actions), WARN (errors)

---

## 9. Phased Rollout

### Phase 1 (MVP)
- Single browser session
- Basic actions: navigate, click, fill, text, snapshot
- Headless mode only
- No multi-instance

### Phase 2 (Enhanced)
- Multi-instance support
- Screenshots
- Stealth mode
- Session persistence

### Phase 3 (Advanced)
- Profile management (cookies, localStorage)
- Proxy support
- Advanced JS evaluation

---

## 10. Dependencies

```go
github.com/chromedp/chromedp v0.14.x
github.com/chromedp/cdproto v0.14.x
```

Only adds ~2-3MB to binary (chromedp + CDP).

---

## 11. File Changes Summary

| File | Action |
|------|--------|
| `internal/browser/browser.go` | New — main Browser type |
| `internal/browser/engine.go` | New — chromedp wrapper |
| `internal/browser/pool.go` | New — session pool |
| `internal/browser/tools.go` | New — tool definitions |
| `internal/browser/config.go` | New — config types |
| `internal/config/config.go` | Modify — add BrowserConfig |
| `config.example.yaml` | Modify — add browser section |
| `cmd/wonderpus/main.go` | Modify — init browser |
| `internal/tool/registry.go` | Modify — register browser tools |

---

## 12. Success Criteria

- [ ] Agent can navigate to a URL and get page text
- [ ] Agent can click elements and fill forms
- [ ] Single binary, no external dependencies at runtime
- [ ] Works in headless mode
- [ ] Proper error messages for failures
- [ ] Integrated with existing tool approval system

---

*Approved: 2026-03-04*
