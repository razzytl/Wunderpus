package rsi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/wunderpus/wunderpus/internal/audit"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"
)

// WeaknessReport contains the results of a weakness analysis cycle.
type WeaknessReport struct {
	GeneratedAt            time.Time       `json:"generated_at"`
	TopCandidates          []WeaknessEntry `json:"top_candidates"`
	TotalFunctionsAnalyzed int             `json:"total_functions_analyzed"`
}

// WeaknessEntry represents a single function identified as weak.
type WeaknessEntry struct {
	FunctionNode  FunctionNode `json:"function_node"`
	WeaknessScore float64      `json:"weakness_score"`
	PrimaryReason string       `json:"primary_reason"`
	ErrorRate     float64      `json:"error_rate"`
	NormalizedP99 float64      `json:"normalized_p99"`
	NormalizedCC  float64      `json:"normalized_cc"`
}

// WeaknessReporter generates weakness reports by combining profiler data
// with the code map to identify functions most in need of improvement.
type WeaknessReporter struct {
	profiler *Profiler
	mapper   *CodeMapper
	db       *sql.DB
	auditLog *audit.AuditLog
	mu       sync.Mutex
	reports  []WeaknessReport // keep last 30
}

// NewWeaknessReporter creates a new weakness reporter.
func NewWeaknessReporter(profiler *Profiler, mapper *CodeMapper) *WeaknessReporter {
	return &WeaknessReporter{
		profiler: profiler,
		mapper:   mapper,
	}
}

