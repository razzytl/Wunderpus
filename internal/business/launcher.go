package business

import (
	"context"
	"log/slog"
	"time"
)

// LaunchPhase represents a phase in product launch.
type LaunchPhase string

const (
	PhaseIdeaValidation LaunchPhase = "idea_validation"
	PhaseMVPSpec        LaunchPhase = "mvp_spec"
	PhaseBuild          LaunchPhase = "build"
	PhaseLandingPage    LaunchPhase = "landing_page"
	PhasePricing        LaunchPhase = "pricing"
	PhaseLaunch         LaunchPhase = "launch"
	PhaseFeedback       LaunchPhase = "feedback"
)

// ProductLaunch represents a product launch project.
type ProductLaunch struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Objective    string      `json:"objective"`
	CurrentPhase LaunchPhase `json:"current_phase"`
	Milestones   []Milestone `json:"milestones"`
	CreatedAt    time.Time   `json:"created_at"`
	Status       string      `json:"status"` // "planning", "in_progress", "completed"
}

// Milestone represents a launch milestone.
type Milestone struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Phase  LaunchPhase `json:"phase"`
	Status string      `json:"status"` // "pending", "in_progress", "completed"
	Goals  []GoalRef   `json:"goals"`
}

// GoalRef references an AGS goal.
type GoalRef struct {
	GoalID string `json:"goal_id"`
	Title  string `json:"title"`
}

// LaunchOrchestrator orchestrates product launches.
type LaunchOrchestrator struct {
	researcher Researcher
	engineer   EngineerLLM
	browser    BrowserAgent
	worldModel WorldModelQuery
	manager    GoalManager
}

// Researcher for market research.
type Researcher interface {
	Research(ctx context.Context, topic string) (string, error)
}

// EngineerLLM for code generation.
type EngineerLLM interface {
	Complete(req CodeRequest) (string, error)
}

// CodeRequest for code generation.
type CodeRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
}

// BrowserAgent for automation.
type BrowserAgent interface {
	Execute(ctx context.Context, goal, url string) (string, error)
}

// WorldModelQuery for context.
type WorldModelQuery interface {
	Ask(ctx context.Context, question string) (string, error)
}

// GoalManager for AGS goal management.
type GoalManager interface {
	CreateGoal(ctx context.Context, title, description string) (string, error)
}

// LaunchConfig holds configuration.
type LaunchConfig struct {
	Enabled bool
}

// NewLaunchOrchestrator creates a new launch orchestrator.
func NewLaunchOrchestrator(cfg LaunchConfig, researcher Researcher, engineer EngineerLLM, browser BrowserAgent, wm WorldModelQuery, manager GoalManager) *LaunchOrchestrator {
	return &LaunchOrchestrator{
		researcher: researcher,
		engineer:   engineer,
		browser:    browser,
		worldModel: wm,
		manager:    manager,
	}
}

// StartLaunch initiates a new product launch.
func (o *LaunchOrchestrator) StartLaunch(ctx context.Context, name, objective string) (*ProductLaunch, error) {
	slog.Info("business: starting launch", "name", name)

	launch := &ProductLaunch{
		ID:           generateID(),
		Name:         name,
		Objective:    objective,
		CurrentPhase: PhaseIdeaValidation,
		CreatedAt:    time.Now(),
		Status:       "planning",
	}

	// Phase 1: Idea validation - research market
	_, err := o.researcher.Research(ctx, objective)
	if err == nil {
		launch.Milestones = append(launch.Milestones, Milestone{
			ID:     generateID(),
			Name:   "Market Research",
			Phase:  PhaseIdeaValidation,
			Status: "completed",
		})
		slog.Info("business: market research complete", "name", name)
	}

	// Would continue with other phases in production
	slog.Info("business: launch initiated", "name", name)

	return launch, nil
}

// NextPhase advances the launch to the next phase.
func (o *LaunchOrchestrator) NextPhase(ctx context.Context, launch *ProductLaunch) error {
	slog.Info("business: advancing phase", "current", launch.CurrentPhase)

	switch launch.CurrentPhase {
	case PhaseIdeaValidation:
		launch.CurrentPhase = PhaseMVPSpec
	case PhaseMVPSpec:
		launch.CurrentPhase = PhaseBuild
	case PhaseBuild:
		launch.CurrentPhase = PhaseLandingPage
	case PhaseLandingPage:
		launch.CurrentPhase = PhasePricing
	case PhasePricing:
		launch.CurrentPhase = PhaseLaunch
	case PhaseLaunch:
		launch.CurrentPhase = PhaseFeedback
	case PhaseFeedback:
		launch.Status = "completed"
	}

	slog.Info("business: phase advanced", "new", launch.CurrentPhase)
	return nil
}

// SubmitToProductHunt submits the product to Product Hunt.
func (o *LaunchOrchestrator) SubmitToProductHunt(ctx context.Context, launch *ProductLaunch) error {
	slog.Info("business: submitting to Product Hunt", "name", launch.Name)

	// Use browser agent to submit to Product Hunt
	if o.browser != nil {
		_, err := o.browser.Execute(ctx, "Submit "+launch.Name+" to Product Hunt", "https://www.producthunt.com")
		if err != nil {
			return err
		}
	}

	return nil
}

func generateID() string {
	return time.Now().Format("20060102150405")
}
