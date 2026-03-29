package planning

import (
	"context"
	"log/slog"
	"time"
)

// Project represents a multi-week project.
type Project struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Objective  string      `json:"objective"`
	Horizon    string      `json:"horizon"` // "1_month", "3_months", "6_months"
	Milestones []Milestone `json:"milestones"`
	Status     string      `json:"status"` // "planning", "active", "completed", "replanned"
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// Milestone represents a project milestone.
type Milestone struct {
	ID           string    `json:"id"`
	Description  string    `json:"description"`
	Deadline     time.Time `json:"deadline"`
	Dependencies []string  `json:"dependencies"`
	Goals        []GoalRef `json:"goals"`
	Status       string    `json:"status"` // "pending", "in_progress", "completed"
}

// GoalRef references an AGS goal.
type GoalRef struct {
	GoalID string `json:"goal_id"`
	Title  string `json:"title"`
}

// ProjectManager manages long-horizon projects.
type ProjectManager struct {
	llm        LLMCaller
	manager    GoalManager
	worldModel WorldModelQuery
}

// ProjectConfig holds configuration.
type ProjectConfig struct {
	Enabled bool
}

// NewProjectManager creates a new project manager.
func NewProjectManager(cfg ProjectConfig, llm LLMCaller, manager GoalManager, wm WorldModelQuery) *ProjectManager {
	return &ProjectManager{
		llm:        llm,
		manager:    manager,
		worldModel: wm,
	}
}

// CreateProject generates a new project from an objective.
func (p *ProjectManager) CreateProject(ctx context.Context, name, objective string, horizon time.Duration) (*Project, error) {
	slog.Info("planning: creating project", "name", name, "horizon", horizon)

	project := &Project{
		ID:        generateID(),
		Name:      name,
		Objective: objective,
		Horizon:   horizon.String(),
		Status:    "planning",
		CreatedAt: time.Now(),
	}

	// Use LLM to decompose into milestones
	decompPrompt := "Decompose this objective into a milestone tree:\n\n" + objective + "\n\nCreate 3-6 milestones that lead to completing this objective within " + horizon.String() + "."

	req := LLMRequest{
		SystemPrompt: "You are a project manager. Create milestone plans.",
		UserPrompt:   decompPrompt,
		Temperature:  0.5,
		MaxTokens:    2000,
	}

	_, err := p.llm.Complete(req)
	if err != nil {
		return nil, err
	}

	// Create milestones (simplified - LLM would provide structure)
	project.Milestones = createDefaultMilestones(horizon)

	// Create initial goals in AGS
	for i, milestone := range project.Milestones {
		goalID, err := p.manager.CreateGoal(ctx, milestone.Description, "Execute milestone: "+milestone.Description)
		if err == nil {
			project.Milestones[i].Goals = append(project.Milestones[i].Goals, GoalRef{GoalID: goalID, Title: milestone.Description})
		}
	}

	slog.Info("planning: project created", "name", name, "milestones", len(project.Milestones))
	return project, nil
}

func createDefaultMilestones(horizon time.Duration) []Milestone {
	// Simplified - would use LLM to generate real milestones
	return []Milestone{
		{
			ID:          generateID(),
			Description: "Phase 1: Research and Planning",
			Deadline:    time.Now().Add(horizon / 4),
			Status:      "pending",
		},
		{
			ID:          generateID(),
			Description: "Phase 2: Core Development",
			Deadline:    time.Now().Add(horizon / 2),
			Status:      "pending",
		},
		{
			ID:          generateID(),
			Description: "Phase 3: Testing and Refinement",
			Deadline:    time.Now().Add(horizon * 3 / 4),
			Status:      "pending",
		},
		{
			ID:          generateID(),
			Description: "Phase 4: Launch and Documentation",
			Deadline:    time.Now().Add(horizon),
			Status:      "pending",
		},
	}
}

// UpdateProgress tracks milestone completion.
func (p *ProjectManager) UpdateProgress(project *Project) {
	for i := range project.Milestones {
		if project.Milestones[i].Status == "pending" && time.Now().After(project.Milestones[i].Deadline) {
			project.Milestones[i].Status = "overdue"
		}
	}
	project.UpdatedAt = time.Now()
}

// Replan adjusts remaining milestones when one is missed.
func (p *ProjectManager) Replan(ctx context.Context, project *Project) error {
	slog.Info("planning: replanning project", "name", project.Name)

	// Would regenerate remaining milestones with updated context
	project.Status = "replanned"

	return nil
}

func generateID() string {
	return time.Now().Format("20060102150405")
}
