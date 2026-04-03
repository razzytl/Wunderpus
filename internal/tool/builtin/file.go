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

// FileRead reads a file within the workspace sandbox.
type FileRead struct {
	allowedPaths []string
	sandbox      *security.WorkspaceSandbox
}

// NewFileRead creates a file read tool with allowed base paths.
func NewFileRead(allowedPaths []string) *FileRead {
	return &FileRead{allowedPaths: allowedPaths}
}

// NewFileReadSandboxed creates a file read tool restricted to the workspace sandbox.
func NewFileReadSandboxed(sandbox *security.WorkspaceSandbox) *FileRead {
	return &FileRead{
		allowedPaths: sandbox.AllowedPaths(),
		sandbox:      sandbox,
	}
}

func (f *FileRead) Name() string { return "file_read" }
func (f *FileRead) Description() string {
	return "Read the contents of a file. Only files within allowed directories can be read."
}
func (f *FileRead) Sensitive() bool                   { return false }
func (f *FileRead) ApprovalLevel() tool.ApprovalLevel { return tool.AutoExecute }
func (f *FileRead) Version() string                   { return "1.0.0" }
func (f *FileRead) Dependencies() []string            { return nil }
func (f *FileRead) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "path", Type: "string", Description: "Path to the file to read", Required: true},
	}
}

func (f *FileRead) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return &tool.Result{Error: "path is required"}, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("invalid path: %v", err)}, nil
	}

	if err := f.checkAccess(absPath, path); err != nil {
		return &tool.Result{Error: err.Error()}, nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("read error: %v", err)}, nil
	}

	// Limit output size
	content := string(data)
	if len(content) > 10000 {
		content = content[:10000] + "\n... (truncated, file too large)"
	}

	return &tool.Result{Output: content}, nil
}

func (f *FileRead) checkAccess(absPath, originalPath string) error {
	if f.sandbox != nil {
		return f.sandbox.ValidatePath(originalPath)
	}
	if !isPathAllowed(absPath, f.allowedPaths) {
		return fmt.Errorf("access denied: %s is outside allowed paths", originalPath)
	}
	return nil
}

// FileWrite writes content to a file within the workspace sandbox.
type FileWrite struct {
	allowedPaths []string
	sandbox      *security.WorkspaceSandbox
}

// NewFileWrite creates a file write tool with allowed base paths.
func NewFileWrite(allowedPaths []string) *FileWrite {
	return &FileWrite{allowedPaths: allowedPaths}
}

// NewFileWriteSandboxed creates a file write tool restricted to the workspace sandbox.
func NewFileWriteSandboxed(sandbox *security.WorkspaceSandbox) *FileWrite {
	return &FileWrite{
		allowedPaths: sandbox.AllowedPaths(),
		sandbox:      sandbox,
	}
}

func (f *FileWrite) Name() string { return "file_write" }
func (f *FileWrite) Description() string {
	return "Write content to a file. Only files within allowed directories can be written. Requires user approval."
}
func (f *FileWrite) Sensitive() bool                   { return true }
func (f *FileWrite) ApprovalLevel() tool.ApprovalLevel { return tool.RequiresApproval }
func (f *FileWrite) Version() string                   { return "1.0.0" }
func (f *FileWrite) Dependencies() []string            { return nil }
func (f *FileWrite) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "path", Type: "string", Description: "Path to the file to write", Required: true},
		{Name: "content", Type: "string", Description: "Content to write to the file", Required: true},
	}
}

func (f *FileWrite) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	if path == "" {
		return &tool.Result{Error: "path is required"}, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("invalid path: %v", err)}, nil
	}

	if err := f.checkAccess(absPath, path); err != nil {
		return &tool.Result{Error: err.Error()}, nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &tool.Result{Error: fmt.Sprintf("cannot create directory: %v", err)}, nil
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return &tool.Result{Error: fmt.Sprintf("write error: %v", err)}, nil
	}

	return &tool.Result{Output: fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)}, nil
}

func (f *FileWrite) checkAccess(absPath, originalPath string) error {
	if f.sandbox != nil {
		return f.sandbox.ValidatePath(originalPath)
	}
	if !isPathAllowed(absPath, f.allowedPaths) {
		return fmt.Errorf("access denied: %s is outside allowed paths", originalPath)
	}
	return nil
}

// FileList lists files in a directory within the workspace sandbox.
type FileList struct {
	allowedPaths []string
	sandbox      *security.WorkspaceSandbox
}

