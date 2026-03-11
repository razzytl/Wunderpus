package tool

import (
	"context"
	"testing"
	"time"
)

func BenchmarkExecutor_AnalyticsRecord(b *testing.B) {
	analytics := &Analytics{
		stats: make(map[string]*ToolStats),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analytics.record("test-tool", time.Millisecond*50, false)
	}
}

func BenchmarkToolRegistry_Get(b *testing.B) {
	registry := NewRegistry()

	for i := 0; i < 50; i++ {
		mockTool := &mockTool{name: "tool-" + string(rune(i))}
		_ = registry.Register(mockTool)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.Get("tool-25")
	}
}

func BenchmarkToolRegistry_List(b *testing.B) {
	registry := NewRegistry()

	for i := 0; i < 50; i++ {
		mockTool := &mockTool{name: "tool-" + string(rune(i))}
		_ = registry.Register(mockTool)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.List()
	}
}

func BenchmarkToolRegistry_Count(b *testing.B) {
	registry := NewRegistry()

	for i := 0; i < 100; i++ {
		mockTool := &mockTool{name: "tool-" + string(rune(i))}
		_ = registry.Register(mockTool)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.Count()
	}
}

type mockTool struct {
	name string
}

func (m *mockTool) Name() string               { return m.name }
func (m *mockTool) Description() string        { return "mock tool" }
func (m *mockTool) Parameters() []ParameterDef { return nil }
func (m *mockTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	return &Result{Output: "ok"}, nil
}
func (m *mockTool) Sensitive() bool        { return false }
func (m *mockTool) Version() string        { return "1.0" }
func (m *mockTool) Dependencies() []string { return nil }
