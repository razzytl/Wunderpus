package heartbeat

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// HeartbeatTask represents a parsed task from HEARTBEAT.md
type HeartbeatTask struct {
	Type     string // "quick" or "long"
	Content  string // The task description
	Line     int    // Line number in the source file
	CronExpr string // Cron expression (e.g., "0 9 * * *")
	Schedule string // Original schedule string (e.g., "daily at 9am")
}

// DefaultIntervals returns default intervals for common schedule patterns
var DefaultIntervals = map[string]string{
	"hourly":  "0 * * * *",
	"daily":   "0 0 * * *",
	"weekly":  "0 0 * * 0",
	"monthly": "0 0 1 * *",
	"yearly":  "0 0 1 1 *",
}

// parseScheduleToCron converts natural language schedule to cron expression
func parseScheduleToCron(schedule string) (string, bool) {
	schedule = strings.ToLower(strings.TrimSpace(schedule))

	// Direct mappings for common schedules (check these first)
	if cron, ok := DefaultIntervals[schedule]; ok {
		return cron, true
	}

	// Parse "weekday at X" or "weekdays at X" - check BEFORE general "at" pattern
	weekdayPattern := regexp.MustCompile(`weekdays?\s+at\s+(\d{1,2})(?::(\d{2}))?\s*(am|pm)?`)
	if match := weekdayPattern.FindStringSubmatch(schedule); match != nil {
		hour, _ := strconv.Atoi(match[1])
		minute := 0
		if match[2] != "" {
			minute, _ = strconv.Atoi(match[2])
		}
		period := match[3]

		if period == "pm" && hour < 12 {
			hour += 12
		} else if period == "am" && hour == 12 {
			hour = 0
		}

		// Monday to Friday (1-5)
		return fmt.Sprintf("%d %d * * 1-5", minute, hour), true
	}

	// Parse just "weekdays" (without at) - weekdays at midnight
	weekdaysOnlyPattern := regexp.MustCompile(`^weekdays?$`)
	if weekdaysOnlyPattern.MatchString(schedule) {
		// Weekdays at midnight
		return "0 0 * * 1-5", true
	}

	// Parse "every X" (e.g., "every 5 minutes", "every 2 hours")
	everyPattern := regexp.MustCompile(`every\s+(\d+)\s+(minute|minutes|hour|hours|day|days|week|weeks)`)
	if match := everyPattern.FindStringSubmatch(schedule); match != nil {
		interval, _ := strconv.Atoi(match[1])
		unit := match[2]

		switch unit {
		case "minute", "minutes":
			// Custom interval - not standard cron, use @every
			return fmt.Sprintf("@every %dm", interval), true
		case "hour", "hours":
			return fmt.Sprintf("0 */%d * * *", interval), true
		case "day", "days":
			return fmt.Sprintf("0 0 */%d * *", interval), true
		case "week", "weeks":
			return fmt.Sprintf("0 0 * * %d", (interval*7)%7), true
		}
	}

	// Parse "at X" (e.g., "at 9am", "at 14:30") - check after more specific patterns
	atPattern := regexp.MustCompile(`at\s+(\d{1,2})(?::(\d{2}))?\s*(am|pm)?`)
	if match := atPattern.FindStringSubmatch(schedule); match != nil {
		hour, _ := strconv.Atoi(match[1])
		minute := 0
		if match[2] != "" {
			minute, _ = strconv.Atoi(match[2])
		}
		period := match[3]

		if period == "pm" && hour < 12 {
			hour += 12
		} else if period == "am" && hour == 12 {
			hour = 0
		}

		return fmt.Sprintf("%d %d * * *", minute, hour), true
	}

	return "", false
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
	schedulePattern := regexp.MustCompile(`\[([^\]]+)\]`)

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
			taskContent := strings.TrimSpace(match[1])

			task := HeartbeatTask{
				Type:    currentSection,
				Content: taskContent,
				Line:    i + 1,
			}

			// Check for schedule in brackets [schedule]
			if schedMatch := schedulePattern.FindStringSubmatch(taskContent); schedMatch != nil {
				schedule := schedMatch[1]
				task.Schedule = schedule
				// Try to convert to cron expression
				if cron, ok := parseScheduleToCron(schedule); ok {
					task.CronExpr = cron
				}
				// Remove schedule from content
				task.Content = schedulePattern.ReplaceAllString(taskContent, "")
				task.Content = strings.TrimSpace(task.Content)
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
