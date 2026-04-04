package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfig_WatchRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Write initial config
	initial := `version: 2
default_provider: openai
agents:
  defaults:
    restrict_to_workspace: true
`
	os.WriteFile(cfgPath, []byte(initial), 0o644)

	var reloadCount int
	var latestCfg *Config

	cfg, cleanup, err := Watch(cfgPath, func(newCfg *Config) {
		reloadCount++
		latestCfg = newCfg
	})
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}
	defer cleanup()

	if !cfg.Agents.Defaults.RestrictToWorkspace {
		t.Fatalf("expected RestrictToWorkspace true, got false")
	}

	// Modify the config file
	updated := `version: 2
default_provider: openai
agents:
  defaults:
    restrict_to_workspace: false
`
	os.WriteFile(cfgPath, []byte(updated), 0o644)

	// Wait for reload
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if reloadCount > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if reloadCount == 0 {
		t.Fatal("config was not reloaded after file change")
	}

	if latestCfg.Agents.Defaults.RestrictToWorkspace {
		t.Fatalf("expected reloaded RestrictToWorkspace false, got true")
	}
}
