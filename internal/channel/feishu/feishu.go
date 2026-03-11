package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/types"
)

// Channel implements the Feishu (Lark) communication channel.
type Channel struct {
	appID             string
	appSecret         string
	verificationToken string
	manager           *agent.Manager
}

// NewChannel creates a new Feishu channel.
func NewChannel(appID, appSecret, verificationToken string, manager *agent.Manager) *Channel {
	return &Channel{
		appID:             appID,
		appSecret:         appSecret,
		verificationToken: verificationToken,
		manager:           manager,
	}
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "feishu"
}

// Start launches the Feishu integration (usually just registers webhook handlers).
func (c *Channel) Start(ctx context.Context) error {
	slog.Info("feishu channel starting (webhook mode)")
	return nil
}

// Stop gracefully stops the channel.
func (c *Channel) Stop() error {
	return nil
}

// HandleWebhook processes incoming Feishu webhooks.
func (c *Channel) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Handle URL verification
	if payload["type"] == "url_verification" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"challenge": payload["challenge"],
		})
		return
	}

	// Handle events
	if header, ok := payload["header"].(map[string]interface{}); ok {
		eventID := header["event_id"]
		slog.Debug("feishu: received event", "event_id", eventID)

		// Map event to internal message
		// Note: Feishu has complex event schemas, this is a simplified version
		if event, ok := payload["event"].(map[string]interface{}); ok {
			if message, ok := event["message"].(map[string]interface{}); ok {
				if contentStr, ok := message["content"].(string); ok {
					var content map[string]string
					json.Unmarshal([]byte(contentStr), &content)
					text := content["text"]

					chatID := message["chat_id"].(string)
					sender := event["sender"].(map[string]interface{})
					senderID := sender["sender_id"].(map[string]interface{})["open_id"].(string)

					c.processMessage(chatID, senderID, text)
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (c *Channel) processMessage(chatID, senderID, text string) {
	sessionID := fmt.Sprintf("feishu_%s", senderID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		_, err := c.manager.ProcessRequest(ctx, types.UserMessage{
			SessionID: sessionID,
			Content:   text,
			ChannelID: chatID,
		})

		if err != nil {
			slog.Error("feishu: failed to process message", "error", err)
			return
		}

		// In a real Feishu bot, we would call the Open Platform API to reply
		// This requires an access token (tenant_access_token)
		slog.Info("feishu: would reply to chat", "chat_id", chatID)
	}()
}
