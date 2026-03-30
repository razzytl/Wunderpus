package engineering

import (
	"context"
	"log/slog"
	"time"
)

// GitHubIssue represents a bug report from GitHub.
type GitHubIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Labels    []string  `json:"labels"`
	Repo      string    `json:"repo"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

// BugFix represents a proposed fix for a bug.
type BugFix struct {
	Issue     GitHubIssue
	PRURL     string
	Status    string // "proposed", "merged", "rejected"
	CreatedAt time.Time
}

// BugHunter monitors and fixes bugs in repositories.
type BugHunter struct {
	github GitHubClient
	coder  CoderLLM
	bounty BountyScanner
}

// GitHubClient interface for GitHub API operations.
type GitHubClient interface {
	GetIssues(ctx context.Context, repo string, labels []string) ([]GitHubIssue, error)
	CreatePR(ctx context.Context, repo string, branch, title, body string) (string, error)
}

// BountyScanner interface for bug bounty programs.
type BountyScanner interface {
	ScanPrograms(ctx context.Context, capabilities []string) ([]BountyProgram, error)
}

// BountyProgram represents a bug bounty program.
type BountyProgram struct {
	Name        string
	URL         string
	Scope       []string
	RewardRange string
}

// BugHunterConfig holds configuration for the bug hunter.
type BugHunterConfig struct {
	Enabled      bool
	WatchedRepos []string
	Labels       []string // "bug", "good first issue", etc.
}

// NewBugHunter creates a new bug hunter.
func NewBugHunter(cfg BugHunterConfig, github GitHubClient, coder CoderLLM, bounty BountyScanner) *BugHunter {
	return &BugHunter{
		github: github,
		coder:  coder,
		bounty: bounty,
	}
}

// ScanAndFix scans repositories for bugs and proposes fixes.
func (b *BugHunter) ScanAndFix(ctx context.Context, repo string) ([]BugFix, error) {
	slog.Info("bughunter: scanning repo", "repo", repo)

	// Get bug issues
	issues, err := b.github.GetIssues(ctx, repo, []string{"bug"})
	if err != nil {
		return nil, err
	}

	fixes := make([]BugFix, 0, len(issues))
	for _, issue := range issues {
		fix, err := b.fixBug(ctx, issue)
		if err != nil {
			slog.Warn("bughunter: failed to fix", "issue", issue.Number, "error", err)
			continue
		}
		fixes = append(fixes, fix)
	}

	slog.Info("bughunter: found fixes", "repo", repo, "count", len(fixes))
	return fixes, nil
}

func (b *BugHunter) fixBug(ctx context.Context, issue GitHubIssue) (BugFix, error) {
	slog.Info("bughunter: fixing bug", "issue", issue.Title)

	// Generate fix using LLM
	fixPrompt := "Analyze this bug report and generate a fix:\n\nTitle: " + issue.Title + "\n\nDescription: " + issue.Body + "\n\nProvide the complete code fix with explanation."

	req := CodeRequest{
		SystemPrompt: "You are an expert developer. Analyze bug reports and generate correct fixes.",
		UserPrompt:   fixPrompt,
		Temperature:  0.2,
		MaxTokens:    2000,
	}

	_, err := b.coder.Complete(req)
	if err != nil {
		return BugFix{}, err
	}

	return BugFix{
		Issue:     issue,
		PRURL:     "", // Would create PR in production
		Status:    "proposed",
		CreatedAt: time.Now(),
	}, nil
}

// ScanBounties scans bug bounty programs for opportunities.
func (b *BugHunter) ScanBounties(ctx context.Context, capabilities []string) ([]BountyProgram, error) {
	if b.bounty == nil {
		return []BountyProgram{}, nil
	}

	programs, err := b.bounty.ScanPrograms(ctx, capabilities)
	if err != nil {
		return nil, err
	}

	slog.Info("bughunter: scanned bounty programs", "count", len(programs))
	return programs, nil
}
