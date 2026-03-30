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
	"github.com/wunderpus/wunderpus/internal/perception"
	"github.com/wunderpus/wunderpus/internal/provider"
	"github.com/wunderpus/wunderpus/internal/security"
	"github.com/wunderpus/wunderpus/internal/skills"
	"github.com/wunderpus/wunderpus/internal/subagent"
	"github.com/wunderpus/wunderpus/internal/swarm"
	"github.com/wunderpus/wunderpus/internal/tool"
	"github.com/wunderpus/wunderpus/internal/tool/builtin"
	"github.com/wunderpus/wunderpus/internal/toolsynth"
	"github.com/wunderpus/wunderpus/internal/worldmodel"
)

// App encapsulates the Wunderpus application state and dependencies.
type App struct {
	Config              *config.Config
	Manager             *agent.Manager
	SubAgentMgr         *subagent.Manager
	MemoryStore         *memory.Store
	EnhancedMemoryStore *memory.EnhancedStore // For RAG
	AuditLogger         *security.AuditLogger
	HeartbeatScheduler  *heartbeat.Scheduler
	HealthServer        *health.Server
	Channels            []channel.Channel
	Registry            *tool.Registry
	Browser             *builtin.BrowserTool
	HumanInTheLoop      *builtin.HumanInTheLoop
	ToolSynth           *toolsynth.SynthSystem       // Tool Synthesis Engine (Section 1)
	WorldModel          *worldmodel.WorldModelSystem // World Model / Knowledge Graph (Section 2)
	Perception          *PerceptionSystem            // Computer Use / GUI Control (Section 3)
	Swarm               *swarm.SwarmSystem           // Agent Swarm Architecture (Section 4)
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
	var browserTool *builtin.BrowserTool
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

		browserTool = builtin.NewBrowserTool()
		registry.Register(browserTool)

		timeout := time.Duration(cfg.Tools.TimeoutSeconds) * time.Second
		approvalFn := func(toolName string, args map[string]any) (bool, error) {
			slog.Warn("SENSITIVE TOOL AUTO-APPROVED", "tool", toolName, "args", args)
			return true, nil
		}
		executor = tool.NewExecutor(registry, audit, approvalFn, timeout)
	}

	// 6. Init memory store with vector search capabilities
	var enhancedMemStore *memory.EnhancedStore
	embedder := router.GetEmbedder()
	if embedder != nil {
		// Use enhanced store with embeddings for RAG
		var enhancedStore *memory.EnhancedStore
		enhancedStore, err = memory.NewEnhancedStore(cfg.Agent.MemoryDBPath, embedder)
		if err != nil {
			audit.Close()
			return nil, fmt.Errorf("failed to init enhanced memory store: %w", err)
		}
		enhancedMemStore = enhancedStore
		slog.Info("Memory store initialized with vector search (RAG enabled)")
	}

	// Also create basic store for manager (needed for message history)
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

	// 8. Init tool synthesis engine (Section 1 — if enabled)
	var toolSynth *toolsynth.SynthSystem
	if cfg.Genesis.ToolSynthEnabled {
		synthCfg := toolsynth.SynthConfig{
			Enabled:        true,
			DBPath:         cfg.Genesis.ToolSynthDBPath,
			MinPassRate:    cfg.Genesis.ToolSynthMinPassRate,
			ScanLimit:      500,
			OutputDir:      filepath.Join("internal", "tool", "generated"),
			RepoRoot:       ".",
			ProfilerDBPath: cfg.Genesis.ProfilerDBPath,
		}
		llmAdapter := &RouterLLMAdapter{router: router}
		toolSynth, err = toolsynth.InitToolSynth(
			synthCfg,
			cfg.Agent.MemoryDBPath,
			llmAdapter,
			nil, // genesis audit log — wired separately if available
			nil, // event bus — wired separately if available
		)
		if err != nil {
			slog.Warn("tool synthesis init failed (non-fatal)", "error", err)
			toolSynth = nil
		}
	}

	// 8b. Init world model / knowledge graph (Section 2 — if enabled)
	var worldModel *worldmodel.WorldModelSystem
	if cfg.Genesis.WorldModelEnabled {
		wmCfg := worldmodel.Config{
			Enabled:       true,
			DBPath:        cfg.Genesis.WorldModelDBPath,
			ScanIntervalH: cfg.Genesis.WorldModelScanIntervalH,
		}
		wmLLM := &WorldModelLLMAdapter{router: router}
		worldModel, err = worldmodel.InitWorldModel(wmCfg, wmLLM, nil)
		if err != nil {
			slog.Warn("world model init failed (non-fatal)", "error", err)
			worldModel = nil
		}
	}

	// 8c. Init perception / computer use (Section 3 — if enabled)
	var perceptionSystem *PerceptionSystem
	if cfg.Genesis.PerceptionEnabled {
		perceptionSystem = initPerception(cfg, router, browserTool)
	}

	// 9. Init agent manager
	manager := agent.NewManager(cfg, router, sanitizer, audit, memStore, registry, executor, skillsLoader)

	// Set enhanced store for RAG if available
	if enhancedMemStore != nil {
		manager.SetEnhancedStore(enhancedMemStore)
	}

	// 9. Init sub-agent manager
	subAgentMgr := subagent.NewManager(manager, router)

	// 9b. Init Human-in-the-Loop manager
	hitl := builtin.NewHumanInTheLoop()

	// Register spawn, message, and ask_human tools
	if cfg.Tools.Enabled {
		registry.Register(builtin.NewSpawnTool(subAgentMgr))
		registry.Register(builtin.NewMessageTool(subAgentMgr))
		registry.Register(builtin.NewAskHumanTool(hitl))
	}

	// 9c. Init agent swarm (Section 4 — if enabled)
	var swarmSystem *swarm.SwarmSystem
	if cfg.Genesis.SwarmEnabled {
		swarmCfg := swarm.Config{Enabled: true}
		defaultAgent := manager.GetAgent("default")
		swarmSystem, err = swarm.InitSwarm(swarmCfg, func(ctx context.Context, goal swarm.Goal, config swarm.AgentConfig) (*swarm.SpecialistResult, error) {
			var resp string
			resp, err = defaultAgent.HandleMessage(ctx, goal.Description)
			if err != nil {
				return nil, err
			}
			return &swarm.SpecialistResult{
				Specialist: config.Name,
				GoalID:     goal.ID,
				Output:     resp,
				Success:    true,
			}, nil
		}, nil)
		if err != nil {
			slog.Warn("swarm init failed (non-fatal)", "error", err)
			swarmSystem = nil
		} else {
			slog.Info("swarm: initialized successfully")
		}
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
		channels = append(channels, telegram.NewChannelWithHITL(cfg.Channels.Telegram.Token, manager, hitl))
	}
	if cfg.Channels.Discord.Enabled {
		channels = append(channels, discord.NewChannelWithHITL(cfg.Channels.Discord.Token, manager, hitl))
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
		Config:              cfg,
		Manager:             manager,
		SubAgentMgr:         subAgentMgr,
		MemoryStore:         memStore,
		EnhancedMemoryStore: enhancedMemStore,
		AuditLogger:         audit,
		HeartbeatScheduler:  heartbeatScheduler,
		HealthServer:        healthSrv,
		Channels:            channels,
		Registry:            registry,
		Browser:             browserTool,
		HumanInTheLoop:      hitl,
		ToolSynth:           toolSynth,
		WorldModel:          worldModel,
		Perception:          perceptionSystem,
		Swarm:               swarmSystem,
	}, nil
}

