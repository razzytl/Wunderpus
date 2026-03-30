package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type KeyBinding struct {
	Key         string
	Description string
	Group       string
}

type KeybindingsHelp struct {
	Bindings []KeyBinding
	Cols     int
	Width    int
}

func NewKeybindingsHelp() *KeybindingsHelp {
	return &KeybindingsHelp{
		Bindings: make([]KeyBinding, 0),
		Cols:     2,
		Width:    60,
	}
}

func (kh *KeybindingsHelp) AddBinding(key, desc, group string) {
	kh.Bindings = append(kh.Bindings, KeyBinding{
		Key:         key,
		Description: desc,
		Group:       group,
	})
}

func (kh *KeybindingsHelp) AddGroup(group string, bindings ...KeyBinding) {
	for _, b := range bindings {
		b.Group = group
		kh.Bindings = append(kh.Bindings, b)
	}
}

func (kh *KeybindingsHelp) View() string {
	lines := make([]string, 0, len(kh.Bindings)*2)

	grouped := make(map[string][]KeyBinding)
	for _, binding := range kh.Bindings {
		grouped[binding.Group] = append(grouped[binding.Group], binding)
	}

	for group, bindings := range grouped {
		groupStyle := lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			Underline(true)

		lines = append(lines, groupStyle.Render(group))
		lines = append(lines, "")

		maxKeyLen := 0
		for _, b := range bindings {
			if len(b.Key) > maxKeyLen {
				maxKeyLen = len(b.Key)
			}
		}

		for _, b := range bindings {
			keyStyle := lipgloss.NewStyle().
				Foreground(systemColor).
				Width(maxKeyLen + 2).
				Align(lipgloss.Right)

			descStyle := lipgloss.NewStyle().
				Foreground(textColor)

			line := keyStyle.Render(b.Key) + "  " + descStyle.Render(b.Description)
			lines = append(lines, line)
		}

		lines = append(lines, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func RenderKeybindingsHelp(title string, kh KeybindingsHelp) string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	content := titleStyle.Render(" "+title+" ") + "\n\n" + kh.View()

	return box.Width(kh.Width).Render(content)
}

func DefaultKeybindings() KeybindingsHelp {
	kh := NewKeybindingsHelp()
	kh.Width = 70

	kh.AddBinding("Enter", "Send message", "Messaging")
	kh.AddBinding("Shift+Enter", "New line in input", "Messaging")
	kh.AddBinding("Escape", "Cancel streaming", "Messaging")
	kh.AddBinding("↑/↓", "Navigate command history", "Navigation")
	kh.AddBinding("Tab", "Cycle providers", "Navigation")
	kh.AddBinding("Ctrl+L", "Clear conversation", "Conversation")
	kh.AddBinding("Ctrl+U", "Clear input line", "Conversation")
	kh.AddBinding("Ctrl+S", "Switch provider", "Providers")
	kh.AddBinding("Ctrl+P", "Toggle command palette", "Commands")
	kh.AddBinding("Ctrl+O", "Orchestrate mode", "Commands")
	kh.AddBinding("Ctrl+C", "Quit application", "General")
	kh.AddBinding("/help", "Show help", "Commands")
	kh.AddBinding("/provider", "Switch provider", "Commands")
	kh.AddBinding("/clear", "Clear conversation", "Commands")
	kh.AddBinding("/status", "Show status", "Commands")
	kh.AddBinding("/settings", "Show settings", "Commands")
	kh.AddBinding("/tools", "List tools", "Commands")
	kh.AddBinding("/exit", "Exit application", "Commands")

	return *kh
}

func (kh *KeybindingsHelp) CompactView() string {
	var lines []string

	grouped := make(map[string][]KeyBinding)
	for _, binding := range kh.Bindings {
		grouped[binding.Group] = append(grouped[binding.Group], binding)
	}

	groupOrder := []string{"Messaging", "Navigation", "Conversation", "Providers", "Commands", "General"}

	for _, group := range groupOrder {
		bindings, ok := grouped[group]
		if !ok || len(bindings) == 0 {
			continue
		}

		groupStyle := lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

		lines = append(lines, groupStyle.Render(group+":"))

		for _, b := range bindings {
			keyStyle := lipgloss.NewStyle().
				Foreground(systemColor)

			descStyle := lipgloss.NewStyle().
				Foreground(dimColor)

			lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render(b.Key), descStyle.Render(b.Description)))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
