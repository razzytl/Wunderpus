package tui

//go:generate stringer -type=LogLevel -linecomment

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Source    string
}

type LogModel struct {
	Entries    []LogEntry
	MaxEntries int
	ShowTime   bool
	ShowLevel  bool
	ShowSource bool
	Width      int
	Filter     LogLevel
}

func NewLog() *LogModel {
	return &LogModel{
		Entries:    make([]LogEntry, 0),
		MaxEntries: 100,
		ShowTime:   true,
		ShowLevel:  true,
		ShowSource: false,
		Width:      60,
		Filter:     LogLevelDebug,
	}
}

func (l *LogModel) AddEntry(level LogLevel, message string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    "",
	}
	l.Entries = append(l.Entries, entry)

	if len(l.Entries) > l.MaxEntries {
		l.Entries = l.Entries[1:]
	}
}

func (l *LogModel) AddEntryWithSource(level LogLevel, message, source string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
	}
	l.Entries = append(l.Entries, entry)

	if len(l.Entries) > l.MaxEntries {
		l.Entries = l.Entries[1:]
	}
}

func (l *LogModel) Debug(message string) {
	l.AddEntry(LogLevelDebug, message)
}

func (l *LogModel) Info(message string) {
	l.AddEntry(LogLevelInfo, message)
}

func (l *LogModel) Warn(message string) {
	l.AddEntry(LogLevelWarn, message)
}

func (l *LogModel) Error(message string) {
	l.AddEntry(LogLevelError, message)
}

func (l LogModel) View() string {
	var lines []string

	for _, entry := range l.Entries {
		if entry.Level < l.Filter {
			continue
		}

		lines = append(lines, l.formatEntry(entry))
	}

	return strings.Join(lines, "\n")
}

func (l LogModel) formatEntry(entry LogEntry) string {
	var parts []string

	if l.ShowTime {
		timeStr := entry.Timestamp.Format("15:04:05")
		timeStyle := lipgloss.NewStyle().Foreground(dimColor)
		parts = append(parts, timeStyle.Render(timeStr))
	}

	if l.ShowLevel {
		levelStr := l.formatLevel(entry.Level)
		parts = append(parts, levelStr)
	}

	if l.ShowSource && entry.Source != "" {
		sourceStyle := lipgloss.NewStyle().Foreground(systemColor)
		parts = append(parts, sourceStyle.Render("["+entry.Source+"]"))
	}

	msgStyle := lipgloss.NewStyle().Foreground(textColor)
	parts = append(parts, msgStyle.Render(entry.Message))

	return strings.Join(parts, " ")
}

func (l LogModel) formatLevel(level LogLevel) string {
	var levelStr string
	var style lipgloss.Style

	switch level {
	case LogLevelDebug:
		levelStr = "DEBUG"
		style = lipgloss.NewStyle().Foreground(dimColor)
	case LogLevelInfo:
		levelStr = "INFO"
		style = lipgloss.NewStyle().Foreground(agentColor)
	case LogLevelWarn:
		levelStr = "WARN"
		style = lipgloss.NewStyle().Foreground(systemColor)
	case LogLevelError:
		levelStr = "ERROR"
		style = lipgloss.NewStyle().Foreground(errorColor)
	default:
		levelStr = "????"
		style = lipgloss.NewStyle().Foreground(dimColor)
	}

	return style.Render(fmt.Sprintf("[%s]", levelStr))
}

func (l LogModel) Clear() {
	l.Entries = make([]LogEntry, 0)
}

func RenderLogBox(title string, log LogModel) string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	content := titleStyle.Render(" "+title+" ") + "\n\n" + log.View()

	return box.Render(content)
}
