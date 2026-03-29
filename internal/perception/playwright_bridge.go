package perception

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/playwright-community/playwright-go"
)

// PlaywrightBridge adapts Playwright to the perception interfaces.
// It wraps a Playwright page for screenshot capture, DOM extraction,
// and action execution.
type PlaywrightBridge struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
}

// NewPlaywrightBridge creates a new Playwright bridge.
func NewPlaywrightBridge() (*PlaywrightBridge, error) {
	slog.Info("perception: initializing Playwright bridge")

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("perception: playwright init: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("perception: browser launch: %w", err)
	}

	page, err := browser.NewPage(playwright.BrowserNewPageOptions{
		Viewport: &playwright.Size{Width: 1280, Height: 720},
	})
	if err != nil {
		browser.Close()
		pw.Stop()
		return nil, fmt.Errorf("perception: page creation: %w", err)
	}

	return &PlaywrightBridge{
		pw:      pw,
		browser: browser,
		page:    page,
	}, nil
}

// Screenshot captures a JPEG screenshot of the current page.
func (b *PlaywrightBridge) Screenshot() ([]byte, error) {
	screenshot, err := b.page.Screenshot(playwright.PageScreenshotOptions{
		Type: playwright.ScreenshotTypeJpeg,
	})
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}
	return screenshot, nil
}

// Navigate navigates to a URL and waits for network idle.
func (b *PlaywrightBridge) Navigate(url string) error {
	_, err := b.page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	return err
}

// Click clicks at coordinates.
func (b *PlaywrightBridge) Click(x, y float64) error {
	return b.page.Mouse().Click(x, y)
}

// Types types text at the current focus.
func (b *PlaywrightBridge) Types(text string) error {
	return b.page.Keyboard().Type(text)
}

// Press presses a keyboard key.
func (b *PlaywrightBridge) Press(key string) error {
	return b.page.Keyboard().Press(key)
}

// Scroll scrolls the page up or down.
func (b *PlaywrightBridge) Scroll(direction string) error {
	delta := 300.0
	if direction == "up" {
		delta = -300.0
	}
	_, err := b.page.Evaluate(fmt.Sprintf("window.scrollBy(0, %f)", delta))
	return err
}

// GetURL returns the current page URL.
func (b *PlaywrightBridge) GetURL() string {
	return b.page.URL()
}

// GetPageContent returns the text content of the page.
func (b *PlaywrightBridge) GetPageContent() string {
	content, err := b.page.Evaluate("document.body.innerText")
	if err != nil {
		return ""
	}
	if s, ok := content.(string); ok {
		return s
	}
	return ""
}

// ExtractDOM extracts interactive elements from the page DOM.
func (b *PlaywrightBridge) ExtractDOM() ([]DOMElement, error) {
	script := `
	(() => {
		const elements = [];
		const interactive = document.querySelectorAll('button, input, a, select, textarea, [role="button"], [onclick]');
		interactive.forEach((el, i) => {
			const rect = el.getBoundingClientRect();
			if (rect.width === 0 || rect.height === 0) return;
			
			const tag = el.tagName.toLowerCase();
			const text = (el.textContent || el.value || el.placeholder || el.innerText || '').trim().substring(0, 100);
			const visible = rect.top >= 0 && rect.top < window.innerHeight;
			
			// Generate a simple selector
			let selector = tag;
			if (el.id) selector = '#' + el.id;
			else if (el.className && typeof el.className === 'string') selector = tag + '.' + el.className.split(' ')[0];
			
			elements.push({
				tag: tag,
				text: text,
				selector: selector,
				is_visible: visible,
				position: rect.top < window.innerHeight / 2 ? 'top' : 'bottom'
			});
		});
		return JSON.stringify(elements);
	})()
	`

	result, err := b.page.Evaluate(script)
	if err != nil {
		return nil, fmt.Errorf("DOM extraction failed: %w", err)
	}

	jsonStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected DOM result type")
	}

	var elements []DOMElement
	if err := json.Unmarshal([]byte(jsonStr), &elements); err != nil {
		return nil, fmt.Errorf("DOM parse failed: %w", err)
	}

	return elements, nil
}

// Close cleans up Playwright resources.
func (b *PlaywrightBridge) Close() {
	if b.page != nil {
		b.page.Close()
	}
	if b.browser != nil {
		b.browser.Close()
	}
	if b.pw != nil {
		b.pw.Stop()
	}
	slog.Info("perception: Playwright bridge closed")
}
