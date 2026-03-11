package provider

import (
	"errors"
	"testing"
	"time"
)

func BenchmarkCooldownTracker_IsInCooldown(b *testing.B) {
	tracker := NewCooldownTracker()
	tracker.StartCooldown("test-provider", 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.IsInCooldown("test-provider")
	}
}

func BenchmarkCooldownTracker_RecordFailure(b *testing.B) {
	tracker := NewCooldownTracker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.RecordFailure("test-provider")
	}
}

func BenchmarkErrorClassifier_Classify(b *testing.B) {
	classifier := &ErrorClassifier{}
	testErrors := []error{
		errors.New("rate limit exceeded"),
		errors.New("timeout waiting for response"),
		errors.New("500 internal server error"),
		errors.New("quota exceeded"),
		errors.New("invalid request parameter"),
		errors.New("connection refused"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, err := range testErrors {
			classifier.Classify(err)
		}
	}
}

func BenchmarkErrorClassifier_IsRetriable(b *testing.B) {
	classifier := &ErrorClassifier{}
	reasons := []FailoverReason{
		FailoverReasonNone,
		FailoverReasonRateLimit,
		FailoverReasonTimeout,
		FailoverReasonServerError,
		FailoverReasonQuotaExceeded,
		FailoverReasonInvalidRequest,
		FailoverReasonRetriableError,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, reason := range reasons {
			classifier.IsRetriable(reason)
		}
	}
}

func BenchmarkCooldownTracker_RecordSuccess(b *testing.B) {
	tracker := NewCooldownTracker()
	tracker.StartCooldown("test-provider", 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.RecordSuccess("test-provider")
	}
}
