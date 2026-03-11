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

type FileEdit struct {
	allowedPaths []string
	sandbox      *security.WorkspaceSandbox
}

func NewFileEdit(allowedPaths []string) *FileEdit {
	return &FileEdit{allowedPaths: allowedPaths}
}

func NewFileEditSandboxed(sandbox *security.WorkspaceSandbox) *FileEdit {
	return &FileEdit{
		allowedPaths: sandbox.AllowedPaths(),
		sandbox:      sandbox,
	}
}

func (f *FileEdit) Name() string { return "file_edit" }
func (f *FileEdit) Description() string {
	return "Edit a file by replacing old_text with new_text. The old_text must exist exactly in the file."
}
func (f *FileEdit) Sensitive() bool        { return true }
func (f *FileEdit) Version() string        { return "1.0.0" }
func (f *FileEdit) Dependencies() []string { return nil }
func (f *FileEdit) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "path", Type: "string", Description: "Path to the file to edit", Required: true},
		{Name: "old_text", Type: "string", Description: "The exact text to find and replace", Required: true},
		{Name: "new_text", Type: "string", Description: "The text to replace with", Required: true},
	}
}

func (f *FileEdit) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	path, _ := args["path"].(string)
	oldText, _ := args["old_text"].(string)
	newText, _ := args["new_text"].(string)

	if path == "" {
		return &tool.Result{Error: "path is required"}, nil
	}
	if oldText == "" {
		return &tool.Result{Error: "old_text is required"}, nil
	}
	if newText == "" {
		return &tool.Result{Error: "new_text is required"}, nil
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

	content := string(data)
	if !strings.Contains(content, oldText) {
		return &tool.Result{Error: "old_text not found in file. Make sure it matches exactly"}, nil
	}

	count := strings.Count(content, oldText)
	if count > 1 {
		return &tool.Result{Error: fmt.Sprintf("old_text appears %d times. Please provide more context to make it unique", count)}, nil
	}

	newContent := strings.Replace(content, oldText, newText, 1)
	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return &tool.Result{Error: fmt.Sprintf("write error: %v", err)}, nil
	}

	return &tool.Result{Output: fmt.Sprintf("File edited: %s", path)}, nil
}

func (f *FileEdit) checkAccess(absPath, originalPath string) error {
	if f.sandbox != nil {
		return f.sandbox.ValidatePath(originalPath)
	}
	if !isPathAllowed(absPath, f.allowedPaths) {
		return fmt.Errorf("access denied: %s is outside allowed paths", originalPath)
	}
	return nil
}

type FileAppend struct {
	allowedPaths []string
	sandbox      *security.WorkspaceSandbox
}

func NewFileAppend(allowedPaths []string) *FileAppend {
	return &FileAppend{allowedPaths: allowedPaths}
}

func NewFileAppendSandboxed(sandbox *security.WorkspaceSandbox) *FileAppend {
	return &FileAppend{
		allowedPaths: sandbox.AllowedPaths(),
		sandbox:      sandbox,
	}
}

func (f *FileAppend) Name() string           { return "file_append" }
func (f *FileAppend) Description() string    { return "Append content to the end of a file" }
func (f *FileAppend) Sensitive() bool        { return true }
func (f *FileAppend) Version() string        { return "1.0.0" }
func (f *FileAppend) Dependencies() []string { return nil }
func (f *FileAppend) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "path", Type: "string", Description: "Path to the file to append to", Required: true},
		{Name: "content", Type: "string", Description: "Content to append", Required: true},
	}
}

func (f *FileAppend) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	if path == "" {
		return &tool.Result{Error: "path is required"}, nil
	}
	if content == "" {
		return &tool.Result{Error: "content is required"}, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("invalid path: %v", err)}, nil
	}

	if err := f.checkAccess(absPath, path); err != nil {
		return &tool.Result{Error: err.Error()}, nil
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &tool.Result{Error: fmt.Sprintf("cannot create directory: %v", err)}, nil
	}

	existing, err := os.ReadFile(absPath)
	if err != nil && !os.IsNotExist(err) {
		return &tool.Result{Error: fmt.Sprintf("read error: %v", err)}, nil
	}

	newContent := append(existing, []byte(content)...)
	if err := os.WriteFile(absPath, newContent, 0644); err != nil {
		return &tool.Result{Error: fmt.Sprintf("write error: %v", err)}, nil
	}

	return &tool.Result{Output: fmt.Sprintf("Appended %d bytes to %s", len(content), path)}, nil
}

func (f *FileAppend) checkAccess(absPath, originalPath string) error {
	if f.sandbox != nil {
		return f.sandbox.ValidatePath(originalPath)
	}
	if !isPathAllowed(absPath, f.allowedPaths) {
		return fmt.Errorf("access denied: %s is outside allowed paths", originalPath)
	}
	return nil
}
