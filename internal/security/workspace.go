package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WorkspaceSandbox enforces workspace directory restrictions.
// When enabled, all file and command operations are restricted to the workspace directory.
type WorkspaceSandbox struct {
	workspace   string // Absolute path to workspace root
	restricted  bool   // Whether restriction is active
}

// NewWorkspaceSandbox creates a workspace sandbox.
// If restricted is true, all path operations must be within the workspace.
// If restricted is false, sandbox allows all paths (no restrictions).
func NewWorkspaceSandbox(workspace string, restricted bool) (*WorkspaceSandbox, error) {
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace path %q: %w", workspace, err)
	}

	// Ensure workspace directory exists (create if needed)
	if restricted {
		if err := os.MkdirAll(absWorkspace, 0755); err != nil {
			return nil, fmt.Errorf("cannot create workspace directory %q: %w", absWorkspace, err)
		}
	}

	return &WorkspaceSandbox{
		workspace:  absWorkspace,
		restricted: restricted,
	}, nil
}

// WorkspacePath returns the absolute workspace path.
func (ws *WorkspaceSandbox) WorkspacePath() string {
	return ws.workspace
}

// IsRestricted returns whether workspace restriction is enabled.
func (ws *WorkspaceSandbox) IsRestricted() bool {
	return ws.restricted
}

// ValidatePath checks if the given path is allowed under the workspace restriction.
// Returns an error with a clear message if access is denied.
func (ws *WorkspaceSandbox) ValidatePath(path string) error {
	if !ws.restricted {
		return nil // No restriction
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("workspace sandbox: invalid path %q: %w", path, err)
	}

	// Block path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("workspace sandbox: access denied — path traversal detected in %q", path)
	}

	// Check that the path is within workspace
	// Use filepath.Rel to properly check containment
	rel, err := filepath.Rel(ws.workspace, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf(
			"workspace sandbox: access denied — %q is outside workspace %q. "+
				"Set restrict_to_workspace: false in config to disable this restriction",
			path, ws.workspace,
		)
	}

	return nil
}

// AllowedPaths returns the workspace path as a single-element allowed paths list.
// This provides backward compatibility with existing tool constructors.
func (ws *WorkspaceSandbox) AllowedPaths() []string {
	if !ws.restricted {
		// When unrestricted, return a broad set of paths
		return []string{"/", "C:\\", "D:\\", "E:\\"}
	}
	return []string{ws.workspace}
}

// ValidateCommand checks that a shell command doesn't attempt to access
// paths outside the workspace. This is a best-effort check that inspects
// common file path patterns in the command string.
func (ws *WorkspaceSandbox) ValidateCommand(command string) error {
	if !ws.restricted {
		return nil
	}

	// Block cd to outside workspace
	lower := strings.ToLower(command)
	cdPatterns := []string{"cd ", "cd\t", "pushd ", "chdir "}
	for _, pat := range cdPatterns {
		if strings.Contains(lower, pat) {
			// Extract the directory argument (very basic parsing)
			idx := strings.Index(lower, pat)
			rest := strings.TrimSpace(command[idx+len(pat):])
			// Remove quotes
			rest = strings.Trim(rest, "\"'")
			if rest != "" {
				if err := ws.ValidatePath(rest); err != nil {
					return fmt.Errorf("workspace sandbox: command attempts to access path outside workspace: %w", err)
				}
			}
		}
	}

	return nil
}
