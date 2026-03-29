package toolsynth

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Detector scans episodic memory for tool gaps.
// It reads recent memory entries and identifies patterns of:
// - Hard gaps: tasks that failed with "no tool available"
// - Soft gaps: tasks that succeeded but took >5x average time (from profiler data)
// - Efficiency gaps: tasks where shell+curl was used instead of a proper tool
type Detector struct {
	dbPath         string
	profilerDBPath string // optional: profiler SQLite for timing-based soft gap detection
	scanLimit      int    // max entries to scan per run (default 500)
	minPriority    float64
	stats          ScanStats
}

// NewDetector creates a detector that reads from the given memory database.
func NewDetector(dbPath string) *Detector {
	return &Detector{
		dbPath:      dbPath,
		scanLimit:   500,
		minPriority: 0.1,
	}
}

// SetProfilerDB configures the profiler database path for timing-based soft gap detection.
func (d *Detector) SetProfilerDB(path string) {
	d.profilerDBPath = path
}

// SetScanLimit overrides the default number of entries scanned per run.
func (d *Detector) SetScanLimit(n int) {
	if n > 0 {
		d.scanLimit = n
	}
}

// Scan reads the last scanLimit entries from episodic memory and detects tool gaps.
// It returns gaps ranked by: frequency × impact × feasibility.
func (d *Detector) Scan() ([]ToolGap, error) {
	start := time.Now()

	db, err := sql.Open("sqlite", d.dbPath)
	if err != nil {
		return nil, fmt.Errorf("detector: open db: %w", err)
	}
	defer db.Close()

	// Read last N messages from episodic memory
	rows, err := db.Query(`
		SELECT role, content, tool_calls, tool_call_id
		FROM messages
		ORDER BY rowid DESC
		LIMIT ?`, d.scanLimit)
	if err != nil {
		return nil, fmt.Errorf("detector: query messages: %w", err)
	}
	defer rows.Close()

	var entries []memoryEntry
	for rows.Next() {
		var e memoryEntry
		var toolCalls, toolCallID sql.NullString
		if err := rows.Scan(&e.Role, &e.Content, &toolCalls, &toolCallID); err != nil {
			continue
		}
		e.ToolCalls = toolCalls.String
		e.ToolCallID = toolCallID.String
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("detector: scan rows: %w", err)
	}

	d.stats.EntriesScanned = len(entries)
	d.stats.ScanDuration = time.Since(start)
	d.stats.LastScan = time.Now()

	// Detect gaps
	gaps := d.detectGaps(entries)
	d.stats.GapsFound = len(gaps)

	slog.Info("detector: scan complete",
		"entries", d.stats.EntriesScanned,
		"gaps", d.stats.GapsFound,
		"duration", d.stats.ScanDuration)

	return gaps, nil
}

// Stats returns statistics from the last scan.
func (d *Detector) Stats() ScanStats {
	return d.stats
}

// memoryEntry represents a single message from the episodic memory store.
type memoryEntry struct {
	Role       string
	Content    string
	ToolCalls  string
	ToolCallID string
}

// gapCandidate is an internal type for accumulating evidence during detection.
type gapCandidate struct {
	name        string
	description string
	gapType     GapType
	evidence    []string
	frequency   int
}

