package agent

import (
	"testing"

	"github.com/wunderpus/wunderpus/internal/provider"
)

func BenchmarkContextManager_AddMessage(b *testing.B) {
	cm := NewContextManager(8000, nil, "test-session", nil)

	longContent := "This is a test message with enough content to matter. " +
		"It contains multiple sentences to simulate real usage. " +
		"We need to see how fast the context manager can add messages. "

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.AddMessage("user", longContent)
	}
}

func BenchmarkContextManager_GetMessages(b *testing.B) {
	cm := NewContextManager(8000, nil, "test-session", nil)

	for i := 0; i < 50; i++ {
		cm.AddMessage("user", "Test message content for benchmarking context manager performance")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.GetMessages()
	}
}

func BenchmarkContextManager_totalTokens(b *testing.B) {
	cm := NewContextManager(8000, nil, "test-session", nil)

	for i := 0; i < 30; i++ {
		cm.AddMessage("user", "This is a test message that has some content for token counting benchmarks")
		cm.AddMessage("assistant", "And this is a response message with additional content for context")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.totalTokens()
	}
}

func BenchmarkContextManager_NeedsSummarization(b *testing.B) {
	cm := NewContextManager(1000, nil, "test-session", nil)

	for i := 0; i < 20; i++ {
		cm.AddMessage("user", "Short message")
		cm.AddMessage("assistant", "Short response")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.NeedsSummarization()
	}
}

func BenchmarkContextManager_Count(b *testing.B) {
	cm := NewContextManager(8000, nil, "test-session", nil)

	for i := 0; i < 100; i++ {
		cm.AddMessage("user", "Message content")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.Count()
	}
}

func BenchmarkMessageBuilding(b *testing.B) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: "You are a helpful assistant."},
		{Role: provider.RoleUser, Content: "Hello, how are you?"},
		{Role: provider.RoleAssistant, Content: "I'm doing well, thank you for asking!"},
		{Role: provider.RoleUser, Content: "Can you help me with something?"},
		{Role: provider.RoleAssistant, Content: "Of course! What would you like help with?"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := make([]provider.Message, len(messages))
		copy(result, messages)
		_ = result
	}
}
