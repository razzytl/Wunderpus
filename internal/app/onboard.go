package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/wunderpus/wunderpus/internal/config"
	"gopkg.in/yaml.v3"
)

// Onboard runs the interactive configuration wizard.
func Onboard(configPath string) error {
	fmt.Println("=== Wunderpus Onboarding ===")
	fmt.Println("Welcome! Let's set up your agentic assistant.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// 1. Providers
	fmt.Println("--- Provider Setup ---")
	openAIKey := prompt(reader, "Enter OpenAI API Key (optional): ", "")
	anthropicKey := prompt(reader, "Enter Anthropic API Key (optional): ", "")
	geminiKey := prompt(reader, "Enter Gemini API Key (optional): ", "")

	// 2. Workspace
	fmt.Println("\n--- Workspace Setup ---")
	workspace := prompt(reader, "Enter workspace path (default: .): ", ".")

	// 3. Channels
	fmt.Println("\n--- Channel Setup ---")
	enableTelegram := promptBool(reader, "Enable Telegram? (y/n): ", false)
	var telegramToken string
	if enableTelegram {
		telegramToken = prompt(reader, "Enter Telegram Bot Token: ", "")
	}

	// Create config
	cfg := &config.Config{}
	// Load defaults first
	// Note: config.Load uses internal helper, but we can manually populate for now
	// or use a helper if we refactor config further.
	// For MVP, we'll just write a basic config.

	cfg.Providers.OpenAI.APIKey = openAIKey
	cfg.Providers.OpenAI.Model = "gpt-4o"
	cfg.Providers.Anthropic.APIKey = anthropicKey
	cfg.Providers.Anthropic.Model = "claude-3-5-sonnet-20240620"
	cfg.Providers.Gemini.APIKey = geminiKey
	cfg.Providers.Gemini.Model = "gemini-1.5-pro"

	cfg.Agents.Defaults.Workspace = workspace
	cfg.Agents.Defaults.RestrictToWorkspace = true

	cfg.Channels.Telegram.Enabled = enableTelegram
	cfg.Channels.Telegram.Token = telegramToken

	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"
	cfg.Logging.Output = "stderr"

	// Save to file
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\nSuccess! Configuration saved to %s\n", configPath)
	fmt.Println("You can now run 'wunderpus agent' to start.")
	return nil
}

func prompt(reader *bufio.Reader, label, defaultValue string) string {
	fmt.Print(label)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func promptBool(reader *bufio.Reader, label string, defaultValue bool) bool {
	input := prompt(reader, label, "")
	if input == "" {
		return defaultValue
	}
	input = strings.ToLower(input)
	return input == "y" || input == "yes" || input == "true"
}
