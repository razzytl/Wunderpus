package rsi

import (
	"context"
	"strings"
	"testing"
)

func TestProposalEngine_Propose(t *testing.T) {
	// Mock LLM that returns valid diffs
	mockLLM := func(ctx context.Context, system, user string, temp float64) (string, error) {
		diff := `--- a/internal/test/file.go
+++ b/internal/test/file.go
@@ -5,3 +5,3 @@
 func example() string {
-	return "old"
+	return "new"
 }
`
		return diff, nil
	}

	engine := NewProposalEngine(mockLLM, "test-model")

	entry := WeaknessEntry{
		FunctionNode: FunctionNode{
			Name:          "example",
			QualifiedName: "test.example",
			File:          "internal/test/file.go",
			SourceCode:    `func example() string { return "old" }`,
		},
		WeaknessScore: 0.5,
		PrimaryReason: "error_rate",
		ErrorRate:     0.3,
	}

	proposals, err := engine.Propose(context.Background(), entry)
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}

	// Should have 3 valid proposals
	for i, p := range proposals {
		if p.Diff == "" {
			t.Fatalf("proposal %d has empty diff", i)
		}
		if !strings.Contains(p.Diff, "internal/test/file.go") {
			t.Fatalf("proposal %d diff doesn't target expected file", i)
		}
		if p.Temperature == 0 {
			t.Fatalf("proposal %d has no temperature set", i)
		}
	}
}

func TestProposalEngine_InvalidDiffRejected(t *testing.T) {
	// Mock LLM that returns invalid diffs (targeting cmd/)
	mockLLM := func(ctx context.Context, system, user string, temp float64) (string, error) {
		diff := `--- a/cmd/main.go
+++ b/cmd/main.go
@@ -5,3 +5,3 @@
 func main() {
-	fmt.Println("old")
+	fmt.Println("new")
 }
`
		return diff, nil
	}

	engine := NewProposalEngine(mockLLM, "test-model")

	entry := WeaknessEntry{
		FunctionNode: FunctionNode{
			Name:          "main",
			QualifiedName: "main.main",
			File:          "cmd/main.go",
			SourceCode:    `func main() { fmt.Println("old") }`,
		},
		WeaknessScore: 0.5,
	}

	proposals, err := engine.Propose(context.Background(), entry)
	// Should get an error because all proposals target cmd/ (outside internal/)
	if err == nil {
		t.Fatal("should have errored when all proposals are invalid")
	}

	// All proposals should have empty diffs
	for i, p := range proposals {
		if p.Diff != "" {
			t.Fatalf("proposal %d should have empty diff (rejected), got: %s", i, p.Diff)
		}
	}
}

func TestValidateDiff(t *testing.T) {
	tests := []struct {
		name    string
		diff    string
		wantErr bool
	}{
		{
			name: "valid diff",
			diff: `--- a/internal/test/file.go
+++ b/internal/test/file.go
@@ -1,3 +1,3 @@
-old line
+new line
`,
			wantErr: false,
		},
		{
			name:    "empty diff",
			diff:    "",
			wantErr: true,
		},
		{
			name: "targets cmd/",
			diff: `--- a/cmd/main.go
+++ b/cmd/main.go
@@ -1,1 +1,1 @@
-old
+new
`,
			wantErr: true,
		},
		{
			name: "no hunk markers",
			diff: `--- a/internal/test/file.go
+++ b/internal/test/file.go
some text without @@ markers
`,
			wantErr: true,
		},
		{
			name: "missing headers",
			diff: `@@ -1,3 +1,3 @@
-old
+new
`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDiff(tc.diff)
			if tc.wantErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

func TestCleanDiffResponse(t *testing.T) {
	input := "```diff\n--- a/internal/test.go\n+++ b/internal/test.go\n@@ -1,1 +1,1 @@\n-old\n+new\n```"
	cleaned := cleanDiffResponse(input)

	if strings.Contains(cleaned, "```") {
		t.Fatalf("markdown fences not removed: %s", cleaned)
	}
	if !strings.HasPrefix(cleaned, "---") {
		t.Fatalf("cleaned diff should start with ---, got: %s", cleaned)
	}
}