// NewFileList creates a file list tool with allowed base paths.
func NewFileList(allowedPaths []string) *FileList {
	return &FileList{allowedPaths: allowedPaths}
}

// NewFileListSandboxed creates a file list tool restricted to the workspace sandbox.
func NewFileListSandboxed(sandbox *security.WorkspaceSandbox) *FileList {
	return &FileList{
		allowedPaths: sandbox.AllowedPaths(),
		sandbox:      sandbox,
	}
}

func (f *FileList) Name() string { return "file_list" }
func (f *FileList) Description() string {
	return "List files and directories in a given path. Only allowed directories can be listed."
}
func (f *FileList) Sensitive() bool                   { return false }
func (f *FileList) ApprovalLevel() tool.ApprovalLevel { return tool.AutoExecute }
func (f *FileList) Version() string                   { return "1.0.0" }
func (f *FileList) Dependencies() []string            { return nil }
func (f *FileList) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "path", Type: "string", Description: "Directory path to list (defaults to current directory)", Required: false},
	}
}

func (f *FileList) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("invalid path: %v", err)}, nil
	}

	if err := f.checkAccess(absPath, path); err != nil {
		return &tool.Result{Error: err.Error()}, nil
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("read dir error: %v", err)}, nil
	}

	var lines []string
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		typeStr := "file"
		if e.IsDir() {
			typeStr = "dir "
		}
		lines = append(lines, fmt.Sprintf("%s  %8d  %s", typeStr, info.Size(), e.Name()))
	}

	if len(lines) == 0 {
		return &tool.Result{Output: "(empty directory)"}, nil
	}

	return &tool.Result{Output: strings.Join(lines, "\n")}, nil
}

func (f *FileList) checkAccess(absPath, originalPath string) error {
	if f.sandbox != nil {
		return f.sandbox.ValidatePath(originalPath)
	}
	if !isPathAllowed(absPath, f.allowedPaths) {
		return fmt.Errorf("access denied: %s is outside allowed paths", originalPath)
	}
	return nil
}

// FileGlob finds files using glob patterns within the workspace sandbox.
type FileGlob struct {
	allowedPaths []string
	sandbox      *security.WorkspaceSandbox
}

// NewFileGlob creates a file glob tool.
func NewFileGlob(allowedPaths []string) *FileGlob {
	return &FileGlob{allowedPaths: allowedPaths}
}

// NewFileGlobSandboxed creates a file glob tool restricted to the workspace sandbox.
func NewFileGlobSandboxed(sandbox *security.WorkspaceSandbox) *FileGlob {
	return &FileGlob{
		allowedPaths: sandbox.AllowedPaths(),
		sandbox:      sandbox,
	}
}

func (f *FileGlob) Name() string { return "file_glob" }
func (f *FileGlob) Description() string {
	return "Find files using a glob pattern (e.g. 'internal/**/*.go'). Only allowed directories are searched."
}
func (f *FileGlob) Sensitive() bool                   { return false }
func (f *FileGlob) ApprovalLevel() tool.ApprovalLevel { return tool.AutoExecute }
func (f *FileGlob) Version() string                   { return "1.0.0" }
func (f *FileGlob) Dependencies() []string            { return nil }
func (f *FileGlob) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "pattern", Type: "string", Description: "The glob pattern to match", Required: true},
	}
}

func (f *FileGlob) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return &tool.Result{Error: "pattern is required"}, nil
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("glob error: %v", err)}, nil
	}

	var allowedMatches []string
	for _, m := range matches {
		abs, err := filepath.Abs(m)
		if err != nil {
			continue
		}
		if f.sandbox != nil {
			if f.sandbox.ValidatePath(m) == nil {
				allowedMatches = append(allowedMatches, m)
			}
		} else if isPathAllowed(abs, f.allowedPaths) {
			allowedMatches = append(allowedMatches, m)
		}
	}

	if len(allowedMatches) == 0 {
		return &tool.Result{Output: "no matches found"}, nil
	}

	return &tool.Result{Output: strings.Join(allowedMatches, "\n")}, nil
}

// isPathAllowed is a shared helper for legacy allowed-paths checking.
func isPathAllowed(absPath string, allowedPaths []string) bool {
	if strings.Contains(absPath, "..") {
		return false
	}
	for _, allowed := range allowedPaths {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, allowedAbs) {
			return true
		}
	}
	return false
}
