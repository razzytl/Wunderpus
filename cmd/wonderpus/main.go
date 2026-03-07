package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wonderpus/wonderpus/internal/agent"
	"github.com/wonderpus/wonderpus/internal/channel"
	"github.com/wonderpus/wonderpus/internal/channel/discord"
	"github.com/wonderpus/wonderpus/internal/channel/telegram"
	"github.com/wonderpus/wonderpus/internal/channel/websocket"
	"github.com/wonderpus/wonderpus/internal/config"
	"github.com/wonderpus/wonderpus/internal/health"
	"github.com/wonderpus/wonderpus/internal/heartbeat"
	"github.com/wonderpus/wonderpus/internal/logging"
	"github.com/wonderpus/wonderpus/internal/memory"
	"github.com/wonderpus/wonderpus/internal/provider"
	"github.com/wonderpus/wonderpus/internal/security"
	"github.com/wonderpus/wonderpus/internal/skills"
	"github.com/wonderpus/wonderpus/internal/subagent"
	"github.com/wonderpus/wonderpus/internal/tool"
	"github.com/wonderpus/wonderpus/internal/tool/builtin"
	"github.com/wonderpus/wonderpus/internal/tui"
)

const Version = "0.2.0"

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "show version information")
	verbose := flag.Bool("verbose", false, "enable debug logging")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Wonderpus v%s\n", Version)
		os.Exit(0)
	}

	// 1. Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Hint: copy config.example.yaml to config.yaml and add your API keys\n")
		os.Exit(1)
	}

	// 2. Init logging
	logLevel := cfg.Logging.Level
	if *verbose {
		logLevel = "debug"
	}
	logging.Init(logLevel, cfg.Logging.Format, cfg.Logging.Output)

	// Prepare encryption key if enabled
	var encKey []byte
	if cfg.Security.Encryption.Enabled && cfg.Security.Encryption.Key != "" {
		var err error
		encKey, err = base64.StdEncoding.DecodeString(cfg.Security.Encryption.Key)
		if err != nil {
			slog.Error("invalid encryption key", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("wonderpus starting",
		"version", Version,
		"providers", cfg.AvailableProviders(),
		"default", cfg.DefaultProvider,
	)

	// 3. Init security
	sanitizer := security.NewSanitizer(cfg.Security.SanitizationEnabled)

	audit, err := security.NewAuditLogger(cfg.Security.AuditDBPath, encKey)
	if err != nil {
		slog.Error("failed to init audit logger", "error", err)
		os.Exit(1)
	}
	defer audit.Close()

	// Initial rotation check
	if err := audit.Rotate(10000); err != nil {
		slog.Warn("audit rotation failed", "error", err)
	}

	// 4. Init providers
	router, err := provider.NewRouter(cfg)
	if err != nil {
		slog.Error("failed to init providers", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\nHint: configure at least one provider in config.yaml\n", err)
		os.Exit(1)
	}

	// 5. Init tools
	registry := tool.NewRegistry()
	var executor *tool.Executor

	// Workspace sandbox
	sandbox, err := security.NewWorkspaceSandbox(cfg.Agents.Defaults.Workspace, cfg.Agents.Defaults.RestrictToWorkspace)
	if err != nil {
		slog.Error("failed to init workspace sandbox", "error", err)
		os.Exit(1)
	}

	if cfg.Tools.Enabled {
		registry.Register(builtin.NewFileReadSandboxed(sandbox))
		registry.Register(builtin.NewFileWriteSandboxed(sandbox))
		registry.Register(builtin.NewFileListSandboxed(sandbox))
		registry.Register(builtin.NewFileGlobSandboxed(sandbox))
		registry.Register(builtin.NewSearchFilesSandboxed(sandbox))
		registry.Register(builtin.NewShellExecSandboxed(cfg.Tools.ShellWhitelist, sandbox))
		registry.Register(builtin.NewHTTPRequest(cfg.Tools.SSRFBlocklist))
		registry.Register(builtin.NewCalculator())
		registry.Register(builtin.NewSystemInfo())

		timeout := time.Duration(cfg.Tools.TimeoutSeconds) * time.Second

		approvalFn := func(toolName string, args map[string]any) (bool, error) {
			// In a full TUI this would pause and ask the user.
			// For Phase 2 MVP, we log a warning but allow it to proceed,
			// relying on the sandboxing (paths/whitelist) for safety.
			slog.Warn("SENSITIVE TOOL AUTO-APPROVED", "tool", toolName, "args", args)
			return true, nil
		}

		executor = tool.NewExecutor(registry, audit, approvalFn, timeout)
	}

	// 6. Init memory store
	memStore, err := memory.NewStore(cfg.Agent.MemoryDBPath)
	if err != nil {
		slog.Error("failed to init memory store", "error", err)
		os.Exit(1)
	}
	defer memStore.Close()

	// 7. Init skills loader
	homeDir, _ := os.UserHomeDir()
	globalSkills := filepath.Join(homeDir, ".wunderpus", "skills")
	builtinSkills := "./skills"
	skillsLoader := skills.NewSkillsLoader(cfg.Agents.Defaults.Workspace, globalSkills, builtinSkills)

	// 8. Init agent manager
	manager := agent.NewManager(
		cfg,
		router,
		sanitizer,
		audit,
		memStore,
		registry,
		executor,
		skillsLoader,
	)

	// 9. Init sub-agent manager
	subAgentMgr := subagent.NewManager(manager, router)

	// Register spawn and message tools
	if cfg.Tools.Enabled {
		registry.Register(builtin.NewSpawnTool(subAgentMgr))
		registry.Register(builtin.NewMessageTool(subAgentMgr))
	}

	// Hardcode a single session ID for the CLI right now
	sessionID := "default_cli_session"
	ag := manager.GetAgent(sessionID)

	// 10. Graceful shutdown handler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 11. Init heartbeat scheduler
	heartbeatCfg := &heartbeat.HeartbeatConfig{
		Enabled:   cfg.Heartbeat.Enabled,
		Interval:  cfg.Heartbeat.Interval,
		Workspace: cfg.Agents.Defaults.Workspace,
	}
	heartbeatParser := heartbeat.NewParser()
	heartbeatScheduler := heartbeat.NewScheduler(heartbeatCfg, heartbeatParser, nil, manager, cfg.Agents.Defaults.Workspace)

	// Start heartbeat scheduler
	if err := heartbeatScheduler.Start(ctx); err != nil {
		slog.Warn("heartbeat scheduler failed to start", "error", err)
	}

	// 12. Init channels
	var channels []channel.Channel
	if cfg.Channels.WebSocket.Enabled {
		channels = append(channels, websocket.NewServer(cfg.Channels.WebSocket.Port, manager))
	}
	if cfg.Channels.Telegram.Enabled {
		channels = append(channels, telegram.NewChannel(cfg.Channels.Telegram.Token, manager))
	}
	if cfg.Channels.Discord.Enabled {
		channels = append(channels, discord.NewChannel(cfg.Channels.Discord.Token, manager))
	}

	// 9. Start health & metrics server
	healthSrv := health.NewServer(cfg.Server.HealthPort)
	// Add Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	healthSrv.Start()

	// 10. Start channels
	for _, ch := range channels {
		if err := ch.Start(ctx); err != nil {
			slog.Error("failed to start channel", "channel", ch.Name(), "error", err)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		cancel()

		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()

		// Stop heartbeat scheduler
		heartbeatScheduler.Stop()

		_ = healthSrv.Shutdown(shutCtx)
		for _, ch := range channels {
			_ = ch.Stop()
		}
		audit.Close()
		slog.Info("wunderpus stopped")
	}()

	_ = ctx // used for future background tasks

	// 11. Launch TUI (runs in main thread)
	if err := tui.Run(ag, memStore); err != nil {
		slog.Error("tui error", "error", err)
		os.Exit(1)
	}
}
