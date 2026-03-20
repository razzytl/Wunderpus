package telegram

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/logging"
	"github.com/wunderpus/wunderpus/internal/tool/builtin"
	"github.com/wunderpus/wunderpus/internal/types"
)

// Channel implements the Telegram communication channel.
type Channel struct {
	token   string
	manager *agent.Manager
	bot     *tgbotapi.BotAPI
	stopCh  chan struct{}
	hitl    *builtin.HumanInTheLoop
}

// NewChannel creates a new Telegram channel.
func NewChannel(token string, manager *agent.Manager) *Channel {
	return &Channel{
		token:   token,
		manager: manager,
		stopCh:  make(chan struct{}),
	}
}

// NewChannelWithHITL creates a new Telegram channel with Human-in-the-Loop support.
func NewChannelWithHITL(token string, manager *agent.Manager, hitl *builtin.HumanInTheLoop) *Channel {
	return &Channel{
		token:   token,
		manager: manager,
		stopCh:  make(chan struct{}),
		hitl:    hitl,
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

	go func() {
		for {
			u := tgbotapi.NewUpdate(0)
			u.Timeout = 60
			updates := c.bot.GetUpdatesChan(u)

			for {
				select {
				case <-c.stopCh:
					return
				case <-ctx.Done():
					return
				case update, ok := <-updates:
					if !ok {
						slog.Warn("telegram: update channel closed, attempting to reconnect...")
						goto reconnect
					}

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

					// Handle HITL responses
					if c.hitl != nil && strings.HasPrefix(text, "/respond ") {
						c.handleHITLResponse(chatID, text)
						continue
					}

					// Standard message processing
					go func(id int64, sid, input string) {
						defer func() {
							if r := recover(); r != nil {
								slog.Error("PANIC in telegram message handler", "panic", r)
							}
						}()

						// Generate correlation ID
						cidBytes := make([]byte, 8)
						_, _ = rand.Read(cidBytes)
						cid := hex.EncodeToString(cidBytes)

						ctx := logging.ContextWithCorrelation(context.Background(), cid)
						ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
						defer cancel()

						logging.L(ctx).Info("processing telegram message", "chat_id", id, "session_id", sid)

						// Show typing indicator
						c.bot.Send(tgbotapi.NewChatAction(id, tgbotapi.ChatTyping))

						respRes, err := c.manager.ProcessRequest(ctx, types.UserMessage{
							SessionID: sid,
							Content:   input,
							ChannelID: fmt.Sprintf("%d", id),
						})
						if err != nil {
							logging.L(ctx).Error("failed to process message", "error", err)
							msg := tgbotapi.NewMessage(id, "Error: "+err.Error())
							c.bot.Send(msg)
							return
						}

						msg := tgbotapi.NewMessage(id, respRes.Content)
						msg.ParseMode = tgbotapi.ModeMarkdown
						c.bot.Send(msg)
					}(chatID, sessionID, text)
				}
			}

		reconnect:
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			case <-c.stopCh:
				return
			}

			newBot, err := tgbotapi.NewBotAPI(c.token)
			if err == nil {
				c.bot = newBot
				slog.Info("telegram: reconnected successfully")
			} else {
				slog.Error("telegram: reconnection failed", "error", err)
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
		msg := tgbotapi.NewMessage(chatID, "I am an AI agent. You can chat with me, or use the buttons below for quick actions:")

		// Create inline keyboard
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Reset Chat 🔄", "/reset"),
				tgbotapi.NewInlineKeyboardButtonData("Status 📊", "/status"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("About 🐙", "/about"),
			),
		)
		msg.ReplyMarkup = keyboard
		c.bot.Send(msg)
	case "/reset":
		sessionID := fmt.Sprintf("tg_%d", chatID)
		ag := c.manager.GetAgent(sessionID)
		ag.ClearContext()
		msg := tgbotapi.NewMessage(chatID, "Conversation history cleared. ✓")
		c.bot.Send(msg)
	case "/pending":
		if c.hitl != nil {
			c.handlePendingCommand(chatID)
		} else {
			msg := tgbotapi.NewMessage(chatID, "Human-in-the-loop not enabled.")
			c.bot.Send(msg)
		}
	default:
		msg := tgbotapi.NewMessage(chatID, "Unknown command: "+command)
		c.bot.Send(msg)
	}
}

// handleHITLResponse processes a human response to a pending HITL request
func (c *Channel) handleHITLResponse(chatID int64, text string) {
	// Format: /respond <index> <response>
	parts := strings.Fields(text)
	if len(parts) < 3 {
		msg := tgbotapi.NewMessage(chatID, "Usage: /respond <index> <your response>\nUse /pending to see pending requests.")
		c.bot.Send(msg)
		return
	}

	// Parse index
	index, err := strconv.Atoi(parts[1])
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Invalid index. Use /pending to see pending requests.")
		c.bot.Send(msg)
		return
	}

	// Get response text
	response := strings.Join(parts[2:], " ")

	// Send response through HITL
	err = c.hitl.SendResponseByIndex(index, response)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Error: %v", err))
		c.bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "✓ Response sent to agent!")
	c.bot.Send(msg)
}

// handlePendingCommand shows pending HITL requests
func (c *Channel) handlePendingCommand(chatID int64) {
	if c.hitl == nil {
		msg := tgbotapi.NewMessage(chatID, "Human-in-the-loop not enabled.")
		c.bot.Send(msg)
		return
	}

	pending := c.hitl.ListPending()
	if len(pending) == 0 {
		msg := tgbotapi.NewMessage(chatID, "No pending human requests.")
		c.bot.Send(msg)
		return
	}

	response := "📋 *Pending Human Requests:*\n"
	for i, req := range pending {
		response += fmt.Sprintf("%d. %s\n", i, req.Question)
		if len(req.ImageData) > 0 {
			response += "   (📎 Includes screenshot)\n"
		}
	}
	response += "\nTo respond: /respond <index> <your response>"

	msg := tgbotapi.NewMessage(chatID, response)
	msg.ParseMode = tgbotapi.ModeMarkdown
	c.bot.Send(msg)
}
