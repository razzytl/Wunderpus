package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/wonderpus/wonderpus/internal/agent"
)

// Channel implements the Discord communication channel.
type Channel struct {
	token   string
	manager *agent.Manager
	session *discordgo.Session
}

// NewChannel creates a new Discord channel.
func NewChannel(token string, manager *agent.Manager) *Channel {
	return &Channel{
		token:   token,
		manager: manager,
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

		resp, err := c.manager.ProcessMessage(ctx, id, input)
		if err != nil {
			s.ChannelMessageSend(channelID, "Error: "+err.Error())
			return
		}

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

func (c *Channel) handleCommand(s *discordgo.Session, m *discordgo.MessageCreate, cmd string) {
	parts := strings.Fields(cmd)
	command := strings.ToLower(parts[0])

	switch command {
	case "!start":
		s.ChannelMessageSend(m.ChannelID, "Welcome to Wonderpus! 🐙\nI am your universal AI assistant. DM me or mention me in a channel to begin.")
	case "!help":
		s.ChannelMessageSend(m.ChannelID, "I am an AI agent. You can chat with me, or use !reset to clear our conversation history.")
	case "!reset":
		sessionID := fmt.Sprintf("discord_%s_%s", m.Author.ID, m.ChannelID)
		ag := c.manager.GetAgent(sessionID)
		ag.ClearContext()
		s.ChannelMessageSend(m.ChannelID, "Conversation history cleared. ✓")
	}
}
