package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// MockFileSender is a mock implementation of FileSender
type MockFileSender struct {
	sentFiles []struct {
		sessionID string
		filePath  string
		caption   string
	}
}

func (m *MockFileSender) SendFile(sessionID, filePath, caption string) error {
	m.sentFiles = append(m.sentFiles, struct {
		sessionID string
		filePath  string
		caption   string
	}{sessionID, filePath, caption})
	return nil
}

func TestSendFile_MissingPath(t *testing.T) {
	tool := NewSendFileTool(nil, nil)

	result, _ := tool.Execute(context.Background(), map[string]any{})

	if result.Error == "" {
		t.Error("expected error for missing file_path")
	}
}

func TestSendFile_FileNotFound(t *testing.T) {
	tool := NewSendFileTool(nil, nil)

	result, _ := tool.Execute(context.Background(), map[string]any{
		"file_path": "/nonexistent/file.txt",
	})

	if result.Error == "" {
		t.Error("expected error for nonexistent file")
	}
}

func TestSendFile_Directory(t *testing.T) {
	tool := NewSendFileTool(nil, nil)

	// Create temp directory
	tmpDir := t.TempDir()

	result, _ := tool.Execute(context.Background(), map[string]any{
		"file_path": tmpDir,
	})

	if result.Error == "" {
		t.Error("expected error for directory")
	}
}

func TestSendFile_WithMockSender(t *testing.T) {
	mock := &MockFileSender{}
	tool := NewSendFileTool(mock, nil)

	// Create a temp file
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	err := os.WriteFile(tmpFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	result, _ := tool.Execute(context.Background(), map[string]any{
		"file_path": tmpFile,
		"caption":   "Test file",
	})

	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}

	if len(mock.sentFiles) != 1 {
		t.Errorf("expected 1 sent file, got %d", len(mock.sentFiles))
	}

	if mock.sentFiles[0].caption != "Test file" {
		t.Errorf("expected caption 'Test file', got %q", mock.sentFiles[0].caption)
	}
}

func TestSendFile_Name(t *testing.T) {
	tool := NewSendFileTool(nil, nil)

	if tool.Name() != "send_file" {
		t.Errorf("expected 'send_file', got %s", tool.Name())
	}
}

func TestSendFile_Description(t *testing.T) {
	tool := NewSendFileTool(nil, nil)

	desc := tool.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestSendFile_Parameters(t *testing.T) {
	tool := NewSendFileTool(nil, nil)

	params := tool.Parameters()
	if len(params) == 0 {
		t.Error("expected non-empty parameters")
	}

	// Check required parameters
	var hasFilePath, hasCaption bool
	for _, p := range params {
		if p.Name == "file_path" {
			hasFilePath = true
			if !p.Required {
				t.Error("file_path should be required")
			}
		}
		if p.Name == "caption" {
			hasCaption = true
			if p.Required {
				t.Error("caption should not be required")
			}
		}
	}

	if !hasFilePath {
		t.Error("missing file_path parameter")
	}
	if !hasCaption {
		t.Error("missing caption parameter")
	}
}

func TestSendFile_Sensitive(t *testing.T) {
	tool := NewSendFileTool(nil, nil)

	if !tool.Sensitive() {
		t.Error("send_file should be marked as sensitive")
	}
}

func TestSendFile_Version(t *testing.T) {
	tool := NewSendFileTool(nil, nil)

	version := tool.Version()
	if version == "" {
		t.Error("expected non-empty version")
	}
}

func TestSendFile_Dependencies(t *testing.T) {
	tool := NewSendFileTool(nil, nil)

	deps := tool.Dependencies()
	// nil or empty slice are both valid for no dependencies
	if deps != nil && len(deps) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(deps))
	}
}
