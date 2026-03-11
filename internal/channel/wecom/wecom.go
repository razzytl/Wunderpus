package wecom

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/agent"
	"github.com/wunderpus/wunderpus/internal/types"
)

type Mode string

const (
	ModeBot   Mode = "bot"
	ModeApp   Mode = "app"
	ModeAIBot Mode = "aibot"
)

type Channel struct {
	corpID         string
	corpSecret     string
	agentID        string
	token          string
	encodingAESKey string
	mode           Mode
	manager        *agent.Manager
	tokenMu        sync.RWMutex
	secret         string
}

func NewChannel(corpID, corpSecret, agentID, encodingAESKey string, mode Mode, manager *agent.Manager) *Channel {
	return &Channel{
		corpID:         corpID,
		corpSecret:     corpSecret,
		agentID:        agentID,
		encodingAESKey: encodingAESKey,
		mode:           mode,
		manager:        manager,
		secret:         encodingAESKey,
	}
}

func (c *Channel) Name() string {
	return "wecom"
}

func (c *Channel) Start(ctx context.Context) error {
	if c.corpID == "" || c.corpSecret == "" {
		return fmt.Errorf("WeCom corp_id and corp_secret required")
	}

	if err := c.refreshToken(ctx); err != nil {
		return fmt.Errorf("failed to get WeCom token: %w", err)
	}

	go c.tokenRefresher(ctx)

	return nil
}

func (c *Channel) Stop() error {
	return nil
}

func (c *Channel) refreshToken(ctx context.Context) error {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		c.corpID, c.corpSecret)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("WeCom API error: %s", result.ErrMsg)
	}

	c.tokenMu.Lock()
	c.token = result.AccessToken
	c.tokenMu.Unlock()

	return nil
}

func (c *Channel) tokenRefresher(ctx context.Context) {
	ticker := time.NewTicker(110 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.refreshToken(context.Background())
		case <-ctx.Done():
			return
		}
	}
}

func (c *Channel) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		c.handleVerify(w, r)
		return
	}

	msgSignature := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")

	var payload struct {
		XMLName struct{} `xml:"xml"`
		Encrypt string   `xml:"Encrypt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	content, err := c.decryptMessage(payload.Encrypt, msgSignature, timestamp, nonce)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var msg struct {
		MsgType      string `xml:"MsgType"`
		Content      string `xml:"Content"`
		FromUserName string `xml:"FromUserName"`
		AgentID      string `xml:"AgentID"`
	}

	if err := xml.Unmarshal([]byte(content), &msg); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	if msg.AgentID != c.agentID {
		w.WriteHeader(http.StatusOK)
		return
	}

	sessionID := fmt.Sprintf("wecom_%s", msg.FromUserName)

	go c.processMessage(sessionID, msg.Content, msg.FromUserName)

	w.WriteHeader(http.StatusOK)
}

func (c *Channel) handleVerify(w http.ResponseWriter, r *http.Request) {
	signature := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")
	echostr := r.URL.Query().Get("echostr")

	decrypted, err := c.decryptEcho(echostr, signature, timestamp, nonce)
	if err != nil {
		fmt.Fprintf(w, "verify failed")
		return
	}

	fmt.Fprintf(w, "%s", decrypted)
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

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)

	payload := map[string]any{
		"touser":  userID,
		"msgtype": "text",
		"agentid": c.agentID,
		"text": map[string]any{
			"content": content,
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	http.DefaultClient.Do(req)
}

func (c *Channel) decryptMessage(encryptStr, signature, timestamp, nonce string) (string, error) {
	if c.encodingAESKey == "" {
		return encryptStr, nil
	}

	sortStr := []string{c.token, timestamp, nonce, encryptStr}
	sort.Strings(sortStr)
	sha := sha256.Sum256([]byte(strings.Join(sortStr, "")))
	expectedSig := base64.StdEncoding.EncodeToString(sha[:])

	if expectedSig != signature {
		return "", fmt.Errorf("signature mismatch")
	}

	key, _ := base64.StdEncoding.DecodeString(c.encodingAESKey + "=")
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	cipherText, _ := base64.StdEncoding.DecodeString(encryptStr)
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	cipherText = cipherText[:len(cipherText)-32]
	result := string(cipherText)[20:]
	return result, nil
}

func (c *Channel) decryptEcho(echostr, signature, timestamp, nonce string) (string, error) {
	if c.encodingAESKey == "" {
		return echostr, nil
	}

	sortStr := []string{c.token, timestamp, nonce, echostr}
	sort.Strings(sortStr)
	sha := sha256.Sum256([]byte(strings.Join(sortStr, "")))
	expectedSig := base64.StdEncoding.EncodeToString(sha[:])

	if expectedSig != signature {
		return "", fmt.Errorf("signature mismatch")
	}

	key, _ := base64.StdEncoding.DecodeString(c.encodingAESKey + "=")
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	cipherText, _ := base64.StdEncoding.DecodeString(echostr)
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	cipherText = cipherText[:len(cipherText)-32]
	result := string(cipherText)[20:]
	return result, nil
}
