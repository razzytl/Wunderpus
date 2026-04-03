package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/tool"
)

// HumanResponse holds the response from a human
type HumanResponse struct {
	Response   string
	ReceivedAt time.Time
}

// HumanInTheLoop manages human intervention requests
type HumanInTheLoop struct {
	mu         sync.RWMutex
	pending    map[string]*PendingRequest // requestID -> pending request
	responseCh map[string]chan HumanResponse
}

// PendingRequest represents a request waiting for human input
type PendingRequest struct {
	ID         string
	Question   string
	ImageData  string // base64 encoded image if provided
	CreatedAt  time.Time
	ResponseCh chan HumanResponse
	Completed  bool
}

// NewHumanInTheLoop creates a new HITL manager
func NewHumanInTheLoop() *HumanInTheLoop {
	return &HumanInTheLoop{
		pending:    make(map[string]*PendingRequest),
		responseCh: make(map[string]chan HumanResponse),
	}
}

// AskHumanTool allows the agent to ask a human for help
type AskHumanTool struct {
	hitl *HumanInTheLoop
}

// NewAskHumanTool creates a new AskHuman tool
func NewAskHumanTool(hitl *HumanInTheLoop) *AskHumanTool {
	return &AskHumanTool{
		hitl: hitl,
	}
}

// Name returns the tool name
func (t *AskHumanTool) Name() string {
	return "ask_human"
}

// Description returns the tool description
func (t *AskHumanTool) Description() string {
	return `Ask a human for help when stuck. Use this tool when:
- You encounter a captcha or verification challenge
- You need clarification on a ambiguous task
- You need approval before proceeding with an action
- You encounter an error you cannot resolve

The agent will pause and wait for human response via Discord/Telegram.`
}

// Parameters returns the tool parameters
func (t *AskHumanTool) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{
			Name:        "question",
			Type:        "string",
			Description: "The question to ask the human",
			Required:    true,
		},
		{
			Name:        "image",
			Type:        "string",
			Description: "Optional base64 encoded screenshot or image to show the human (e.g., captcha)",
			Required:    false,
		},
		{
			Name:        "timeout_seconds",
			Type:        "number",
			Description: "Maximum time to wait for human response (default: 300 seconds)",
			Required:    false,
		},
	}
}

// Execute runs the ask_human tool
func (t *AskHumanTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	question, ok := args["question"].(string)
	if !ok || question == "" {
		return &tool.Result{Error: "question is required"}, nil
	}

	// Optional image
	imageData := ""
	if img, ok := args["image"].(string); ok {
		imageData = img
	}

	// Timeout (default 5 minutes)
	timeout := 300
	if to, ok := args["timeout_seconds"].(float64); ok {
		timeout = int(to)
	}

	// Create pending request
	reqID := fmt.Sprintf("hitl_%d", time.Now().UnixNano())
	responseCh := make(chan HumanResponse, 1)

	pending := &PendingRequest{
		ID:         reqID,
		Question:   question,
		ImageData:  imageData,
		CreatedAt:  time.Now(),
		ResponseCh: responseCh,
	}

	t.hitl.AddRequest(reqID, pending)

	// Wait for response with timeout
	select {
	case response := <-responseCh:
		t.hitl.RemoveRequest(reqID)
		return &tool.Result{
			Output: fmt.Sprintf("Human response: %s", response.Response),
		}, nil
	case <-time.After(time.Duration(timeout) * time.Second):
		t.hitl.RemoveRequest(reqID)
		return &tool.Result{
			Error: "Human response timeout - no response received within timeout period",
		}, nil
	case <-ctx.Done():
		t.hitl.RemoveRequest(reqID)
		return &tool.Result{
			Error: "Context cancelled while waiting for human response",
		}, nil
	}
}

// Sensitive returns whether this tool requires approval
func (t *AskHumanTool) Sensitive() bool {
	return false
}

// ApprovalLevel returns the policy-based approval level for this tool.
func (t *AskHumanTool) ApprovalLevel() tool.ApprovalLevel { return tool.AutoExecute }

// Version returns the tool version
func (t *AskHumanTool) Version() string {
	return "1.0.0"
}

// Dependencies returns tool dependencies
func (t *AskHumanTool) Dependencies() []string {
	return nil
}

// AddRequest adds a pending human request
func (h *HumanInTheLoop) AddRequest(id string, req *PendingRequest) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending[id] = req
	h.responseCh[id] = req.ResponseCh
}

// RemoveRequest removes a pending human request
func (h *HumanInTheLoop) RemoveRequest(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.pending, id)
	delete(h.responseCh, id)
}

// GetRequest gets a pending request by ID
func (h *HumanInTheLoop) GetRequest(id string) (*PendingRequest, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	req, ok := h.pending[id]
	return req, ok
}

// ListPending returns all pending requests
func (h *HumanInTheLoop) ListPending() []*PendingRequest {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*PendingRequest, 0, len(h.pending))
	for _, req := range h.pending {
		result = append(result, req)
	}
	return result
}

// SendResponse sends a human response to a pending request
func (h *HumanInTheLoop) SendResponse(id string, response string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	req, ok := h.pending[id]
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}

	if req.Completed {
		return fmt.Errorf("request already completed: %s", id)
	}

	req.Completed = true
	req.ResponseCh <- HumanResponse{
		Response:   response,
		ReceivedAt: time.Now(),
	}
	close(req.ResponseCh)
	return nil
}

// SendResponseByIndex sends a response by list index (for CLI/UI)
func (h *HumanInTheLoop) SendResponseByIndex(index int, response string) error {
	pending := h.ListPending()
	if index < 0 || index >= len(pending) {
		return fmt.Errorf("invalid index: %d", index)
	}
	return h.SendResponse(pending[index].ID, response)
}
