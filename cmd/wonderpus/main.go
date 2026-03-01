package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
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

	// Hardcode a single session ID for the CLI right now
	// In the future this can be generated per connection/run
	sessionID := "default_cli_session"

	// 7. Init agent
	ag := agent.NewAgent(
		router,
		sanitizer,
		audit,
		memStore,
		registry,
		executor,
		cfg.Agent.SystemPrompt,
		cfg.Agent.MaxContextTokens,
		cfg.Agent.Temperature,
		sessionID,
	)

	// 8. Start health server
	healthSrv := health.NewServer(cfg.Server.HealthPort)
	healthSrv.Start()

	// 7. Graceful shutdown handler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		cancel()

		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()

		_ = healthSrv.Shutdown(shutCtx)
		audit.Close()
		slog.Info("wonderpus stopped")
	}()

	_ = ctx // used for future background tasks

	// 8. Launch TUI
	if err := tui.Run(ag); err != nil {
		slog.Error("tui error", "error", err)
		os.Exit(1)
	}
}
