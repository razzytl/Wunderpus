package agent

import (
	"log/slog"

	"github.com/pkoukk/tiktoken-go"
	"github.com/wunderpus/wunderpus/internal/memory"
	"github.com/wunderpus/wunderpus/internal/provider"
)

// ContextManager manages conversation history with token-based truncation and sqlite persistence.
type ContextManager struct {
	messages  []provider.Message
	maxTokens int
	store     *memory.Store
	sessionID string
	tke       *tiktoken.Tiktoken
	encKey    []byte
}

// NewContextManager creates a new context manager.
func NewContextManager(maxTokens int, store *memory.Store, sessionID string, encKey []byte) *ContextManager {
	// Initialize tiktoken (using cl100k_base which is used by gpt-4/o)
	tke, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		slog.Error("failed to get tiktoken encoding, falling back to basic length counter", "error", err)
	}

	cm := &ContextManager{
		maxTokens: maxTokens,
		store:     store,
		sessionID: sessionID,
		tke:       tke,
		encKey:    encKey,
	}

	if store != nil && sessionID != "" {
		msgs, err := store.LoadSession(sessionID, encKey)
		if err == nil && len(msgs) > 0 {
			cm.messages = msgs
		}
	}

	return cm
}

// AddMessage appends a message to the conversation history.
func (c *ContextManager) AddMessage(role, content string) {
	msg := provider.Message{
		Role:    role,
		Content: content,
	}
	c.messages = append(c.messages, msg)
	if c.store != nil && c.sessionID != "" {
		_ = c.store.SaveMessage(c.sessionID, msg, c.encKey)
	}
	c.truncate()
}

// AddToolCallMessage appends an assistant message containing tool calls.
func (c *ContextManager) AddToolCallMessage(content string, toolCalls []provider.ToolCallInfo) {
	msg := provider.Message{
		Role:      provider.RoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}
	c.messages = append(c.messages, msg)
	if c.store != nil && c.sessionID != "" {
		_ = c.store.SaveMessage(c.sessionID, msg, c.encKey)
	}
	c.truncate()
}

// AddToolResultMessage appends a message containing the result of a tool execution.
func (c *ContextManager) AddToolResultMessage(toolCallID string, content string) {
	msg := provider.Message{
		Role:       provider.RoleTool,
		Content:    content,
		ToolCallID: toolCallID,
	}
	c.messages = append(c.messages, msg)
	if c.store != nil && c.sessionID != "" {
		_ = c.store.SaveMessage(c.sessionID, msg, c.encKey)
	}
	c.truncate()
}

// GetMessages returns all messages within the token budget.
func (c *ContextManager) GetMessages() []provider.Message {
	c.truncate()
	return c.messages
}

// NeedsSummarization returns true if the context is over 80% full.
func (c *ContextManager) NeedsSummarization() bool {
	return float64(c.totalTokens()) > float64(c.maxTokens)*0.8
}

// SummarizeOldest replaces the first N messages with a summary.
func (c *ContextManager) SummarizeOldest(summary string) {
	if len(c.messages) < 4 {
		return
	}
	// Keep the system prompt if we had one (usually handled by Agent.buildMessages)
	// But ContextManager only sees what was added to it.
	newMsgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "Summary of previous conversation: " + summary},
	}
	// Keep the last 2 messages for immediate continuity
	newMsgs = append(newMsgs, c.messages[len(c.messages)-2:]...)
	c.messages = newMsgs
}

// Clear removes all in-memory messages but doesn't delete from SQLite.
func (c *ContextManager) Clear() {
	c.messages = nil
}

// Count returns the number of messages.
func (c *ContextManager) Count() int {
	return len(c.messages)
}

// truncate removes oldest messages when the context exceeds the token limit.
func (c *ContextManager) truncate() {
	for c.totalTokens() > c.maxTokens && len(c.messages) > 2 {
		// Remove the oldest message (keep at least the last 2)
		c.messages = c.messages[1:]
		// We don't delete from the DB here; the DB is the full un-truncated history
	}
}

// totalTokens uses tiktoken to count tokens accurately if available.
func (c *ContextManager) totalTokens() int {
	total := 0
	for _, m := range c.messages {
		if c.tke != nil {
			total += len(c.tke.Encode(m.Content, nil, nil))
			// roughly 4 tokens for scaffolding
			total += 4
		} else {
			// Fallback heuristic if tiktoken failed to load
			total += len(m.Content)/4 + 4
		}
	}
	return total
}
