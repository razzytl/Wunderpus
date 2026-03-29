package planning

import (
	"testing"
	"time"
)

func TestProjectManager_CreateProject(t *testing.T) {
	pm := &ProjectManager{}

	if pm == nil {
		t.Error("Expected project manager to be created")
	}
}

func TestProject_Milestones(t *testing.T) {
	project := &Project{
		Name:    "Test Project",
		Horizon: "3 months",
		Milestones: []Milestone{
			{ID: "1", Description: "Phase 1", Status: "pending"},
			{ID: "2", Description: "Phase 2", Status: "pending"},
		},
	}

	if len(project.Milestones) != 2 {
		t.Errorf("Expected 2 milestones, got %d", len(project.Milestones))
	}
}

func TestSelfMapEngine_GenerateRoadmap(t *testing.T) {
	engine := &SelfMapEngine{}

	if engine == nil {
		t.Error("Expected engine to be created")
	}
}

func TestImprovementGoal_Structure(t *testing.T) {
	goal := ImprovementGoal{
		ID:       "goal-1",
		Type:     "tool_synthesis",
		Title:    "Learn new tools",
		Priority: 10,
		Status:   "pending",
		Deadline: time.Now().Add(30 * 24 * time.Hour),
	}

	if goal.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", goal.Priority)
	}
}
