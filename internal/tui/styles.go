package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette — dark-friendly, muted purple accent
var (
	accentColor   = lipgloss.Color("#9D7CD8") // muted purple
	dimColor      = lipgloss.Color("#565F89") // dim gray
	textColor     = lipgloss.Color("#C0CAF5") // soft white
	userColor     = lipgloss.Color("#7AA2F7") // calm blue
	agentColor    = lipgloss.Color("#9ECE6A") // soft green
	systemColor   = lipgloss.Color("#E0AF68") // warm yellow
	errorColor    = lipgloss.Color("#F7768E") // soft red
	bgColor       = lipgloss.Color("#1A1B26") // deep dark
	statusBgColor = lipgloss.Color("#24283B") // slightly lighter
)

// Styles used throughout the TUI.
var (
	// Container styles
	AppStyle = lipgloss.NewStyle().
			Background(bgColor)

	// Status bar at bottom
	StatusBarStyle = lipgloss.NewStyle().
			Background(statusBgColor).
			Foreground(dimColor).
			Padding(0, 1)

	StatusActiveStyle = lipgloss.NewStyle().
				Background(statusBgColor).
				Foreground(accentColor).
				Bold(true).
				Padding(0, 1)

	// Chat messages
	UserPrefixStyle = lipgloss.NewStyle().
			Foreground(userColor).
			Bold(true)

	AgentPrefixStyle = lipgloss.NewStyle().
				Foreground(agentColor).
				Bold(true)

	SystemPrefixStyle = lipgloss.NewStyle().
				Foreground(systemColor).
				Bold(true)

	MessageStyle = lipgloss.NewStyle().
			Foreground(textColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	// Input area
	InputPromptStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	// Banner
	BannerStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			Align(lipgloss.Center)

	DimStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Spinner
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	NotifyStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(accentColor)
)
