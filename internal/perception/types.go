package perception

import "time"

// BrowserAction represents an action the LLM wants to take on a page.
type BrowserAction struct {
	Action      string     `json:"action"`                // click, type, scroll, navigate, wait, extract, screenshot, press
	Coordinates [2]float64 `json:"coordinates,omitempty"` // [x, y] for click
	Text        string     `json:"text,omitempty"`        // text for type, URL for navigate
	Direction   string     `json:"direction,omitempty"`   // up, down for scroll
	Key         string     `json:"key,omitempty"`         // keyboard key for press (Enter, Tab, etc.)
	Selector    string     `json:"selector,omitempty"`    // CSS selector (DOM agent)
	Reasoning   string     `json:"reasoning,omitempty"`   // why the LLM chose this action
}

// BrowserResult is the outcome of a browser agent execution.
type BrowserResult struct {
	Success       bool          `json:"success"`
	FinalURL      string        `json:"final_url"`
	ExtractedData string        `json:"extracted_data,omitempty"` // structured data from final state
	Screenshots   []string      `json:"screenshots,omitempty"`    // base64 screenshots taken
	ActionsTaken  int           `json:"actions_taken"`
	Duration      time.Duration `json:"duration"`
	Error         string        `json:"error,omitempty"`
	ActionLog     []ActionLog   `json:"action_log,omitempty"`
}

// ActionLog records a single action and its result.
type ActionLog struct {
	Action   BrowserAction `json:"action"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// DOMElement represents an interactive element found in the DOM.
type DOMElement struct {
	Tag        string            `json:"tag"`      // button, input, a, select, etc.
	Text       string            `json:"text"`     // visible text or label
	Selector   string            `json:"selector"` // CSS selector to target it
	Position   string            `json:"position"` // "top-left", "center", etc.
	IsVisible  bool              `json:"is_visible"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// DOMDescription is a semantic description of a page's interactive elements.
type DOMDescription struct {
	URL      string       `json:"url"`
	Title    string       `json:"title"`
	Elements []DOMElement `json:"elements"`
}

// VisionLLMCaller abstracts LLM calls with vision support.
type VisionLLMCaller interface {
	// CompleteWithVision sends a text prompt with an image and returns the response.
	CompleteWithVision(prompt string, imageData []byte, mimeType string) (string, error)
	// Complete sends a text-only prompt and returns the response.
	CompleteText(prompt string) (string, error)
}

// ScreenshotCapturer abstracts screenshot capture.
type ScreenshotCapturer interface {
	// Screenshot captures a screenshot of the current page.
	Screenshot() ([]byte, error)
	// Navigate navigates to a URL and waits for load.
	Navigate(url string) error
}

// ActionExecutor executes browser actions.
type ActionExecutor interface {
	// Click performs a mouse click at coordinates.
	Click(x, y float64) error
	// Type types text at the current focus.
	Types(text string) error
	// Press presses a keyboard key.
	Press(key string) error
	// Scroll scrolls the page.
	Scroll(direction string) error
	// Navigate navigates to a URL.
	Navigate(url string) error
	// GetURL returns the current page URL.
	GetURL() string
	// GetPageContent returns the page's text content.
	GetPageContent() string
	// ExtractDOM returns the interactive DOM elements.
	ExtractDOM() ([]DOMElement, error)
}
