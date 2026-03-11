package constants

import "time"

const (
	// Version is the current version of Wunderpus
	Version = "0.2.0"

	// Default timeouts
	DefaultTimeout      = 2 * time.Minute
	DefaultShortTimeout = 30 * time.Second
	DefaultLongTimeout  = 10 * time.Minute

	// Context and Message limits
	MaxInputLength    = 4000
	MaxResponseLength = 32000
	DefaultMaxHistory = 20

	// Agency logic
	MaxIterations = 5
)

// Model names
const (
	ModelGPT4o         = "gpt-4o"
	ModelGPT4v         = "gpt-4-vision-preview"
	ModelClaude3Opus   = "claude-3-opus-20240229"
	ModelClaude3Sonnet = "claude-3-sonnet-20240229"
	ModelGeminiPro     = "gemini-pro"
	ModelLlama3        = "llama3"
)
