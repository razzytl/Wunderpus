package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wunderpus/wunderpus/internal/security"
	"github.com/wunderpus/wunderpus/internal/tool"
)

// FileSender is an interface for sending files back to users
type FileSender interface {
	SendFile(sessionID, filePath, caption string) error
}

// SendFileTool sends files or media to the active chat
type SendFileTool struct {
	fileSender FileSender
	sandbox    *security.WorkspaceSandbox
}

// NewSendFileTool creates a new send_file tool
func NewSendFileTool(fileSender FileSender, sandbox *security.WorkspaceSandbox) *SendFileTool {
	return &SendFileTool{
		fileSender: fileSender,
		sandbox:    sandbox,
	}
}

// Name returns the tool name
func (t *SendFileTool) Name() string {
	return "send_file"
}

// Description returns the tool description
func (t *SendFileTool) Description() string {
	return "Send a file or media attachment to the active chat. Supports images, videos, audio, documents, and other file types. Use file_path to specify the file to send."
}

// Parameters returns the tool parameters
func (t *SendFileTool) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{
			Name:        "file_path",
			Type:        "string",
			Description: "Path to the file to send (relative to workspace or absolute path)",
			Required:    true,
		},
		{
			Name:        "caption",
			Type:        "string",
			Description: "Optional caption to include with the file",
			Required:    false,
		},
		{
			Name:        "session_id",
			Type:        "string",
			Description: "Session ID to send the file to (defaults to current session)",
			Required:    false,
		},
	}
}

// Execute runs the send_file tool
func (t *SendFileTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	// Get file path
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return &tool.Result{Error: "file_path is required"}, nil
	}

	// Get optional caption
	caption := ""
	if c, ok := args["caption"].(string); ok {
		caption = c
	}

	// Get optional session ID
	sessionID := ""
	if s, ok := args["session_id"].(string); ok {
		sessionID = s
	}

	// Resolve the file path - check sandbox first
	var resolvedPath string
	if t.sandbox != nil && t.sandbox.IsRestricted() {
		// Resolve relative to workspace
		workspace := t.sandbox.WorkspacePath()
		if !filepath.IsAbs(filePath) {
			resolvedPath = filepath.Join(workspace, filePath)
		} else {
			resolvedPath = filePath
		}
		// Validate the path is within workspace
		if err := t.sandbox.ValidatePath(resolvedPath); err != nil {
			return &tool.Result{Error: fmt.Sprintf("path validation failed: %v", err)}, nil
		}
	} else {
		resolvedPath = filePath
	}

	// Check if file exists
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &tool.Result{Error: fmt.Sprintf("file not found: %s", filePath)}, nil
		}
		return &tool.Result{Error: fmt.Sprintf("cannot access file: %v", err)}, nil
	}

	if info.IsDir() {
		return &tool.Result{Error: "cannot send a directory, please provide a file path"}, nil
	}

	// Check file size (limit to 50MB for most platforms)
	if info.Size() > 50*1024*1024 {
		return &tool.Result{Error: "file too large (max 50MB)"}, nil
	}

	// If we have a file sender, use it
	if t.fileSender != nil {
		err := t.fileSender.SendFile(sessionID, resolvedPath, caption)
		if err != nil {
			return &tool.Result{Error: fmt.Sprintf("failed to send file: %v", err)}, nil
		}
		return &tool.Result{
			Output: fmt.Sprintf("File sent successfully: %s (%s)", filepath.Base(resolvedPath), formatFileSize(info.Size())),
		}, nil
	}

	// No file sender - return file info for manual handling
	return &tool.Result{
		Output: fmt.Sprintf("File ready to send: %s\nSize: %s\nNote: File sender not configured, please check platform integration.",
			filepath.Base(resolvedPath), formatFileSize(info.Size())),
	}, nil
}

// Sensitive returns whether this tool requires approval
func (t *SendFileTool) Sensitive() bool {
	return true
}

// ApprovalLevel returns the policy-based approval level for this tool.
func (t *SendFileTool) ApprovalLevel() tool.ApprovalLevel { return tool.RequiresApproval }

// Version returns the tool version
func (t *SendFileTool) Version() string {
	return "1.0.0"
}

// Dependencies returns tool dependencies
func (t *SendFileTool) Dependencies() []string {
	return nil
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
