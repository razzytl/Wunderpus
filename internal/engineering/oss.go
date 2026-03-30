package engineering

import (
	"context"
	"log/slog"
	"time"
)

// OSSIssue represents an open source contribution opportunity.
type OSSIssue struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	Labels     []string  `json:"labels"`
	Repo       string    `json:"repo"`
	URL        string    `json:"url"`
	Difficulty string    `json:"difficulty"` // "good first issue", "help wanted", "medium", "hard"
	CreatedAt  time.Time `json:"created_at"`
}

// OSSContribution represents a contribution to an OSS project.
type OSSContribution struct {
	Issue     OSSIssue
	PRURL     string
	Status    string // "proposed", "merged", "rejected"
	CreatedAt time.Time
}

// OSSEngine finds and contributes to open source projects.
type OSSEngine struct {
	github     GitHubClient
	coder      CoderLLM
	worldModel WorldModelQuery
}

// OSSConfig holds configuration for OSS engine.
type OSSConfig struct {
	Enabled        bool
	LanguageFilter []string // "go", "python", "rust"
	ExcludeOrgs    []string
	MinStars       int
}

// NewOSSEngine creates a new OSS contribution engine.
func NewOSSEngine(cfg OSSConfig, github GitHubClient, coder CoderLLM, wm WorldModelQuery) *OSSEngine {
	return &OSSEngine{
		github:     github,
		coder:      coder,
		worldModel: wm,
	}
}

// ScanIssues scans for good first issues in Go projects.
func (e *OSSEngine) ScanIssues(ctx context.Context, language string) ([]OSSIssue, error) {
	slog.Info("oss: scanning for issues", "language", language)

	// Search for issues with "good first issue" or "help wanted" labels
	issues, err := e.github.GetIssues(ctx, "golang/go", []string{"good first issue", "help wanted"})
	if err != nil {
		return nil, err
	}

	result := make([]OSSIssue, 0, len(issues))
	for _, issue := range issues {
		result = append(result, OSSIssue{
			Number:     issue.Number,
			Title:      issue.Title,
			Body:       issue.Body,
			Labels:     issue.Labels,
			Repo:       "golang/go",
			URL:        "https://github.com/golang/go/issues/" + string(rune(issue.Number)),
			Difficulty: "good first issue",
			CreatedAt:  issue.CreatedAt,
		})
	}

	slog.Info("oss: found issues", "count", len(result))
	return result, nil
}

// ScoreIssues ranks issues by capability match and project impact.
func (e *OSSEngine) ScoreIssues(ctx context.Context, issues []OSSIssue, capabilities []string) ([]RankedIssue, error) {
	ranked := make([]RankedIssue, 0, len(issues))

	for _, issue := range issues {
		// Score based on skill match
		skillScore := e.calculateSkillMatch(issue, capabilities)

		// Would also consider project stars in production
		impactScore := 0.5 // Default

		totalScore := skillScore*0.7 + impactScore*0.3

		ranked = append(ranked, RankedIssue{
			Issue: issue,
			Score: totalScore,
			Rank:  0,
		})
	}

	// Sort by score descending
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].Score > ranked[i].Score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	// Set ranks
	for i := range ranked {
		ranked[i].Rank = i + 1
	}

	return ranked, nil
}

func (e *OSSEngine) calculateSkillMatch(issue OSSIssue, capabilities []string) float64 {
	// Simple keyword matching
	body := issue.Title + " " + issue.Body
	matched := 0
	for _, cap := range capabilities {
		for _, label := range issue.Labels {
			if len(cap) > 0 && (len(body) > 0 || len(label) > 0) {
				_ = cap // Placeholder for actual matching
				matched++
			}
		}
	}
	return float64(matched) / float64(len(capabilities)*10+1)
}

// RankedIssue represents a scored issue.
type RankedIssue struct {
	Issue OSSIssue
	Score float64
	Rank  int
}

// Contribute attempts to fix an issue and submit a PR.
func (e *OSSEngine) Contribute(ctx context.Context, issue OSSIssue) (OSSContribution, error) {
	slog.Info("oss: contributing to", "issue", issue.Title)

	// Generate implementation
	implPrompt := "Implement a fix for this GitHub issue:\n\nTitle: " + issue.Title + "\n\nDescription: " + issue.Body + "\n\nProvide complete, working code."

	req := CodeRequest{
		SystemPrompt: "You are an expert developer in Go. Write clean, idiomatic code.",
		UserPrompt:   implPrompt,
		Temperature:  0.3,
		MaxTokens:    2000,
	}

	_, err := e.coder.Complete(req)
	if err != nil {
		return OSSContribution{}, err
	}

	// Update world model with contribution
	if e.worldModel != nil {
		_ = e.worldModel // Would record contribution for social proof
	}

	return OSSContribution{
		Issue:     issue,
		PRURL:     "", // Would create PR in production
		Status:    "proposed",
		CreatedAt: time.Now(),
	}, nil
}
