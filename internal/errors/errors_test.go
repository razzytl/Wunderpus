package errors

import (
	"errors"
	"testing"
)

func TestWunderpusError_Error(t *testing.T) {
	e := &WunderpusError{
		Type:    ProviderError,
		Message: "test error",
	}

	if e.Error() != "[PROVIDER] test error" {
		t.Errorf("expected [PROVIDER] test error, got %s", e.Error())
	}
}

func TestWunderpusError_WithErr(t *testing.T) {
	inner := errors.New("inner error")
	e := &WunderpusError{
		Type:    ProviderError,
		Message: "test error",
		Err:     inner,
	}

	if e.Error() != "[PROVIDER] test error: inner error" {
		t.Errorf("unexpected error string: %s", e.Error())
	}
}

func TestWunderpusError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	e := &WunderpusError{
		Type:    ProviderError,
		Message: "test error",
		Err:     inner,
	}

	if !errors.Is(e, inner) {
		t.Error("expected unwrap to return inner error")
	}
}

func TestNew(t *testing.T) {
	e := New(ProviderError, "test error")
	if !IsType(e, ProviderError) {
		t.Error("expected provider error type")
	}
}

func TestWrap(t *testing.T) {
	inner := errors.New("inner error")
	e := Wrap(ProviderError, "wrapped", inner)

	if !IsType(e, ProviderError) {
		t.Error("expected provider error type")
	}
	if !errors.Is(e, inner) {
		t.Error("expected inner error to be wrapped")
	}
}

func TestMarkRetryable(t *testing.T) {
	e := New(ProviderError, "test error")
	retryable := MarkRetryable(e)

	if !IsRetryable(retryable) {
		t.Error("expected error to be retryable")
	}
}

func TestIsRetryable_NilError(t *testing.T) {
	if IsRetryable(nil) {
		t.Error("expected nil to not be retryable")
	}
}

func TestIsType_NilError(t *testing.T) {
	if IsType(nil, ProviderError) {
		t.Error("expected nil to not match type")
	}
}
