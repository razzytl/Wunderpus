package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/wunderpus/wunderpus/internal/security"
	"github.com/wunderpus/wunderpus/internal/tool"
)

// defaultWhitelist is the set of commands allowed by default.
var defaultWhitelist = map[string]bool{
	"ls": true, "dir": true, "cat": true, "type": true, "echo": true,
	"head": true, "tail": true, "wc": true, "grep": true, "find": true,
	"pwd": true, "date": true, "whoami": true, "hostname": true,
	"tree": true, "sort": true, "uniq": true, "diff": true,
}

// defaultDenyPatterns is a comprehensive list of regex patterns for dangerous commands.
// These patterns use word boundaries and are more precise than simple substring matching.
var defaultDenyPatterns = []*regexp.Regexp{
	// Recursive delete with flags
	regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`),
	regexp.MustCompile(`\bdel\s+/[fq]\b`),
	regexp.MustCompile(`\brmdir\s+/s\b`),
	// Disk wiping commands
	regexp.MustCompile(`\b(format|mkfs|diskpart)\b\s`),
	// Direct disk writes
	regexp.MustCompile(`\bdd\s+if=`),
	// Block device writes
	regexp.MustCompile(`>\s*/dev/(sd[a-z]|hd[a-z]|vd[a-z]|xvd[a-z]|nvme\d|mmcblk\d|loop\d|dm-\d|md\d|sr\d|nbd\d)`),
	// System shutdown/reboot
	regexp.MustCompile(`\b(shutdown|reboot|poweroff)\b`),
	// Fork bombs
	regexp.MustCompile(`:\(\)\s*\{.*\};\s*:`),
	// Command substitution
	regexp.MustCompile(`\$\([^)]+\)`),
	regexp.MustCompile(`\$\{[^}]+\}`),
	regexp.MustCompile("`[^`]+`"),
	// Pipe to shell
	regexp.MustCompile(`\|\s*sh\b`),
	regexp.MustCompile(`\|\s*bash\b`),
	// Chained delete
	regexp.MustCompile(`;\s*rm\s+-[rf]`),
	regexp.MustCompile(`&&\s*rm\s+-[rf]`),
	regexp.MustCompile(`\|\|\s*rm\s+-[rf]`),
	// Here-docs
	regexp.MustCompile(`<<\s*EOF`),
	// Command substitution with dangerous commands
	regexp.MustCompile(`\$\(\s*cat\s+`),
	regexp.MustCompile(`\$\(\s*curl\s+`),
	regexp.MustCompile(`\$\(\s*wget\s+`),
	regexp.MustCompile(`\$\(\s*which\s+`),
	// Privilege escalation
	regexp.MustCompile(`\bsudo\b`),
	regexp.MustCompile(`\bchmod\s+[0-7]{3,4}\b`),
	regexp.MustCompile(`\bchown\b`),
	// Process termination
	regexp.MustCompile(`\bpkill\b`),
	regexp.MustCompile(`\bkillall\b`),
	regexp.MustCompile(`\bkill\s+-[9]\b`),
	// Pipe curl/wget to shell (common attack vector)
	regexp.MustCompile(`\bcurl\b.*\|\s*(sh|bash)`),
	regexp.MustCompile(`\bwget\b.*\|\s*(sh|bash)`),
	// Sensitive file reads (direct file reads, not just command substitution)
	regexp.MustCompile(`\bcat\b\s+/etc/(passwd|shadow|group|sudoers|gshadow)`),
	regexp.MustCompile(`\bcat\b\s+/Windows/System32/(config|SAM|system)`),
	regexp.MustCompile(`\btype\b\s+C:\\Windows\\System32\\(config|SAM|system)`),
	// Package managers
	regexp.MustCompile(`\bnpm\s+install\s+-g\b`),
	regexp.MustCompile(`\bpip\s+install\s+--user\b`),
	regexp.MustCompile(`\bapt\s+(install|remove|purge)\b`),
	regexp.MustCompile(`\byum\s+(install|remove)\b`),
	regexp.MustCompile(`\bdnf\s+(install|remove)\b`),
	// Docker
	regexp.MustCompile(`\bdocker\s+run\b`),
	regexp.MustCompile(`\bdocker\s+exec\b`),
	// Git force push
	regexp.MustCompile(`\bgit\s+push\b`),
	regexp.MustCompile(`\bgit\s+force\b`),
	// SSH
	regexp.MustCompile(`\bssh\b.*@`),
	// Eval
	regexp.MustCompile(`\beval\b`),
	// Source shell scripts
	regexp.MustCompile(`\bsource\s+.*\.sh\b`),
}

// absolutePathPattern matches absolute file paths in commands (Unix and Windows).
var absolutePathPattern = regexp.MustCompile(`[A-Za-z]:\\[^\\"']+|/[^\s"']+`)

// safePaths are kernel pseudo-devices that are always safe to reference in commands.
var safePaths = map[string]bool{
	"/dev/null":    true,
	"/dev/zero":    true,
	"/dev/random":  true,
	"/dev/urandom": true,
	"/dev/stdin":   true,
	"/dev/stdout":  true,
	"/dev/stderr":  true,
}

// ShellExec executes whitelisted shell commands within the workspace sandbox.
type ShellExec struct {
	whitelist    map[string]bool
	denyPatterns []*regexp.Regexp
	sandbox      *security.WorkspaceSandbox
}

// NewShellExec creates a shell executor with the given command whitelist.
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
	return &ShellExec{
		whitelist:    wl,
		denyPatterns: defaultDenyPatterns,
	}
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
func (s *ShellExec) Sensitive() bool        { return true }
func (s *ShellExec) Version() string        { return "1.1.0" }
func (s *ShellExec) Dependencies() []string { return nil }
func (s *ShellExec) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "command", Type: "string", Description: "The shell command to execute", Required: true},
	}
}

// isPathAllowed checks if a given path is allowed (either in safe list or within workspace).
func (s *ShellExec) isPathAllowed(path string) bool {
	// Check safe paths
	if safePaths[strings.ToLower(path)] {
		return true
	}
	// If sandboxed, let the sandbox validate
	if s.sandbox != nil && s.sandbox.IsRestricted() {
		return s.sandbox.ValidatePath(path) == nil
	}
	// No sandbox - allow absolute paths (but dangerous ones are caught by deny patterns)
	return true
}

// validatePaths checks all absolute paths in the command against the allowlist.
func (s *ShellExec) validatePaths(command string) error {
	matches := absolutePathPattern.FindAllString(command, -1)
	for _, path := range matches {
		if !s.isPathAllowed(path) {
			return fmt.Errorf("blocked: path %q is not allowed", path)
		}
	}
	return nil
}

// checkDenyPatterns checks the command against all dangerous regex patterns.
func (s *ShellExec) checkDenyPatterns(command string) error {
	for _, pattern := range s.denyPatterns {
		if pattern.MatchString(command) {
			return fmt.Errorf("blocked: command contains dangerous pattern (%s)", pattern.String())
		}
	}
	return nil
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

	// Check against dangerous regex patterns (word-boundary aware)
	if err := s.checkDenyPatterns(command); err != nil {
		return &tool.Result{Error: err.Error()}, nil
	}

	// Validate absolute paths
	if err := s.validatePaths(command); err != nil {
		return &tool.Result{Error: err.Error()}, nil
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
