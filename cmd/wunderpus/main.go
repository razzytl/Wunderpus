package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/wunderpus/wunderpus/internal/app"
	"github.com/wunderpus/wunderpus/internal/skills"
	"github.com/wunderpus/wunderpus/internal/tui"
)

var (
	configPath string
	verbose    bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "wunderpus",
	Short: "Wunderpus - Your expert agentic AI assistant",
	Long: `Wunderpus is a powerful, vendor-agnostic AI agent framework with 
security sandboxing, skills system, and multi-channel support.`,
	Run: func(cmd *cobra.Command, args []string) {
		runAgent(cmd, args)
	},
}

func init() {
	if envConfig := os.Getenv("WUNDERPUS_CONFIG"); envConfig != "" {
		configPath = envConfig
	} else {
		configPath = "config.yaml"
	}
	rootCmd.PersistentFlags().StringVar(&configPath, "config", configPath, "path to config file (or set WUNDERPUS_CONFIG env var)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")

	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(onboardCmd)
	rootCmd.AddCommand(cronCmd)
	rootCmd.AddCommand(skillsCmd)
	rootCmd.AddCommand(authCmd)

	agentCmd.Flags().StringP("message", "m", "", "one-shot message to agent")

	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronAddCmd)
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsInstallCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLoginCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Wunderpus status",
	Run: func(cmd *cobra.Command, args []string) {
		application, err := app.Bootstrap(configPath, verbose)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer application.Close()

		fmt.Printf("Wunderpus Status\n")
		fmt.Printf("Workspace: %s\n", application.Config.Agents.Defaults.Workspace)
		fmt.Printf("Providers: %v\n", application.Config.AvailableProviders())
		fmt.Printf("DefaultProvider: %s\n", application.Config.DefaultProvider)

		fmt.Printf("Uptime: %v\n", time.Since(application.HealthServer.StartTime))
	},
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run the Wunderpus agent",
	Run: func(cmd *cobra.Command, args []string) {
		msg, _ := cmd.Flags().GetString("message")
		if msg != "" {
			runOneShot(msg)
			return
		}
		runAgent(cmd, args)
	},
}

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Start the Wunderpus gateway (background services)",
	Run: func(cmd *cobra.Command, args []string) {
		application, err := app.Bootstrap(configPath, verbose)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer application.Close()

		fmt.Println("Wunderpus Gateway starting...")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start channels
		for _, ch := range application.Channels {
			if err := ch.Start(ctx); err != nil {
				fmt.Printf("Failed to start channel %s: %v\n", ch.Name(), err)
			} else {
				fmt.Printf("Channel %s started\n", ch.Name())
			}
		}

		// Start heartbeat
		if err := application.HeartbeatScheduler.Start(ctx); err != nil {
			fmt.Printf("Heartbeat scheduler failed: %v\n", err)
		}

		fmt.Println("Gateway running. Press Ctrl+C to stop.")
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		fmt.Println("Shutting down gateway...")
	},
}

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Interactive onboarding and configuration setup",
	Run: func(cmd *cobra.Command, args []string) {
		if err := app.Onboard(configPath); err != nil {
			fmt.Printf("Onboarding error: %v\n", err)
		}
	},
}

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage periodic (heartbeat) tasks",
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scheduled tasks in HEARTBEAT.md",
	Run: func(cmd *cobra.Command, args []string) {
		application, err := app.Bootstrap(configPath, verbose)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer application.Close()

		status := application.HeartbeatScheduler.GetStatus()
		fmt.Printf("Periodic Tasks Status: %v\n", status)
		fmt.Printf("Interval: %v minutes\n", status["interval"])
		fmt.Printf("Quick Tasks: %v\n", status["quick_tasks"])
		fmt.Printf("Long Tasks: %v\n", status["long_tasks"])
	},
}

var cronAddCmd = &cobra.Command{
	Use:   "add [task description]",
	Short: "Add a new periodic task to HEARTBEAT.md",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		application, err := app.Bootstrap(configPath, verbose)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer application.Close()

		taskDesc := args[0]
		workspace := application.Config.Agents.Defaults.Workspace
		heartbeatPath := workspace + "/HEARTBEAT.md"

		content := fmt.Sprintf("\n## %s\n- [ ] %s\n", time.Now().Format("2006-01-02 15:04"), taskDesc)

		f, err := os.OpenFile(heartbeatPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer f.Close()

		if _, err := f.WriteString(content); err != nil {
			fmt.Printf("Error writing to HEARTBEAT.md: %v\n", err)
			return
		}

		fmt.Printf("Task added to %s\n", heartbeatPath)
	},
}

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage and discover agent skills",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all installed skills",
	Run: func(cmd *cobra.Command, args []string) {
		application, err := app.Bootstrap(configPath, verbose)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer application.Close()

		allSkills := application.Manager.GetSkillsLoader().ListSkills()
		fmt.Printf("Installed Skills (%d):\n", len(allSkills))
		for _, s := range allSkills {
			fmt.Printf("- %s (%s): %s\n", s.Name, s.Source, s.Description)
		}
	},
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install [source]",
	Short: "Install a skill from GitHub or local path",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		application, err := app.Bootstrap(configPath, verbose)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer application.Close()

		installer := skills.NewSkillInstaller(application.Config.Agents.Defaults.Workspace)

		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") || strings.Contains(source, "/") {
			err = installer.InstallFromGitHub(context.Background(), source)
		} else {
			err = installer.InstallFromLocalPath(context.Background(), source)
		}

		if err != nil {
			fmt.Printf("Error installing skill: %v\n", err)
			return
		}
		fmt.Printf("Skill installed successfully from %s\n", source)
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication and API keys",
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status for all providers",
	Run: func(cmd *cobra.Command, args []string) {
		application, err := app.Bootstrap(configPath, verbose)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer application.Close()

		fmt.Println("Authentication Status:")
		providers := application.Config.AvailableProviders()
		for _, p := range providers {
			fmt.Printf("- %s: Authenticated\n", p)
		}
		if len(providers) == 0 {
			fmt.Println("No providers authenticated. Use 'wunderpus onboard' to setup.")
		}
	},
}

var authLoginCmd = &cobra.Command{
	Use:   "login [provider]",
	Short: "Login to a provider (opens browser for OAuth or prompts for API key)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]
		fmt.Printf("Authentication for %s\n", provider)
		fmt.Println("Use 'wunderpus onboard' to configure provider API keys interactively.")
		fmt.Println("Or set environment variables:")
		fmt.Printf("  - %s_API_KEY\n", strings.ToUpper(provider))
		fmt.Println("  - Or add to config.yaml under providers")
	},
}

func runAgent(cmd *cobra.Command, args []string) {
	application, err := app.Bootstrap(configPath, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer application.Close()

	// Launch TUI
	sessionID := "default_cli_session"
	ag := application.Manager.GetAgent(sessionID)

	if err := tui.Run(ag, application.MemoryStore); err != nil {
		fmt.Fprintf(os.Stderr, "TUI Error: %v\n", err)
		os.Exit(1)
	}
}

func runOneShot(msg string) {
	application, err := app.Bootstrap(configPath, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer application.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := application.Manager.ProcessMessage(ctx, "oneshot_cli", msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(resp)
}
