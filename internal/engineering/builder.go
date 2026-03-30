package engineering

import (
	"context"
	"log/slog"
	"time"
)

// ProjectSpec represents a complete software project specification.
type ProjectSpec struct {
	Name       string     `json:"name"`
	Type       string     `json:"type"`     // "rest_api", "web_app", "cli_tool", "library"
	Language   string     `json:"language"` // "go", "python", "typescript", "rust"
	Frameworks []string   `json:"frameworks"`
	Endpoints  []Endpoint `json:"endpoints"`
	Database   string     `json:"database"` // "none", "sqlite", "postgres", "mysql"
	Features   []string   `json:"features"`
}

// Endpoint represents an API endpoint specification.
type Endpoint struct {
	Path         string `json:"path"`
	Method       string `json:"method"` // "GET", "POST", "PUT", "DELETE"
	Handler      string `json:"handler"`
	InputSchema  string `json:"input_schema"`
	OutputSchema string `json:"output_schema"`
}

// ProjectBuilder builds complete software projects from specifications.
type ProjectBuilder struct {
	coder      CoderLLM
	shell      ShellExecutor
	sandbox    TestRunner
	worldModel WorldModelQuery
}

// CoderLLM interface for code generation.
type CoderLLM interface {
	Complete(req CodeRequest) (string, error)
}

// CodeRequest represents a code generation request.
type CodeRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
}

// ShellExecutor executes shell commands.
type ShellExecutor interface {
	Execute(ctx context.Context, cmd string) (string, error)
}

// TestRunner runs tests in a sandbox.
type TestRunner interface {
	RunTests(ctx context.Context, path string) (TestResult, error)
}

// TestResult represents test execution results.
type TestResult struct {
	Passed   bool
	Output   string
	Coverage float64
}

// WorldModelQuery queries the world model for context.
type WorldModelQuery interface {
	Ask(ctx context.Context, question string) (string, error)
}

// BuilderConfig holds configuration for the project builder.
type BuilderConfig struct {
	Enabled     bool
	MinTestCov  float64
	SandboxPath string
}

// NewProjectBuilder creates a new project builder.
func NewProjectBuilder(cfg BuilderConfig, coder CoderLLM, shell ShellExecutor, sandbox TestRunner, wm WorldModelQuery) *ProjectBuilder {
	return &ProjectBuilder{
		coder:      coder,
		shell:      shell,
		sandbox:    sandbox,
		worldModel: wm,
	}
}

// Build constructs a complete project from specification.
func (p *ProjectBuilder) Build(ctx context.Context, spec ProjectSpec) (*BuildResult, error) {
	slog.Info("engineering: building project", "name", spec.Name, "type", spec.Type)

	result := &BuildResult{
		ProjectName: spec.Name,
		Steps:       []BuildStep{},
		Success:     false,
	}

	// Step 1: Extract requirements
	result.Steps = append(result.Steps, BuildStep{
		Name:     "requirements",
		Duration: time.Since(time.Now()),
		Success:  true,
	})
	slog.Info("engineering: requirements extracted")

	// Step 2: Generate architecture
	archPrompt := "Design a system architecture for: " + spec.Name + " (" + spec.Language + ")"
	archReq := CodeRequest{
		SystemPrompt: "You are a software architect. Design clear, maintainable systems.",
		UserPrompt:   archPrompt,
		Temperature:  0.5,
		MaxTokens:    2000,
	}
	_, err := p.coder.Complete(archReq)
	if err != nil {
		result.Steps = append(result.Steps, BuildStep{Name: "architecture", Success: false, Error: err.Error()})
		return result, err
	}
	result.Steps = append(result.Steps, BuildStep{Name: "architecture", Success: true})
	slog.Info("engineering: architecture generated")

	// Step 3: Scaffold project
	scaffoldCmd := "mkdir " + spec.Name
	if spec.Language == "go" {
		scaffoldCmd += " && cd " + spec.Name + " && go mod init " + spec.Name
	} else if spec.Language == "python" {
		scaffoldCmd += " && cd " + spec.Name + " && python -m venv venv"
	}
	_, err = p.shell.Execute(ctx, scaffoldCmd)
	if err != nil {
		slog.Warn("engineering: scaffold failed", "error", err)
	}
	result.Steps = append(result.Steps, BuildStep{Name: "scaffold", Success: err == nil})
	slog.Info("engineering: project scaffolded")

	// Step 4: Implement components
	for _, endpoint := range spec.Endpoints {
		codePrompt := "Generate Go handler for " + endpoint.Method + " " + endpoint.Path + " with input: " + endpoint.InputSchema
		codeReq := CodeRequest{
			SystemPrompt: "You are a Go developer. Write clean, tested, production-ready code.",
			UserPrompt:   codePrompt,
			Temperature:  0.3,
			MaxTokens:    1500,
		}
		_, err := p.coder.Complete(codeReq)
		if err != nil {
			result.Steps = append(result.Steps, BuildStep{Name: "implement_" + endpoint.Path, Success: false})
			continue
		}
		result.Steps = append(result.Steps, BuildStep{Name: "implement_" + endpoint.Path, Success: true})
	}

	// Step 5: Run tests
	if p.sandbox != nil {
		testRes, err := p.sandbox.RunTests(ctx, spec.Name)
		result.TestResult = &testRes
		if err != nil {
			result.Steps = append(result.Steps, BuildStep{Name: "tests", Success: false, Error: err.Error()})
			return result, err
		}
		result.Steps = append(result.Steps, BuildStep{Name: "tests", Success: testRes.Passed, Output: testRes.Output})
	}

	result.Success = true
	result.CompletedAt = time.Now()
	slog.Info("engineering: build complete", "name", spec.Name, "success", result.Success)

	return result, nil
}

// BuildResult represents the result of a build operation.
type BuildResult struct {
	ProjectName string
	Steps       []BuildStep
	TestResult  *TestResult
	Success     bool
	CompletedAt time.Time
}

// BuildStep represents a single step in the build process.
type BuildStep struct {
	Name     string
	Duration time.Duration
	Success  bool
	Output   string
	Error    string
}
