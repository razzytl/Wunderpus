package tool

import (
	"testing"
)

func TestAnalytics(t *testing.T) {
	analytics := &Analytics{
		stats: make(map[string]*ToolStats),
	}

	_ = analytics // analytics is now initialized
}
