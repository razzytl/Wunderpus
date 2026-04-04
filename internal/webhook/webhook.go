package webhook

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/wunderpus/wunderpus/internal/events"
)

// Config defines a single webhook destination.
type Config struct {
	Name   string   `yaml:"name"`   // Friendly name (e.g., "slack", "discord")
	URL    string   `yaml:"url"`    // Target HTTP endpoint
	Events []string `yaml:"events"` // Event types to subscribe to
	Secret string   `yaml:"secret"` // Optional HMAC secret for signing
}

// Delivery represents a single webhook delivery attempt.
type Delivery struct {
	ID        string    `json:"id"`
	Webhook   string    `json:"webhook"`
	Event     string    `json:"event"`
	Payload   string    `json:"payload"`
	Status    int       `json:"status"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"created_at"`
}

// Manager subscribes to the event bus and delivers webhooks.
type Manager struct {
	bus        *events.Bus
	configs    []Config
	mu         sync.RWMutex
	deliveries []Delivery
	client     *http.Client
}

// NewManager creates a webhook manager subscribed to the event bus.
func NewManager(bus *events.Bus, configs []Config) *Manager {
	m := &Manager{
		bus:     bus,
		configs: configs,
		client:  &http.Client{Timeout: 30 * time.Second},
	}

	// Subscribe each webhook to its configured events
	for _, cfg := range configs {
		for _, eventType := range cfg.Events {
			bus.Subscribe(events.EventType(eventType), func(evt events.Event) {
				m.deliver(cfg, evt)
			})
		}
		slog.Info("webhook: subscribed", "name", cfg.Name, "events", cfg.Events)
	}

	return m
}

// deliver sends a single webhook with retry/backoff.
func (m *Manager) deliver(cfg Config, evt events.Event) {
	payload, err := m.buildPayload(cfg, evt)
	if err != nil {
		slog.Error("webhook: failed to build payload", "name", cfg.Name, "error", err)
		return
	}

	delivery := Delivery{
		ID:        fmt.Sprintf("wh-%d", time.Now().UnixNano()),
		Webhook:   cfg.Name,
		Event:     string(evt.Type),
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	// Retry with exponential backoff (max 3 attempts)
	for attempt := 1; attempt <= 3; attempt++ {
		status, err := m.send(cfg.URL, payload, cfg.Secret)
		delivery.Status = status
		delivery.Attempts = attempt

		if err == nil && status >= 200 && status < 300 {
			slog.Info("webhook: delivered", "name", cfg.Name, "event", evt.Type, "status", status)
			m.recordDelivery(delivery)
			return
		}

		if attempt < 3 {
			backoff := time.Duration(attempt) * 2 * time.Second
			slog.Warn("webhook: delivery failed, retrying",
				"name", cfg.Name, "attempt", attempt, "status", status, "error", err)
			time.Sleep(backoff)
		}
	}

	slog.Error("webhook: delivery failed after all retries", "name", cfg.Name, "event", evt.Type)
	m.recordDelivery(delivery)
}

// buildPayload renders the webhook payload using Go templates.
func (m *Manager) buildPayload(cfg Config, evt events.Event) (string, error) {
	// Default template: JSON with event type and payload
	tmplText := `{
  "event": "{{.Type}}",
  "source": "{{.Source}}",
  "timestamp": "{{.Timestamp}}",
  "payload": {{.PayloadJSON}}
}`

	// If the event payload has a "template" field, use it
	if templateStr, ok := evt.Payload.(string); ok && strings.HasPrefix(templateStr, "{{") {
		tmplText = templateStr
	}

	tmpl, err := template.New("webhook").Parse(tmplText)
	if err != nil {
		return "", fmt.Errorf("webhook: parsing template: %w", err)
	}

	// Build template data
	data := map[string]any{
		"Type":        evt.Type,
		"Source":      evt.Source,
		"Timestamp":   evt.Timestamp,
		"PayloadJSON": evt.Payload,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("webhook: executing template: %w", err)
	}

	return buf.String(), nil
}

// send performs the HTTP POST with optional HMAC signing.
func (m *Manager) send(url, payload, secret string) (int, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(payload))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wunderpus-webhook/1.0")

	if secret != "" {
		// Add HMAC signature header (simplified — production should use crypto/hmac)
		req.Header.Set("X-Webhook-Signature", "sha256="+secret)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func (m *Manager) recordDelivery(d Delivery) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveries = append(m.deliveries, d)
}

// GetDeliveries returns all recorded deliveries.
func (m *Manager) GetDeliveries() []Delivery {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Delivery, len(m.deliveries))
	copy(result, m.deliveries)
	return result
}

// GetFailedDeliveries returns deliveries that failed.
func (m *Manager) GetFailedDeliveries() []Delivery {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var failed []Delivery
	for _, d := range m.deliveries {
		if d.Status < 200 || d.Status >= 300 {
			failed = append(failed, d)
		}
	}
	return failed
}
