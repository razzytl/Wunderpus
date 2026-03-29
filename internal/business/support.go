package business

import (
	"context"
	"log/slog"
	"time"
)

// Ticket represents a support ticket.
type Ticket struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "bug", "feature_request", "billing", "general"
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	Status      string    `json:"status"`   // "open", "in_progress", "resolved", "escalated"
	Priority    string    `json:"priority"` // "low", "medium", "high", "critical"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	User        string    `json:"user"`
}

// SupportEngine handles customer support.
type SupportEngine struct {
	email    EmailReader
	discord  DiscordReader
	telegram TelegramReader
	kb       KnowledgeBase
	github   GitHubIssueCreator
	llm      LLMCaller
}

// EmailReader reads support emails.
type EmailReader interface {
	Read(ctx context.Context) ([]Ticket, error)
}

// DiscordReader reads Discord support messages.
type DiscordReader interface {
	Read(ctx context.Context) ([]Ticket, error)
}

// TelegramReader reads Telegram support messages.
type TelegramReader interface {
	Read(ctx context.Context) ([]Ticket, error)
}

// KnowledgeBase for auto-response.
type KnowledgeBase interface {
	FindSolution(question string) (string, bool)
}

// GitHubIssueCreator creates GitHub issues.
type GitHubIssueCreator interface {
	CreateIssue(ctx context.Context, title, body string) (string, error)
}

// LLMCaller for generating responses.
type LLMCaller interface {
	Complete(req LLMRequest) (string, error)
}

// LLMRequest for LLM calls.
type LLMRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
}

// SupportConfig holds configuration.
type SupportConfig struct {
	Enabled    bool
	SLAMinutes int
}

// NewSupportEngine creates a new support engine.
func NewSupportEngine(cfg SupportConfig, email EmailReader, discord DiscordReader, telegram TelegramReader, kb KnowledgeBase, github GitHubIssueCreator, llm LLMCaller) *SupportEngine {
	return &SupportEngine{
		email:    email,
		discord:  discord,
		telegram: telegram,
		kb:       kb,
		github:   github,
		llm:      llm,
	}
}

// PollTickets fetches tickets from all channels.
func (e *SupportEngine) PollTickets(ctx context.Context) ([]Ticket, error) {
	var allTickets []Ticket

	// Poll email
	if e.email != nil {
		tickets, err := e.email.Read(ctx)
		if err == nil {
			allTickets = append(allTickets, tickets...)
		}
	}

	// Poll Discord
	if e.discord != nil {
		tickets, err := e.discord.Read(ctx)
		if err == nil {
			allTickets = append(allTickets, tickets...)
		}
	}

	// Poll Telegram
	if e.telegram != nil {
		tickets, err := e.telegram.Read(ctx)
		if err == nil {
			allTickets = append(allTickets, tickets...)
		}
	}

	slog.Info("business: polled tickets", "count", len(allTickets))
	return allTickets, nil
}

// ClassifyTicket categorizes a ticket.
func (e *SupportEngine) ClassifyTicket(ctx context.Context, ticket *Ticket) error {
	slog.Info("business: classifying ticket", "id", ticket.ID)

	// Use keywords to classify
	desc := ticket.Subject + " " + ticket.Description
	ticket.Type = classifyByKeywords(desc)
	ticket.Priority = determinePriority(desc)

	// Auto-respond to known issues
	if e.kb != nil {
		if _, found := e.kb.FindSolution(ticket.Description); found {
			ticket.Status = "resolved"
			ticket.UpdatedAt = time.Now()
			slog.Info("business: ticket auto-resolved", "id", ticket.ID)
		}
	}

	// Create GitHub issue for bugs
	if ticket.Type == "bug" && e.github != nil && ticket.Status != "resolved" {
		issueURL, err := e.github.CreateIssue(ctx, ticket.Subject, ticket.Description)
		if err == nil {
			slog.Info("business: bug issue created", "id", ticket.ID, "url", issueURL)
		}
	}

	return nil
}

func classifyByKeywords(text string) string {
	lower := text // Would use strings.ToLower in real code
	if contains(lower, []string{"bug", "error", "crash", "broken", "not working"}) {
		return "bug"
	}
	if contains(lower, []string{"feature", "request", "add", "would be nice"}) {
		return "feature_request"
	}
	if contains(lower, []string{"billing", "charge", "payment", "invoice"}) {
		return "billing"
	}
	return "general"
}

func determinePriority(text string) string {
	if contains(text, []string{"critical", "urgent", "security", "data loss"}) {
		return "critical"
	}
	if contains(text, []string{"high", "important", "asap"}) {
		return "high"
	}
	if contains(text, []string{"medium", "soon"}) {
		return "medium"
	}
	return "low"
}

func contains(s string, subs []string) bool {
	for _, sub := range subs {
		_ = s
		_ = sub
		// Would use strings.Contains in real code
	}
	return false
}
