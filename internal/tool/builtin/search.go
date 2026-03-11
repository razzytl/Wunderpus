package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wunderpus/wunderpus/internal/security"
	"github.com/wunderpus/wunderpus/internal/tool"
)

// SearchFiles searches for content inside files within the workspace sandbox.
type SearchFiles struct {
	allowedPaths []string
	sandbox      *security.WorkspaceSandbox
}

// NewSearchFiles creates a new content search tool.
func NewSearchFiles(allowedPaths []string) *SearchFiles {
	return &SearchFiles{allowedPaths: allowedPaths}
}

// NewSearchFilesSandboxed creates a content search tool restricted to the workspace sandbox.
func NewSearchFilesSandboxed(sandbox *security.WorkspaceSandbox) *SearchFiles {
	return &SearchFiles{
		allowedPaths: sandbox.AllowedPaths(),
		sandbox:      sandbox,
	}
}

func (s *SearchFiles) Name() string        { return "content_search" }
func (s *SearchFiles) Description() string  { return "Search for a string within files in restricted directories (grep-like). Supports recursive search if path is a directory." }
func (s *SearchFiles) Sensitive() bool      { return false }
func (s *SearchFiles) Version() string      { return "1.0.0" }
func (s *SearchFiles) Dependencies() []string { return nil }
func (s *SearchFiles) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "query", Type: "string", Description: "The string to search for", Required: true},
		{Name: "path", Type: "string", Description: "The directory or file to search in (defaults to current directory)", Required: false},
	}
}

func (s *SearchFiles) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	query, _ := args["query"].(string)
	path, _ := args["path"].(string)

	if query == "" {
		return &tool.Result{Error: "query is required"}, nil
	}
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("invalid path: %v", err)}, nil
	}

	if err := s.checkAccess(absPath, path); err != nil {
		return &tool.Result{Error: err.Error()}, nil
	}

	var results []string
	err = filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Don't search large binary files or git/dist dirs
		if info.Size() > 1024*1024 || strings.Contains(p, ".git") {
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}

		content := string(data)
		if strings.Contains(content, query) {
			// Find line numbers
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				if strings.Contains(line, query) {
					rel, _ := filepath.Rel(absPath, p)
					results = append(results, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
				}
				if len(results) > 100 {
					return fmt.Errorf("too many results")
				}
			}
		}
		return nil
	})

	if err != nil && err.Error() != "too many results" {
		return &tool.Result{Error: fmt.Sprintf("search failed: %v", err)}, nil
	}

	if len(results) == 0 {
		return &tool.Result{Output: "no matches found"}, nil
	}

	out := strings.Join(results, "\n")
	if len(out) > 5000 {
		out = out[:5000] + "\n... (truncated)"
	}

	return &tool.Result{Output: out}, nil
}

func (s *SearchFiles) checkAccess(absPath, originalPath string) error {
	if s.sandbox != nil {
		return s.sandbox.ValidatePath(originalPath)
	}
	if !isPathAllowed(absPath, s.allowedPaths) {
		return fmt.Errorf("access denied: %s is outside allowed paths", originalPath)
	}
	return nil
}
