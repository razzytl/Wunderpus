package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wonderpus/wonderpus/internal/agent"
	"github.com/wonderpus/wonderpus/internal/provider"
)

const banner = `
  🐙 Wonderpus
  Universal AI Agent
`

// streamDoneMsg signals streaming is complete.
type streamDoneMsg struct{ full string }

// streamChunkMsg carries a chunk of streamed text.
type streamChunkMsg struct{ text string }

// streamErrMsg carries a stream error.
type streamErrMsg struct{ err error }

// toolMsg carries info about a tool execution.
type toolMsg struct {
	name   string
	args   map[string]any
	result string
}

// Model is the Bubbletea model for the TUI.
type Model struct {
	agent    *agent.Agent
	viewport viewport.Model
	input    textarea.Model
	spinner  spinner.Model
	chatLog  []string
	width    int
	height   int
	streaming bool
	streamBuf string
	ready    bool
	quitting bool
}

// New creates a new TUI model.
func New(ag *agent.Agent) Model {
	// Input area
	ta := textarea.New()
	ta.Placeholder = "Type a message... (Enter to send, /help for commands)"
	ta.Focus()
	ta.CharLimit = 4000
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1)
	ta.BlurredStyle.Base = ta.FocusedStyle.Base

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	return Model{
		agent:   ag,
		input:   ta,
		spinner: sp,
		chatLog: []string{
			BannerStyle.Render(banner),
			DimStyle.Render("  Provider: " + ag.Router().ActiveName() + " │ Type /help for commands"),
			"",
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 0
		inputHeight := 5 // textarea + border
		statusHeight := 1
		vpHeight := m.height - headerHeight - inputHeight - statusHeight - 1

		if !m.ready {
			m.viewport = viewport.New(m.width, vpHeight)
			m.viewport.SetContent(strings.Join(m.chatLog, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpHeight
		}

		m.input.SetWidth(m.width - 4)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.streaming {
				break
			}
			val := strings.TrimSpace(m.input.Value())
			if val == "" {
				break
			}
			m.input.Reset()

			// Handle commands
			if strings.HasPrefix(val, "/") {
				return m.handleCommand(val)
			}

			// Send message
			m.appendChat(UserPrefixStyle.Render("You") + " " + MessageStyle.Render(val))
			m.streaming = true
			m.streamBuf = ""
			return m, m.sendStreamCmd(val)
		}

	case streamChunkMsg:
		m.streamBuf += msg.text
		m.updateStreamLine()
		return m, nil

	case streamDoneMsg:
		m.streaming = false
		// Replace the streaming line with final version
		if len(m.chatLog) > 0 && strings.HasPrefix(m.chatLog[len(m.chatLog)-1], AgentPrefixStyle.Render("🐙")) {
			m.chatLog[len(m.chatLog)-1] = AgentPrefixStyle.Render("🐙") + " " + MessageStyle.Render(msg.full)
		} else {
			m.appendChat(AgentPrefixStyle.Render("🐙") + " " + MessageStyle.Render(msg.full))
		}
		m.appendChat("") // blank line separator
		m.refreshViewport()
		return m, nil

	case streamErrMsg:
		m.streaming = false
		m.appendChat(ErrorStyle.Render("Error: " + msg.err.Error()))
		m.appendChat("")
		return m, nil

	case toolMsg:
		argStr := fmt.Sprintf("%v", msg.args)
		if len(argStr) > 50 {
			argStr = argStr[:47] + "..."
		}
		m.appendChat(DimStyle.Render(fmt.Sprintf("  🔧 Tool Executed: %s( %s )", msg.name, argStr)))
		resultLine := msg.result
		if len(resultLine) > 100 {
			resultLine = resultLine[:97] + "..."
		}
		m.appendChat(DimStyle.Render(fmt.Sprintf("     ↳ %s", strings.ReplaceAll(resultLine, "\n", " "))))
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update textarea
	if !m.streaming {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}
	if m.quitting {
		return DimStyle.Render("Goodbye! 🐙\n")
	}

	// Status bar
	providerInfo := StatusActiveStyle.Render("⚡ " + m.agent.Router().ActiveName())
	msgCount := StatusBarStyle.Render(fmt.Sprintf("│ msgs: %d", m.agent.MessageCount()))
	streamIndicator := ""
	if m.streaming {
		streamIndicator = StatusBarStyle.Render(" │ ") + SpinnerStyle.Render(m.spinner.View()+" thinking...")
	}
	statusBar := lipgloss.JoinHorizontal(lipgloss.Top, providerInfo, msgCount, streamIndicator)
	statusLine := StatusBarStyle.Width(m.width).Render(statusBar)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		statusLine,
		m.input.View(),
	)
}

func (m *Model) appendChat(line string) {
	m.chatLog = append(m.chatLog, line)
	m.refreshViewport()
}

func (m *Model) refreshViewport() {
	m.viewport.SetContent(strings.Join(m.chatLog, "\n"))
	m.viewport.GotoBottom()
}

func (m *Model) updateStreamLine() {
	line := AgentPrefixStyle.Render("🐙") + " " + MessageStyle.Render(m.streamBuf) + SpinnerStyle.Render("▌")
	if len(m.chatLog) > 0 && strings.HasPrefix(m.chatLog[len(m.chatLog)-1], AgentPrefixStyle.Render("🐙")) {
		m.chatLog[len(m.chatLog)-1] = line
	} else {
		m.chatLog = append(m.chatLog, line)
	}
	m.refreshViewport()
}

func (m Model) sendStreamCmd(input string) tea.Cmd {
	return func() tea.Msg {
		ch, err := m.agent.StreamMessage(context.Background(), input)
		if err != nil {
			return streamErrMsg{err: err}
		}

		var full string
		for chunk := range ch {
			if chunk.Error != nil {
				return streamErrMsg{err: chunk.Error}
			}
			if chunk.Done {
				break
			}
			full += chunk.Content
			// We can't send individual messages from here in a blocking func,
			// so we accumulate and return the full result.
		}
		return streamDoneMsg{full: full}
	}
}

func (m Model) handleCommand(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	command := strings.ToLower(parts[0])

	switch command {
	case "/help":
		help := []string{
			"",
			SystemPrefixStyle.Render("Commands:"),
			DimStyle.Render("  /help              Show this help"),
			DimStyle.Render("  /clear             Clear conversation"),
			DimStyle.Render("  /status            Show agent status"),
			DimStyle.Render("  /provider <name>   Switch provider (openai, anthropic, ollama, gemini)"),
			DimStyle.Render("  /providers         List available providers"),
			DimStyle.Render("  /tools             List available tools"),
			DimStyle.Render("  /exit              Quit"),
			"",
		}
		for _, l := range help {
			m.appendChat(l)
		}

	case "/clear":
		m.agent.ClearContext()
		m.chatLog = m.chatLog[:3] // keep banner
		m.appendChat(SystemPrefixStyle.Render("✓ Conversation cleared"))
		m.appendChat("")
		m.refreshViewport()

	case "/status":
		status := []string{
			"",
			SystemPrefixStyle.Render("Status:"),
			DimStyle.Render(fmt.Sprintf("  Provider:  %s", m.agent.Router().ActiveName())),
			DimStyle.Render(fmt.Sprintf("  Messages:  %d", m.agent.MessageCount())),
			DimStyle.Render(fmt.Sprintf("  Available: %s", strings.Join(m.agent.Router().List(), ", "))),
			"",
		}
		for _, l := range status {
			m.appendChat(l)
		}

	case "/provider":
		if len(parts) < 2 {
			m.appendChat(ErrorStyle.Render("Usage: /provider <name>"))
			m.appendChat("")
		} else {
			name := parts[1]
			if err := m.agent.Router().SetActive(name); err != nil {
				m.appendChat(ErrorStyle.Render("Error: " + err.Error()))
			} else {
				m.appendChat(SystemPrefixStyle.Render("✓ Switched to " + name))
			}
			m.appendChat("")
		}

	case "/providers":
		providers := m.agent.Router().List()
		m.appendChat("")
		m.appendChat(SystemPrefixStyle.Render("Available providers:"))
		for _, p := range providers {
			marker := "  "
			if p == m.agent.Router().ActiveName() {
				marker = "▸ "
			}
			m.appendChat(DimStyle.Render("  " + marker + p))
		}
		m.appendChat("")

	case "/tools":
		m.appendChat("")
		m.appendChat(SystemPrefixStyle.Render("Available tools:"))
		// Not directly exposed on agent cleanly, so we just list the typical built-ins for Phase 2
		m.appendChat(DimStyle.Render("  ▸ file_read      (Read file contents)"))
		m.appendChat(DimStyle.Render("  ▸ file_write     (Write to file - sandbox)"))
		m.appendChat(DimStyle.Render("  ▸ file_list      (List directory contents)"))
		m.appendChat(DimStyle.Render("  ▸ shell_exec     (Execute whitelisted commands)"))
		m.appendChat(DimStyle.Render("  ▸ http_request   (Make HTTP requests - SSRF protected)"))
		m.appendChat(DimStyle.Render("  ▸ calculator     (Evaluate math expressions)"))
		m.appendChat("")

	case "/exit":
		m.quitting = true
		return m, tea.Quit

	default:
		m.appendChat(ErrorStyle.Render("Unknown command: " + command + " (try /help)"))
		m.appendChat("")
	}

	return m, nil
}

// Run starts the TUI application.
func Run(ag *agent.Agent) error {
	_ = provider.RoleSystem // ensure import
	m := New(ag)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Wire up tool callback to send messages to the TUI
	ag.SetToolCallback(func(name string, args map[string]any, result string) {
		p.Send(toolMsg{name: name, args: args, result: result})
	})

	_, err := p.Run()
	return err
}
