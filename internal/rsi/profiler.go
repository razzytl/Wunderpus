package rsi

import (
	"database/sql"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SpanStats holds telemetry data for a single tracked function or operation.
type SpanStats struct {
	FunctionName    string
	CallCount       int64
	TotalDurationNs int64
	ErrorCount      int64
	SuccessCount    int64
	P99LatencyNs    int64
	LastSeen        time.Time
}

// ringBuffer is a fixed-size circular buffer for latency samples.
type ringBuffer struct {
	data   []int64
	size   int
	idx    int
	filled bool
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{
		data: make([]int64, size),
		size: size,
	}
}

func (rb *ringBuffer) add(val int64) {
	rb.data[rb.idx] = val
	rb.idx = (rb.idx + 1) % rb.size
	if rb.idx == 0 {
		rb.filled = true
	}
}

func (rb *ringBuffer) p99() int64 {
	count := rb.size
	if !rb.filled {
		count = rb.idx
		if count == 0 {
			return 0
		}
	}

	sorted := make([]int64, count)
	copy(sorted, rb.data[:count])
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := int(math.Ceil(float64(count)*0.99)) - 1
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}

// Profiler wraps agent functions with automatic telemetry collection.
// It tracks call counts, durations, errors, and P99 latency per function.
type Profiler struct {
	spans  map[string]*spanState
	mu     sync.RWMutex
	dbPath string
	db     *sql.DB
}

type spanState struct {
	stats  *SpanStats
	buffer *ringBuffer
	mu     sync.Mutex
}

// NewProfiler creates a new profiler. If dbPath is non-empty, telemetry
// snapshots are persisted to SQLite.
func NewProfiler(dbPath string) (*Profiler, error) {
	p := &Profiler{
		spans:  make(map[string]*spanState),
		dbPath: dbPath,
	}

	if dbPath != "" {
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, err
		}
		_, _ = db.Exec("PRAGMA journal_mode=WAL;")

		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS profiler_snapshots (
				id             INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp      TEXT NOT NULL,
				function_name  TEXT NOT NULL,
				call_count     INTEGER,
				total_duration_ns INTEGER,
				error_count    INTEGER,
				success_count  INTEGER,
				p99_latency_ns INTEGER
			);
			CREATE INDEX IF NOT EXISTS idx_profiler_fn ON profiler_snapshots(function_name);
		`)
		if err != nil {
			db.Close()
			return nil, err
		}
		p.db = db
	}

	return p, nil
}

// Track wraps a function call, recording its duration and error status.
// Usage: err = profiler.Track("myFunc", func() error { return doWork() })
func (p *Profiler) Track(name string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	p.record(name, duration.Nanoseconds(), err != nil)
	return err
}

// TrackDuration records an already-completed operation's duration.
func (p *Profiler) TrackDuration(name string, durationNs int64, hadError bool) {
	p.record(name, durationNs, hadError)
}

func (p *Profiler) record(name string, durationNs int64, hadError bool) {
	p.mu.Lock()
	state, ok := p.spans[name]
	if !ok {
		state = &spanState{
			stats:  &SpanStats{FunctionName: name},
			buffer: newRingBuffer(1000),
		}
		p.spans[name] = state
	}
	p.mu.Unlock()

	state.mu.Lock()
	defer state.mu.Unlock()

	state.stats.CallCount++
	state.stats.TotalDurationNs += durationNs
	state.stats.LastSeen = time.Now()

	if hadError {
		state.stats.ErrorCount++
	} else {
		state.stats.SuccessCount++
	}

	state.buffer.add(durationNs)
	state.stats.P99LatencyNs = state.buffer.p99()
}

// Snapshot returns a copy of current stats for all tracked functions.
func (p *Profiler) Snapshot() map[string]SpanStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]SpanStats, len(p.spans))
	for name, state := range p.spans {
		state.mu.Lock()
		result[name] = *state.stats
		state.mu.Unlock()
	}
	return result
}

// GetStats returns stats for a single function, or false if not tracked.
func (p *Profiler) GetStats(name string) (SpanStats, bool) {
	p.mu.RLock()
	state, ok := p.spans[name]
	p.mu.RUnlock()
	if !ok {
		return SpanStats{}, false
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	return *state.stats, true
}

// PersistSnapshot writes current stats to the SQLite database.
// Called on a background goroutine every 5 minutes.
func (p *Profiler) PersistSnapshot() error {
	if p.db == nil {
		return nil
	}

	snapshot := p.Snapshot()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO profiler_snapshots
		(timestamp, function_name, call_count, total_duration_ns, error_count, success_count, p99_latency_ns)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, stats := range snapshot {
		_, err := stmt.Exec(
			now,
			stats.FunctionName,
			stats.CallCount,
			stats.TotalDurationNs,
			stats.ErrorCount,
			stats.SuccessCount,
			stats.P99LatencyNs,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// StartPersistence begins a background goroutine that persists snapshots
// every interval. Returns a stop function.
func (p *Profiler) StartPersistence(interval time.Duration) func() {
	if p.db == nil {
		return func() {}
	}

	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := p.PersistSnapshot(); err != nil {
					slog.Error("profiler: persist snapshot failed", "error", err)
				}
			case <-stop:
				return
			}
		}
	}()

	return func() { close(stop) }
}

// Close shuts down the profiler and its database connection.
func (p *Profiler) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
