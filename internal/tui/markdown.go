package tui

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	mdWidth int
)

func InitMarkdownRenderer(width int) error {
	mdWidth = width
	return nil
}

func InitMarkdownRendererWithTheme(width int) error {
	mdWidth = width
	return nil
}

func RenderMarkdown(content string) string {
	rendered, err := glamour.Render(content, "dark")
	if err != nil {
		return content
	}
	return stripTrailingNewlines(rendered)
}

func RenderMarkdownWithWidth(content string, width int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	rendered, err := r.Render(content)
	if err != nil {
		return content
	}
	return stripTrailingNewlines(rendered)
}

func stripTrailingNewlines(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func GetProviderIcon(name string) string {
	switch name {
	case "openai":
		return "⚡"
	case "anthropic":
		return "🧠"
	case "ollama":
		return "🦙"
	case "gemini":
		return "🌟"
	default:
		return "○"
	}
}

var (
	ProviderIconStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9D7CD8"))

	CostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ECE6A"))

	TokenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7AA2F7"))

	ToolStatusIdleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#565F89"))

	ToolStatusRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0AF68"))

	LatencyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565F89"))
)
