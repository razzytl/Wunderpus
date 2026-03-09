package builtin

import (
	"context"
	"testing"
)

func TestFileEditBasic(t *testing.T) {
	edit := NewFileEdit([]string{"/tmp"})

	if edit.Name() != "file_edit" {
		t.Errorf("expected file_edit, got %s", edit.Name())
	}

	if edit.Sensitive() != true {
		t.Error("expected Sensitive() to be true")
	}

	if len(edit.Parameters()) == 0 {
		t.Error("expected Parameters() to be non-empty")
	}
}

func TestFileAppendBasic(t *testing.T) {
	append := NewFileAppend([]string{"/tmp"})

	if append.Name() != "file_append" {
		t.Errorf("expected file_append, got %s", append.Name())
	}

	if append.Sensitive() != true {
		t.Error("expected Sensitive() to be true")
	}
}

func TestWebSearchBasic(t *testing.T) {
	search, err := NewWebSearch(WebSearchConfig{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if search == nil {
		t.Fatal("expected non-nil search")
	}

	if search.Name() != "web_search" {
		t.Errorf("expected web_search, got %s", search.Name())
	}

	result, err := search.Execute(context.Background(), map[string]any{
		"query": "test query",
		"count": 3,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestWebFetchBasic(t *testing.T) {
	fetch, err := NewWebFetch("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if fetch == nil {
		t.Fatal("expected non-nil fetch")
	}

	if fetch.Name() != "web_fetch" {
		t.Errorf("expected web_fetch, got %s", fetch.Name())
	}
}

func TestCronToolBasic(t *testing.T) {
	cron := NewCronTool()

	if cron.Name() != "cron" {
		t.Errorf("expected cron, got %s", cron.Name())
	}

	if len(cron.Parameters()) == 0 {
		t.Error("expected Parameters() to be non-empty")
	}

	result, _ := cron.Execute(context.Background(), map[string]any{
		"action": "list",
	})

	if result == nil {
		t.Error("expected non-nil result")
	}

	result2, _ := cron.Execute(context.Background(), map[string]any{
		"action": "invalid",
	})

	if result2.Error == "" {
		t.Error("expected error for invalid action")
	}
}

func TestSearchFilesBasic(t *testing.T) {
	search := NewSearchFiles([]string{"/tmp"})

	if search.Name() != "content_search" {
		t.Errorf("expected content_search, got %s", search.Name())
	}

	if search.Sensitive() != false {
		t.Error("expected Sensitive() to be false")
	}
}
