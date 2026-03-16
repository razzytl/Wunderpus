package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/channel"
	"github.com/wunderpus/wunderpus/internal/channel/discord"
	"github.com/wunderpus/wunderpus/internal/channel/feishu"
	"github.com/wunderpus/wunderpus/internal/channel/slack"
	"github.com/wunderpus/wunderpus/internal/channel/telegram"
	"github.com/wunderpus/wunderpus/internal/channel/websocket"
	"github.com/wunderpus/wunderpus/internal/channel/whatsapp"
	"github.com/wunderpus/wunderpus/internal/config"
	"github.com/wunderpus/wunderpus/internal/health"
	"github.com/wunderpus/wunderpus/internal/heartbeat"
	"github.com/wunderpus/wunderpus/internal/logging"
	"github.com/wunderpus/wunderpus/internal/memory"
	"github.com/wunderpus/wunderpus/internal/provider"
	"github.com/wunderpus/wunderpus/internal/security"
	"github.com/wunderpus/wunderpus/internal/skills"
	"github.com/wunderpus/wunderpus/internal/subagent"
	"github.com/wunderpus/wunderpus/internal/tool"
	"github.com/wunderpus/wunderpus/internal/tool/builtin"
)

// App encapsulates the Wunderpus application state and dependencies.
type App struct {
	Config             *config.Config
	Manager            *agent.Manager
	SubAgentMgr        *subagent.Manager
	MemoryStore        *memory.Store
	AuditLogger        *security.AuditLogger
	HeartbeatScheduler *heartbeat.Scheduler
	HealthServer       *health.Server
	Channels           []channel.Channel
	Registry           *tool.Registry
	Browser            *builtin.BrowserTool
}

