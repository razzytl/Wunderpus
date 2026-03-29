package edge

import (
	"log/slog"
	"runtime"
)

// EdgeMode represents edge computing configuration.
type EdgeMode struct {
	Enabled        bool
	WASSandbox     bool   // Use WASM sandbox
	VisionEnabled  bool   // Enable vision agent
	DistillEnabled bool   // Enable model distillation
	MaxMemoryMB    int    // Memory limit
	ModelSelection string // Model selection strategy
}

// MinimalModeConfig holds minimal mode configuration.
type MinimalModeConfig struct {
	Enabled        bool
	AutoDetect     bool // Auto-detect edge environment
	WASSandbox     bool
	VisionEnabled  bool
	DistillEnabled bool
}

// NewMinimalMode creates edge mode with feature flags.
func NewMinimalMode(cfg MinimalModeConfig) *EdgeMode {
	// Auto-detect if enabled
	detectEnabled := cfg.Enabled
	if cfg.AutoDetect {
		detectEnabled = autoDetectEdge()
	}

	edge := &EdgeMode{
		Enabled:        detectEnabled,
		WASSandbox:     !detectEnabled && cfg.WASSandbox, // Default disabled on edge
		VisionEnabled:  !detectEnabled && cfg.VisionEnabled,
		DistillEnabled: !detectEnabled && cfg.DistillEnabled,
		MaxMemoryMB:    getDefaultMemory(),
		ModelSelection: selectModelStrategy(),
	}

	slog.Info("edge: mode configured",
		"enabled", edge.Enabled,
		"memory_mb", edge.MaxMemoryMB,
		"model", edge.ModelSelection)

	return edge
}

func autoDetectEdge() bool {
	// Detect if running on constrained environment
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// If less than 2GB RAM and less than 4 CPU cores
	totalMemoryGB := memStats.TotalAlloc / (1024 * 1024 * 1024)
	numCPU := runtime.NumCPU()

	if totalMemoryGB < 2 || numCPU < 4 {
		return true
	}
	return false
}

func getDefaultMemory() int {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	totalGB := memStats.TotalAlloc / (1024 * 1024 * 1024)
	if totalGB < 2 {
		return 512 // Conservative on small devices
	}
	return 1024
}

func selectModelStrategy() string {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	totalGB := memStats.TotalAlloc / (1024 * 1024 * 1024)
	if totalGB < 2 {
		return "qwen2.5-3b" // Small model for constrained env
	}
	if totalGB < 4 {
		return "llama3.2-3b" // Medium model
	}
	return "llama3.1-8b" // Full model
}

// EdgeEnvironment represents current edge environment info.
type EdgeEnvironment struct {
	Platform      string `json:"platform"`
	CPUCores      int    `json:"cpu_cores"`
	MemoryTotalGB int    `json:"memory_gb"`
	IsEdge        bool   `json:"is_edge"`
}

// DetectEnvironment detects current hardware capabilities.
func DetectEnvironment() *EdgeEnvironment {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	env := &EdgeEnvironment{
		Platform:      runtime.GOOS,
		CPUCores:      runtime.NumCPU(),
		MemoryTotalGB: int(memStats.TotalAlloc / (1024 * 1024 * 1024)),
		IsEdge:        false,
	}

	// Check if qualifies as edge
	if env.MemoryTotalGB < 2 || env.CPUCores < 4 {
		env.IsEdge = true
	}

	slog.Info("edge: environment detected", "platform", env.Platform, "is_edge", env.IsEdge)
	return env
}

// CheckFeatureAvailable checks if a feature is available in current mode.
func (e *EdgeMode) CheckFeatureAvailable(feature string) bool {
	if !e.Enabled {
		return true // Full feature set in normal mode
	}

	switch feature {
	case "sandbox":
		return e.WASSandbox
	case "vision":
		return e.VisionEnabled
	case "distillation":
		return e.DistillEnabled
	default:
		return false
	}
}
