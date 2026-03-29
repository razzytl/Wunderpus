package perception

import (
	"context"
	"testing"
)

// --- Mock Implementations ---

type mockCapturer struct {
	screenshots [][]byte
	navigated   []string
	idx         int
}

func (m *mockCapturer) Screenshot() ([]byte, error) {
	if m.idx < len(m.screenshots) {
		data := m.screenshots[m.idx]
		m.idx++
		return data, nil
	}
	return []byte("fake-screenshot"), nil
}

func (m *mockCapturer) Navigate(url string) error {
	m.navigated = append(m.navigated, url)
	return nil
}

type mockExecutor struct {
	url         string
	content     string
	clicks      [][2]float64
	typed       []string
	pressed     []string
	scrolled    []string
	domElements []DOMElement
}

func (m *mockExecutor) Click(x, y float64) error {
	m.clicks = append(m.clicks, [2]float64{x, y})
	return nil
}

func (m *mockExecutor) Types(text string) error {
	m.typed = append(m.typed, text)
	return nil
}

func (m *mockExecutor) Press(key string) error {
	m.pressed = append(m.pressed, key)
	return nil
}

func (m *mockExecutor) Scroll(direction string) error {
	m.scrolled = append(m.scrolled, direction)
	return nil
}

func (m *mockExecutor) Navigate(url string) error {
	m.url = url
	return nil
}

func (m *mockExecutor) GetURL() string {
	if m.url == "" {
		return "https://example.com"
	}
	return m.url
}

func (m *mockExecutor) GetPageContent() string {
	return m.content
}

func (m *mockExecutor) ExtractDOM() ([]DOMElement, error) {
	return m.domElements, nil
}

type mockVisionLLM struct {
	responses []string
	idx       int
}

func (m *mockVisionLLM) CompleteWithVision(prompt string, imageData []byte, mimeType string) (string, error) {
	if m.idx < len(m.responses) {
		resp := m.responses[m.idx]
		m.idx++
		return resp, nil
	}
	return `{"action": "click", "coordinates": [100, 200], "reasoning": "test"}`, nil
}

func (m *mockVisionLLM) CompleteText(prompt string) (string, error) {
	return m.CompleteWithVision(prompt, nil, "")
}

// --- Vision Tests ---

func TestVisionPlanAction(t *testing.T) {
	capturer := &mockCapturer{screenshots: [][]byte{[]byte("img1")}}
	executor := &mockExecutor{}
	llm := &mockVisionLLM{responses: []string{
		`{"action": "click", "coordinates": [150, 300], "reasoning": "Click submit button"}`,
	}}

	vision := NewVision(capturer, executor, llm)

	action, err := vision.PlanAction("Click the submit button")
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	if action.Action != "click" {
		t.Errorf("expected action 'click', got %q", action.Action)
	}
	if action.Coordinates[0] != 150 || action.Coordinates[1] != 300 {
		t.Errorf("expected coordinates [150, 300], got %v", action.Coordinates)
	}
}

func TestVisionExecuteAction_Click(t *testing.T) {
	capturer := &mockCapturer{}
	executor := &mockExecutor{}
	llm := &mockVisionLLM{}

	vision := NewVision(capturer, executor, llm)

	err := vision.ExecuteAction(&BrowserAction{
		Action:      "click",
		Coordinates: [2]float64{100, 200},
	})
	if err != nil {
		t.Fatalf("execute click failed: %v", err)
	}

	if len(executor.clicks) != 1 {
		t.Errorf("expected 1 click, got %d", len(executor.clicks))
	}
}

func TestVisionExecuteAction_Type(t *testing.T) {
	capturer := &mockCapturer{}
	executor := &mockExecutor{}
	llm := &mockVisionLLM{}

	vision := NewVision(capturer, executor, llm)

	err := vision.ExecuteAction(&BrowserAction{
		Action: "type",
		Text:   "hello world",
	})
	if err != nil {
		t.Fatalf("execute type failed: %v", err)
	}

	if len(executor.typed) != 1 || executor.typed[0] != "hello world" {
		t.Errorf("expected typed 'hello world', got %v", executor.typed)
	}
}

