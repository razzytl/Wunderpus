package tui

//go:generate stringer -type=BoxStyle -linecomment

import (
	"github.com/charmbracelet/lipgloss"
)

type BoxStyle int

const (
	BoxStyleRounded BoxStyle = iota
	BoxStyleDouble
	BoxStyleSingle
	BoxStyleBold
	BoxStyleRound
)

type BoxModel struct {
	Content   string
	Title     string
	Style     BoxStyle
	Width     int
	Height    int
	ShowTitle bool
}

func NewBox(content string) *BoxModel {
	return &BoxModel{
		Content:   content,
		Title:     "",
		Style:     BoxStyleRounded,
		Width:     0,
		Height:    0,
		ShowTitle: false,
	}
}

func (b *BoxModel) SetTitle(title string) {
	b.Title = title
	b.ShowTitle = true
}

func (b *BoxModel) SetStyle(style BoxStyle) {
	b.Style = style
}

func (b *BoxModel) SetWidth(width int) {
	b.Width = width
}

func (b *BoxModel) SetHeight(height int) {
	b.Height = height
}

func (b *BoxModel) View() string {
	borderStyle := b.getBorderStyle()

	box := lipgloss.NewStyle().
		BorderStyle(borderStyle).
		BorderForeground(accentColor).
		Padding(0, 1)

	if b.Width > 0 {
		box = box.Width(b.Width)
	}
	if b.Height > 0 {
		box = box.Height(b.Height)
	}

	if b.ShowTitle && b.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

		titleBar := titleStyle.Render(" " + b.Title + " ")
		return box.Render(titleBar + "\n" + b.Content)
	}

	return box.Render(b.Content)
}

func (b *BoxModel) getBorderStyle() lipgloss.Border {
	switch b.Style {
	case BoxStyleRounded:
		return lipgloss.RoundedBorder()
	case BoxStyleDouble:
		return lipgloss.DoubleBorder()
	case BoxStyleSingle:
		return lipgloss.NormalBorder()
	case BoxStyleBold:
		return lipgloss.NormalBorder()
	case BoxStyleRound:
		return lipgloss.ThickBorder()
	default:
		return lipgloss.RoundedBorder()
	}
}

func RenderBox(content string) string {
	return NewBox(content).View()
}

func RenderTitledBox(title, content string) string {
	box := NewBox(content)
	box.SetTitle(title)
	return box.View()
}

func RenderDoubleBox(title, content string) string {
	box := NewBox(content)
	box.SetTitle(title)
	box.SetStyle(BoxStyleDouble)
	return box.View()
}

func RenderAlertBox(alertType NotificationType, title, message string) string {
	var borderColor lipgloss.Color
	var icon string

	switch alertType {
	case NotificationSuccess:
		borderColor = agentColor
		icon = "✓"
	case NotificationWarning:
		borderColor = systemColor
		icon = "⚠"
	case NotificationError:
		borderColor = errorColor
		icon = "✗"
	default:
		borderColor = accentColor
		icon = "ℹ"
	}

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(40)

	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true)

	contentStyle := lipgloss.NewStyle().
		Foreground(textColor)

	return box.Render(titleStyle.Render(icon+" "+title) + "\n\n" + contentStyle.Render(message))
}