// detectGaps analyzes memory entries to find tool gaps.
func (d *Detector) detectGaps(entries []memoryEntry) []ToolGap {
	candidates := make(map[string]*gapCandidate)

	// Patterns for detecting hard gaps (no tool available)
	hardGapPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)no\s+tool\s+(available|found|exists)\s+(for|to)\s+(.+)`),
		regexp.MustCompile(`(?i)cannot\s+(find|locate|use)\s+(?:a\s+)?tool\s+(?:for|to)\s+(.+)`),
		regexp.MustCompile(`(?i)i\s+(don'?t|do\s+not)\s+have\s+(?:a\s+)?tool\s+(?:for|to)\s+(.+)`),
		regexp.MustCompile(`(?i)no\s+such\s+tool|tool\s+not\s+found|unknown\s+tool`),
	}

	// Patterns for detecting soft gaps (workarounds)
	softGapPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)using\s+shell.*?(?:instead|workaround|alternative)\s+(?:of|for)\s+(.+)`),
		regexp.MustCompile(`(?i)curl\s+.*?(?:because|since|as)\s+(?:there'?s?\s+)?no\s+(.+)`),
	}

	// Patterns for detecting efficiency gaps (shell+curl instead of HTTP tool)
	efficiencyPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)exec.*?curl\s+`),
		regexp.MustCompile(`(?i)shell.*?wget\s+`),
	}

	// Analyze tool response messages for failure indicators
	toolFailurePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)tool\s+execution\s+failed`),
		regexp.MustCompile(`(?i)error\s+executing\s+tool`),
		regexp.MustCompile(`(?i)tool\s+.+?\s+(timed?\s*out|timeout)`),
	}

	for _, entry := range entries {
		content := strings.ToLower(entry.Content)
		_ = content // used for soft gap patterns

		// Check for hard gaps in assistant/tool messages
		if entry.Role == "assistant" || entry.Role == "tool" {
			for _, pat := range hardGapPatterns {
				matches := pat.FindStringSubmatch(entry.Content)
				if len(matches) > 0 {
					var desc string
					if len(matches) > 3 {
						desc = strings.TrimSpace(matches[3])
					} else if len(matches) > 2 {
						desc = strings.TrimSpace(matches[2])
					} else {
						desc = "unknown capability"
					}
					key := normalizeGapKey(desc)
					if _, exists := candidates[key]; !exists {
						candidates[key] = &gapCandidate{
							name:        key,
							description: desc,
							gapType:     HardGap,
						}
					}
					candidates[key].frequency++
					candidates[key].evidence = append(candidates[key].evidence, truncate(entry.Content, 200))
				}
			}

			// Check for soft gaps
			for _, pat := range softGapPatterns {
				if matches := pat.FindStringSubmatch(entry.Content); len(matches) > 1 {
					desc := strings.TrimSpace(matches[1])
					key := normalizeGapKey(desc)
					if _, exists := candidates[key]; !exists {
						candidates[key] = &gapCandidate{
							name:        key,
							description: desc,
							gapType:     SoftGap,
						}
					}
					candidates[key].frequency++
					candidates[key].evidence = append(candidates[key].evidence, truncate(entry.Content, 200))
				}
			}

			// Check for efficiency gaps
			for _, pat := range efficiencyPatterns {
				if pat.MatchString(entry.Content) {
					key := "http_client_workaround"
					if _, exists := candidates[key]; !exists {
						candidates[key] = &gapCandidate{
							name:        key,
							description: "Agent using shell commands for HTTP requests instead of proper HTTP tool",
							gapType:     EfficiencyGap,
						}
					}
					candidates[key].frequency++
					candidates[key].evidence = append(candidates[key].evidence, truncate(entry.Content, 200))
				}
			}

			// Check for tool failures (potential soft gap indicators)
			for _, pat := range toolFailurePatterns {
				if pat.MatchString(content) {
					key := "tool_failure"
					if _, exists := candidates[key]; !exists {
						candidates[key] = &gapCandidate{
							name:        key,
							description: "Tool execution failures detected in episodic memory",
							gapType:     SoftGap,
						}
					}
					candidates[key].frequency++
					candidates[key].evidence = append(candidates[key].evidence, truncate(entry.Content, 200))
				}
			}
		}
	}

	// Soft gap detection from profiler timing data (>5x average time)
	d.detectTimingSoftGaps(candidates)

	// Convert candidates to ToolGaps with priority scoring
	gaps := make([]ToolGap, 0, len(candidates))
	for _, c := range candidates {
		priority := d.calculatePriority(c)
		if priority < d.minPriority {
			continue
		}

		// Deduplicate evidence (keep max 10)
		evidence := dedupStrings(c.evidence)
		if len(evidence) > 10 {
			evidence = evidence[:10]
		}

		gaps = append(gaps, ToolGap{
			Name:               c.name,
			Description:        c.description,
			GapType:            c.gapType,
			Evidence:           evidence,
			Priority:           priority,
			SuggestedInterface: inferInterface(c.name, c.description),
		})
	}

	// Sort by priority descending
	sortGapsByPriority(gaps)

	return gaps
}