func TestVisionExecuteAction_Scroll(t *testing.T) {
	capturer := &mockCapturer{}
	executor := &mockExecutor{}
	llm := &mockVisionLLM{}

	vision := NewVision(capturer, executor, llm)

	err := vision.ExecuteAction(&BrowserAction{
		Action:    "scroll",
		Direction: "down",
	})
	if err != nil {
		t.Fatalf("execute scroll failed: %v", err)
	}

	if len(executor.scrolled) != 1 || executor.scrolled[0] != "down" {
		t.Errorf("expected scroll down, got %v", executor.scrolled)
	}
}

func TestVisionExecuteAction_Press(t *testing.T) {
	capturer := &mockCapturer{}
	executor := &mockExecutor{}
	llm := &mockVisionLLM{}

	vision := NewVision(capturer, executor, llm)

	err := vision.ExecuteAction(&BrowserAction{
		Action: "press",
		Key:    "Enter",
	})
	if err != nil {
		t.Fatalf("execute press failed: %v", err)
	}

	if len(executor.pressed) != 1 || executor.pressed[0] != "Enter" {
		t.Errorf("expected press Enter, got %v", executor.pressed)
	}
}

func TestVisionExecuteAction_Unknown(t *testing.T) {
	capturer := &mockCapturer{}
	executor := &mockExecutor{}
	llm := &mockVisionLLM{}

	vision := NewVision(capturer, executor, llm)

	err := vision.ExecuteAction(&BrowserAction{Action: "dance"})
	if err == nil {
		t.Error("expected error for unknown action")
	}
}

// --- Parse Action Tests ---

func TestParseAction_Valid(t *testing.T) {
	response := `{"action": "type", "text": "search query", "reasoning": "fill search box"}`

	action, err := parseAction(response)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if action.Action != "type" {
		t.Errorf("expected 'type', got %q", action.Action)
	}
	if action.Text != "search query" {
		t.Errorf("expected 'search query', got %q", action.Text)
	}
}

func TestParseAction_WithFences(t *testing.T) {
	response := "```json\n{\"action\": \"scroll\", \"direction\": \"down\"}\n```"

	action, err := parseAction(response)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if action.Action != "scroll" {
		t.Errorf("expected 'scroll', got %q", action.Action)
	}
}

func TestParseAction_EmptyAction(t *testing.T) {
	response := `{"reasoning": "something"}`

	_, err := parseAction(response)
	if err == nil {
		t.Error("expected error for missing action field")
	}
}

// --- Browser Agent Tests ---

func TestBrowserAgentExecute(t *testing.T) {
	capturer := &mockCapturer{screenshots: [][]byte{[]byte("img")}}
	executor := &mockExecutor{content: "Page content", domElements: []DOMElement{
		{Tag: "button", Text: "Submit", IsVisible: true},
	}}
	llm := &mockVisionLLM{responses: []string{
		`{"action": "click", "coordinates": [100, 200], "reasoning": "click submit"}`,
		`{"action": "extract", "reasoning": "goal achieved"}`,
	}}

	vision := NewVision(capturer, executor, llm)
	agent := NewBrowserAgent(vision)
	agent.SetMaxActions(5)

	result, err := agent.Execute(context.Background(), "Click submit and extract data", "https://example.com")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// Should succeed via extract action
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.ExtractedData != "Page content" {
		t.Errorf("expected extracted data, got %q", result.ExtractedData)
	}
}

