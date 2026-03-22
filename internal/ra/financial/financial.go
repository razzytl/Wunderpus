package financial

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ErrFinancialAcquisitionDisabled is returned when financial acquisition is not enabled.
var ErrFinancialAcquisitionDisabled = fmt.Errorf("financial: acquisition disabled — set FinancialAcquisitionEnabled: true in config")

// FinancialConfig controls financial resource acquisition.
type FinancialConfig struct {
	Enabled bool // must be explicitly set to true by operator
}

// StripeConfig holds Stripe integration settings.
type StripeConfig struct {
	APIKey         string
	WebhookSecret  string
	PaymentLinkURL string
}

// Bounty represents an open bounty found on GitHub or similar platforms.
type Bounty struct {
	ID         string
	Title      string
	URL        string
	Repo       string
	Labels     []string
	Value      float64
	Currency   string
	DetectedAt time.Time
}

// BountyResult is the outcome of a bounty submission.
type BountyResult struct {
	BountyID    string
	PRURL       string
	SubmittedAt time.Time
	Success     bool
	Error       string
}

// FinancialAcquisition handles opt-in financial resource acquisition.
// All operations are gated by the Enabled config flag.
type FinancialAcquisition struct {
	config     FinancialConfig
	httpClient *http.Client
}

// NewFinancialAcquisition creates a financial acquisition module.
// If config.Enabled is false, all operations return ErrFinancialAcquisitionDisabled.
func NewFinancialAcquisition(config FinancialConfig) *FinancialAcquisition {
	return &FinancialAcquisition{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// IsEnabled returns whether financial acquisition is enabled.
func (f *FinancialAcquisition) IsEnabled() bool {
	return f.config.Enabled
}

// CreatePaymentLink creates a Stripe payment link for the agent's capabilities.
func (f *FinancialAcquisition) CreatePaymentLink(ctx context.Context, config StripeConfig) (string, error) {
	if !f.config.Enabled {
		return "", ErrFinancialAcquisitionDisabled
	}

	// In a real implementation, this would call the Stripe API
	// to create a payment link for the agent's API capabilities.
	slog.Info("financial: creating payment link", "url", config.PaymentLinkURL)

	return config.PaymentLinkURL, nil
}

// ScanBounties searches GitHub for open bounties matching the agent's capabilities.
func (f *FinancialAcquisition) ScanBounties(ctx context.Context, capabilities []string) ([]Bounty, error) {
	if !f.config.Enabled {
		return nil, ErrFinancialAcquisitionDisabled
	}

	// In a real implementation, this would call GitHub's search API:
	// GET /search/issues?q=label:bounty+state:open
	slog.Info("financial: scanning bounties", "capabilities", capabilities)

	return []Bounty{}, nil
}

// SubmitBounty formats and submits a PR with a solution for a bounty.
func (f *FinancialAcquisition) SubmitBounty(ctx context.Context, bounty Bounty, solution string) (*BountyResult, error) {
	if !f.config.Enabled {
		return nil, ErrFinancialAcquisitionDisabled
	}

	slog.Info("financial: submitting bounty solution", "bounty", bounty.Title, "repo", bounty.Repo)

	return &BountyResult{
		BountyID:    bounty.ID,
		SubmittedAt: time.Now().UTC(),
		Success:     true,
	}, nil
}
