package agent

import (
	"github.com/wonderpus/wonderpus/internal/provider"
)

// ContextManager manages conversation history with token-based truncation.
type ContextManager struct {
	messages  []provider.Message
	maxTokens int
}

// NewContextManager creates a new context manager.
func NewContextManager(maxTokens int) *ContextManager {
	return &ContextManager{
		maxTokens: maxTokens,
	}
}

// AddMessage appends a message to the conversation history.
func (c *ContextManager) AddMessage(role, content string) {
	c.messages = append(c.messages, provider.Message{
		Role:    role,
		Content: content,
	})
	c.truncate()
}

// GetMessages returns all messages within the token budget.
func (c *ContextManager) GetMessages() []provider.Message {
	return c.messages
}

// Clear removes all messages.
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
	}
}

// totalTokens estimates token count across all messages.
// Uses a simple heuristic: ~4 chars per token.
func (c *ContextManager) totalTokens() int {
	total := 0
	for _, m := range c.messages {
		// ~4 chars per token + overhead per message
		total += len(m.Content)/4 + 4
	}
	return total
}
