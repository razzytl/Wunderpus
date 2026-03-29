package social

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Email represents an outreach email.
type Email struct {
	ID          string     `json:"id"`
	To          string     `json:"to"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
	Status      string     `json:"status"` // "draft", "sent", "opened", "replied"
	SentAt      *time.Time `json:"sent_at"`
	OpenedAt    *time.Time `json:"opened_at"`
	RepliedAt   *time.Time `json:"replied_at"`
	FollowUpNum int        `json:"follow_up_num"` // 0, 1, 2
}

// OutreachEngine handles email and cold outreach.
type OutreachEngine struct {
	smtp       SMTPClient
	researcher Researcher
	writer     LLMCaller
	worldModel WorldModelQuery
	tracker    EmailTracker
}

// SMTPClient for sending emails.
type SMTPClient interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Researcher for target identification.
type Researcher interface {
	Research(ctx context.Context, topic string) (string, error)
}

// EmailTracker tracks email engagement.
type EmailTracker struct {
	Emails []Email
}

// OutreachConfig holds configuration.
type OutreachConfig struct {
	Enabled      bool
	MaxFollowUps int
	SMTPHost     string
	SMTPPort     int
}

// NewOutreachEngine creates a new outreach engine.
func NewOutreachEngine(cfg OutreachConfig, smtp SMTPClient, researcher Researcher, writer LLMCaller, wm WorldModelQuery) *OutreachEngine {
	return &OutreachEngine{
		smtp:       smtp,
		researcher: researcher,
		writer:     writer,
		worldModel: wm,
		tracker:    EmailTracker{},
	}
}

// CreateOutreach creates personalized outreach email.
func (e *OutreachEngine) CreateOutreach(ctx context.Context, target string) (*Email, error) {
	slog.Info("social: creating outreach", "target", target)

	// Get context about target from world model
	var contextInfo string
	if e.worldModel != nil {
		result, err := e.worldModel.Ask(ctx, "information about "+target)
		if err == nil {
			contextInfo = result
		}
	}

	// Build prompt with context
	prompt := fmt.Sprintf("Write a personalized outreach email to: %s\n", target)
	if contextInfo != "" {
		prompt = prompt + fmt.Sprintf("Context I have about them: %s\n", contextInfo)
	}
	prompt = prompt + "Make it professional, personalized, and not spammy. Keep under 200 words."

	req := LLMRequest{
		SystemPrompt: "You write professional outreach emails.",
		UserPrompt:   prompt,
		Temperature:  0.6,
		MaxTokens:    500,
	}

	body, err := e.writer.Complete(req)
	if err != nil {
		return nil, err
	}

	email := &Email{
		ID:     generateID(),
		To:     target,
		Body:   body,
		Status: "draft",
	}

	slog.Info("social: outreach created", "target", target)
	return email, nil
}

// Send sends an email.
func (e *OutreachEngine) Send(ctx context.Context, email *Email) error {
	slog.Info("social: sending email", "to", email.To)

	if e.smtp == nil {
		return fmt.Errorf("no SMTP configured")
	}

	err := e.smtp.Send(ctx, email.To, email.Subject, email.Body)
	if err != nil {
		return err
	}

	email.Status = "sent"
	now := time.Now()
	email.SentAt = &now

	e.tracker.Emails = append(e.tracker.Emails, *email)

	return nil
}

// FollowUp sends a follow-up email for no reply.
func (e *OutreachEngine) FollowUp(ctx context.Context, email *Email) error {
	if email.FollowUpNum >= 2 {
		slog.Info("social: max follow-ups reached", "to", email.To)
		return nil
	}

	email.FollowUpNum++
	email.Body = email.Body + "\n\n[Follow-up]"

	return e.Send(ctx, email)
}

// TrackOpens checks for email opens (via tracking pixel or API).
func (e *OutreachEngine) TrackOpens(ctx context.Context) {
	for i := range e.tracker.Emails {
		email := &e.tracker.Emails[i]
		if email.Status == "sent" && email.OpenedAt == nil {
			// Would check tracking pixel or API
			// For now, just mark as potential
			_ = i
		}
	}
}
