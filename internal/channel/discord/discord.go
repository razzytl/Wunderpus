package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/tool/builtin"
	"github.com/wunderpus/wunderpus/internal/types"
)

// Channel implements the Discord communication channel.
type Channel struct {
	token   string
	manager *agent.Manager
	session *discordgo.Session
	hitl    *builtin.HumanInTheLoop
}

// NewChannel creates a new Discord channel.
func NewChannel(token string, manager *agent.Manager) *Channel {
	return &Channel{
		token:   token,
		manager: manager,
	}
}

// NewChannelWithHITL creates a new Discord channel with Human-in-the-Loop support.
func NewChannelWithHITL(token string, manager *agent.Manager, hitl *builtin.HumanInTheLoop) *Channel {
	return &Channel{
		token:   token,
		manager: manager,
		hitl:    hitl,
	}
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "discord"
}

// Start launches the Discord bot.
func (c *Channel) Start(ctx context.Context) error {
	if c.token == "" {
		return fmt.Errorf("discord: token is empty")
	}

	dg, err := discordgo.New("Bot " + c.token)
	if err != nil {
		return fmt.Errorf("discord: creating session: %w", err)
	}

	dg.AddHandler(c.messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	err = dg.Open()
	if err != nil {
		return fmt.Errorf("discord: opening connection: %w", err)
	}

	c.session = dg
	slog.Info("discord channel starting", "user", dg.State.User.Username)

	return nil
}

// Stop gracefully stops the bot.
func (c *Channel) Stop() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

func (c *Channel) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Basic session isolation: UserID + ChannelID
	sessionID := fmt.Sprintf("discord_%s_%s", m.Author.ID, m.ChannelID)
	text := m.Content

	if text == "" {
		return
	}

	// Handle HITL responses first - check if this is a response to a pending request
	if c.hitl != nil && strings.HasPrefix(text, "!respond ") {
		c.handleHITLResponse(s, m, text)
		return
	}

	// Handle commands
	if strings.HasPrefix(text, "!") {
		c.handleCommand(s, m, text)
		return
	}

	// Only respond to DMs or mentions in servers
	isDM := m.GuildID == ""
	mentioned := false
	for _, u := range m.Mentions {
		if u.ID == s.State.User.ID {
			mentioned = true
			break
		}
	}

	if !isDM && !mentioned {
		return
	}

	// Clean mention from text
	if mentioned {
		text = strings.ReplaceAll(text, fmt.Sprintf("<@%s>", s.State.User.ID), "")
		text = strings.ReplaceAll(text, fmt.Sprintf("<@!%s>", s.State.User.ID), "")
		text = strings.TrimSpace(text)
	}

	go func(id string, input string, channelID string) {
		// Typing indicator
		s.ChannelTyping(channelID)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		respRes, err := c.manager.ProcessRequest(ctx, types.UserMessage{
			SessionID: id,
			Content:   input,
			ChannelID: channelID,
		})
		if err != nil {
			s.ChannelMessageSend(channelID, "Error: "+err.Error())
			return
		}

		resp := respRes.Content
		// Discord has a 2000 char limit
		if len(resp) > 2000 {
			for i := 0; i < len(resp); i += 1900 {
				end := i + 1900
				if end > len(resp) {
					end = len(resp)
				}
				s.ChannelMessageSend(channelID, resp[i:end])
			}
		} else {
			s.ChannelMessageSend(channelID, resp)
		}
	}(sessionID, text, m.ChannelID)
}

// handleHITLResponse processes a human response to a pending HITL request
func (c *Channel) handleHITLResponse(s *discordgo.Session, m *discordgo.MessageCreate, text string) {
	// Format: !respond <index> <response>
	parts := strings.Fields(text)
	if len(parts) < 3 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!respond <index> <your response>`\nUse `!pending` to see pending requests.")
		return
	}

	// Parse index
	var index int
	_, err := fmt.Sscanf(parts[1], "%d", &index)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Invalid index. Use `!pending` to see pending requests.")
		return
	}

	// Get response text
	response := strings.Join(parts[2:], " ")

	// Send response through HITL
	err = c.hitl.SendResponseByIndex(index, response)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %v", err))
		return
	}

	s.ChannelMessageSend(m.ChannelID, "✓ Response sent to agent!")
}

// HandlePendingCommand shows pending HITL requests
func (c *Channel) HandlePendingCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if c.hitl == nil {
		s.ChannelMessageSend(m.ChannelID, "Human-in-the-loop not enabled.")
		return
	}

	pending := c.hitl.ListPending()
	if len(pending) == 0 {
		s.ChannelMessageSend(m.ChannelID, "No pending human requests.")
		return
	}

	msg := "📋 **Pending Human Requests:**\n"
	for i, req := range pending {
		msg += fmt.Sprintf("%d. %s\n", i, req.Question)
		if len(req.ImageData) > 0 {
			msg += "   (📎 Includes screenshot)\n"
		}
	}
	msg += "\nTo respond: `!respond <index> <your response>`"

	s.ChannelMessageSend(m.ChannelID, msg)
}

func (c *Channel) handleCommand(s *discordgo.Session, m *discordgo.MessageCreate, cmd string) {
	parts := strings.Fields(cmd)
	command := strings.ToLower(parts[0])

	switch command {
	case "!start":
		s.ChannelMessageSend(m.ChannelID, "Welcome to Wonderpus! 🐙\nI am your universal AI assistant. DM me or mention me in a channel to begin.")
	case "!help":
		msg := &discordgo.MessageSend{
			Content: "I am an AI agent. You can chat with me, or use the buttons below for quick actions:",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Reset Chat",
							Style:    discordgo.SecondaryButton,
							CustomID: "reset_chat",
							Emoji:    &discordgo.ComponentEmoji{Name: "🔄"},
						},
						discordgo.Button{
							Label:    "Status",
							Style:    discordgo.SecondaryButton,
							CustomID: "view_status",
							Emoji:    &discordgo.ComponentEmoji{Name: "📊"},
						},
					},
				},
			},
		}
		s.ChannelMessageSendComplex(m.ChannelID, msg)
	case "!reset":
		sessionID := fmt.Sprintf("discord_%s_%s", m.Author.ID, m.ChannelID)
		ag := c.manager.GetAgent(sessionID)
		ag.ClearContext()
		s.ChannelMessageSend(m.ChannelID, "Conversation history cleared. ✓")
	case "!pending":
		if c.hitl != nil {
			c.HandlePendingCommand(s, m)
		} else {
			s.ChannelMessageSend(m.ChannelID, "Human-in-the-loop not enabled.")
		}
	}
}
