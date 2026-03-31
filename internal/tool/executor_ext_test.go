package tool

import (
	"testing"
)

func TestAnalytics(t *testing.T) {
	analytics := &Analytics{
		stats: make(map[string]*ToolStats),
	}

	if analytics.stats == nil {
		t.Error("expected non-nil stats map")
	}
}
