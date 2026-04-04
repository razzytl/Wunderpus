package planning

import (
	"context"
	"log/slog"
	"time"
)

// SelfImprovementRoadmap represents Wunderpus's self-improvement plan.
type SelfImprovementRoadmap struct {
	ID        string            `json:"id"`
	StartDate time.Time         `json:"start_date"`
	EndDate   time.Time         `json:"end_date"`
	Goals     []ImprovementGoal `json:"goals"`
	Status    string            `json:"status"` // "generating", "active", "completed"
}

// ImprovementGoal represents a self-improvement goal.
type ImprovementGoal struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "tool_synthesis", "learning"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    int       `json:"priority"` // 1-10
	Status      string    `json:"status"`   // "pending", "in_progress", "completed"
	Deadline    time.Time `json:"deadline"`
}

// SelfMapEngine generates self-improvement roadmaps.
type SelfMapEngine struct {
	toolGapDetector ToolGapDetector
	agsFailures     FailureAnalyzer
	revenueData     RevenueAnalyzer
	llm             LLMCaller
}

// ToolGapDetector finds capability gaps.
type ToolGapDetector interface {
	DetectGaps(ctx context.Context) ([]Gap, error)
}

// Gap represents a capability gap.
type Gap struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Priority    float64 `json:"priority"`
}

// FailureAnalyzer analyzes AGS failures.
type FailureAnalyzer interface {
	GetRecentFailures(ctx context.Context) ([]string, error)
}

// RevenueAnalyzer gets revenue data.
type RevenueAnalyzer interface {
	GetRevenueData(ctx context.Context) (RevenueData, error)
}

// RevenueData represents revenue metrics.
type RevenueData struct {
	TotalRevenue   float64  `json:"total_revenue"`
	RevenueSources []string `json:"sources"`
}

// SelfMapConfig holds configuration.
type SelfMapConfig struct {
	Enabled     bool
	RoadmapDays int // typically 90
}

// NewSelfMapEngine creates a new self-map engine.
func NewSelfMapEngine(cfg SelfMapConfig, gapDetector ToolGapDetector, failures FailureAnalyzer, revenue RevenueAnalyzer, llm LLMCaller) *SelfMapEngine {
	return &SelfMapEngine{
		toolGapDetector: gapDetector,
		agsFailures:     failures,
		revenueData:     revenue,
		llm:             llm,
	}
}

// GenerateRoadmap creates a 90-day self-improvement roadmap.
func (e *SelfMapEngine) GenerateRoadmap(ctx context.Context) (*SelfImprovementRoadmap, error) {
	slog.Info("planning: generating self-improvement roadmap")

	roadmap := &SelfImprovementRoadmap{
		ID:        generateID(),
		StartDate: time.Now(),
		EndDate:   time.Now().Add(90 * 24 * time.Hour),
		Status:    "generating",
	}

	// Gather inputs
	var inputs []string

	// 1. Tool gaps
	if e.toolGapDetector != nil {
		gaps, _ := e.toolGapDetector.DetectGaps(ctx)
		for _, gap := range gaps {
			inputs = append(inputs, "Gap: "+gap.Description)
		}
	}

	// 2. Recent failures
	if e.agsFailures != nil {
		failures, _ := e.agsFailures.GetRecentFailures(ctx)
		for _, f := range failures {
			inputs = append(inputs, "Failure: "+f)
		}
	}

	// 3. Revenue signals
	if e.revenueData != nil {
		revenue, _ := e.revenueData.GetRevenueData(ctx)
		inputs = append(inputs, "Revenue sources: "+joinStrings(revenue.RevenueSources))
	}

	// Generate roadmap using LLM
	prompt := "Generate a 90-day self-improvement roadmap for an AI agent. Consider:\n"
	prompt = prompt + joinStrings(inputs) + "\n\nPrioritize skills that would make the agent most valuable and technically improve its capabilities."

	req := LLMRequest{
		SystemPrompt: "You generate strategic self-improvement roadmaps.",
		UserPrompt:   prompt,
		Temperature:  0.6,
		MaxTokens:    3000,
	}

	_, err := e.llm.Complete(req)
	if err != nil {
		return nil, err
	}

	// Create improvement goals (simplified)
	roadmap.Goals = []ImprovementGoal{
		{
			ID:       generateID(),
			Type:     "tool_synthesis",
			Title:    "Fill top capability gaps",
			Priority: 10,
			Status:   "pending",
			Deadline: time.Now().Add(30 * 24 * time.Hour),
		},
		{
			ID:       generateID(),
			Type:     "learning",
			Title:    "Improve code generation quality",
			Priority: 8,
			Status:   "pending",
			Deadline: time.Now().Add(60 * 24 * time.Hour),
		},
		{
			ID:       generateID(),
			Type:     "learning",
			Title:    "Learn new integration patterns",
			Priority: 6,
			Status:   "pending",
			Deadline: time.Now().Add(90 * 24 * time.Hour),
		},
	}

	roadmap.Status = "active"
	slog.Info("planning: roadmap generated", "goals", len(roadmap.Goals))

	return roadmap, nil
}

// ReviewRoadmap reviews and updates the roadmap monthly.
func (e *SelfMapEngine) ReviewRoadmap(roadmap *SelfImprovementRoadmap) {
	slog.Info("planning: reviewing roadmap")

	for i := range roadmap.Goals {
		if roadmap.Goals[i].Status == "pending" && time.Now().After(roadmap.Goals[i].Deadline) {
			roadmap.Goals[i].Status = "overdue"
		}
	}

	// Would regenerate if needed in production
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result = result + ", "
		}
		result = result + s
	}
	return result
}
