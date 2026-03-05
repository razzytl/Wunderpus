package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ProgressBarModel struct {
	Label       string
	Total       float64
	Current     float64
	Width       int
	ShowPercent bool
	Color       lipgloss.Color
}

func NewProgressBar(label string, total float64) ProgressBarModel {
	return ProgressBarModel{
		Label:       label,
		Total:       total,
		Current:     0,
		Width:       40,
		ShowPercent: true,
		Color:       accentColor,
	}
}

func (p *ProgressBarModel) SetCurrent(current float64) {
	p.Current = math.Min(current, p.Total)
}

func (p *ProgressBarModel) Increment(delta float64) {
	p.Current = math.Min(p.Current+delta, p.Total)
}

func (p ProgressBarModel) View() string {
	if p.Total == 0 {
		return p.renderEmpty()
	}

	percent := p.Current / p.Total
	filled := int(float64(p.Width) * percent)

	barStyle := lipgloss.NewStyle().
		Foreground(p.Color).
		Bold(true)

	emptyStyle := lipgloss.NewStyle().
		Foreground(dimColor)

	filledStr := strings.Repeat("█", filled)
	emptyStr := strings.Repeat("░", p.Width-filled)

	bar := barStyle.Render(filledStr) + emptyStyle.Render(emptyStr)

	var result string
	if p.ShowPercent {
		percentStr := fmt.Sprintf("%.1f%%", percent*100)
		result = fmt.Sprintf("%s [%s] %s", p.Label, bar, percentStr)
	} else {
		currentStr := fmt.Sprintf("%.0f", p.Current)
		totalStr := fmt.Sprintf("%.0f", p.Total)
		result = fmt.Sprintf("%s [%s] %s/%s", p.Label, bar, currentStr, totalStr)
	}

	return result
}

func (p ProgressBarModel) renderEmpty() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(dimColor)

	bar := emptyStyle.Render(strings.Repeat("░", p.Width))
	return fmt.Sprintf("%s [%s]", p.Label, bar)
}

func (p ProgressBarModel) IsComplete() bool {
	return p.Current >= p.Total
}

type MultiProgressModel struct {
	Bars  map[string]*ProgressBarModel
	Width int
}

func NewMultiProgress() MultiProgressModel {
	return MultiProgressModel{
		Bars:  make(map[string]*ProgressBarModel),
		Width: 40,
	}
}

func (m *MultiProgressModel) AddBar(id string, label string, total float64) {
	m.Bars[id] = &ProgressBarModel{
		Label: label,
		Total: total,
		Width: m.Width,
	}
}

func (m *MultiProgressModel) UpdateBar(id string, current float64) {
	if bar, ok := m.Bars[id]; ok {
		bar.SetCurrent(current)
	}
}

func (m MultiProgressModel) View() string {
	var lines []string
	for _, bar := range m.Bars {
		lines = append(lines, bar.View())
	}
	return strings.Join(lines, "\n")
}

func (m MultiProgressModel) IsComplete() bool {
	for _, bar := range m.Bars {
		if !bar.IsComplete() {
			return false
		}
	}
	return true
}

func RenderProgressBox(title string, progress *ProgressBarModel) string {
	content := progress.View()

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	return boxStyle.Width(60).Render(
		titleStyle.Render(" "+title+" ") + "\n\n" + content,
	)
}
