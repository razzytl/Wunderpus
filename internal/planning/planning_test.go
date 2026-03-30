package planning

import (
	"testing"
	"time"
)

func TestProjectManager_CreateProject(t *testing.T) {
	pm := &ProjectManager{}

	_ = pm // pm is now initialized
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

	if project.Name != "Test Project" {
		t.Errorf("Expected name 'Test Project', got %s", project.Name)
	}
	if project.Horizon != "3 months" {
		t.Errorf("Expected horizon '3 months', got %s", project.Horizon)
	}
	if len(project.Milestones) != 2 {
		t.Errorf("Expected 2 milestones, got %d", len(project.Milestones))
	}
}

func TestSelfMapEngine_GenerateRoadmap(t *testing.T) {
	engine := &SelfMapEngine{}

	_ = engine // engine is now initialized
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

	if goal.ID != "goal-1" {
		t.Errorf("Expected ID 'goal-1', got %s", goal.ID)
	}
	if goal.Type != "tool_synthesis" {
		t.Errorf("Expected type 'tool_synthesis', got %s", goal.Type)
	}
	if goal.Title != "Learn new tools" {
		t.Errorf("Expected title 'Learn new tools', got %s", goal.Title)
	}
	if goal.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", goal.Status)
	}
	if goal.Deadline.IsZero() {
		t.Error("Expected deadline to be set")
	}
	if goal.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", goal.Priority)
	}
}
