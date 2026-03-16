package memory

import (
	"testing"
)

// TestSearchResult tests SearchResult structure
func TestSearchResult(t *testing.T) {
	result := SearchResult{
		SessionID: "session-1",
		Content:   "some content",
		Score:     0.85,
	}

	if result.SessionID != "session-1" {
		t.Errorf("expected SessionID 'session-1', got %q", result.SessionID)
	}
	if result.Content != "some content" {
		t.Errorf("expected Content 'some content', got %q", result.Content)
	}
	if result.Score != 0.85 {
		t.Errorf("expected Score 0.85, got %f", result.Score)
	}
}

// TestSearchResult_EmptyContent tests SearchResult with empty content
func TestSearchResult_EmptyContent(t *testing.T) {
	result := SearchResult{
		SessionID: "session-1",
		Content:   "",
		Score:     0.0,
	}

	if result.Content != "" {
		t.Errorf("expected empty Content, got %q", result.Content)
	}
	if result.Score != 0.0 {
		t.Errorf("expected default Score 0.0, got %f", result.Score)
	}
}

// TestSearchResult_DefaultScore tests default score is zero
func TestSearchResult_DefaultScore(t *testing.T) {
	result := SearchResult{}

	if result.Score != 0.0 {
		t.Errorf("expected default Score 0.0, got %f", result.Score)
	}
	if result.SessionID != "" {
		t.Errorf("expected empty SessionID, got %q", result.SessionID)
	}
}

// TestSearchResult_EmptySession tests empty session ID
func TestSearchResult_EmptySession(t *testing.T) {
	result := SearchResult{
		Content: "test",
	}

	if result.SessionID != "" {
		t.Errorf("expected empty SessionID, got %q", result.SessionID)
	}
	if result.Content != "test" {
		t.Errorf("expected Content 'test', got %q", result.Content)
	}
}

// TestSearchResult_HighScore tests high score value
func TestSearchResult_HighScore(t *testing.T) {
	result := SearchResult{
		Score: 1.0,
	}

	if result.Score != 1.0 {
		t.Errorf("expected Score 1.0, got %f", result.Score)
	}
}

// TestSearchResult_NegativeScore tests negative score value
func TestSearchResult_NegativeScore(t *testing.T) {
	result := SearchResult{
		Score: -0.5,
	}

	if result.Score != -0.5 {
		t.Errorf("expected Score -0.5, got %f", result.Score)
	}
}

// TestSearchResults_Slice tests slice of search results
func TestSearchResults_Slice(t *testing.T) {
	results := []SearchResult{
		{SessionID: "s1", Content: "content1", Score: 0.9},
		{SessionID: "s2", Content: "content2", Score: 0.8},
		{SessionID: "s3", Content: "content3", Score: 0.7},
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	if results[0].Score != 0.9 {
		t.Errorf("expected first score 0.9, got %f", results[0].Score)
	}
}
