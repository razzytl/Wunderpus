package errors

import (
	"fmt"
)

// ErrorType defines the category of the error.
type ErrorType string

const (
	ConfigError   ErrorType = "CONFIG"
	ProviderError ErrorType = "PROVIDER"
	SecurityError ErrorType = "SECURITY"
	ToolError     ErrorType = "TOOL"
	InternalError ErrorType = "INTERNAL"
	ChannelError  ErrorType = "CHANNEL"
	RateLimitError ErrorType = "RATE_LIMIT"
	TimeoutError   ErrorType = "TIMEOUT"
	MemoryError    ErrorType = "MEMORY"
)

// WunderpusError represents a structured error in the system.
type WunderpusError struct {
	Type      ErrorType
	Message   string
	Err       error
	Retryable bool
}

func (e *WunderpusError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

func (e *WunderpusError) Unwrap() error {
	return e.Err
}

// New creates a new structured error.
func New(t ErrorType, msg string) error {
	return &WunderpusError{
		Type:    t,
		Message: msg,
	}
}

// Wrap wraps an existing error with a type and message.
func Wrap(t ErrorType, msg string, err error) error {
	return &WunderpusError{
		Type:    t,
		Message: msg,
		Err:     err,
	}
}

// MarkRetryable marks an error as retryable.
func MarkRetryable(err error) error {
	if we, ok := err.(*WunderpusError); ok {
		we.Retryable = true
		return we
	}
	return &WunderpusError{
		Type:      InternalError,
		Message:   "Retryable error",
		Err:       err,
		Retryable: true,
	}
}

// IsRetryable checks if an error has been marked as retryable.
func IsRetryable(err error) bool {
	if we, ok := err.(*WunderpusError); ok {
		return we.Retryable
	}
	return false
}

// IsType checks if an error is of a specific WunderpusError type.
func IsType(err error, t ErrorType) bool {
	if we, ok := err.(*WunderpusError); ok {
		return we.Type == t
	}
	return false
}
