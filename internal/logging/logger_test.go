package logging

import (
	"context"
	"testing"
)

func TestInitDebug(t *testing.T) {
	// Should not panic with valid inputs
	Init("debug", "json", "stderr")
	Init("DEBUG", "JSON", "STDOUT")
	Init("debug", "text", "stdout")
}

func TestInitInfo(t *testing.T) {
	Init("info", "json", "stderr")
	Init("INFO", "JSON", "STDERR")
}

func TestInitWarn(t *testing.T) {
	Init("warn", "json", "stderr")
	Init("WARN", "text", "stdout")
}

func TestInitError(t *testing.T) {
	Init("error", "json", "stderr")
	Init("ERROR", "text", "stdout")
}

func TestInitInvalidLevel(t *testing.T) {
	// Invalid levels should default to info
	Init("invalid", "json", "stderr")
	Init("", "json", "stderr")
}

func TestInitInvalidFormat(t *testing.T) {
	// Invalid formats should default to JSON
	Init("info", "invalid", "stderr")
	Init("info", "", "stderr")
}

func TestInitInvalidOutput(t *testing.T) {
	// Invalid outputs should default to stderr
	Init("info", "json", "invalid")
	Init("info", "json", "")
}

func TestInitTextFormat(t *testing.T) {
	Init("info", "text", "stderr")
}

func TestInitStdoutOutput(t *testing.T) {
	Init("info", "json", "stdout")
}

func TestWithCorrelation(t *testing.T) {
	logger := WithCorrelation("test-id-123")
	if logger == nil {
		t.Fatal("WithCorrelation should not return nil")
	}
}

func TestContextWithCorrelation(t *testing.T) {
	ctx := context.Background()
	id := "correlation-123"

	ctx = ContextWithCorrelation(ctx, id)

	got := ctx.Value(correlationIDKey)
	if got != id {
		t.Errorf("Expected %q, got %v", id, got)
	}
}

func TestLWithCorrelation(t *testing.T) {
	ctx := context.Background()
	id := "correlation-456"

	ctx = ContextWithCorrelation(ctx, id)
	logger := L(ctx)

	if logger == nil {
		t.Fatal("L should not return nil")
	}
}

func TestLWithoutCorrelation(t *testing.T) {
	ctx := context.Background()
	logger := L(ctx)

	if logger == nil {
		t.Fatal("L should not return nil")
	}
}

func TestCorrelationIDKey(t *testing.T) {
	// Verify the key is properly defined
	key := correlationIDKey
	if string(key) != "correlation_id" {
		t.Errorf("Expected correlation_id key, got %s", string(key))
	}
}

func TestMultipleCorrelationIDs(t *testing.T) {
	ctx := context.Background()

	// Add multiple correlations
	ctx = ContextWithCorrelation(ctx, "first")
	ctx = ContextWithCorrelation(ctx, "second")

	// Should have the last one
	got := ctx.Value(correlationIDKey)
	if got != "second" {
		t.Errorf("Expected 'second', got %v", got)
	}
}
