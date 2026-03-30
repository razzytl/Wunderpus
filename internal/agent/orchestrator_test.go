package agent

import (
	"testing"

	"github.com/wunderpus/wunderpus/internal/provider"
	"github.com/wunderpus/wunderpus/internal/tool"
)

// TestTaskGraph tests TaskGraph structure
func TestTaskGraph(t *testing.T) {
	graph := &TaskGraph{
		Goal: "test goal",
	}

	if graph.Goal != "test goal" {
		t.Errorf("expected Goal 'test goal', got %q", graph.Goal)
	}
}

// TestSubtask tests Subtask structure
func TestSubtask(t *testing.T) {
	subtask := Subtask{
		ID:           "subtask-1",
		Description:  "do something",
		Type:         "worker",
		Dependencies: []string{"dep-1", "dep-2"},
	}

	if subtask.ID != "subtask-1" {
		t.Errorf("expected ID 'subtask-1', got %q", subtask.ID)
	}
	if subtask.Description != "do something" {
		t.Errorf("expected Description 'do something', got %q", subtask.Description)
	}
	if subtask.Type != "worker" {
		t.Errorf("expected Type 'worker', got %q", subtask.Type)
	}
	if len(subtask.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(subtask.Dependencies))
	}
}

// TestOrchestrator_NewOrchestrator tests creating a new orchestrator
func TestOrchestrator_NewOrchestrator(t *testing.T) {
	router := &provider.Router{}
	registry := tool.NewRegistry()

	orch := NewOrchestrator(router, registry, nil, "gpt-4")

	if orch == nil {
		t.Fatal("expected non-nil orchestrator")
	}
	if orch.model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %q", orch.model)
	}
}

// TestWorkerArm tests WorkerArm creation
func TestWorkerArm(t *testing.T) {
	router := &provider.Router{}
	registry := tool.NewRegistry()

	arm := NewWorkerArm("task-1", "worker", router, registry, nil)

	if arm == nil {
		t.Fatal("expected non-nil worker arm")
	}
}

// TestMessage tests provider.Message structure
func TestMessage(t *testing.T) {
	msg := provider.Message{
		Role:    "user",
		Content: "Hello, world!",
	}

	if msg.Role != "user" {
		t.Errorf("expected Role 'user', got %q", msg.Role)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("expected Content 'Hello, world!', got %q", msg.Content)
	}
}

// TestMessage_Roles tests different message roles
func TestMessage_Roles(t *testing.T) {
	roles := []string{"system", "user", "assistant", "tool"}

	for _, role := range roles {
		msg := provider.Message{
			Role: role,
		}
		if msg.Role != role {
			t.Errorf("expected Role %q, got %q", role, msg.Role)
		}
	}
}
