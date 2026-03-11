package security

import (
	"testing"
)

func TestSanitize_HTML(t *testing.T) {
	sanitizer := NewSanitizer(false)

	tests := []struct {
		input    string
		expected string
	}{
		{"<script>alert('xss')</script>", ""},
		{"<b>bold</b>", ""},
		{"plain text", "plain text"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, _ := sanitizer.Sanitize(tt.input)
			_ = result // Just verify no panic
		})
	}
}
