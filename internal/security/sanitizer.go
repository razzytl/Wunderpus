package security

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Threat describes a detected prompt injection attempt.
type Threat struct {
	Pattern     string
	Description string
	Severity    string // low, medium, high
}

// injectionPatterns are compiled regexes for common prompt injection techniques.
var injectionPatterns = []struct {
	re          *regexp.Regexp
	description string
	severity    string
}{
	{
		re:          regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|prior|above)\s+(instructions?|prompts?|context)`),
		description: "Attempt to override system instructions",
		severity:    "high",
	},
	{
		re:          regexp.MustCompile(`(?i)you\s+are\s+now\s+(a|an|in)\s+`),
		description: "Role reassignment attempt",
		severity:    "high",
	},
	{
		re:          regexp.MustCompile(`(?i)(system|assistant)\s*:\s*`),
		description: "System/assistant role injection",
		severity:    "medium",
	},
	{
		re:          regexp.MustCompile(`(?i)reveal\s+(your|the)\s+(system\s+)?prompt`),
		description: "System prompt extraction attempt",
		severity:    "medium",
	},
	{
		re:          regexp.MustCompile(`(?i)pretend\s+(you('re|\s+are)\s+)?(not\s+)?(an?\s+)?ai`),
		description: "Identity override attempt",
		severity:    "medium",
	},
	{
		re:          regexp.MustCompile(`(?i)disregard\s+(all\s+)?(safety|security|guidelines|rules)`),
		description: "Safety bypass attempt",
		severity:    "high",
	},
	{
		re:          regexp.MustCompile(`(?i)\[INST\]|\[\/INST\]|<<SYS>>|<\|im_start\|>`),
		description: "Raw prompt template injection",
		severity:    "high",
	},
	{
		re:          regexp.MustCompile(`(?i)(thought|observation|action|action\s+input)\s*:\s*`),
		description: "Chain-of-thought/ReAct pattern injection",
		severity:    "medium",
	},
	{
		re:          regexp.MustCompile(`(?i)translation\s+to\s+l33t|gibberish|base64`),
		description: "Obfuscated payload attempt",
		severity:    "low",
	},
}

// Sanitizer checks and cleans user input.
type Sanitizer struct {
	enabled bool
}

// NewSanitizer creates a new input sanitizer.
func NewSanitizer(enabled bool) *Sanitizer {
	return &Sanitizer{enabled: enabled}
}

// Sanitize checks input for prompt injection patterns.
// Returns the cleaned input and any detected threats.
func (s *Sanitizer) Sanitize(input string) (string, []Threat) {
	if !s.enabled {
		return input, nil
	}

	// 1. Normalize Unicode (NFC)
	normalized := norm.NFC.String(input)

	var threats []Threat
	cleaned := normalized

	// 2. Check for injection patterns
	for _, p := range injectionPatterns {
		if p.re.MatchString(normalized) {
			threats = append(threats, Threat{
				Pattern:     p.re.String(),
				Description: p.description,
				Severity:    p.severity,
			})
		}
	}

	// 3. Strip null bytes and control chars (except newlines and tabs)
	cleaned = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		if r < 32 {
			return -1
		}
		return r
	}, cleaned)

	return cleaned, threats
}

// LimitLength truncates input if it exceeds the maximum allowed length.
func (s *Sanitizer) LimitLength(input string, max int) (string, bool) {
	if len(input) <= max {
		return input, false
	}
	return input[:max], true
}

// HasHighSeverity returns true if any threat is high severity.
func HasHighSeverity(threats []Threat) bool {
	for _, t := range threats {
		if t.Severity == "high" {
			return true
		}
	}
	return false
}
