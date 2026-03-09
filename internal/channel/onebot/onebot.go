package onebot

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/wonderpus/wonderpus/internal/agent"
	"github.com/wonderpus/wonderpus/internal/types"
)

type Channel struct {
	wsURL    string
	manager  *agent.Manager
	conn     *websocket.Conn
	mu       sync.Mutex
	stopChan chan struct{}
}

func NewChannel(wsURL string, manager *agent.Manager) *Channel {
	return &Channel{
		wsURL:    wsURL,
		manager:  manager,
		stopChan: make(chan struct{}),
	}
}

func (c *Channel) Name() string {
	return "onebot"
}

func (c *Channel) Start(ctx context.Context) error {
	if c.wsURL == "" {
		return fmt.Errorf("OneBot WebSocket URL required")
	}

	go c.connect(ctx)

	return nil
}

func (c *Channel) Stop() error {
	close(c.stopChan)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Channel) connect(ctx context.Context) {
	var err error
	dialer := websocket.Dialer{}
	c.conn, _, err = dialer.DialContext(ctx, c.wsURL, nil)
	if err != nil {
		fmt.Printf("OneBot WebSocket connection failed: %v\n", err)
		return
	}
	defer c.conn.Close()

	go c.readLoop()

	<-c.stopChan
}

func (c *Channel) readLoop() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var event map[string]any
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		c.handleEvent(event)
	}
}

func (c *Channel) handleEvent(event map[string]any) {
	msgType, _ := event["message_type"].(string)
	if msgType == "" {
		return
	}

	var content string
	switch msgType {
	case "private":
		if raw, ok := event["message"].(string); ok {
			content = raw
		}
	case "group":
		if raw, ok := event["message"].(string); ok {
			content = raw
		}
	}

	if content == "" {
		return
	}

	userID, _ := event["user_id"].(float64)
	sessionID := fmt.Sprintf("onebot_%d", int(userID))

	go c.processMessage(sessionID, content)
}

func (c *Channel) processMessage(sessionID, content string) {
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
		return
	}

	c.sendPrivateMessage(sessionID, resp.Content)
}

func (c *Channel) sendPrivateMessage(userID string, content string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return
	}

	msg := map[string]any{
		"action": "send_private_msg",
		"params": map[string]any{
			"user_id": userID,
			"message": content,
		},
	}

	c.conn.WriteJSON(msg)
}
