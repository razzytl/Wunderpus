package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/wonderpus/wonderpus/internal/tool"
)

// defaultWhitelist is the set of commands allowed by default.
var defaultWhitelist = map[string]bool{
	"ls": true, "dir": true, "cat": true, "type": true, "echo": true,
	"head": true, "tail": true, "wc": true, "grep": true, "find": true,
	"pwd": true, "date": true, "whoami": true, "hostname": true,
	"tree": true, "sort": true, "uniq": true, "diff": true,
}

// ShellExec executes whitelisted shell commands.
type ShellExec struct {
	whitelist map[string]bool
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

func (s *ShellExec) Name() string        { return "shell_exec" }
func (s *ShellExec) Description() string  { return "Execute a whitelisted shell command. Only safe, read-only commands are allowed. Requires user approval." }
func (s *ShellExec) Sensitive() bool      { return true }
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

	// Block dangerous patterns regardless of whitelist
	lower := strings.ToLower(command)
	for _, dangerous := range []string{"rm ", "del ", "rmdir", "format", "mkfs", "> /dev", "sudo", "chmod", "chown"} {
		if strings.Contains(lower, dangerous) {
			return &tool.Result{Error: fmt.Sprintf("blocked: command contains dangerous pattern %q", dangerous)}, nil
		}
	}

	// Check whitelist
	if !s.whitelist[baseCmd] {
		allowed := make([]string, 0, len(s.whitelist))
		for k := range s.whitelist {
			allowed = append(allowed, k)
		}
		return &tool.Result{Error: fmt.Sprintf("command %q is not whitelisted. Allowed: %s", baseCmd, strings.Join(allowed, ", "))}, nil
	}

	// Execute
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
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
