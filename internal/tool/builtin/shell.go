package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/wonderpus/wonderpus/internal/security"
	"github.com/wonderpus/wonderpus/internal/tool"
)

// defaultWhitelist is the set of commands allowed by default.
var defaultWhitelist = map[string]bool{
	"ls": true, "dir": true, "cat": true, "type": true, "echo": true,
	"head": true, "tail": true, "wc": true, "grep": true, "find": true,
	"pwd": true, "date": true, "whoami": true, "hostname": true,
	"tree": true, "sort": true, "uniq": true, "diff": true,
}

// ShellExec executes whitelisted shell commands within the workspace sandbox.
type ShellExec struct {
	whitelist map[string]bool
	sandbox   *security.WorkspaceSandbox
}

// NewShellExec creates a shell executor with the given command whitelist.
// If whitelist is nil, the default whitelist is used.
func NewShellExec(whitelist []string) *ShellExec {
	wl := make(map[string]bool)
	if len(whitelist) == 0 {
		for k, v := range defaultWhitelist {
			wl[k] = v
		}
	} else {
		for _, cmd := range whitelist {
			wl[cmd] = true
		}
	}
	return &ShellExec{whitelist: wl}
}

// NewShellExecSandboxed creates a shell executor restricted to the workspace sandbox.
func NewShellExecSandboxed(whitelist []string, sandbox *security.WorkspaceSandbox) *ShellExec {
	s := NewShellExec(whitelist)
	s.sandbox = sandbox
	return s
}

func (s *ShellExec) Name() string { return "shell_exec" }
func (s *ShellExec) Description() string {
	return "Execute a whitelisted shell command. Only safe, read-only commands are allowed. Requires user approval."
}
func (s *ShellExec) Sensitive() bool { return true }
func (s *ShellExec) Version() string { return "1.0.0" }
func (s *ShellExec) Dependencies() []string { return nil }
func (s *ShellExec) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "command", Type: "string", Description: "The shell command to execute", Required: true},
	}
}

func (s *ShellExec) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return &tool.Result{Error: "command is required"}, nil
	}

	// Parse the command to get the base command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return &tool.Result{Error: "empty command"}, nil
	}

	baseCmd := parts[0]

	// Additional dangerous patterns from Phase 3 requirements
	lower := strings.ToLower(command)
	
	// Exact substring matches
	dangerousSubstrings := []string{
		"rm -rf", "del /f", "rmdir /s",
		"format", "mkfs", "diskpart",
		"dd if=",
		"shutdown", "reboot", "poweroff",
		":(){ :|:& };:", // fork bomb
		"> /dev", ">nul", "> null",
		"sudo", "chmod", "chown",
		"eval", "exec ", "bash -c",
		"powershell", "cmd.exe",
	}

	for _, dangerous := range dangerousSubstrings {
		if strings.Contains(lower, dangerous) {
			return &tool.Result{Error: fmt.Sprintf("blocked: command contains dangerous pattern %q", dangerous)}, nil
		}
	}

	// Regex matches (e.g. for /dev/sd[a-z])
	matched, _ := regexp.MatchString(`/dev/sd[a-z]`, lower)
	if matched {
		return &tool.Result{Error: "blocked: command contains dangerous pattern for direct disk writes"}, nil
	}

	// Check whitelist
	if !s.whitelist[baseCmd] {
		allowed := make([]string, 0, len(s.whitelist))
		for k := range s.whitelist {
			allowed = append(allowed, k)
		}
		return &tool.Result{Error: fmt.Sprintf("command %q is not whitelisted. Allowed: %s", baseCmd, strings.Join(allowed, ", "))}, nil
	}

	// Workspace sandbox: restrict command execution paths
	if s.sandbox != nil {
		if err := s.sandbox.ValidateCommand(command); err != nil {
			return &tool.Result{Error: err.Error()}, nil
		}
	}

	// Execute — use workspace as working directory when sandboxed
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	// Set working directory to workspace when sandbox is active
	if s.sandbox != nil && s.sandbox.IsRestricted() {
		cmd.Dir = s.sandbox.WorkspacePath()
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n[stderr]: " + stderr.String()
	}

	// Truncate large output
	if len(output) > 10000 {
		output = output[:10000] + "\n... (truncated)"
	}

	if err != nil {
		return &tool.Result{Output: output, Error: fmt.Sprintf("command failed: %v", err)}, nil
	}

	return &tool.Result{Output: output}, nil
}
