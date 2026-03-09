package builtin

import (
	"context"
	"testing"
)

func TestShellExec_Whitelist(t *testing.T) {
	wl := []string{"echo", "ls", "cat"}
	sh := NewShellExec(wl)

	result, _ := sh.Execute(context.Background(), map[string]any{
		"command": "echo hello",
	})

	if result.Error != "" {
		t.Errorf("expected no error, got %s", result.Error)
	}

	if result.Output == "" {
		t.Error("expected output, got empty")
	}
}

func TestShellExec_Blocked(t *testing.T) {
	wl := []string{"echo"}
	sh := NewShellExec(wl)

	result, _ := sh.Execute(context.Background(), map[string]any{
		"command": "ls /",
	})

	if result.Error == "" {
		t.Error("expected blocked error")
	}
}

func TestShellExec_DangerousPatterns(t *testing.T) {
	wl := []string{"echo", "cat", "ls"}
	sh := NewShellExec(wl)

	dangerous := []string{
		"echo test | sh",
		"echo $(whoami)",
		"cat /etc/passwd",
		"rm -rf /",
	}

	for _, cmd := range dangerous {
		result, _ := sh.Execute(context.Background(), map[string]any{
			"command": cmd,
		})
		if result.Error == "" {
			t.Errorf("expected dangerous command %q to be blocked", cmd)
		}
	}
}

func TestFileRead_Basic(t *testing.T) {
	fr := NewFileRead([]string{"/tmp", "/home"})

	if fr.Name() != "file_read" {
		t.Errorf("expected file_read, got %s", fr.Name())
	}

	if fr.Sensitive() != false {
		t.Error("expected Sensitive() to be false")
	}
}

func TestFileWrite_Basic(t *testing.T) {
	fw := NewFileWrite([]string{"/tmp"})

	if fw.Name() != "file_write" {
		t.Errorf("expected file_write, got %s", fw.Name())
	}

	if fw.Sensitive() != true {
		t.Error("expected Sensitive() to be true")
	}
}

func TestFileList_Basic(t *testing.T) {
	fl := NewFileList([]string{"/tmp"})

	if fl.Name() != "file_list" {
		t.Errorf("expected file_list, got %s", fl.Name())
	}
}

func TestHTTPRequest_Blocked(t *testing.T) {
	http := NewHTTPRequest(nil)

	blocked := []string{
		"http://localhost:8080",
		"http://127.0.0.1",
		"http://169.254.169.254",
		"http://10.0.0.1",
		"http://192.168.1.1",
	}

	for _, url := range blocked {
		result, _ := http.Execute(context.Background(), map[string]any{
			"url": url,
		})
		if result.Error == "" {
			t.Errorf("expected blocked URL %s to have error", url)
		}
	}
}

func TestHTTPRequest_Allowed(t *testing.T) {
	http := NewHTTPRequest(nil)

	result, _ := http.Execute(context.Background(), map[string]any{
		"url": "https://api.openai.com",
	})

	if result.Error != "" {
		t.Errorf("expected allowed URL, got error: %s", result.Error)
	}
}

func TestCalculator_Basic(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		expr     string
		expected string
		hasErr   bool
	}{
		{"1 + 1", "2", false},
		{"10 - 5", "5", false},
		{"3 * 4", "12", false},
		{"10 / 2", "5", false},
		{"10 / 0", "", true},
		{"sqrt(4)", "2", false},
		{"pow(2, 3)", "8", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		res, _ := calc.Execute(context.Background(), map[string]any{
			"expression": tt.expr,
		})

		if tt.hasErr {
			if res.Error == "" {
				t.Errorf("expected error for %s", tt.expr)
			}
		} else {
			if res.Output != tt.expected {
				t.Errorf("expected %s, got %s for %s", tt.expected, res.Output, tt.expr)
			}
		}
	}
}

func TestSpawnTool_Basic(t *testing.T) {
	// Skip - requires subagent.Manager which is complex to mock
}