func TestBrowserAgentMaxActions(t *testing.T) {
	capturer := &mockCapturer{screenshots: [][]byte{[]byte("img")}}
	executor := &mockExecutor{domElements: []DOMElement{}}
	llm := &mockVisionLLM{responses: []string{
		`{"action": "scroll", "direction": "down", "reasoning": "scroll"}`,
	}}

	vision := NewVision(capturer, executor, llm)
	agent := NewBrowserAgent(vision)
	agent.SetMaxActions(2)

	result, err := agent.Execute(context.Background(), "Find something", "https://example.com")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if result.Success {
		t.Error("should not succeed when max actions reached")
	}
	if result.ActionsTaken != 2 {
		t.Errorf("expected 2 actions, got %d", result.ActionsTaken)
	}
}

func TestBrowserAgentTimeout(t *testing.T) {
	capturer := &mockCapturer{screenshots: [][]byte{[]byte("img")}}
	executor := &mockExecutor{}
	llm := &mockVisionLLM{responses: []string{
		`{"action": "wait", "reasoning": "waiting"}`,
	}}

	vision := NewVision(capturer, executor, llm)
	agent := NewBrowserAgent(vision)
	agent.SetMaxActions(100)

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	result, err := agent.Execute(ctx, "Do something", "https://example.com")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if result.Success {
		t.Error("should not succeed on timeout")
	}
}

// --- DOM Agent Tests ---

func TestDOMAgentPlanAction(t *testing.T) {
	executor := &mockExecutor{
		domElements: []DOMElement{
			{Tag: "button", Text: "Search", Selector: "#search-btn", IsVisible: true},
			{Tag: "input", Text: "", Selector: "#query", IsVisible: true},
		},
	}
	llm := &mockVisionLLM{responses: []string{
		`{"action": "type", "selector": "#query", "text": "hello", "reasoning": "fill search"}`,
	}}

	agent := NewDOMAgent(executor, llm)

	action, err := agent.PlanAction("Search for hello")
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	if action.Action != "type" {
		t.Errorf("expected 'type', got %q", action.Action)
	}
	if action.Selector != "#query" {
		t.Errorf("expected selector '#query', got %q", action.Selector)
	}
}

func TestDOMAgentCanHandle(t *testing.T) {
	executor := &mockExecutor{
		domElements: []DOMElement{
			{Tag: "button", Text: "Click", IsVisible: true},
		},
	}
	llm := &mockVisionLLM{}

	agent := NewDOMAgent(executor, llm)
	if !agent.CanHandle() {
		t.Error("should be able to handle page with visible button")
	}
}

func TestDOMAgentCannotHandle(t *testing.T) {
	executor := &mockExecutor{
		domElements: []DOMElement{
			{Tag: "div", Text: "Just text", IsVisible: true},
		},
	}
	llm := &mockVisionLLM{}

	agent := NewDOMAgent(executor, llm)
	if agent.CanHandle() {
		t.Error("should not be able to handle page without interactive elements")
	}
}

// --- Desktop Agent Tests ---

func TestDesktopAgentPlatform(t *testing.T) {
	agent := NewDesktopAgent()
	if agent.Platform() == "" {
		t.Error("platform should not be empty")
	}
}

// --- Utility Tests ---

func TestBuildActionPrompt(t *testing.T) {
	prompt := buildActionPrompt("Click submit", "https://example.com", `[{"tag":"button","text":"Submit"}]`)

	if prompt == "" {
		t.Error("prompt should not be empty")
	}
	if !contains(prompt, "Click submit") {
		t.Error("prompt should contain instruction")
	}
	if !contains(prompt, "https://example.com") {
		t.Error("prompt should contain URL")
	}
}

func TestBuildDOMPrompt(t *testing.T) {
	prompt := buildDOMPrompt("Search for X", "https://example.com", "Button 'Search'")

	if !contains(prompt, "Search for X") {
		t.Error("prompt should contain instruction")
	}
	if !contains(prompt, "Button 'Search'") {
		t.Error("prompt should contain DOM description")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfStr(s, substr) >= 0
}

func indexOfStr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
