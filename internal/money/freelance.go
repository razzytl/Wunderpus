package money

import (
	"context"
	"log/slog"
	"time"
)

// Job represents a freelance job opportunity.
type Job struct {
	ID           string    `json:"id"`
	Platform     string    `json:"platform"` // "upwork", "fiverr", "freelancer", "github"
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Skills       []string  `json:"skills"`
	Budget       float64   `json:"budget"`
	Currency     string    `json:"currency"`
	PostedAt     time.Time `json:"posted_at"`
	URL          string    `json:"url"`
	ClientRating float64   `json:"client_rating"` // 0-5
}

// Bid represents a submitted proposal for a job.
type Bid struct {
	JobID       string    `json:"job_id"`
	Amount      float64   `json:"amount"`
	CoverLetter string    `json:"cover_letter"`
	SubmittedAt time.Time `json:"submitted_at"`
	Status      string    `json:"status"` // "pending", "accepted", "rejected"
}

// JobScanner scans job boards for matching opportunities.
type JobScanner interface {
	Scan(ctx context.Context, skills []string) ([]Job, error)
}

// BidEvaluator scores job opportunities based on capability match.
type BidEvaluator struct{}

// Score calculates a match score for a job.
func (e *BidEvaluator) Score(job Job, capabilities []string) float64 {
	// Count matching skills
	matched := 0
	jobSkills := make(map[string]bool)
	for _, s := range job.Skills {
		jobSkills[s] = true
	}
	for _, c := range capabilities {
		if jobSkills[c] {
			matched++
		}
	}
	skillScore := float64(matched) / float64(len(job.Skills))
	if len(job.Skills) == 0 {
		skillScore = 0
	}

	// Budget score (normalized to typical range)
	budgetScore := job.Budget / 1000.0 // Assuming $1000 is reference
	if budgetScore > 1 {
		budgetScore = 1
	}

	// Client rating score
	ratingScore := job.ClientRating / 5.0

	// Weighted combination
	return skillScore*0.5 + budgetScore*0.3 + ratingScore*0.2
}

// FreelanceEngine manages freelance job discovery and bidding.
type FreelanceEngine struct {
	scanner      JobScanner
	evaluator    *BidEvaluator
	capabilities []string
	minScore     float64
}

// FreelanceConfig holds configuration for the freelance engine.
type FreelanceConfig struct {
	Enabled       bool
	MinMatchScore float64
	Capabilities  []string
}

// NewFreelanceEngine creates a new freelance engine.
func NewFreelanceEngine(scanner JobScanner, cfg FreelanceConfig) *FreelanceEngine {
	return &FreelanceEngine{
		scanner:      scanner,
		evaluator:    &BidEvaluator{},
		capabilities: cfg.Capabilities,
		minScore:     cfg.MinMatchScore,
	}
}

// ScanAndScore scans for jobs and returns scored opportunities.
func (f *FreelanceEngine) ScanAndScore(ctx context.Context) ([]Job, error) {
	if f.scanner == nil {
		return nil, nil // No scanner configured
	}

	jobs, err := f.scanner.Scan(ctx, f.capabilities)
	if err != nil {
		slog.Warn("freelance: scan failed", "error", err)
		return nil, err
	}

	// Filter by minimum score
	var scoredJobs []Job
	for _, job := range jobs {
		score := f.evaluator.Score(job, f.capabilities)
		if score >= f.minScore {
			scoredJobs = append(scoredJobs, job)
		}
	}

	slog.Info("freelance: scanned jobs", "total", len(jobs), "matched", len(scoredJobs))
	return scoredJobs, nil
}

// PlatformScanner is a mock implementation that scans multiple platforms.
type PlatformScanner struct {
	llmCaller    LLMCaller
	browserAgent BrowserAgent
}

// LLMCaller interface for LLM calls.
type LLMCaller interface {
	Complete(req LLMRequest) (string, error)
}

// LLMRequest represents an LLM request.
type LLMRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
}

// BrowserAgent interface for browser automation.
type BrowserAgent interface {
	Execute(ctx context.Context, goal, url string) (string, error)
}

// NewPlatformScanner creates a new platform scanner.
func NewPlatformScanner(llm LLMCaller, browser BrowserAgent) *PlatformScanner {
	return &PlatformScanner{
		llmCaller:    llm,
		browserAgent: browser,
	}
}

// Scan implements JobScanner. It searches multiple job platforms.
func (s *PlatformScanner) Scan(ctx context.Context, skills []string) ([]Job, error) {
	// In production, this would scan real job boards
	// For now, return empty list - real implementation would use browser agent
	return []Job{}, nil
}

// MockScanner is a test-friendly scanner that returns predefined jobs.
type MockScanner struct {
	Jobs []Job
}

// NewMockScanner creates a mock scanner.
func NewMockScanner(jobs []Job) *MockScanner {
	return &MockScanner{Jobs: jobs}
}

// Scan returns the predefined jobs.
func (s *MockScanner) Scan(ctx context.Context, skills []string) ([]Job, error) {
	return s.Jobs, nil
}