// Bootstrap initializes the application with the given config path.
func Bootstrap(configPath string, verbose bool) (*App, error) {
	// 1. Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Init logging
	logLevel := cfg.Logging.Level
	if verbose {
		logLevel = "debug"
	}
	logging.Init(logLevel, cfg.Logging.Format, cfg.Logging.Output)

	// Prepare encryption key if enabled
	var encKey []byte
	if cfg.Security.Encryption.Enabled && cfg.Security.Encryption.Key != "" {
		encKey, err = base64.StdEncoding.DecodeString(cfg.Security.Encryption.Key)
		if err != nil {
			return nil, fmt.Errorf("invalid encryption key: %w", err)
		}
	}

	// 3. Init security
	sanitizer := security.NewSanitizer(cfg.Security.SanitizationEnabled)
	audit, err := security.NewAuditLogger(cfg.Security.AuditDBPath, encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init audit logger: %w", err)
	}

	// Initial rotation check
	_ = audit.Rotate(10000)

	// 4. Init providers
	router, err := provider.NewRouter(cfg)
	if err != nil {
		audit.Close()
		return nil, fmt.Errorf("failed to init providers: %w", err)
	}

	// 5. Init tools
	registry := tool.NewRegistry()
	sandbox, err := security.NewWorkspaceSandbox(cfg.Agents.Defaults.Workspace, cfg.Agents.Defaults.RestrictToWorkspace)
	if err != nil {
		audit.Close()
		return nil, fmt.Errorf("failed to init workspace sandbox: %w", err)
	}

	var executor *tool.Executor
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

		browserTool := builtin.NewBrowserTool()
		registry.Register(browserTool)

		timeout := time.Duration(cfg.Tools.TimeoutSeconds) * time.Second
		approvalFn := func(toolName string, args map[string]any) (bool, error) {
			slog.Warn("SENSITIVE TOOL AUTO-APPROVED", "tool", toolName, "args", args)
			return true, nil
		}
		executor = tool.NewExecutor(registry, audit, approvalFn, timeout)
	}

	// 6. Init memory store
	memStore, err := memory.NewStore(cfg.Agent.MemoryDBPath)
	if err != nil {
		audit.Close()
		return nil, fmt.Errorf("failed to init memory store: %w", err)
	}

	// 7. Init skills loader
	homeDir, _ := os.UserHomeDir()
	globalSkills := filepath.Join(homeDir, ".wunderpus", "skills")
	builtinSkills := "./skills"
	skillsLoader := skills.NewSkillsLoader(cfg.Agents.Defaults.Workspace, globalSkills, builtinSkills)

	// 8. Init agent manager
	manager := agent.NewManager(cfg, router, sanitizer, audit, memStore, registry, executor, skillsLoader)

	// 9. Init sub-agent manager
	subAgentMgr := subagent.NewManager(manager, router)

	// Register spawn and message tools
	if cfg.Tools.Enabled {
		registry.Register(builtin.NewSpawnTool(subAgentMgr))
		registry.Register(builtin.NewMessageTool(subAgentMgr))
	}

	// 10. Init heartbeat scheduler
	heartbeatCfg := &heartbeat.HeartbeatConfig{
		Enabled:   cfg.Heartbeat.Enabled,
		Interval:  cfg.Heartbeat.Interval,
		Workspace: cfg.Agents.Defaults.Workspace,
	}
	heartbeatParser := heartbeat.NewParser()
	heartbeatExecutor := heartbeat.NewHeartbeatExecutor(manager, subAgentMgr)
	heartbeatScheduler := heartbeat.NewScheduler(heartbeatCfg, heartbeatParser, heartbeatExecutor, manager, cfg.Agents.Defaults.Workspace)

	// 11. Init health server
	healthSrv := health.NewServer(cfg.Server.HealthPort)

	// 12. Channels
	var channels []channel.Channel

	if cfg.Channels.Telegram.Enabled {
		channels = append(channels, telegram.NewChannel(cfg.Channels.Telegram.Token, manager))
	}
	if cfg.Channels.Discord.Enabled {
		channels = append(channels, discord.NewChannel(cfg.Channels.Discord.Token, manager))
	}
	if cfg.Channels.WebSocket.Enabled {
		channels = append(channels, websocket.NewServer(cfg.Channels.WebSocket.Port, manager))
	}
	if cfg.Channels.Slack.Enabled {
		channels = append(channels, slack.NewChannel(cfg.Channels.Slack.Token, cfg.Channels.Slack.AppToken, cfg.Channels.Slack.SocketMode, manager))
	}
	if cfg.Channels.WhatsApp.Enabled {
		channels = append(channels, whatsapp.NewChannel(cfg.Channels.WhatsApp.SessionPath, manager))
	}
	if cfg.Channels.Feishu.Enabled {
		channels = append(channels, feishu.NewChannel(cfg.Channels.Feishu.AppID, cfg.Channels.Feishu.AppSecret, cfg.Channels.Feishu.VerificationToken, manager))
	}

	// Create channel aggregator for file sending
	channelAggregator := channel.NewChannelAggregator(channels)

	// Register send_file tool if enabled
	if cfg.Tools.Enabled && cfg.Tools.SendFile.Enabled {
		registry.Register(builtin.NewSendFileTool(channelAggregator, sandbox))
	}

	return &App{
		Config:             cfg,
		Manager:            manager,
		SubAgentMgr:        subAgentMgr,
		MemoryStore:        memStore,
		AuditLogger:        audit,
		HeartbeatScheduler: heartbeatScheduler,
		HealthServer:       healthSrv,
		Channels:           channels,
		Registry:           registry,
		Browser:            browserTool,
	}, nil
}

// Close gracefully shuts down all application components.
func (a *App) Close() {
	if a.Browser != nil {
		a.Browser.Close()
	}
	if a.HeartbeatScheduler != nil {
		a.HeartbeatScheduler.Stop()
	}
	if a.HealthServer != nil {
		_ = a.HealthServer.Shutdown(context.Background())
	}
	for _, ch := range a.Channels {
		_ = ch.Stop()
	}
	if a.MemoryStore != nil {
		_ = a.MemoryStore.Close()
	}
	if a.AuditLogger != nil {
		_ = a.AuditLogger.Close()
	}
}
