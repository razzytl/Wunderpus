package toolsynth

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// MarketplaceScanner scans external sources (MCP registry, GitHub) for
// existing tool implementations that could replace synthesized tools.
type MarketplaceScanner interface {
	// SearchMCP queries the MCP registry for tools matching the given query.
	SearchMCP(query string) ([]MCPServerResult, error)
	// SearchGitHub searches GitHub for repositories matching the given topic/query.
	SearchGitHub(query string) ([]GitHubRepo, error)
}

// MCPServerResult represents a found MCP server implementation.
type MCPServerResult struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	URL         string  `json:"url"`
	Score       float64 `json:"score"` // relevance score
}

// GitHubRepo represents a found GitHub repository.
type GitHubRepo struct {
	Name        string  `json:"name"`
	FullName    string  `json:"full_name"`
	Description string  `json:"description"`
	URL         string  `json:"url"`
	Stars       int     `json:"stars"`
	Language    string  `json:"language"`
	Score       float64 `json:"score"`
}

// ToolUsage tracks how often a synthesized tool is used.
type ToolUsage struct {
	Name      string    `json:"name"`
	CallCount int64     `json:"call_count"`
	LastUsed  time.Time `json:"last_used"`
	Errors    int64     `json:"errors"`
}

// ImprovementCandidate represents a potential replacement for a synthesized tool.
type ImprovementCandidate struct {
	ToolName  string      `json:"tool_name"`
	Source    string      `json:"source"` // "mcp" or "github"
	Candidate interface{} `json:"candidate"`
	Score     float64     `json:"score"`
	Reason    string      `json:"reason"`
}

// ImprovementLoop manages the continuous improvement of synthesized tools.
// It:
//  1. Tracks tool usage via profiler integration (after 100+ uses, tools become
//     eligible for RSI improvement cycles)
//  2. Scans MCP registry and GitHub for existing implementations
//  3. Auto-PRs to the repo when a strong candidate is found
type ImprovementLoop struct {
	scanner        MarketplaceScanner
	registrar      *Registrar
	profilerDBPath string // optional: profiler SQLite for syncing usage
	usage          map[string]*ToolUsage
	mu             sync.RWMutex
	minUses        int64 // minimum uses before RSI eligibility (default 100)
	scanInterval   time.Duration
	stopCh         chan struct{}
}

// NewImprovementLoop creates a new improvement loop.
func NewImprovementLoop(scanner MarketplaceScanner, registrar *Registrar) *ImprovementLoop {
	return &ImprovementLoop{
		scanner:      scanner,
		registrar:    registrar,
		usage:        make(map[string]*ToolUsage),
		minUses:      100,
		scanInterval: 24 * time.Hour, // daily scan by default
		stopCh:       make(chan struct{}),
	}
}

// SetProfilerDB configures the profiler database path for syncing usage data.
func (l *ImprovementLoop) SetProfilerDB(path string) {
	l.profilerDBPath = path
}

// SetMinUses overrides the minimum use count for RSI eligibility.
func (l *ImprovementLoop) SetMinUses(n int64) {
	if n > 0 {
		l.minUses = n
	}
}

// SetScanInterval overrides the default scan interval.
func (l *ImprovementLoop) SetScanInterval(d time.Duration) {
	if d > 0 {
		l.scanInterval = d
	}
}

// RecordCall records a tool call for usage tracking.
// This should be wired to the profiler's Track() function.
func (l *ImprovementLoop) RecordCall(name string, hadError bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	u, ok := l.usage[name]
	if !ok {
		u = &ToolUsage{Name: name}
		l.usage[name] = u
	}
	u.CallCount++
	u.LastUsed = time.Now()
	if hadError {
		u.Errors++
	}
}

