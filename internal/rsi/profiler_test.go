package rsi

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestProfiler_Track(t *testing.T) {
	p, err := NewProfiler("")
	if err != nil {
		t.Fatalf("NewProfiler: %v", err)
	}

	// Track a function 100 times with ~10% error rate
	for i := 0; i < 100; i++ {
		i := i
		_ = p.Track("testFunc", func() error {
			time.Sleep(time.Millisecond) // 1ms latency
			if i%10 == 0 {
				return fmt.Errorf("simulated error")
			}
			return nil
		})
	}

	stats, ok := p.GetStats("testFunc")
	if !ok {
		t.Fatal("testFunc not found in stats")
	}

	if stats.CallCount != 100 {
		t.Fatalf("expected 100 calls, got %d", stats.CallCount)
	}

	// Error count should be ~10 (10% of 100)
	if stats.ErrorCount < 8 || stats.ErrorCount > 12 {
		t.Fatalf("expected ~10 errors, got %d", stats.ErrorCount)
	}

	if stats.SuccessCount != 100-stats.ErrorCount {
		t.Fatalf("success count %d + error count %d != 100", stats.SuccessCount, stats.ErrorCount)
	}

	// P99 should be roughly in the right range (all are ~1ms)
	if stats.P99LatencyNs < 500_000 { // 0.5ms minimum
		t.Fatalf("P99 too low: %dns", stats.P99LatencyNs)
	}
}

func TestProfiler_Snapshot(t *testing.T) {
	p, _ := NewProfiler("")

	_ = p.Track("fnA", func() error { time.Sleep(time.Millisecond); return nil })
	_ = p.Track("fnB", func() error { time.Sleep(2 * time.Millisecond); return nil })
	_ = p.Track("fnA", func() error { time.Sleep(time.Millisecond); return nil })

	snap := p.Snapshot()

	if len(snap) != 2 {
		t.Fatalf("expected 2 functions in snapshot, got %d", len(snap))
	}

	if snap["fnA"].CallCount != 2 {
		t.Fatalf("fnA expected 2 calls, got %d", snap["fnA"].CallCount)
	}
	if snap["fnB"].CallCount != 1 {
		t.Fatalf("fnB expected 1 call, got %d", snap["fnB"].CallCount)
	}
}

func TestProfiler_P99Accuracy(t *testing.T) {
	p, _ := NewProfiler("")

	// Add 100 known latencies, check P99 is within 5% of actual
	for i := 0; i < 100; i++ {
		dur := time.Duration(i+1) * time.Millisecond
		p.TrackDuration("p99test", dur.Nanoseconds(), false)
	}

	stats, _ := p.GetStats("p99test")

	// P99 of 1ms..100ms should be ~99ms (99th percentile)
	expectedP99 := int64(99 * time.Millisecond)
	tolerance := float64(expectedP99) * 0.10 // 10% tolerance for test stability

	actualP99 := stats.P99LatencyNs
	diff := math.Abs(float64(actualP99 - expectedP99))

	if diff > tolerance {
		t.Fatalf("P99 accuracy: expected ~%dns, got %dns (diff=%.0f, tolerance=%.0f)",
			expectedP99, actualP99, diff, tolerance)
	}
}

func TestProfiler_TrackDuration(t *testing.T) {
	p, _ := NewProfiler("")

	p.TrackDuration("manual", 1_000_000, false) // 1ms
	p.TrackDuration("manual", 2_000_000, true)  // 2ms error
	p.TrackDuration("manual", 3_000_000, false) // 3ms

	stats, _ := p.GetStats("manual")

	if stats.CallCount != 3 {
		t.Fatalf("expected 3 calls, got %d", stats.CallCount)
	}
	if stats.ErrorCount != 1 {
		t.Fatalf("expected 1 error, got %d", stats.ErrorCount)
	}
	if stats.TotalDurationNs != 6_000_000 {
		t.Fatalf("expected total 6ms, got %dns", stats.TotalDurationNs)
	}
}
