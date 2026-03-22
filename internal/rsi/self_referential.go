package rsi

import (
	"fmt"
	"strings"
)

// SelfReferentialConfig controls self-referential RSI capabilities.
type SelfReferentialConfig struct {
	Enabled bool // whether RSI can modify internal/rsi/ package

	// Track record requirements for enabling
	MinSuccessfulCycles int // minimum successful RSI cycles before unlock
	MaxRecentRollbacks  int // max rollbacks in last 5 cycles
}

// DefaultSelfReferentialConfig returns safe defaults (all disabled).
func DefaultSelfReferentialConfig() SelfReferentialConfig {
	return SelfReferentialConfig{
		Enabled:             false,
		MinSuccessfulCycles: 10,
		MaxRecentRollbacks:  0,
	}
}

// SelfReferentialFirewall extends the standard RSI firewall to allow
// self-referential modifications when explicitly enabled.
type SelfReferentialFirewall struct {
	config SelfReferentialConfig
}

// NewSelfReferentialFirewall creates a firewall with the given config.
func NewSelfReferentialFirewall(config SelfReferentialConfig) *SelfReferentialFirewall {
	return &SelfReferentialFirewall{config: config}
}

// CheckDiff validates a diff against the RSI firewall rules.
// Even when self-referential RSI is enabled, these paths are ALWAYS blocked:
// - cmd/ (main entrypoint)
// - The firewall itself (internal/rsi/self_referential.go)
func (f *SelfReferentialFirewall) CheckDiff(diff string) error {
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		if !strings.HasPrefix(line, "+++ ") && !strings.HasPrefix(line, "--- ") {
			continue
		}

		path := strings.TrimPrefix(line, "+++ ")
		path = strings.TrimPrefix(path, "--- ")
		path = strings.TrimPrefix(path, "a/")
		path = strings.TrimPrefix(path, "b/")

		// Always block: cmd/ directory
		if strings.HasPrefix(path, "cmd/") || strings.HasPrefix(path, "cmd\\") {
			return &FirewallError{
				Path:    path,
				Reason:  "cmd/ is ALWAYS blocked — main entrypoint cannot be modified",
				Blocked: true,
			}
		}

		// Always block: the firewall itself
		if strings.Contains(path, "self_referential") {
			return &FirewallError{
				Path:    path,
				Reason:  "the RSI firewall itself cannot be modified",
				Blocked: true,
			}
		}

		// Check RSI package modifications
		if strings.HasPrefix(path, "internal/rsi/") || strings.HasPrefix(path, "internal\\rsi\\") {
			if !f.config.Enabled {
				return &FirewallError{
					Path:    path,
					Reason:  "self-referential RSI is disabled — cannot modify internal/rsi/",
					Blocked: true,
				}
			}
			// Self-referential RSI is enabled — allow RSI package modifications
			continue
		}

		// Standard firewall: must be under internal/
		if !strings.HasPrefix(path, "internal/") && path != "/dev/null" {
			return &FirewallError{
				Path:    path,
				Reason:  "path is outside internal/ — blocked by RSI firewall",
				Blocked: true,
			}
		}
	}

	return nil
}

// FirewallError represents a firewall violation.
type FirewallError struct {
	Path    string
	Reason  string
	Blocked bool
}

func (e *FirewallError) Error() string {
	return fmt.Sprintf("RSI firewall: %s — %s", e.Path, e.Reason)
}

// IsEligibleForSelfReferential checks whether the agent's track record
// qualifies for self-referential RSI unlock.
func IsEligibleForSelfReferential(
	config SelfReferentialConfig,
	successfulCycles int,
	recentRollbacks int,
) bool {
	if config.Enabled {
		return true // already enabled
	}
	return successfulCycles >= config.MinSuccessfulCycles &&
		recentRollbacks <= config.MaxRecentRollbacks
}
