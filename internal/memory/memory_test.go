package memory

import (
	"testing"
	"time"
)

func TestSearchResult_Structure(t *testing.T) {
	sr := SearchResult{
		SessionID: "test-123",
		Content:   "test content",
		Score:     0.95,
	}

	if sr.SessionID != "test-123" {
		t.Errorf("expected SessionID test-123, got %s", sr.SessionID)
	}

	if sr.Content != "test content" {
		t.Errorf("expected Content test content, got %s", sr.Content)
	}

	if sr.Score != 0.95 {
		t.Errorf("expected Score 0.95, got %f", sr.Score)
	}
}

func TestSession_Structure(t *testing.T) {
	s := Session{
		ID:        "test-123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Title:     "Test Session",
	}

	if s.ID != "test-123" {
		t.Errorf("expected ID test-123, got %s", s.ID)
	}

	if s.Title != "Test Session" {
		t.Errorf("expected Title Test Session, got %s", s.Title)
	}
}

func TestStore_Initialization(t *testing.T) {
	store := &Store{}

	if store.db != nil {
		t.Error("expected nil db initially")
	}
}
