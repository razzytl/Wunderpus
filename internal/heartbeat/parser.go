package heartbeat

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// HeartbeatTask represents a parsed task from HEARTBEAT.md
type HeartbeatTask struct {
	Type    string // "quick" or "long"
	Content string // The task description
	Line    int    // Line number in the source file
}

// HeartbeatConfig holds the heartbeat configuration
type HeartbeatConfig struct {
	Enabled   bool
	Interval  int // minutes
	Workspace string
}

// ParseResult contains the parsed HEARTBEAT.md content
type ParseResult struct {
	QuickTasks   []HeartbeatTask
	LongTasks    []HeartbeatTask
	LastModified time.Time
}

// Parser parses HEARTBEAT.md files
type Parser struct{}

// NewParser creates a new HEARTBEAT.md parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads and parses a HEARTBEAT.md file from the given path
func (p *Parser) Parse(path string) (*ParseResult, error) {
	data, modTime, err := p.readFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	result := &ParseResult{
		LastModified: modTime,
	}

	// Parse the content
	lines := strings.Split(content, "\n")
	currentSection := ""
	taskPattern := regexp.MustCompile(`^-\s+(.+)$`)
	headerPattern := regexp.MustCompile(`^##?\s+(.+)$`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for section headers
		if headerPattern.MatchString(trimmed) {
			match := headerPattern.FindStringSubmatch(trimmed)
			sectionTitle := strings.ToLower(match[1])
			if strings.Contains(sectionTitle, "quick") {
				currentSection = "quick"
			} else if strings.Contains(sectionTitle, "long") {
				currentSection = "long"
			}
			continue
		}

		// Check for task items
		if taskPattern.MatchString(trimmed) && currentSection != "" {
			match := taskPattern.FindStringSubmatch(trimmed)
			task := HeartbeatTask{
				Type:    currentSection,
				Content: strings.TrimSpace(match[1]),
				Line:    i + 1,
			}

			if currentSection == "quick" {
				result.QuickTasks = append(result.QuickTasks, task)
			} else if currentSection == "long" {
				result.LongTasks = append(result.LongTasks, task)
			}
		}
	}

	return result, nil
}

// readFile reads the file and returns its contents and modification time
func (p *Parser) readFile(path string) ([]byte, time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, time.Time{}, err
	}

	return data, stat.ModTime(), nil
}

// FindHeartbeatFile searches for HEARTBEAT.md in the workspace
func FindHeartbeatFile(workspace string) (string, error) {
	// First check workspace root
	path := filepath.Join(workspace, "HEARTBEAT.md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Check common locations
	locations := []string{
		filepath.Join(workspace, ".wunderpus", "HEARTBEAT.md"),
		filepath.Join(workspace, ".wunderpus", "heartbeat.md"),
		filepath.Join(os.Getenv("HOME"), ".wunderpus", "HEARTBEAT.md"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	return "", fmt.Errorf("HEARTBEAT.md not found in workspace: %s", workspace)
}
