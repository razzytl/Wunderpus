package agent

import (
	"testing"

	"github.com/wonderpus/wonderpus/internal/provider"
)

func TestAgentCreation(t *testing.T) {
	agent := &Agent{
		sessionID: "test-session",
		sysPrompt: "You are helpful.",
		ctx:       NewContextManager(4096, nil, "test", nil),
	}

	if agent.sessionID != "test-session" {
		t.Errorf("expected test-session, got %s", agent.sessionID)
	}
	if agent.sysPrompt != "You are helpful." {
		t.Errorf("expected You are helpful., got %s", agent.sysPrompt)
	}
}

func TestContextManager(t *testing.T) {
	t.Run("add_and_count", func(t *testing.T) {
		ctx := NewContextManager(4096, nil, "test", nil)

		ctx.AddMessage(provider.RoleUser, "Hello")
		if ctx.Count() != 1 {
			t.Errorf("expected 1, got %d", ctx.Count())
		}

		ctx.AddMessage(provider.RoleAssistant, "Hi there!")
		if ctx.Count() != 2 {
			t.Errorf("expected 2, got %d", ctx.Count())
		}
	})

	t.Run("clear", func(t *testing.T) {
		ctx := NewContextManager(4096, nil, "test", nil)

		ctx.AddMessage(provider.RoleUser, "Hello")
		ctx.Clear()

		if ctx.Count() != 0 {
			t.Errorf("expected 0, got %d", ctx.Count())
		}
	})

	t.Run("tool_call_message", func(t *testing.T) {
		ctx := NewContextManager(4096, nil, "test", nil)

		toolCalls := []provider.ToolCallInfo{
			{ID: "call_1", Type: "function", Function: provider.ToolCallFunc{Name: "test_tool", Arguments: "{}"}},
		}

		ctx.AddToolCallMessage("test call", toolCalls)
		if ctx.Count() != 1 {
			t.Errorf("expected 1, got %d", ctx.Count())
		}
	})

	t.Run("tool_result_message", func(t *testing.T) {
		ctx := NewContextManager(4096, nil, "test", nil)

		ctx.AddToolResultMessage("call_1", "tool result")
		if ctx.Count() != 1 {
			t.Errorf("expected 1, got %d", ctx.Count())
		}
	})
}

func TestBuildMessages(t *testing.T) {
	agent := &Agent{
		sysPrompt: "System prompt",
		ctx:       NewContextManager(4096, nil, "test", nil),
	}

	agent.ctx.AddMessage(provider.RoleUser, "User message")
	agent.ctx.AddMessage(provider.RoleAssistant, "Assistant response")

	messages := agent.buildMessages()

	if len(messages) < 3 {
		t.Errorf("expected at least 3 messages, got %d", len(messages))
	}
	if len(messages) > 0 && messages[0].Role != provider.RoleSystem {
		t.Errorf("expected first message to be system, got %s", messages[0].Role)
	}
}

func TestToolCallback(t *testing.T) {
	agent := &Agent{}

	var called bool
	var name string
	var args map[string]any
	var result string

	agent.SetToolCallback(func(n string, a map[string]any, r string) {
		called = true
		name = n
		args = a
		result = r
	})

	agent.toolFunc("tool_name", map[string]any{"key": "value"}, "output")

	if !called {
		t.Error("expected callback to be called")
	}
	if name != "tool_name" {
		t.Errorf("expected tool_name, got %s", name)
	}
	if args["key"] != "value" {
		t.Errorf("expected value, got %v", args["key"])
	}
	if result != "output" {
		t.Errorf("expected output, got %s", result)
	}
}

func TestEncryptionKey(t *testing.T) {
	t.Run("valid_base64_key", func(t *testing.T) {
		agent := &Agent{}
		agent.SetEncryptionKey("YK+rgohyGJEmiyS6QJAv8/gYFKVoP0jVhnZSSCZA2hY=")

		if agent.encryptionKey == nil {
			t.Error("expected encryptionKey to be set")
		}
		if len(agent.encryptionKey) != 32 {
			t.Errorf("expected 32 bytes, got %d", len(agent.encryptionKey))
		}
	})

	t.Run("invalid_key", func(t *testing.T) {
		agent := &Agent{}
		agent.SetEncryptionKey("not-valid-base64")

		if agent.encryptionKey != nil {
			t.Error("expected encryptionKey to be nil")
		}
	})

	t.Run("empty_key", func(t *testing.T) {
		agent := &Agent{}
		agent.SetEncryptionKey("")

		if agent.encryptionKey != nil {
			t.Error("expected encryptionKey to be nil")
		}
	})
}
