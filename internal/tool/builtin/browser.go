package builtin

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/playwright-community/playwright-go"
	"github.com/wunderpus/wunderpus/internal/tool"
)

// BrowserTool allows the agent to interact with a real browser.
type BrowserTool struct {
	mu      sync.Mutex
	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
}

func NewBrowserTool() *BrowserTool {
	return &BrowserTool{}
}

func (b *BrowserTool) Name() string { return "browser_action" }

func (b *BrowserTool) Description() string {
	return "Interact with a web browser to visually navigate pages, click, type, and take screenshots."
}

func (b *BrowserTool) Sensitive() bool                   { return false }
func (b *BrowserTool) ApprovalLevel() tool.ApprovalLevel { return tool.AutoExecute }
func (b *BrowserTool) Version() string                   { return "1.0.0" }

func (b *BrowserTool) Dependencies() []string { return nil }

func (b *BrowserTool) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "action", Type: "string", Description: "The action to perform: 'navigate', 'click', 'type', 'screenshot', 'close'.", Required: true},
		{Name: "url", Type: "string", Description: "The URL to navigate to (required for 'navigate' action).", Required: false},
		{Name: "x", Type: "number", Description: "X coordinate of the element (required for 'click' action).", Required: false},
		{Name: "y", Type: "number", Description: "Y coordinate of the element (required for 'click' action).", Required: false},
		{Name: "text", Type: "string", Description: "Text to type (required for 'type' action).", Required: false},
	}
}

// ensureBrowser ensures a browser page is open.
func (b *BrowserTool) ensureBrowser() error {
	if b.page != nil {
		return nil
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %w", err)
	}
	b.pw = pw

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	b.browser = browser

	page, err := browser.NewPage(playwright.BrowserNewPageOptions{
		Viewport: &playwright.Size{Width: 1280, Height: 720},
	})
	if err != nil {
		return fmt.Errorf("could not create page: %w", err)
	}
	b.page = page

	return nil
}

func (b *BrowserTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	action, ok := args["action"].(string)
	if !ok || action == "" {
		return &tool.Result{Error: "action is required"}, nil
	}

	// Close action can be done even if browser isn't fully initialized (it handles cleanup)
	if action == "close" {
		b.Close()
		return &tool.Result{Output: "Browser closed successfully."}, nil
	}

	// For other actions, ensure browser is open
	if err := b.ensureBrowser(); err != nil {
		return &tool.Result{Error: fmt.Sprintf("failed to init browser: %v", err)}, nil
	}

	switch action {
	case "navigate":
		url, _ := args["url"].(string)
		if url == "" {
			return &tool.Result{Error: "url is required for navigate action"}, nil
		}

		// Add http protocol if missing
		if len(url) > 4 && url[:4] != "http" {
			url = "https://" + url
		}

		if _, err := b.page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		}); err != nil {
			return &tool.Result{Error: fmt.Sprintf("failed to navigate: %v", err)}, nil
		}
		return &tool.Result{Output: fmt.Sprintf("Successfully navigated to %s", url)}, nil

	case "screenshot":
		screenshotBytes, err := b.page.Screenshot(playwright.PageScreenshotOptions{
			Type: playwright.ScreenshotTypeJpeg,
		})
		if err != nil {
			return &tool.Result{Error: fmt.Sprintf("failed to take screenshot: %v", err)}, nil
		}

		encodedString := base64.StdEncoding.EncodeToString(screenshotBytes)
		return &tool.Result{Output: fmt.Sprintf("Screenshot taken:\ndata:image/jpeg;base64,%s", encodedString)}, nil

	case "click":
		xFloat, okX := args["x"].(float64)
		yFloat, okY := args["y"].(float64)
		if !okX || !okY {
			return &tool.Result{Error: "x and y coordinates are required for click action"}, nil
		}

		if err := b.page.Mouse().Click(xFloat, yFloat); err != nil {
			return &tool.Result{Error: fmt.Sprintf("failed to click at (%v, %v): %v", xFloat, yFloat, err)}, nil
		}
		return &tool.Result{Output: fmt.Sprintf("Clicked at coordinates (%v, %v)", xFloat, yFloat)}, nil

	case "type":
		text, _ := args["text"].(string)
		if text == "" {
			return &tool.Result{Error: "text is required for type action"}, nil
		}

		err := b.page.Keyboard().Type(text)
		if err != nil {
			return &tool.Result{Error: fmt.Sprintf("failed to type text: %v", err)}, nil
		}
		return &tool.Result{Output: fmt.Sprintf("Typed text: %q", text)}, nil

	default:
		return &tool.Result{Error: fmt.Sprintf("unknown action: %s", action)}, nil
	}
}

// Close gracefully closes the Playwright resources.
func (b *BrowserTool) Close() {
	if b.page != nil {
		_ = b.page.Close()
		b.page = nil
	}
	if b.browser != nil {
		_ = b.browser.Close()
		b.browser = nil
	}
	if b.pw != nil {
		_ = b.pw.Stop()
		b.pw = nil
	}
}
