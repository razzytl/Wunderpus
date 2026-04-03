package toolsynth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Designer produces a ToolSpec from a ToolGap using LLM calls.
// It generates a JSON specification that the Coder can use to produce Go source.
type Designer struct {
	llm LLMCaller
	db  *sql.DB // optional: shared core DB connection for storing ToolSpecs
}

// NewDesigner creates a designer with the given LLM caller.
func NewDesigner(llm LLMCaller) *Designer {
	return &Designer{llm: llm}
}

// SetDB configures the shared database connection for persisting ToolSpecs.
func (d *Designer) SetDB(db *sql.DB) {
	d.db = db
}

// Design takes a ToolGap and produces a complete ToolSpec via LLM.
// The output includes: name, description, Go interface, parameters,
// return type, dependencies, and test cases.
func (d *Designer) Design(gap ToolGap) (*ToolSpec, error) {
	slog.Info("designer: designing tool for gap", "gap", gap.Name, "type", gap.GapType)

	prompt := buildDesignPrompt(gap)

	response, err := d.llm.Complete(CompletionRequest{
		SystemPrompt: designSystemPrompt,
		UserPrompt:   prompt,
		Temperature:  0.3,
		MaxTokens:    2000,
	})
	if err != nil {
		return nil, fmt.Errorf("designer: LLM call failed: %w", err)
	}

	spec, err := parseToolSpec(response)
	if err != nil {
		return nil, fmt.Errorf("designer: parse spec: %w", err)
	}

	// Validate the spec
	if err := validateSpec(spec); err != nil {
		return nil, fmt.Errorf("designer: invalid spec: %w", err)
	}

	// Stamp it
	spec.Origin = "synthesized"
	spec.CreatedAt = time.Now()
	spec.GapEvidence = gap.Evidence

	// Persist ToolSpec to SQLite if configured
	if err := d.storeSpec(spec); err != nil {
		slog.Warn("designer: failed to persist spec (non-fatal)", "error", err)
	}

	slog.Info("designer: spec produced",
		"name", spec.Name,
		"params", len(spec.Parameters),
		"deps", len(spec.Dependencies),
		"tests", len(spec.TestCases))

	return spec, nil
}

// storeSpec persists a ToolSpec to SQLite.
func (d *Designer) storeSpec(spec *ToolSpec) error {
	if d.db == nil {
		return nil
	}

	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS synth_tool_specs (
			name TEXT PRIMARY KEY,
			spec_json TEXT,
			created_at TEXT
		)`)
	if err != nil {
		return err
	}

	specJSON, _ := json.Marshal(spec)
	_, err = d.db.Exec(`
		INSERT OR REPLACE INTO synth_tool_specs (name, spec_json, created_at)
		VALUES (?, ?, ?)`,
		spec.Name, string(specJSON), spec.CreatedAt.Format(time.RFC3339))

	return err
}

const designSystemPrompt = `You are a Go tool designer for the Wunderpus autonomous agent.
Given a tool gap description, produce a JSON ToolSpec for a new Go tool.

The tool must:
1. Be a single Go function with clear inputs and outputs
2. Use only standard library or common Go packages (no CGO by default)
3. Handle errors properly
4. Be testable with deterministic inputs/outputs

Respond with ONLY a valid JSON object matching this schema:
{
  "name": "tool_name_in_snake_case",
  "description": "What this tool does",
  "go_interface": "func FunctionName(params) (returnType, error)",
  "parameters": [
    {"name": "param1", "type": "string", "description": "what it does", "required": true}
  ],
  "return_type": "Go type string",
  "dependencies": ["github.com/some/package"],
  "test_cases": [
    {"input": {"param1": "value"}, "expected": "expected output or substring"}
  ]
}

IMPORTANT: Output ONLY the JSON. No markdown, no explanation.`

// buildDesignPrompt creates the user prompt for the LLM.
func buildDesignPrompt(gap ToolGap) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Tool Gap: %s\n", gap.Name))
	sb.WriteString(fmt.Sprintf("Type: %s\n", gap.GapType))
	sb.WriteString(fmt.Sprintf("Description: %s\n", gap.Description))
	sb.WriteString(fmt.Sprintf("Suggested interface: %s\n", gap.SuggestedInterface))

	if len(gap.Evidence) > 0 {
		sb.WriteString("\nEvidence from episodic memory:\n")
		for i, e := range gap.Evidence {
			if i >= 5 {
				break // limit evidence in prompt
			}
			sb.WriteString(fmt.Sprintf("- %s\n", e))
		}
	}

	sb.WriteString("\nDesign a Go tool that fills this gap. Include at least 2 test cases.\n")

	return sb.String()
}

// parseToolSpec extracts a ToolSpec from an LLM response.
func parseToolSpec(response string) (*ToolSpec, error) {
	// Strip markdown code fences if present
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		// Remove first and last lines (code fences)
		if len(lines) >= 2 {
			response = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	response = strings.TrimSpace(response)

	var spec ToolSpec
	if err := json.Unmarshal([]byte(response), &spec); err != nil {
		return nil, fmt.Errorf("invalid JSON in LLM response: %w", err)
	}

	return &spec, nil
}

// validateSpec checks that a ToolSpec has all required fields.
func validateSpec(spec *ToolSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("missing tool name")
	}
	if spec.Description == "" {
		return fmt.Errorf("missing description")
	}
	if spec.GoInterface == "" {
		return fmt.Errorf("missing Go interface")
	}
	if spec.ReturnType == "" {
		return fmt.Errorf("missing return type")
	}
	if len(spec.TestCases) == 0 {
		return fmt.Errorf("no test cases provided")
	}

	// Validate parameter names are non-empty
	for i, p := range spec.Parameters {
		if p.Name == "" {
			return fmt.Errorf("parameter %d has empty name", i)
		}
		if p.Type == "" {
			return fmt.Errorf("parameter %q has empty type", p.Name)
		}
	}

	// Validate dependency format (if present)
	for _, dep := range spec.Dependencies {
		if dep != "" && !strings.Contains(dep, "/") && !isStandardLib(dep) {
			return fmt.Errorf("invalid dependency format: %q", dep)
		}
	}

	return nil
}

// isStandardLib checks if a package is in Go's standard library.
func isStandardLib(pkg string) bool {
	stdPkgs := []string{
		"fmt", "os", "io", "strings", "strconv", "errors", "context",
		"encoding/json", "encoding/csv", "encoding/xml", "net/http",
		"path/filepath", "time", "sync", "bytes", "bufio", "log",
		"math", "regexp", "sort", "database/sql",
	}
	for _, s := range stdPkgs {
		if pkg == s {
			return true
		}
	}
	return false
}
