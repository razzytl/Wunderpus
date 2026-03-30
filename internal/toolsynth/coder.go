package toolsynth

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CandidateTemperatures defines the temperatures used for generating tool candidates.
var CandidateTemperatures = []float64{0.2, 0.5, 0.8}

// Coder generates Go source code for synthesized tools.
// It produces multiple candidates at different temperatures and validates
// each with go build and go vet before accepting.
type Coder struct {
	llm       LLMCaller
	validator CodeValidator
	outputDir string // where validated sources are written (internal/tool/generated/)
	workDir   string // temp dir for compilation validation
}

// NewCoder creates a coder with the given LLM caller and validator.
// outputDir is where validated .go files are written.
func NewCoder(llm LLMCaller, validator CodeValidator, outputDir string) *Coder {
	return &Coder{
		llm:       llm,
		validator: validator,
		outputDir: outputDir,
		workDir:   os.TempDir(),
	}
}

// SetWorkDir overrides the temp directory used for build validation.
func (c *Coder) SetWorkDir(dir string) {
	c.workDir = dir
}

// Generate produces Go source code candidates from a ToolSpec.
// It generates candidates at temperatures [0.2, 0.5, 0.8] and returns
// only those that compile successfully.
func (c *Coder) Generate(spec ToolSpec) ([]GeneratedCandidate, error) {
	slog.Info("coder: generating candidates", "tool", spec.Name)

	// Ensure output directory exists
	if err := os.MkdirAll(c.outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("coder: create output dir: %w", err)
	}

	candidates := make([]GeneratedCandidate, 0, len(CandidateTemperatures))

	for _, temp := range CandidateTemperatures {
		slog.Debug("coder: generating candidate", "tool", spec.Name, "temperature", temp)

		source, err := c.generateCandidate(spec, temp)
		if err != nil {
			slog.Warn("coder: candidate generation failed",
				"tool", spec.Name, "temperature", temp, "error", err)
			continue
		}

		// Validate the candidate
		buildOutput, buildErr := c.validator.Validate(source, c.workDir)

		candidate := GeneratedCandidate{
			Source:      source,
			Temperature: temp,
			BuildError:  "",
		}
		if buildErr != nil {
			candidate.BuildError = buildOutput
			slog.Debug("coder: candidate failed validation",
				"tool", spec.Name, "temperature", temp, "error", buildErr)
			// Still include it — the tester may want to see all candidates
		}

		candidates = append(candidates, candidate)
	}

	// Count valid candidates
	validCount := 0
	for _, c := range candidates {
		if c.BuildError == "" {
			validCount++
		}
	}

	slog.Info("coder: generation complete",
		"tool", spec.Name,
		"total", len(candidates),
		"valid", validCount)

	if validCount == 0 {
		return candidates, fmt.Errorf("coder: no valid candidates produced for %s", spec.Name)
	}

	return candidates, nil
}

// WriteValidSource writes the first valid (compiling) candidate to the output directory.
// Returns the file path of the written source.
func (c *Coder) WriteValidSource(spec ToolSpec, candidates []GeneratedCandidate) (string, error) {
	for _, candidate := range candidates {
		if candidate.BuildError != "" {
			continue
		}

		fileName := fmt.Sprintf("%s.go", spec.Name)
		filePath := filepath.Join(c.outputDir, fileName)

		if err := os.WriteFile(filePath, []byte(candidate.Source), 0o644); err != nil {
			return "", fmt.Errorf("coder: write source: %w", err)
		}

		slog.Info("coder: wrote validated source", "path", filePath)
		return filePath, nil
	}

	return "", fmt.Errorf("coder: no valid candidates to write")
}

// generateCandidate calls the LLM to produce one candidate source file.
func (c *Coder) generateCandidate(spec ToolSpec, temperature float64) (string, error) {
	prompt := buildCodePrompt(spec)

	response, err := c.llm.Complete(CompletionRequest{
		SystemPrompt: coderSystemPrompt,
		UserPrompt:   prompt,
		Temperature:  temperature,
		MaxTokens:    4000,
	})
	if err != nil {
		return "", err
	}

	// Clean up the response
	source := cleanSourceResponse(response)
	return source, nil
}

const coderSystemPrompt = `You are a Go code generator for the Wunderpus autonomous agent.
Given a ToolSpec, produce a complete, compilable Go source file.

Requirements:
1. Package must be "generated" (package generated)
2. Must implement the tool.Tool interface:
   - Name() string
   - Description() string
   - Parameters() []tool.ParameterDef
   - Execute(ctx context.Context, args map[string]any) (*tool.ToolResult, error)
   - Sensitive() bool
   - Version() string
   - Dependencies() []string
3. Proper error handling — never panic
4. Context cancellation support
5. No global state
6. Timeout support via context
7. Import the correct package path: "github.com/wunderpus/wunderpus/internal/tool"

Respond with ONLY the Go source code. No markdown fences, no explanation.
The file must compile with 'go build' without errors.`

