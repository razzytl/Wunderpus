package engineering

import (
	"context"
	"testing"
	"time"
)

func TestProjectBuilder_Build_RESTAPI(t *testing.T) {
	builder := &ProjectBuilder{
		coder:   &mockCoder{},
		shell:   &mockShell{},
		sandbox: &mockRunner{},
	}

	spec := ProjectSpec{
		Name:     "test-api",
		Type:     "rest_api",
		Language: "go",
		Endpoints: []Endpoint{
			{Path: "/users", Method: "GET", Handler: "GetUsers"},
			{Path: "/users", Method: "POST", Handler: "CreateUser"},
		},
	}

	ctx := context.Background()
	result, err := builder.Build(ctx, spec)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if result.ProjectName != "test-api" {
		t.Errorf("Expected project name 'test-api', got '%s'", result.ProjectName)
	}

	if len(result.Steps) == 0 {
		t.Error("Expected build steps")
	}

	// Verify key steps were executed
	stepNames := make(map[string]bool)
	for _, step := range result.Steps {
		stepNames[step.Name] = true
	}

	if !stepNames["requirements"] {
		t.Error("Expected requirements step")
	}
	if !stepNames["architecture"] {
		t.Error("Expected architecture step")
	}
}

func TestProjectBuilder_Build_EmptySpec(t *testing.T) {
	builder := &ProjectBuilder{
		coder: &mockCoder{},
		shell: &mockShell{},
	}

	spec := ProjectSpec{
		Name:     "empty",
		Type:     "cli_tool",
		Language: "python",
	}

	ctx := context.Background()
	result, err := builder.Build(ctx, spec)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected build to succeed with default settings")
	}
}

func TestBugHunter_FixBug(t *testing.T) {
	hunter := &BugHunter{
		coder: &mockCoder{},
	}

	issue := GitHubIssue{
		Number:    123,
		Title:     "Fix nil pointer",
		Body:      "This causes a crash when nil is passed",
		Labels:    []string{"bug"},
		Repo:      "test/repo",
		State:     "open",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	fix, err := hunter.fixBug(ctx, issue)
	if err != nil {
		t.Fatalf("Fix failed: %v", err)
	}

	if fix.Issue.Number != 123 {
		t.Errorf("Expected issue 123, got %d", fix.Issue.Number)
	}
}

func TestOSSEngine_ScoreIssues(t *testing.T) {
	engine := &OSSEngine{
		coder: &mockCoder{},
	}

	issues := []OSSIssue{
		{Number: 1, Title: "Add feature X", Body: "Should implement feature X", Labels: []string{"good first issue"}},
		{Number: 2, Title: "Fix bug Y", Body: "Bug in production", Labels: []string{"bug"}},
	}

	capabilities := []string{"go", "api", "数据库"}

	ctx := context.Background()
	ranked, err := engine.ScoreIssues(ctx, issues, capabilities)
	if err != nil {
		t.Fatalf("Scoring failed: %v", err)
	}

	if len(ranked) != len(issues) {
		t.Errorf("Expected %d ranked issues, got %d", len(issues), len(ranked))
	}

	// Check ranks are assigned
	for _, r := range ranked {
		if r.Rank == 0 {
			t.Error("Expected rank to be assigned")
		}
	}
}

func TestProjectSpec_Endpoint(t *testing.T) {
	spec := ProjectSpec{
		Name:     "test",
		Type:     "rest_api",
		Language: "go",
		Endpoints: []Endpoint{
			{Path: "/api/v1/users", Method: "GET", Handler: "GetUsers"},
		},
	}

	if spec.Name != "test" {
		t.Errorf("Expected name 'test', got %s", spec.Name)
	}
	if spec.Type != "rest_api" {
		t.Errorf("Expected type 'rest_api', got %s", spec.Type)
	}
	if spec.Language != "go" {
		t.Errorf("Expected language 'go', got %s", spec.Language)
	}
	if len(spec.Endpoints) != 1 {
		t.Errorf("Expected 1 endpoint, got %d", len(spec.Endpoints))
	}

	if spec.Endpoints[0].Method != "GET" {
		t.Errorf("Expected GET method, got %s", spec.Endpoints[0].Method)
	}
}

// Mock implementations
type mockCoder struct{}

func (c *mockCoder) Complete(req CodeRequest) (string, error) {
	return "mock code", nil
}

type mockShell struct{}

func (s *mockShell) Execute(ctx context.Context, cmd string) (string, error) {
	return "OK", nil
}

type mockRunner struct{}

func (r *mockRunner) RunTests(ctx context.Context, path string) (TestResult, error) {
	return TestResult{Passed: true, Output: "all tests passed", Coverage: 85.0}, nil
}
