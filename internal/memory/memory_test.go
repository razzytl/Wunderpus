package memory

import (
	"os"
	"testing"

	"github.com/wunderpus/wunderpus/internal/provider"
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
		ID:    "test-123",
		Title: "Test Session",
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

func TestNewStore(t *testing.T) {
	tmpFile := t.TempDir() + "/test_memory.db"
	defer os.Remove(tmpFile)

	store, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("Store should not be nil")
	}

	if store.db == nil {
		t.Error("Database should not be nil")
	}
}

func TestEnsureSession(t *testing.T) {
	tmpFile := t.TempDir() + "/test_ensure.db"
	defer os.Remove(tmpFile)

	store, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create a new session
	err = store.EnsureSession("test-session-1")
	if err != nil {
		t.Errorf("EnsureSession failed: %v", err)
	}

	// Creating same session should not error (INSERT OR IGNORE)
	err = store.EnsureSession("test-session-1")
	if err != nil {
		t.Errorf("EnsureSession duplicate failed: %v", err)
	}
}

func TestSaveAndLoadMessage(t *testing.T) {
	tmpFile := t.TempDir() + "/test_msg.db"
	defer os.Remove(tmpFile)

	store, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	sessionID := "test-session-msg"
	msg := provider.Message{
		Role:    "user",
		Content: "Hello, world!",
	}

	// Save message
	err = store.SaveMessage(sessionID, msg, nil)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// Load messages
	messages, err := store.LoadSession(sessionID, nil)
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got %s", messages[0].Role)
	}

	if messages[0].Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %s", messages[0].Content)
	}
}

func TestSaveMultipleMessages(t *testing.T) {
	tmpFile := t.TempDir() + "/test_multi_msg.db"
	defer os.Remove(tmpFile)

	store, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	sessionID := "test-session-multi"

	// Save multiple messages
	messages := []provider.Message{
		{Role: "user", Content: "First message"},
		{Role: "assistant", Content: "Second message"},
		{Role: "user", Content: "Third message"},
	}

	for _, msg := range messages {
		err = store.SaveMessage(sessionID, msg, nil)
		if err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}
	}

	// Load and verify order
	loaded, err := store.LoadSession(sessionID, nil)
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(loaded))
	}

	// Verify order is preserved
	if loaded[0].Content != "First message" {
		t.Errorf("Expected 'First message', got %s", loaded[0].Content)
	}
	if loaded[1].Content != "Second message" {
		t.Errorf("Expected 'Second message', got %s", loaded[1].Content)
	}
	if loaded[2].Content != "Third message" {
		t.Errorf("Expected 'Third message', got %s", loaded[2].Content)
	}
}

func TestLoadNonExistentSession(t *testing.T) {
	tmpFile := t.TempDir() + "/test_nonexistent.db"
	defer os.Remove(tmpFile)

	store, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	messages, err := store.LoadSession("non-existent-session", nil)
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages for non-existent session, got %d", len(messages))
	}
}

func TestPreferences(t *testing.T) {
	tmpFile := t.TempDir() + "/test_prefs.db"
	defer os.Remove(tmpFile)

	store, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Set preference
	err = store.SetPreference("theme", "dark")
	if err != nil {
		t.Fatalf("SetPreference failed: %v", err)
	}

	// Get preference
	val := store.GetPreference("theme", "light")
	if val != "dark" {
		t.Errorf("Expected 'dark', got %s", val)
	}

	// Get non-existent preference with default
	val = store.GetPreference("nonexistent", "default-value")
	if val != "default-value" {
		t.Errorf("Expected 'default-value', got %s", val)
	}
}

func TestClose(t *testing.T) {
	tmpFile := t.TempDir() + "/test_close.db"

	store, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Close should not error
	err = store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Double close should not panic
	err = store.Close()
	if err != nil {
		t.Errorf("Double close failed: %v", err)
	}
}
