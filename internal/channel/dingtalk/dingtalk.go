package dingtalk

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/types"
)

type Channel struct {
	appKey    string
	appSecret string
	agentID   string
	manager   *agent.Manager
	token     string
	tokenMu   sync.RWMutex
	secret    string
}

func NewChannel(appKey, appSecret, agentID, secret string, manager *agent.Manager) *Channel {
	return &Channel{
		appKey:    appKey,
		appSecret: appSecret,
		agentID:   agentID,
		manager:   manager,
		secret:    secret,
	}
}

func (c *Channel) Name() string {
	return "dingtalk"
}

func (c *Channel) Start(ctx context.Context) error {
	if c.appKey == "" || c.appSecret == "" {
		return fmt.Errorf("DingTalk app_key and app_secret required")
	}

	if err := c.refreshToken(ctx); err != nil {
		return fmt.Errorf("failed to get DingTalk token: %w", err)
	}

	go func() {
		ticker := time.NewTicker(2 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.refreshToken(context.Background())
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (c *Channel) Stop() error {
	return nil
}

func (c *Channel) refreshToken(ctx context.Context) error {
	url := fmt.Sprintf("https://api.dingtalk.com/v1.0/oauth2/token?grantType=client_credentials&clientId=%s&clientSecret=%s",
		c.appKey, c.appSecret)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int    `json:"expireIn"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.tokenMu.Lock()
	c.token = result.AccessToken
	c.tokenMu.Unlock()

	return nil
}

func (c *Channel) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var event struct {
		Header struct {
			EventType string `json:"eventType"`
		} `json:"header"`
		Body struct {
			FromUserID string `json:"fromUserId"`
			MsgType    string `json:"msgType"`
			Content    string `json:"content"`
			SessionID  string `json:"sessionId"`
		} `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if event.Header.EventType != "im Message" {
		w.WriteHeader(http.StatusOK)
		return
	}

	content := event.Body.Content
	if content == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	sessionID := fmt.Sprintf("dingtalk_%s", event.Body.SessionID)

	go c.processMessage(sessionID, content, event.Body.FromUserID)

	w.WriteHeader(http.StatusOK)
}

func (c *Channel) processMessage(sessionID, content, userID string) {
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
		c.sendMessage(userID, fmt.Sprintf("Error: %v", err))
		return
	}

	c.sendMessage(userID, resp.Content)
}

func (c *Channel) sendMessage(userID, content string) {
	c.tokenMu.RLock()
	token := c.token
	c.tokenMu.RUnlock()

	if token == "" {
		return
	}

	url := "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"

	payload := map[string]any{
		"robotCode": c.agentID,
		"userIds":   []string{userID},
		"msg": map[string]any{
			"msgType": "text",
			"text": map[string]any{
				"content": content,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)

	http.DefaultClient.Do(req)
}

func (c *Channel) VerifySignature(timestamp, signature string) bool {
	if c.secret == "" {
		return true
	}

	stringToSign := fmt.Sprintf("%s\n%s", timestamp, c.secret)
	h := sha256.Sum256([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(h[:])

	return sign == signature
}
