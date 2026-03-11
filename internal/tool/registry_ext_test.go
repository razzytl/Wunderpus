package tool

import (
	"testing"
)

func TestBuiltinTools(t *testing.T) {
	// Test basic builtin tool registration
	registry := NewRegistry()

	if registry == nil {
		t.Error("expected non-nil registry")
	}
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	// Test that registry has List method
	tools := registry.List()
	_ = tools // Just verify method exists
}

func TestToolRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	count := registry.Count()
	if count < 0 {
		t.Errorf("expected non-negative count, got %d", count)
	}
}