// SetDB sets the SQLite database for persisting reports (keeps last 30).
func (w *WeaknessReporter) SetDB(db *sql.DB) {
	w.db = db
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS weakness_reports (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			generated_at TEXT NOT NULL,
			data         TEXT NOT NULL
		)
	`)
}

// SetAuditLog sets the audit log for publishing events.
func (w *WeaknessReporter) SetAuditLog(l *audit.AuditLog) {
	w.auditLog = l
}

// StartScheduler runs Generate() on a background goroutine.
// taskCount channel triggers a cycle on every Nth task completion.
// Returns a stop function.
func (w *WeaknessReporter) StartScheduler(
	taskCount <-chan int,
	everyNTasks int,
	codeMapFn func() *CodeMap,
) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case count := <-taskCount:
				if count > 0 && count%everyNTasks == 0 {
					if cm := codeMapFn(); cm != nil {
						report := w.Generate(cm)
						slog.Info("rsi weakness: report generated (task trigger)",
							"candidates", len(report.TopCandidates))
						w.persist(report)
					}
				}
			case <-ticker.C:
				if cm := codeMapFn(); cm != nil {
					report := w.Generate(cm)
					slog.Info("rsi weakness: report generated (hourly)",
						"candidates", len(report.TopCandidates))
					w.persist(report)
				}
			case <-stop:
				return
			}
		}
	}()
	return func() { close(stop) }
}

func (w *WeaknessReporter) persist(report *WeaknessReport) {
	if w.db == nil {
		return
	}
	data, _ := json.Marshal(report)
	_, _ = w.db.Exec(
		`INSERT INTO weakness_reports (generated_at, data) VALUES (?, ?)`,
		report.GeneratedAt.Format(time.RFC3339Nano), string(data),
	)
	// Prune old entries (keep last 30)
	_, _ = w.db.Exec(
		`DELETE FROM weakness_reports WHERE id NOT IN (SELECT id FROM weakness_reports ORDER BY id DESC LIMIT 30)`,
	)
}

// Generate produces a WeaknessReport by combining SpanStats from the profiler
// with the CodeMap's cyclomatic complexity data.
// Returns the top 10 weakest functions ranked by composite score.
func (w *WeaknessReporter) Generate(codeMap *CodeMap) *WeaknessReport {
	snapshot := w.profiler.Snapshot()

	if len(snapshot) == 0 {
		return &WeaknessReport{
			GeneratedAt:            time.Now().UTC(),
			TotalFunctionsAnalyzed: 0,
		}
	}

	// First pass: collect raw metrics for normalization
	type rawMetrics struct {
		fn         *FunctionNode
		errorRate  float64
		p99Ns      int64
		complexity int
	}

	raw := make([]rawMetrics, 0, len(snapshot))
	var maxP99 float64
	var maxCC float64

	for name, stats := range snapshot {
		fn, exists := codeMap.Functions[name]
		if !exists {
			continue
		}

		var errorRate float64
		if stats.CallCount > 0 {
			errorRate = float64(stats.ErrorCount) / float64(stats.CallCount)
		}

		p99 := float64(stats.P99LatencyNs)
		cc := float64(fn.CyclomaticComp)

		if p99 > maxP99 {
			maxP99 = p99
		}
		if cc > maxCC {
			maxCC = cc
		}

		raw = append(raw, rawMetrics{
			fn:         fn,
			errorRate:  errorRate,
			p99Ns:      stats.P99LatencyNs,
			complexity: fn.CyclomaticComp,
		})
	}

	if len(raw) == 0 {
		return &WeaknessReport{
			GeneratedAt:            time.Now().UTC(),
			TotalFunctionsAnalyzed: len(snapshot),
		}
	}

	// Avoid division by zero in normalization
	if maxP99 == 0 {
		maxP99 = 1
	}
	if maxCC == 0 {
		maxCC = 1
	}

	// Second pass: compute composite weakness scores
	entries := make([]WeaknessEntry, 0, len(raw))
	for _, r := range raw {
		normP99 := float64(r.p99Ns) / maxP99
		normCC := float64(r.complexity) / maxCC

		// Composite: error_rate * 0.5 + normalized_p99 * 0.3 + normalized_complexity * 0.2
		score := (r.errorRate * 0.5) + (normP99 * 0.3) + (normCC * 0.2)

		// Determine primary reason
		reason := "complexity"
		contributions := map[string]float64{
			"error_rate": r.errorRate * 0.5,
			"latency":    normP99 * 0.3,
			"complexity": normCC * 0.2,
		}
		maxContrib := 0.0
		for k, v := range contributions {
			if v > maxContrib {
				maxContrib = v
				reason = k
			}
		}

		entries = append(entries, WeaknessEntry{
			FunctionNode:  *r.fn,
			WeaknessScore: math.Round(score*10000) / 10000,
			PrimaryReason: reason,
			ErrorRate:     math.Round(r.errorRate*10000) / 10000,
			NormalizedP99: math.Round(normP99*10000) / 10000,
			NormalizedCC:  math.Round(normCC*10000) / 10000,
		})
	}

	// Sort by weakness score descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].WeaknessScore > entries[j].WeaknessScore
	})

	// Return top 10
	topN := 10
	if topN > len(entries) {
		topN = len(entries)
	}

	report := &WeaknessReport{
		GeneratedAt:            time.Now().UTC(),
		TopCandidates:          entries[:topN],
		TotalFunctionsAnalyzed: len(raw),
	}

	// Keep last 30 reports
	w.mu.Lock()
	w.reports = append(w.reports, *report)
	if len(w.reports) > 30 {
		w.reports = w.reports[len(w.reports)-30:]
	}
	w.mu.Unlock()

	if w.auditLog != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"functions_analyzed": report.TotalFunctionsAnalyzed,
			"top_candidates":     len(report.TopCandidates),
		})
		_ = w.auditLog.Write(audit.AuditEntry{
			Timestamp: report.GeneratedAt,
			Subsystem: "rsi",
			EventType: audit.EventRSICycleStarted,
			Payload:   payload,
		})
	}

	return report
}

// Reports returns a copy of stored reports.
func (w *WeaknessReporter) Reports() []WeaknessReport {
	w.mu.Lock()
	defer w.mu.Unlock()
	cp := make([]WeaknessReport, len(w.reports))
	copy(cp, w.reports)
	return cp
}

// String returns a human-readable summary of the weakness report.
func (r *WeaknessReport) String() string {
	s := fmt.Sprintf("Weakness Report — %s (%d functions analyzed)\n",
		r.GeneratedAt.Format(time.RFC3339), r.TotalFunctionsAnalyzed)
	for i, e := range r.TopCandidates {
		s += fmt.Sprintf("  %d. %s (score=%.4f, reason=%s, err=%.2f%%, p99=%.4f, cc=%d)\n",
			i+1, e.FunctionNode.QualifiedName, e.WeaknessScore, e.PrimaryReason,
			e.ErrorRate*100, e.NormalizedP99, e.FunctionNode.CyclomaticComp)
	}
	return s
}
