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
genesis:
  rsi_enabled: false
  trust_budget_max: 500
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

	if cfg.Genesis.TrustBudgetMax != 500 {
		t.Fatalf("expected 500, got %d", cfg.Genesis.TrustBudgetMax)
	}

	// Modify the config file
	updated := `version: 2
default_provider: openai
genesis:
  rsi_enabled: true
  trust_budget_max: 800
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

	if latestCfg.Genesis.TrustBudgetMax != 800 {
		t.Fatalf("expected reloaded value 800, got %d", latestCfg.Genesis.TrustBudgetMax)
	}

	if !latestCfg.Genesis.RSIEnabled {
		t.Fatal("expected RSIEnabled to be true after reload")
	}
}
