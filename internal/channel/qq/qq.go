package qq

import (
	"context"
	"fmt"
	"sync"

	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"golang.org/x/oauth2"

	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/types"
)

type Channel struct {
	appID        string
	appSecret    string
	manager      *agent.Manager
	api          openapi.OpenAPI
	tokenSource  oauth2.TokenSource
	ctx          context.Context
	cancel       context.CancelFunc
	processedIDs map[string]bool
	mu           sync.RWMutex
}

func NewChannel(appID, appSecret string, manager *agent.Manager) *Channel {
	return &Channel{
		appID:        appID,
		appSecret:    appSecret,
		manager:      manager,
		processedIDs: make(map[string]bool),
	}
}

func (c *Channel) Name() string {
	return "qq"
}

func (c *Channel) Start(ctx context.Context) error {
	if c.appID == "" || c.appSecret == "" {
		return fmt.Errorf("QQ app_id and app_secret not configured")
	}

	credentials := &token.QQBotCredentials{
		AppID:     c.appID,
		AppSecret: c.appSecret,
	}
	c.tokenSource = token.NewQQBotTokenSource(credentials)

	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := token.StartRefreshAccessToken(c.ctx, c.tokenSource); err != nil {
		return fmt.Errorf("failed to start token refresh: %w", err)
	}

	c.api = botgo.NewOpenAPI(c.appID, c.tokenSource).WithTimeout(5)

	intent := event.RegisterHandlers(
		c.handleC2CMessage(),
		c.handleGroupATMessage(),
	)

	wsInfo, err := c.api.WS(c.ctx, nil, "")
	if err != nil {
		return fmt.Errorf("failed to get websocket info: %w", err)
	}

	sessionManager := botgo.NewSessionManager()
	go func() {
		if err := sessionManager.Start(wsInfo, c.tokenSource, &intent); err != nil {
			fmt.Printf("QQ WebSocket error: %v\n", err)
		}
	}()

	return nil
}

func (c *Channel) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func (c *Channel) handleC2CMessage() event.C2CMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSC2CMessageData) error {
		if c.isDuplicate(data.ID) {
			return nil
		}

		if data.Author == nil || data.Author.ID == "" {
			return nil
		}

		content := data.Content
		if content == "" {
			return nil
		}

		senderID := data.Author.ID
		sessionID := fmt.Sprintf("qq_%s", senderID)

		go c.processMessage(sessionID, content, senderID)

		return nil
	}
}

func (c *Channel) handleGroupATMessage() event.GroupATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
		if c.isDuplicate(data.ID) {
			return nil
		}

		if data.Author == nil || data.Author.ID == "" {
			return nil
		}

		content := data.Content
		if content == "" {
			return nil
		}

		senderID := data.Author.ID
		sessionID := fmt.Sprintf("qq_group_%s", data.GroupID)

		go c.processMessage(sessionID, content, senderID)

		return nil
	}
}

func (c *Channel) processMessage(sessionID, content, senderID string) {
	if c.manager == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*60)
	defer cancel()

	resp, err := c.manager.ProcessRequest(ctx, types.UserMessage{
		SessionID: sessionID,
		Content:   content,
	})

	if err != nil {
		c.sendMessage(senderID, fmt.Sprintf("Error: %v", err))
		return
	}

	c.sendMessage(senderID, resp.Content)
}

func (c *Channel) sendMessage(targetID, content string) {
	if c.api == nil {
		return
	}

	msg := &dto.MessageToCreate{
		Content: content,
	}

	_, err := c.api.PostC2CMessage(context.Background(), targetID, msg)
	if err != nil {
		fmt.Printf("QQ send error: %v\n", err)
	}
}

func (c *Channel) isDuplicate(messageID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.processedIDs[messageID] {
		return true
	}

	c.processedIDs[messageID] = true

	if len(c.processedIDs) > 10000 {
		count := 0
		for id := range c.processedIDs {
			if count >= 5000 {
				break
			}
			delete(c.processedIDs, id)
			count++
		}
	}

	return false
}