// Close gracefully shuts down all application components.
func (a *App) Close() {
	if a.ToolSynth != nil && a.ToolSynth.Improver != nil {
		a.ToolSynth.Improver.Stop()
	}
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

// RouterLLMAdapter adapts the provider.Router to the toolsynth.LLMCaller interface.
type RouterLLMAdapter struct {
	router *provider.Router
}

// Complete sends an LLM request through the provider router with fallback.
func (a *RouterLLMAdapter) Complete(req toolsynth.CompletionRequest) (string, error) {
	ctx := context.Background()
	providerReq := &provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: req.SystemPrompt},
			{Role: provider.RoleUser, Content: req.UserPrompt},
		},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	resp, err := a.router.CompleteWithFallback(ctx, providerReq)
	if err != nil {
		return "", fmt.Errorf("llm adapter: %w", err)
	}
	return resp.Content, nil
}

// WorldModelLLMAdapter adapts the provider.Router to the worldmodel.LLMCaller interface.
type WorldModelLLMAdapter struct {
	router *provider.Router
}

// Complete sends an LLM request through the provider router with fallback.
func (a *WorldModelLLMAdapter) Complete(req worldmodel.LLMRequest) (string, error) {
	ctx := context.Background()
	providerReq := &provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: req.SystemPrompt},
			{Role: provider.RoleUser, Content: req.UserPrompt},
		},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	resp, err := a.router.CompleteWithFallback(ctx, providerReq)
	if err != nil {
		return "", fmt.Errorf("worldmodel llm adapter: %w", err)
	}
	return resp.Content, nil
}

