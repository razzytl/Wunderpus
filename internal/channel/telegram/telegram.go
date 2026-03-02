package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/wonderpus/wonderpus/internal/agent"
)

// Channel implements the Telegram communication channel.
type Channel struct {
	token   string
	manager *agent.Manager
	bot     *tgbotapi.BotAPI
	stopCh  chan struct{}
}

// NewChannel creates a new Telegram channel.
func NewChannel(token string, manager *agent.Manager) *Channel {
	return &Channel{
		token:   token,
		manager: manager,
		stopCh:  make(chan struct{}),
	}
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "telegram"
}

// Start launches the Telegram bot polling loop.
func (c *Channel) Start(ctx context.Context) error {
	if c.token == "" {
		return fmt.Errorf("telegram: token is empty")
	}

	bot, err := tgbotapi.NewBotAPI(c.token)
	if err != nil {
		return fmt.Errorf("telegram: initializing bot: %w", err)
	}

	c.bot = bot
	slog.Info("telegram channel starting", "bot", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := c.bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-c.stopCh:
				return
			case <-ctx.Done():
				return
			case update := <-updates:
				if update.Message == nil {
					continue
				}

				chatID := update.Message.Chat.ID
				sessionID := fmt.Sprintf("tg_%d", chatID)
				text := update.Message.Text

				if text == "" {
					continue
				}

				// Handle commands
				if strings.HasPrefix(text, "/") {
					c.handleCommand(chatID, text)
					continue
				}

				// Standard message processing
				go func(id int64, sid, input string) {
					// Show typing indicator
					c.bot.Send(tgbotapi.NewChatAction(id, tgbotapi.ChatTyping))

					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer cancel()

					resp, err := c.manager.ProcessMessage(ctx, sid, input)
					if err != nil {
						msg := tgbotapi.NewMessage(id, "Error: "+err.Error())
						c.bot.Send(msg)
						return
					}

					msg := tgbotapi.NewMessage(id, resp)
					msg.ParseMode = tgbotapi.ModeMarkdown
					c.bot.Send(msg)
				}(chatID, sessionID, text)
			}
		}
	}()

	return nil
}

// Stop gracefully stops the bot.
func (c *Channel) Stop() error {
	close(c.stopCh)
	if c.bot != nil {
		c.bot.StopReceivingUpdates()
	}
	return nil
}

func (c *Channel) handleCommand(chatID int64, cmd string) {
	parts := strings.Fields(cmd)
	command := strings.ToLower(parts[0])

	switch command {
	case "/start":
		msg := tgbotapi.NewMessage(chatID, "Welcome to Wonderpus! 🐙\nI am your universal AI assistant. Send me a message to begin.")
		c.bot.Send(msg)
	case "/help":
		msg := tgbotapi.NewMessage(chatID, "I am an AI agent. You can chat with me, or use /reset to clear our conversation history.")
		c.bot.Send(msg)
	case "/reset":
		sessionID := fmt.Sprintf("tg_%d", chatID)
		ag := c.manager.GetAgent(sessionID)
		ag.ClearContext()
		msg := tgbotapi.NewMessage(chatID, "Conversation history cleared. ✓")
		c.bot.Send(msg)
	default:
		msg := tgbotapi.NewMessage(chatID, "Unknown command: "+command)
		c.bot.Send(msg)
	}
}

