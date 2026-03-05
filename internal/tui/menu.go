package tui

import (
	"github.com/charmbracelet/lipgloss"
)

type MenuItem struct {
	Label    string
	Shortcut string
	Action   func() string
}

type MenuModel struct {
	Title    string
	Items    []MenuItem
	Selected int
	Width    int
}

func NewMenu(title string) *MenuModel {
	return &MenuModel{
		Title:    title,
		Items:    make([]MenuItem, 0),
		Selected: 0,
		Width:    40,
	}
}

func (m *MenuModel) AddItem(label, shortcut string, action func() string) {
	m.Items = append(m.Items, MenuItem{
		Label:    label,
		Shortcut: shortcut,
		Action:   action,
	})
}

func (m *MenuModel) MoveUp() {
	if m.Selected > 0 {
		m.Selected--
	}
}

func (m *MenuModel) MoveDown() {
	if m.Selected < len(m.Items)-1 {
		m.Selected++
	}
}

func (m *MenuModel) SelectItem(index int) {
	if index >= 0 && index < len(m.Items) {
		m.Selected = index
	}
}

func (m *MenuModel) SelectedItem() *MenuItem {
	if m.Selected >= 0 && m.Selected < len(m.Items) {
		return &m.Items[m.Selected]
	}
	return nil
}

func (m MenuModel) View() string {
	var lines []string

	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Align(lipgloss.Center)

	lines = append(lines, titleStyle.Render(m.Title))
	lines = append(lines, "")

	for i, item := range m.Items {
		var row string

		if i == m.Selected {
			selectedStyle := lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Background(statusBgColor)

			shortcutStyle := lipgloss.NewStyle().
				Foreground(dimColor).
				Background(statusBgColor)

			row = " " + selectedStyle.Render("▸ "+item.Label) + " " + shortcutStyle.Render(item.Shortcut)
		} else {
			itemStyle := lipgloss.NewStyle().
				Foreground(textColor)

			shortcutStyle := lipgloss.NewStyle().
				Foreground(dimColor)

			row = "   " + itemStyle.Render(item.Label) + " " + shortcutStyle.Render(item.Shortcut)
		}

		lines = append(lines, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func RenderMenuBox(title string, menu *MenuModel) string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1).
		Width(menu.Width)

	return box.Render(menu.View())
}