// PerceptionSystem holds all perception components (Section 3).
type PerceptionSystem struct {
	Vision       *perception.Vision
	BrowserAgent *perception.BrowserAgent
	DOMAgent     *perception.DOMAgent
	DesktopAgent *perception.DesktopAgent
}

// initPerception initializes the perception system if Playwright is available.
func initPerception(cfg *config.Config, router *provider.Router, browserTool *builtin.BrowserTool) *PerceptionSystem {
	// Try to create Playwright bridge
	bridge, err := perception.NewPlaywrightBridge()
	if err != nil {
		slog.Warn("perception: Playwright bridge unavailable, using existing browser tool", "error", err)
		return nil
	}

	// Vision LLM adapter
	visionLLM := &PerceptionLLMAdapter{router: router}

	// Create vision interface
	vision := perception.NewVision(bridge, bridge, visionLLM)

	// Create browser agent
	browserAgent := perception.NewBrowserAgent(vision)
	if cfg.Genesis.PerceptionMaxActions > 0 {
		browserAgent.SetMaxActions(cfg.Genesis.PerceptionMaxActions)
	}

	// Create DOM agent
	domAgent := perception.NewDOMAgent(bridge, visionLLM)

	// Create desktop agent
	desktopAgent := perception.NewDesktopAgent()

	slog.Info("perception: initialized",
		"maxActions", cfg.Genesis.PerceptionMaxActions,
		"platform", desktopAgent.Platform())

	return &PerceptionSystem{
		Vision:       vision,
		BrowserAgent: browserAgent,
		DOMAgent:     domAgent,
		DesktopAgent: desktopAgent,
	}
}

// PerceptionLLMAdapter adapts provider.Router to perception.VisionLLMCaller.
type PerceptionLLMAdapter struct {
	router *provider.Router
}

func (a *PerceptionLLMAdapter) CompleteWithVision(prompt string, imageData []byte, mimeType string) (string, error) {
	ctx := context.Background()
	req := &provider.CompletionRequest{
		Messages: []provider.Message{
			{
				Role: provider.RoleUser,
				MultiContent: []provider.ContentPart{
					{Type: "text", Text: prompt},
					{Type: "image_url", ImageURL: &provider.ImageURL{
						URL: fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imageData)),
					}},
				},
			},
		},
		MaxTokens: 1000,
	}
	resp, err := a.router.CompleteWithFallback(ctx, req)
	if err != nil {
		return "", fmt.Errorf("perception llm: %w", err)
	}
	return resp.Content, nil
}

func (a *PerceptionLLMAdapter) CompleteText(prompt string) (string, error) {
	ctx := context.Background()
	req := &provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: "You are a browser automation agent."},
			{Role: provider.RoleUser, Content: prompt},
		},
		MaxTokens: 500,
	}
	resp, err := a.router.CompleteWithFallback(ctx, req)
	if err != nil {
		return "", fmt.Errorf("perception llm: %w", err)
	}
	return resp.Content, nil
}