// detectTimingSoftGaps reads profiler data and flags tools that take >5x average time.
func (d *Detector) detectTimingSoftGaps(candidates map[string]*gapCandidate) {
	if d.profilerDBPath == "" {
		return
	}

	db, err := sql.Open("sqlite", d.profilerDBPath)
	if err != nil {
		slog.Warn("detector: cannot open profiler DB for timing analysis", "error", err)
		return
	}
	defer db.Close()

	// Get latest snapshot per function
	rows, err := db.Query(`
		SELECT function_name, call_count, total_duration_ns, error_count, success_count
		FROM profiler_snapshots
		WHERE id IN (
			SELECT MAX(id) FROM profiler_snapshots GROUP BY function_name
		)
		AND call_count > 0
	`)
	if err != nil {
		slog.Warn("detector: profiler query failed", "error", err)
		return
	}
	defer rows.Close()

	type toolStats struct {
		name      string
		callCount int64
		totalNs   int64
		errCount  int64
		avgNs     float64
	}

	var allTools []toolStats
	var totalAvgNs float64
	var toolCount int

	for rows.Next() {
		var ts toolStats
		if err := rows.Scan(&ts.name, &ts.callCount, &ts.totalNs, &ts.errCount); err != nil {
			continue
		}
		ts.avgNs = float64(ts.totalNs) / float64(ts.callCount)
		allTools = append(allTools, ts)
		totalAvgNs += ts.avgNs
		toolCount++
	}

	if toolCount == 0 {
		return
	}

	overallAvg := totalAvgNs / float64(toolCount)

	// Flag tools >5x average as soft gaps
	for _, ts := range allTools {
		if ts.avgNs > overallAvg*5 && ts.callCount >= 5 {
			key := normalizeGapKey(ts.name + " too slow")
			if _, exists := candidates[key]; !exists {
				candidates[key] = &gapCandidate{
					name: key,
					description: fmt.Sprintf("Tool %q takes %.0fx average time (%.1fms avg vs %.1fms overall avg)",
						ts.name, ts.avgNs/overallAvg, ts.avgNs/1e6, overallAvg/1e6),
					gapType: SoftGap,
				}
			}
			candidates[key].frequency++
			candidates[key].evidence = append(candidates[key].evidence,
				fmt.Sprintf("Profiler: %s avg %.1fms (%d calls, %d errors)",
					ts.name, ts.avgNs/1e6, ts.callCount, ts.errCount))
		}
	}
}

// calculatePriority scores a gap by: frequency × impact × feasibility.
func (d *Detector) calculatePriority(c *gapCandidate) float64 {
	// Frequency: normalized by scan limit
	frequency := float64(c.frequency) / float64(d.scanLimit)

	// Impact: hard gaps are highest impact, then soft, then efficiency
	var impact float64
	switch c.gapType {
	case HardGap:
		impact = 1.0
	case SoftGap:
		impact = 0.7
	case EfficiencyGap:
		impact = 0.4
	}

	// Feasibility: heuristic based on gap name
	feasibility := 0.8 // default: most tools are feasible to synthesize
	if strings.Contains(c.name, "browser") || strings.Contains(c.name, "gui") {
		feasibility = 0.3 // harder to synthesize
	}
	if strings.Contains(c.name, "crypto") || strings.Contains(c.name, "trading") {
		feasibility = 0.5 // moderate complexity
	}

	raw := frequency * impact * feasibility
	return math.Round(raw*1000) / 1000
}

// normalizeGapKey converts a description into a consistent gap key.
func normalizeGapKey(desc string) string {
	desc = strings.ToLower(desc)
	// Remove articles and filler words
	replacements := []string{"a ", "", "an ", "", "the ", "", "some ", ""}
	for i := 0; i < len(replacements); i += 2 {
		desc = strings.ReplaceAll(desc, replacements[i], replacements[i+1])
	}
	// Replace non-alphanumeric with underscore
	re := regexp.MustCompile(`[^a-z0-9]+`)
	desc = re.ReplaceAllString(desc, "_")
	desc = strings.Trim(desc, "_")
	if desc == "" {
		desc = "unknown_gap"
	}
	return desc
}

// inferInterface suggests a Go interface based on the gap name and description.
func inferInterface(name, description string) string {
	_ = strings.ToLower(description) // used for future heuristic refinement
	name = strings.ToLower(name)

	switch {
	case strings.Contains(name, "pdf"):
		return "func ParsePDF(path string) (string, error)"
	case strings.Contains(name, "excel") || strings.Contains(name, "spreadsheet"):
		return "func ParseExcel(path string) ([]Sheet, error)"
	case strings.Contains(name, "csv"):
		return "func ParseCSV(path string) ([][]string, error)"
	case strings.Contains(name, "json"):
		return "func ParseJSON(path string) (map[string]any, error)"
	case strings.Contains(name, "xml"):
		return "func ParseXML(path string) (map[string]any, error)"
	case strings.Contains(name, "image") || strings.Contains(name, "ocr"):
		return "func ProcessImage(path string) (ImageResult, error)"
	case strings.Contains(name, "http") || strings.Contains(name, "api"):
		return "func CallAPI(method, url string, body []byte) ([]byte, error)"
	case strings.Contains(name, "database") || strings.Contains(name, "sql"):
		return "func QueryDB(connStr, query string) ([]map[string]any, error)"
	case strings.Contains(name, "email") || strings.Contains(name, "mail"):
		return "func SendEmail(to, subject, body string) error"
	default:
		return "func Execute(args map[string]any) (string, error)"
	}
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// dedupStrings removes duplicate strings while preserving order.
func dedupStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// sortGapsByPriority sorts gaps in descending order of priority.
func sortGapsByPriority(gaps []ToolGap) {
	for i := 1; i < len(gaps); i++ {
		for j := i; j > 0 && gaps[j].Priority > gaps[j-1].Priority; j-- {
			gaps[j], gaps[j-1] = gaps[j-1], gaps[j]
		}
	}
}
