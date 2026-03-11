package builtin

import (
	"context"
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/wunderpus/wunderpus/internal/tool"
	"runtime"
)

// SystemInfo provides information about the host system.
type SystemInfo struct{}

// NewSystemInfo creates a new system info tool.
func NewSystemInfo() *SystemInfo {
	return &SystemInfo{}
}

func (s *SystemInfo) Name() string { return "system_info" }
func (s *SystemInfo) Description() string {
	return "Get information about the system (OS, Arch, CPU, Memory)."
}
func (s *SystemInfo) Sensitive() bool                 { return false }
func (s *SystemInfo) Version() string                 { return "1.0.0" }
func (s *SystemInfo) Dependencies() []string          { return nil }
func (s *SystemInfo) Parameters() []tool.ParameterDef { return nil }

func (s *SystemInfo) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	v, _ := mem.VirtualMemoryWithContext(ctx)
	c, _ := cpu.InfoWithContext(ctx)

	cpuModel := "Unknown"
	if len(c) > 0 {
		cpuModel = c[0].ModelName
	}

	output := fmt.Sprintf("OS: %s\nArch: %s\nCPU: %s\nCores: %d\nMemory Total: %d MB\nMemory Available: %d MB\n",
		runtime.GOOS,
		runtime.GOARCH,
		cpuModel,
		runtime.NumCPU(),
		v.Total/1024/1024,
		v.Available/1024/1024,
	)

	return &tool.Result{Output: output}, nil
}
