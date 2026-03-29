package perception

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
)

// DesktopAgent controls native desktop applications.
// Platform-specific implementations:
// - Linux: xdotool + xwd for X11
// - macOS: osascript + screencapture
// - Windows: PowerShell + SendKeys
type DesktopAgent struct {
	platform string
}

// NewDesktopAgent creates a desktop agent for the current platform.
func NewDesktopAgent() *DesktopAgent {
	return &DesktopAgent{
		platform: runtime.GOOS,
	}
}

// Platform returns the detected platform.
func (d *DesktopAgent) Platform() string {
	return d.platform
}

// Screenshot captures the entire desktop screen.
func (d *DesktopAgent) Screenshot() ([]byte, error) {
	slog.Debug("perception: desktop screenshot", "platform", d.platform)

	switch d.platform {
	case "linux":
		return exec.Command("xwd", "-root", "-silent", "-display", ":0").Output()
	case "darwin":
		return exec.Command("screencapture", "-x", "-").Output()
	case "windows":
		return exec.Command("powershell", "-Command",
			`[Reflection.Assembly]::LoadWithPartialName('System.Drawing') | Out-Null; `+
				`$bmp = New-Object Drawing.Bitmap(1,1); `+
				`$bmp.Save([Console]::OpenStandardOutput(), 'Png')`).Output()
	default:
		return nil, fmt.Errorf("desktop: unsupported platform %s", d.platform)
	}
}

// Click performs a mouse click at screen coordinates.
func (d *DesktopAgent) Click(x, y int) error {
	slog.Debug("perception: desktop click", "x", x, "y", y)

	switch d.platform {
	case "linux":
		return exec.Command("xdotool", "mousemove", itoa(x), itoa(y), "click", "1").Run()
	case "darwin":
		script := fmt.Sprintf(`tell application "System Events" to click at {%d, %d}`, x, y)
		return exec.Command("osascript", "-e", script).Run()
	case "windows":
		script := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms
			[System.Windows.Forms.Cursor]::Position = New-Object System.Drawing.Point(%d, %d)
		`, x, y)
		return exec.Command("powershell", "-Command", script).Run()
	default:
		return fmt.Errorf("desktop: unsupported platform %s", d.platform)
	}
}

// Types types text using keyboard simulation.
func (d *DesktopAgent) Types(text string) error {
	slog.Debug("perception: desktop type", "text", truncate(text, 50))

	switch d.platform {
	case "linux":
		return exec.Command("xdotool", "type", "--delay", "50", text).Run()
	case "darwin":
		script := fmt.Sprintf(`tell application "System Events" to keystroke %q`, text)
		return exec.Command("osascript", "-e", script).Run()
	case "windows":
		script := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms
			[System.Windows.Forms.SendKeys]::SendWait(%q)
		`, text)
		return exec.Command("powershell", "-Command", script).Run()
	default:
		return fmt.Errorf("desktop: unsupported platform %s", d.platform)
	}
}

// PressKey presses a single keyboard key.
func (d *DesktopAgent) PressKey(key string) error {
	slog.Debug("perception: desktop press", "key", key)

	keyMap := map[string]map[string]string{
		"linux":   {"Enter": "Return", "Tab": "Tab", "Escape": "Escape"},
		"darwin":  {"Enter": "return", "Tab": "tab", "Escape": "escape"},
		"windows": {"Enter": "{ENTER}", "Tab": "{TAB}", "Escape": "{ESC}"},
	}

	platformKeys := keyMap[d.platform]
	if platformKeys == nil {
		return fmt.Errorf("desktop: unsupported platform %s", d.platform)
	}

	mappedKey, ok := platformKeys[key]
	if !ok {
		mappedKey = key
	}

	switch d.platform {
	case "linux":
		return exec.Command("xdotool", "key", mappedKey).Run()
	case "darwin":
		script := fmt.Sprintf(`tell application "System Events" to keystroke %q`, mappedKey)
		return exec.Command("osascript", "-e", script).Run()
	case "windows":
		script := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms
			[System.Windows.Forms.SendKeys]::SendWait(%q)
		`, mappedKey)
		return exec.Command("powershell", "-Command", script).Run()
	default:
		return fmt.Errorf("desktop: unsupported platform %s", d.platform)
	}
}

// OpenApp opens a desktop application.
func (d *DesktopAgent) OpenApp(appPath string) error {
	slog.Info("perception: opening app", "path", appPath)

	switch d.platform {
	case "linux":
		cmd := exec.Command(appPath)
		return cmd.Start()
	case "darwin":
		return exec.Command("open", "-a", appPath).Run()
	case "windows":
		return exec.Command("cmd", "/C", "start", appPath).Run()
	default:
		return fmt.Errorf("desktop: unsupported platform %s", d.platform)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	n := i
	if n < 0 {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if i < 0 {
		s = "-" + s
	}
	return s
}
