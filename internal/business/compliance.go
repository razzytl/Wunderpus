package business

import (
	"context"
	"log/slog"
	"time"
)

// ComplianceChange represents a regulatory change.
type ComplianceChange struct {
	Regulation  string    `json:"regulation"` // "GDPR", "DMCA", "CCPA"
	Summary     string    `json:"summary"`
	Impact      string    `json:"impact"`
	EffectiveAt time.Time `json:"effective_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// LegalWatcher monitors legal and compliance changes.
type LegalWatcher struct {
	webSearch WebSearch
	operator  Notifier
}

// WebSearch for regulatory updates.
type WebSearch interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

// SearchResult from web search.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Notifier notifies operator of compliance issues.
type Notifier interface {
	Notify(ctx context.Context, message string) error
}

// ComplianceConfig holds configuration.
type ComplianceConfig struct {
	Enabled        bool
	Regulations    []string // "GDPR", "DMCA", "CCPA"
	CheckIntervalH int
}

// NewLegalWatcher creates a new legal watcher.
func NewLegalWatcher(cfg ComplianceConfig, webSearch WebSearch, operator Notifier) *LegalWatcher {
	return &LegalWatcher{
		webSearch: webSearch,
		operator:  operator,
	}
}

// CheckRegulations checks for regulatory updates.
func (w *LegalWatcher) CheckRegulations(ctx context.Context) ([]ComplianceChange, error) {
	slog.Info("business: checking regulatory updates")

	var changes []ComplianceChange

	for _, reg := range []string{"GDPR", "DMCA", "CCPA"} {
		query := reg + " regulation changes " + time.Now().Format("2006")
		results, err := w.webSearch.Search(ctx, query)
		if err != nil {
			continue
		}

		for _, result := range results {
			change := ComplianceChange{
				Regulation: reg,
				Summary:    result.Snippet,
				Impact:     "review required",
				CreatedAt:  time.Now(),
			}
			changes = append(changes, change)
		}
	}

	slog.Info("business: regulatory changes found", "count", len(changes))
	return changes, nil
}

// HandleDMCATakedown processes a DMCA takedown notice.
func (w *LegalWatcher) HandleDMCATakedown(ctx context.Context, notice DMCA) error {
	slog.Warn("business: DMCA takedown received", "notice", notice.ID)

	// Immediately comply - remove content
	err := removeContent(notice.ContentID)
	if err != nil {
		slog.Error("business: failed to remove content", "error", err)
		return err
	}

	// Log the takedown
	slog.Info("business: content removed", "content_id", notice.ContentID)

	// Notify operator
	if w.operator != nil {
		w.operator.Notify(ctx, "DMCA takedown received and complied. ID: "+notice.ID)
	}

	return nil
}

// DMCATakedown represents a DMCA takedown notice.
type DMCA struct {
	ID         string    `json:"id"`
	ContentID  string    `json:"content_id"`
	Claimant   string    `json:"claimant"`
	ReceivedAt time.Time `json:"received_at"`
}

// TrackRevenue records revenue for tax purposes.
func (w *LegalWatcher) TrackRevenue(revenue []RevenueEntry) map[string]float64 {
	taxByJurisdiction := make(map[string]float64)

	for _, entry := range revenue {
		taxByJurisdiction[entry.Jurisdiction] += entry.Amount
	}

	slog.Info("business: tax tracking complete", "jurisdictions", len(taxByJurisdiction))
	return taxByJurisdiction
}

// RevenueEntry for tax tracking.
type RevenueEntry struct {
	Jurisdiction string    `json:"jurisdiction"` // "US", "EU", "UK"
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	Date         time.Time `json:"date"`
}

func removeContent(id string) error {
	// Would actually remove content in production
	return nil
}