// SyncFromProfiler reads tool usage data from the profiler database
// and updates internal usage tracking. Call this periodically to sync
// profiler data with the improvement loop.
func (l *ImprovementLoop) SyncFromProfiler() error {
	if l.profilerDBPath == "" {
		return nil
	}

	db, err := openProfilerDB(l.profilerDBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT function_name, call_count, error_count
		FROM profiler_snapshots
		WHERE id IN (
			SELECT MAX(id) FROM profiler_snapshots GROUP BY function_name
		)
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	l.mu.Lock()
	defer l.mu.Unlock()

	for rows.Next() {
		var name string
		var callCount, errCount int64
		if err := rows.Scan(&name, &callCount, &errCount); err != nil {
			continue
		}

		u, ok := l.usage[name]
		if !ok {
			u = &ToolUsage{Name: name}
			l.usage[name] = u
		}
		// Use the higher of tracked vs profiler counts
		if callCount > u.CallCount {
			u.CallCount = callCount
			u.Errors = errCount
			u.LastUsed = time.Now()
		}
	}

	return rows.Err()
}

// EligibleForRSI returns synthesized tools that have been used enough
// to be eligible for RSI improvement cycles.
func (l *ImprovementLoop) EligibleForRSI() []ToolUsage {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var eligible []ToolUsage
	for _, u := range l.usage {
		if u.CallCount >= l.minUses && l.registrar.IsSynthesized(u.Name) {
			eligible = append(eligible, *u)
		}
	}
	return eligible
}

// ScanMarketplace scans MCP registry and GitHub for tool implementations
// that could replace or improve existing synthesized tools.
func (l *ImprovementLoop) ScanMarketplace() ([]ImprovementCandidate, error) {
	slog.Info("improvement: scanning marketplace for tool alternatives")

	var candidates []ImprovementCandidate

	// Get all synthesized tools
	synthesized := l.registrar.List()

	for _, tool := range synthesized {
		// Search MCP registry
		if l.scanner != nil {
			mcpResults, err := l.scanner.SearchMCP(tool.Name)
			if err != nil {
				slog.Debug("improvement: MCP search failed", "tool", tool.Name, "error", err)
			} else {
				for _, result := range mcpResults {
					if result.Score > 0.7 {
						candidates = append(candidates, ImprovementCandidate{
							ToolName:  tool.Name,
							Source:    "mcp",
							Candidate: result,
							Score:     result.Score,
							Reason:    fmt.Sprintf("MCP server '%s' provides similar functionality", result.Name),
						})
					}
				}
			}

			// Search GitHub for battle-tested implementations
			searchQuery := fmt.Sprintf("%s go tool", tool.Name)
			ghResults, err := l.scanner.SearchGitHub(searchQuery)
			if err != nil {
				slog.Debug("improvement: GitHub search failed", "tool", tool.Name, "error", err)
			} else {
				for _, repo := range ghResults {
					// Only consider well-starred Go repos
					if repo.Stars >= 100 && strings.EqualFold(repo.Language, "Go") {
						score := float64(repo.Stars) / 10000.0 // normalize
						if score > 1.0 {
							score = 1.0
						}
						candidates = append(candidates, ImprovementCandidate{
							ToolName:  tool.Name,
							Source:    "github",
							Candidate: repo,
							Score:     score,
							Reason:    fmt.Sprintf("GitHub repo '%s' has %d stars — battle-tested", repo.FullName, repo.Stars),
						})
					}
				}
			}
		}
	}

	if len(candidates) > 0 {
		slog.Info("improvement: marketplace scan found candidates", "count", len(candidates))
	}

	return candidates, nil
}

// Start begins the periodic marketplace scan in a background goroutine.
func (l *ImprovementLoop) Start() {
	go l.scanLoop()
	slog.Info("improvement: loop started", "interval", l.scanInterval, "minUses", l.minUses)
}

// Stop stops the background scan loop.
func (l *ImprovementLoop) Stop() {
	close(l.stopCh)
	slog.Info("improvement: loop stopped")
}

// scanLoop runs the periodic marketplace scan.
func (l *ImprovementLoop) scanLoop() {
	ticker := time.NewTicker(l.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			candidates, err := l.ScanMarketplace()
			if err != nil {
				slog.Error("improvement: marketplace scan failed", "error", err)
				continue
			}
			if len(candidates) > 0 {
				slog.Info("improvement: found candidates", "count", len(candidates))
				// Future: auto-PR generation for strong candidates
			}
		case <-l.stopCh:
			return
		}
	}
}

// UsageStats returns usage statistics for all tracked tools.
func (l *ImprovementLoop) UsageStats() map[string]ToolUsage {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := make(map[string]ToolUsage, len(l.usage))
	for k, v := range l.usage {
		stats[k] = *v
	}
	return stats
}

// openProfilerDB opens the profiler's SQLite database for reading usage data.
func openProfilerDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("improvement: open profiler db: %w", err)
	}
	return db, nil
}
