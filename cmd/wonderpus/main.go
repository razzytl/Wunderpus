package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wonderpus/wonderpus/internal/agent"
	"github.com/wonderpus/wonderpus/internal/config"
	"github.com/wonderpus/wonderpus/internal/health"
	"github.com/wonderpus/wonderpus/internal/logging"
	"github.com/wonderpus/wonderpus/internal/memory"
	"github.com/wonderpus/wonderpus/internal/provider"
	"github.com/wonderpus/wonderpus/internal/security"
	"github.com/wonderpus/wonderpus/internal/tool"
	"github.com/wonderpus/wonderpus/internal/tool/builtin"
	"github.com/wonderpus/wonderpus/internal/tui"
	"github.com/wonderpus/wonderpus/internal/channel"
	"github.com/wonderpus/wonderpus/internal/channel/websocket"
	"github.com/wonderpus/wonderpus/internal/channel/telegram"
	"github.com/wonderpus/wonderpus/internal/channel/discord"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// 1. Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Hint: copy config.example.yaml to config.yaml and add your API keys\n")
		os.Exit(1)
	}

	// 2. Init logging
	logging.Init(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output)
	slog.Info("wonderpus starting",
		"version", "0.1.0",
		"providers", cfg.AvailableProviders(),
		"default", cfg.DefaultProvider,
	)

	// 3. Init security
	sanitizer := security.NewSanitizer(cfg.Security.SanitizationEnabled)

	audit, err := security.NewAuditLogger(cfg.Security.AuditDBPath)
	if err != nil {
		slog.Error("failed to init audit logger", "error", err)
		os.Exit(1)
	}
	defer audit.Close()

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

	if cfg.Tools.Enabled {
		registry.Register(builtin.NewFileRead(cfg.Tools.AllowedPaths))
		registry.Register(builtin.NewFileWrite(cfg.Tools.AllowedPaths))
		registry.Register(builtin.NewFileList(cfg.Tools.AllowedPaths))
		registry.Register(builtin.NewShellExec(cfg.Tools.ShellWhitelist))
		registry.Register(builtin.NewHTTPRequest())
		registry.Register(builtin.NewCalculator())

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

	// 7. Init agent manager
	manager := agent.NewManager(
		cfg,
		router,
		sanitizer,
		audit,
		memStore,
		registry,
		executor,
	)

	// Hardcode a single session ID for the CLI right now
	sessionID := "default_cli_session"
	ag := manager.GetAgent(sessionID)

	// 7. Graceful shutdown handler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 8. Init channels
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

		_ = healthSrv.Shutdown(shutCtx)
		for _, ch := range channels {
			_ = ch.Stop()
		}
		audit.Close()
		slog.Info("wonderpus stopped")
	}()

	_ = ctx // used for future background tasks

	// 11. Launch TUI (runs in main thread)
	if err := tui.Run(ag); err != nil {
		slog.Error("tui error", "error", err)
		os.Exit(1)
	}
}
