package toolsynth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/wunderpus/wunderpus/internal/audit"
	"github.com/wunderpus/wunderpus/internal/events"
)

// AuditBridge bridges the tool synthesis system to the genesis audit logger.
type AuditBridge struct {
	auditLog *audit.AuditLog
}

// NewAuditBridge creates a bridge to the genesis audit log.
func NewAuditBridge(auditLog *audit.AuditLog) *AuditBridge {
	return &AuditBridge{auditLog: auditLog}
}

// WriteToolSynthesized writes an EventToolSynthesized entry to the audit log.
func (b *AuditBridge) WriteToolSynthesized(name, sourcePath string, spec ToolSpec, testResult ToolTestResult) error {
	if b.auditLog == nil {
		return nil
	}

	payload, _ := json.Marshal(map[string]any{
		"tool_name":   name,
		"source_path": sourcePath,
		"pass_rate":   testResult.PassRate,
		"origin":      "synthesized",
	})

	return b.auditLog.Write(audit.AuditEntry{
		ID:        fmt.Sprintf("toolsynth-%s-%d", name, time.Now().UnixNano()),
		Timestamp: time.Now(),
		Subsystem: "toolsynth",
		EventType: audit.EventToolSynthesized,
		ActorID:   "toolsynth-pipeline",
		Payload:   payload,
	})
}

// WriteToolGapDetected writes an EventToolGapDetected entry to the audit log.
func (b *AuditBridge) WriteToolGapDetected(gap ToolGap) error {
	if b.auditLog == nil {
		return nil
	}

	payload, _ := json.Marshal(map[string]any{
		"gap_name": gap.Name,
		"gap_type": string(gap.GapType),
		"priority": gap.Priority,
	})

	return b.auditLog.Write(audit.AuditEntry{
		ID:        fmt.Sprintf("toolsynth-gap-%s-%d", gap.Name, time.Now().UnixNano()),
		Timestamp: time.Now(),
		Subsystem: "toolsynth",
		EventType: audit.EventToolGapDetected,
		ActorID:   "toolsynth-detector",
		Payload:   payload,
	})
}

// EventBridge bridges the tool synthesis system to the event bus.
type EventBridge struct {
	bus *events.Bus
}

// NewEventBridge creates a bridge to the event bus.
func NewEventBridge(bus *events.Bus) *EventBridge {
	return &EventBridge{bus: bus}
}

// PublishToolSynthesized publishes an EventToolSynthesized on the bus.
func (b *EventBridge) PublishToolSynthesized(name string, spec ToolSpec) {
	if b.bus == nil {
		return
	}

	b.bus.Publish(events.Event{
		Type:      audit.EventToolSynthesized,
		Payload:   map[string]any{"tool_name": name, "spec": spec},
		Timestamp: time.Now(),
		Source:    "toolsynth",
	})
}

// PublishToolGapDetected publishes an EventToolGapDetected on the bus.
func (b *EventBridge) PublishToolGapDetected(gap ToolGap) {
	if b.bus == nil {
		return
	}

	b.bus.Publish(events.Event{
		Type:      audit.EventToolGapDetected,
		Payload:   map[string]any{"gap_name": gap.Name, "gap_type": gap.GapType, "priority": gap.Priority},
		Timestamp: time.Now(),
		Source:    "toolsynth-detector",
	})
}

// SynthConfig holds configuration for the complete tool synthesis system.
type SynthConfig struct {
	Enabled        bool
	DBPath         string
	MinPassRate    float64
	ScanLimit      int
	OutputDir      string // where generated .go files go
	RepoRoot       string // project root for sandbox compilation
	ProfilerDBPath string // optional: profiler SQLite for timing analysis
}

// DefaultSynthConfig returns sensible defaults for tool synthesis.
func DefaultSynthConfig(homeDir, repoRoot string) SynthConfig {
	return SynthConfig{
		Enabled:     false,
		DBPath:      filepath.Join(homeDir, "wunderpus_toolsynth.db"),
		MinPassRate: 0.8,
		ScanLimit:   500,
		OutputDir:   filepath.Join(repoRoot, "internal", "tool", "generated"),
		RepoRoot:    repoRoot,
	}
}

// SynthSystem holds all tool synthesis components wired together.
type SynthSystem struct {
	Detector    *Detector
	Designer    *Designer
	Coder       *Coder
	Tester      *Tester
	Registrar   *Registrar
	Improver    *ImprovementLoop
	Pipeline    *Pipeline
	AuditBridge *AuditBridge
	EventBridge *EventBridge
}

// InitToolSynth initializes the complete tool synthesis system.
// memoryDBPath is the path to the memory store's SQLite database.
// llm is the LLM caller for design and code generation.
func InitToolSynth(
	cfg SynthConfig,
	memoryDBPath string,
	llm LLMCaller,
	auditLog *audit.AuditLog,
	bus *events.Bus,
) (*SynthSystem, error) {
	if !cfg.Enabled {
		slog.Info("toolsynth: disabled by config")
		return nil, nil
	}

	slog.Info("toolsynth: initializing",
		"db", cfg.DBPath,
		"output", cfg.OutputDir,
		"minPassRate", cfg.MinPassRate)

	// 1. Detector — scans episodic memory for gaps
	detector := NewDetector(memoryDBPath)
	detector.SetScanLimit(cfg.ScanLimit)
	if cfg.ProfilerDBPath != "" {
		detector.SetProfilerDB(cfg.ProfilerDBPath)
	}

	// 2. Designer — LLM-based tool specification generator
	designer := NewDesigner(llm)
	designer.SetDBPath(cfg.DBPath)

	// 3. Coder — LLM-based Go source code generator
	validator := NewDefaultValidator()
	coder := NewCoder(llm, validator, cfg.OutputDir)

	// 4. Tester — sandbox-based tool testing
	// We use a mock runner initially; real runners can be wired later
	tester := NewTester(&NoOpRunner{})
	tester.SetMinPassRate(cfg.MinPassRate)

	// 5. Registrar — tool registration + metadata persistence
	registrar := NewRegistrar(cfg.OutputDir, cfg.DBPath)

	// 6. Improvement Loop — MCP/GitHub marketplace scanning
	improver := NewImprovementLoop(nil, registrar) // scanner can be wired later
	if cfg.ProfilerDBPath != "" {
		improver.SetProfilerDB(cfg.ProfilerDBPath)
	}

	// 7. Pipeline — orchestrates the full cycle
	pipeline := NewPipeline(detector, designer, coder, tester, registrar)
	pipeline.SetImprover(improver)

	// 8. Bridges
	var ab *AuditBridge
	if auditLog != nil {
		ab = NewAuditBridge(auditLog)
		registrar.SetAuditWriter(ab)
	}

	var eb *EventBridge
	if bus != nil {
		eb = NewEventBridge(bus)
		registrar.SetEventPublisher(eb)
	}

	system := &SynthSystem{
		Detector:    detector,
		Designer:    designer,
		Coder:       coder,
		Tester:      tester,
		Registrar:   registrar,
		Improver:    improver,
		Pipeline:    pipeline,
		AuditBridge: ab,
		EventBridge: eb,
	}

	slog.Info("toolsynth: initialized successfully")
	return system, nil
}

// NoOpRunner is a placeholder tool runner for testing.
// Wire a real runner (e.g., tool.Executor) for production use.
type NoOpRunner struct{}

func (r *NoOpRunner) Run(_ context.Context, _ string, _ map[string]any) (string, error) {
	return "", fmt.Errorf("noop runner: not wired")
}