// buildCodePrompt creates the user prompt for code generation.
func buildCodePrompt(spec ToolSpec) string {
	var sb strings.Builder

	sb.WriteString("Generate a Go source file for this tool:\n\n")

	specJSON, _ := formatSpecForPrompt(spec)
	sb.WriteString(specJSON)

	sb.WriteString("\n\nRequirements:\n")
	sb.WriteString("- Package: generated\n")
	sb.WriteString("- Must implement tool.Tool interface\n")
	sb.WriteString("- Must compile without errors\n")
	sb.WriteString("- Must handle context cancellation\n")
	sb.WriteString("- Must not use global state\n")

	if len(spec.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("- Dependencies to use: %s\n", strings.Join(spec.Dependencies, ", ")))
	}

	sb.WriteString("\nProduce ONLY the Go source code.\n")

	return sb.String()
}

// formatSpecForPrompt creates a readable spec summary for the LLM prompt.
func formatSpecForPrompt(spec ToolSpec) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Name: %s\n", spec.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n", spec.Description))
	sb.WriteString(fmt.Sprintf("Interface: %s\n", spec.GoInterface))
	sb.WriteString(fmt.Sprintf("Return type: %s\n", spec.ReturnType))

	if len(spec.Parameters) > 0 {
		sb.WriteString("Parameters:\n")
		for _, p := range spec.Parameters {
			req := ""
			if p.Required {
				req = " (required)"
			}
			sb.WriteString(fmt.Sprintf("  - %s (%s)%s: %s\n", p.Name, p.Type, req, p.Description))
		}
	}

	if len(spec.TestCases) > 0 {
		sb.WriteString("Test cases to support:\n")
		for i, tc := range spec.TestCases {
			sb.WriteString(fmt.Sprintf("  %d. Input: %v → Expected: %s\n", i+1, tc.Input, tc.Expected))
		}
	}

	return sb.String(), nil
}

// cleanSourceResponse strips markdown code fences and whitespace from LLM output.
func cleanSourceResponse(response string) string {
	response = strings.TrimSpace(response)

	// Remove ```go or ``` fences
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) >= 2 {
			// Find end fence
			endIdx := len(lines) - 1
			for i := len(lines) - 1; i > 0; i-- {
				if strings.TrimSpace(lines[i]) == "```" {
					endIdx = i
					break
				}
			}
			response = strings.Join(lines[1:endIdx], "\n")
		}
	}

	return strings.TrimSpace(response)
}

// DefaultValidator implements CodeValidator using go build and go vet.
type DefaultValidator struct {
	goBinary string // path to go binary, defaults to "go"
}

// NewDefaultValidator creates a validator that uses the system Go toolchain.
func NewDefaultValidator() *DefaultValidator {
	goBin := "go"
	if p, err := exec.LookPath("go"); err == nil {
		goBin = p
	}
	return &DefaultValidator{goBinary: goBin}
}

// Validate compiles and vets the given Go source in a temp directory.
// Returns the build output and any error.
func (v *DefaultValidator) Validate(source string, workDir string) (string, error) {
	// Create temp directory for compilation
	tmpDir, err := os.MkdirTemp(workDir, "toolsynth-validate-*")
	if err != nil {
		return "", fmt.Errorf("validate: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod
	goMod := `module toolsynth-validate

go 1.25.0

require github.com/wunderpus/wunderpus v0.0.0

replace github.com/wunderpus/wunderpus => ` + filepath.Join(workDir, "..", "..") + "\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return "", fmt.Errorf("validate: write go.mod: %w", err)
	}

	// Create package directory
	pkgDir := filepath.Join(tmpDir, "generated")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		return "", fmt.Errorf("validate: create pkg dir: %w", err)
	}

	// Write source file
	if err := os.WriteFile(filepath.Join(pkgDir, "tool.go"), []byte(source), 0o644); err != nil {
		return "", fmt.Errorf("validate: write source: %w", err)
	}

	// Run go build
	buildCmd := exec.Command(v.goBinary, "build", "./generated")
	buildCmd.Dir = tmpDir
	buildOutput, buildErr := buildCmd.CombinedOutput()

	if buildErr != nil {
		return string(buildOutput), fmt.Errorf("go build failed: %w", buildErr)
	}

	// Run go vet
	vetCmd := exec.Command(v.goBinary, "vet", "./generated")
	vetCmd.Dir = tmpDir
	vetOutput, vetErr := vetCmd.CombinedOutput()

	if vetErr != nil {
		return string(vetOutput), fmt.Errorf("go vet failed: %w", vetErr)
	}

	// Run staticcheck if available
	if scBin, err := exec.LookPath("staticcheck"); err == nil {
		scCmd := exec.Command(scBin, "./generated/...")
		scCmd.Dir = tmpDir
		scOutput, scErr := scCmd.CombinedOutput()
		if scErr != nil {
			return string(scOutput), fmt.Errorf("staticcheck failed: %w", scErr)
		}
	}

	return "", nil
}
