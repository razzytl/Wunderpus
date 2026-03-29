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
	workspace  string // Absolute path to workspace root
	restricted bool   // Whether restriction is active
}

// NewWorkspaceSandbox creates a workspace sandbox.
// If restricted is true, all path operations must be within the workspace.
// If restricted is false, sandbox allows all paths (no restrictions).
func NewWorkspaceSandbox(workspace string, restricted bool) (*WorkspaceSandbox, error) {
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace path %q: %w", workspace, err)
	}

	return &WorkspaceSandbox{
		workspace:  absWorkspace,
		restricted: restricted,
	}, nil
}

// Initialize ensures the workspace directory exists.
func (ws *WorkspaceSandbox) Initialize() error {
	if ws.restricted {
		if err := os.MkdirAll(ws.workspace, 0755); err != nil {
			return fmt.Errorf("cannot create workspace directory %q: %w", ws.workspace, err)
		}
	}
	return nil
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

	// Resolve the path relative to the workspace if it's not absolute
	var absPath string
	if !filepath.IsAbs(path) {
		// On Windows, paths starting with / or \ are root-relative, not relative to CWD.
		// filepath.Join will handle this correctly on Unix, but on Windows we need to be careful.
		if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
			// This is a root-relative path. Resolve it to an absolute path on the current drive.
			var err error
			absPath, err = filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("workspace sandbox: invalid path %q: %w", path, err)
			}
		} else {
			absPath = filepath.Clean(filepath.Join(ws.workspace, path))
		}
	} else {
		absPath = filepath.Clean(path)
	}

	// Check if the path is within workspace
	rel, err := filepath.Rel(ws.workspace, absPath)
	if err != nil {
		return fmt.Errorf("workspace sandbox: invalid path %q: %w", path, err)
	}

	// Check if rel starts with ".." or is ".."
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
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
// paths outside the workspace or use dangerous command chaining.
func (ws *WorkspaceSandbox) ValidateCommand(command string) error {
	if !ws.restricted {
		return nil
	}

	// Block command chaining characters that could bypass simple checks
	chainingChars := []string{";", "&&", "||", "|", "`", "$(", ">", "<"}
	for _, char := range chainingChars {
		if strings.Contains(command, char) {
			return fmt.Errorf("workspace sandbox: access denied — command chaining or redirection (%q) is prohibited in restricted mode", char)
		}
	}

	// Block cd to outside workspace
	lower := strings.ToLower(strings.TrimSpace(command))

	// Basic field splitting to get command and arguments
	fields := strings.Fields(lower)
	if len(fields) == 0 {
		return nil
	}

	cmd := fields[0]
	cdCommands := map[string]bool{"cd": true, "pushd": true, "chdir": true}

	if cdCommands[cmd] {
		if len(fields) > 1 {
			// Check the first argument as a path
			path := strings.Trim(fields[1], "\"'")
			if err := ws.ValidatePath(path); err != nil {
				return fmt.Errorf("workspace sandbox: command attempts to access path outside workspace: %w", err)
			}
		}
	}

	return nil
}
