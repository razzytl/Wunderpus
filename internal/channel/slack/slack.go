package slack

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/wonderpus/wonderpus/internal/agent"
	"github.com/wonderpus/wonderpus/internal/types"
)

// Channel implements the Slack communication channel.
type Channel struct {
	token      string
	appToken   string
	socketMode bool
	manager    *agent.Manager
	client     *slack.Client
	socket     *socketmode.Client
}

// NewChannel creates a new Slack channel.
func NewChannel(token, appToken string, socketMode bool, manager *agent.Manager) *Channel {
	client := slack.New(token, slack.OptionAppLevelToken(appToken))
	return &Channel{
		token:      token,
		appToken:   appToken,
		socketMode: socketMode,
		manager:    manager,
		client:     client,
	}
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "slack"
}

// Start launches the Slack bot.
func (c *Channel) Start(ctx context.Context) error {
	if c.token == "" {
		return fmt.Errorf("slack: token is empty")
	}

	if c.socketMode {
		if c.appToken == "" {
			return fmt.Errorf("slack: app token is required for socket mode")
		}
		c.socket = socketmode.New(
			c.client,
			socketmode.OptionDebug(false),
		)
		go c.runSocketMode(ctx)
		slog.Info("slack channel starting (socket mode)")
	} else {
		slog.Info("slack channel starting (webhook mode - not fully implemented)")
	}

	return nil
}

// Stop gracefully stops the bot.
func (c *Channel) Stop() error {
	// Socket mode doesn't need explicit stop usually, but we could close connections
	return nil
}

func (c *Channel) runSocketMode(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-c.socket.Events:
				switch evt.Type {
				case socketmode.EventTypeEventsAPI:
					eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
					if !ok {
						continue
					}
					c.socket.Ack(*evt.Request)
					c.handleEventsAPI(eventsAPIEvent)
				}
			}
		}
	}()

	c.socket.Run()
}

func (c *Channel) handleEventsAPI(event slackevents.EventsAPIEvent) {
	if event.Type != slackevents.CallbackEvent {
		return
	}

	innerEvent := event.InnerEvent.Data
	switch ev := innerEvent.(type) {
	case *slackevents.AppMentionEvent:
		c.processMessage(ev.Channel, ev.User, ev.Text)
	case *slackevents.MessageEvent:
		// Ignore messages from bots
		if ev.BotID != "" {
			return
		}
		// Handle DMs
		if ev.ChannelType == "im" {
			c.processMessage(ev.Channel, ev.User, ev.Text)
		}
	}
}

func (c *Channel) processMessage(channelID, userID, text string) {
	// Clean text (remove bot mention)
	// Slack mentions are like <@U12345678>
	// For simplicity, we just pass the text as is or do minimal cleaning
	
	sessionID := fmt.Sprintf("slack_%s_%s", userID, channelID)
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		respRes, err := c.manager.ProcessRequest(ctx, types.UserMessage{
			SessionID: sessionID,
			Content:   text,
			ChannelID: channelID,
		})
		
		var reply string
		if err != nil {
			reply = "Error: " + err.Error()
		} else {
			reply = respRes.Content
		}

		_, _, err = c.client.PostMessage(channelID, slack.MsgOptionText(reply, false))
		if err != nil {
			slog.Error("slack: failed to post message", "error", err)
		}
	}()
}
