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
	"github.com/wonderpus/wonderpus/internal/provider"
	"github.com/wonderpus/wonderpus/internal/security"
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

	// 5. Init agent
	ag := agent.NewAgent(
		router,
		sanitizer,
		audit,
		cfg.Agent.SystemPrompt,
		cfg.Agent.MaxContextTokens,
		cfg.Agent.Temperature,
	)

	// 6. Start health server
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
