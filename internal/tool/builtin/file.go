package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wonderpus/wonderpus/internal/tool"
)

// FileRead reads a file within allowed paths.
type FileRead struct {
	allowedPaths []string
}

// NewFileRead creates a file read tool with allowed base paths.
func NewFileRead(allowedPaths []string) *FileRead {
	return &FileRead{allowedPaths: allowedPaths}
}

func (f *FileRead) Name() string        { return "file_read" }
func (f *FileRead) Description() string  { return "Read the contents of a file. Only files within allowed directories can be read." }
func (f *FileRead) Sensitive() bool      { return false }
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

	if !f.isAllowed(absPath) {
		return &tool.Result{Error: fmt.Sprintf("access denied: %s is outside allowed paths", path)}, nil
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

func (f *FileRead) isAllowed(absPath string) bool {
	// Block path traversal
	if strings.Contains(absPath, "..") {
		return false
	}

	for _, allowed := range f.allowedPaths {
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

// FileWrite writes content to a file within allowed paths.
type FileWrite struct {
	allowedPaths []string
}

// NewFileWrite creates a file write tool with allowed base paths.
func NewFileWrite(allowedPaths []string) *FileWrite {
	return &FileWrite{allowedPaths: allowedPaths}
}

func (f *FileWrite) Name() string        { return "file_write" }
func (f *FileWrite) Description() string  { return "Write content to a file. Only files within allowed directories can be written. Requires user approval." }
func (f *FileWrite) Sensitive() bool      { return true }
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

	if !f.isAllowed(absPath) {
		return &tool.Result{Error: fmt.Sprintf("access denied: %s is outside allowed paths", path)}, nil
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

func (f *FileWrite) isAllowed(absPath string) bool {
	if strings.Contains(absPath, "..") {
		return false
	}
	for _, allowed := range f.allowedPaths {
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

// FileList lists files in a directory within allowed paths.
type FileList struct {
	allowedPaths []string
}

// NewFileList creates a file list tool with allowed base paths.
func NewFileList(allowedPaths []string) *FileList {
	return &FileList{allowedPaths: allowedPaths}
}

func (f *FileList) Name() string        { return "file_list" }
func (f *FileList) Description() string  { return "List files and directories in a given path. Only allowed directories can be listed." }
func (f *FileList) Sensitive() bool      { return false }
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

	if !f.isAllowed(absPath) {
		return &tool.Result{Error: fmt.Sprintf("access denied: %s is outside allowed paths", path)}, nil
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

func (f *FileList) isAllowed(absPath string) bool {
	if strings.Contains(absPath, "..") {
		return false
	}
	for _, allowed := range f.allowedPaths {
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
