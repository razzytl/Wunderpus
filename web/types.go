package web

import "time"

// WebSocket message types used for communication between
// the React frontend and Go backend.
const (
	// Client → Server
	MsgTypeUserMessage  = "user_message"
	MsgTypeBranchSwitch = "branch_switch"
	MsgTypeListBranches = "list_branches"

	// Server → Client
	MsgTypeChatToken           = "chat_token"
	MsgTypeChatComplete        = "chat_complete"
	MsgTypeToolExecutionStart  = "tool_execution_start"
	MsgTypeToolExecutionResult = "tool_execution_result"
	MsgTypeSystemLog           = "system_log"
	MsgTypeError               = "error"
)

// WSMessage is the envelope for all WebSocket messages.
type WSMessage struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id,omitempty"`
	Payload   any       `json:"payload"`
}

// --- Client → Server Payloads ---

// UserMessagePayload is sent by the frontend when the user submits a message.
type UserMessagePayload struct {
	Content   string `json:"content"`
	SessionID string `json:"session_id,omitempty"`
	BranchID  string `json:"branch_id,omitempty"`
}

// BranchSwitchPayload is sent by the frontend to switch the active branch.
type BranchSwitchPayload struct {
	SessionID string `json:"session_id"`
	BranchID  string `json:"branch_id"`
}

// --- Server → Client Payloads ---

// ChatTokenPayload represents a streaming text chunk from the LLM.
type ChatTokenPayload struct {
	Token    string `json:"token"`
	Done     bool   `json:"done"`
	BranchID string `json:"branch_id,omitempty"`
}

// ChatCompletePayload is sent when the full response is assembled.
type ChatCompletePayload struct {
	Content    string `json:"content"`
	TokenCount int    `json:"token_count,omitempty"`
	BranchID   string `json:"branch_id,omitempty"`
}

// ToolExecutionStartPayload is sent when the agent invokes a tool/skill.
type ToolExecutionStartPayload struct {
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	RequestID string         `json:"request_id"`
}

// ToolExecutionResultPayload is sent when a tool/skill completes.
type ToolExecutionResultPayload struct {
	ToolName  string `json:"tool_name"`
	RequestID string `json:"request_id"`
	Success   bool   `json:"success"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
	Duration  string `json:"duration,omitempty"`
}

// SystemLogPayload carries backend status/log messages.
type SystemLogPayload struct {
	Level   string `json:"level"` // "info", "warn", "error"
	Message string `json:"message"`
}

// ErrorPayload is sent on errors.
type ErrorPayload struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}
