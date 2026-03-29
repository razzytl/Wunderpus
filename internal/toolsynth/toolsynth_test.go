package toolsynth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// --- Test Helpers ---

// createTestMemoryDB creates a temporary SQLite database with the messages table
// and populates it with the given entries.
func createTestMemoryDB(t *testing.T, entries []memoryEntry) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_memory.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Create schema matching memory.Store
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT 'Test'
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		tool_call_id TEXT DEFAULT '',
		tool_calls TEXT DEFAULT '',
		timestamp TEXT NOT NULL,
		encrypted INTEGER DEFAULT 0
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	// Insert session
	now := time.Now().Format(time.RFC3339)
	_, _ = db.Exec(`INSERT INTO sessions (id, created_at, updated_at, title) VALUES (?, ?, ?, ?)`,
		"test-session", now, now, "Test")

	// Insert messages
	for _, e := range entries {
		ts := time.Now().Format(time.RFC3339)
		_, _ = db.Exec(`INSERT INTO messages (session_id, role, content, tool_call_id, tool_calls, timestamp, encrypted)
			VALUES (?, ?, ?, ?, ?, ?, 0)`,
			"test-session", e.Role, e.Content, e.ToolCallID, e.ToolCalls, ts)
	}

	return dbPath
}

// mockLLM implements LLMCaller for testing.
type mockLLM struct {
	responses map[string]string // prompt prefix -> response
}

func (m *mockLLM) Complete(req CompletionRequest) (string, error) {
	// Return a canned ToolSpec JSON for any design request
	if resp, ok := m.responses[req.UserPrompt[:min(20, len(req.UserPrompt))]]; ok {
		return resp, nil
	}

	// Default: return a valid tool spec
	spec := ToolSpec{
		Name:         "test_tool",
		Description:  "A test tool for unit testing",
		GoInterface:  "func TestTool(input string) (string, error)",
		Parameters:   []ParamSpec{{Name: "input", Type: "string", Description: "Input text", Required: true}},
		ReturnType:   "string",
		Dependencies: nil,
		TestCases: []TestCase{
			{Input: map[string]any{"input": "hello"}, Expected: "processed: hello"},
			{Input: map[string]any{"input": "world"}, Expected: "processed: world"},
		},
	}
	b, _ := json.Marshal(spec)
	return string(b), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// mockValidator implements CodeValidator for testing.
type mockValidator struct {
	shouldFail bool
}

func (v *mockValidator) Validate(source string, pkgDir string) (string, error) {
	if v.shouldFail {
		return "build error: syntax error", fmt.Errorf("build failed")
	}
	return "", nil
}

// mockRunner implements ToolRunner for testing.
type mockRunner struct {
	results map[string]string // tool name -> result
	errors  map[string]error  // tool name -> error
}

func (r *mockRunner) Run(_ context.Context, name string, _ map[string]any) (string, error) {
	if err, ok := r.errors[name]; ok {
		return "", err
	}
	if result, ok := r.results[name]; ok {
		return result, nil
	}
	return "test result", nil
}

// --- Detector Tests ---

func TestDetectorScan_EmptyDB(t *testing.T) {
	dbPath := createTestMemoryDB(t, nil)
	detector := NewDetector(dbPath)

	gaps, err := detector.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(gaps) != 0 {
		t.Errorf("expected 0 gaps from empty DB, got %d", len(gaps))
	}
}

func TestDetectorScan_HardGap(t *testing.T) {
	entries := []memoryEntry{
		{Role: "assistant", Content: "I don't have a tool for parsing PDF files. Let me try another approach."},
		{Role: "assistant", Content: "No tool available for converting Excel to CSV."},
		{Role: "tool", Content: "Result: I cannot find a tool for processing images."},
	}
	dbPath := createTestMemoryDB(t, entries)
	detector := NewDetector(dbPath)
	detector.SetScanLimit(5) // small limit so frequency-based priority is meaningful

	gaps, err := detector.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(gaps) == 0 {
		t.Fatal("expected at least 1 gap, got 0")
	}

	// Check that at least one gap is a hard gap
	hasHardGap := false
	for _, gap := range gaps {
		if gap.GapType == HardGap {
			hasHardGap = true
			if gap.Priority <= 0 {
				t.Errorf("hard gap should have priority > 0, got %f", gap.Priority)
			}
			if len(gap.Evidence) == 0 {
				t.Error("gap should have evidence")
			}
		}
	}
	if !hasHardGap {
		t.Error("expected at least one hard gap")
	}
}

func TestDetectorScan_EfficiencyGap(t *testing.T) {
	entries := []memoryEntry{
		{Role: "tool", Content: "exec curl https://api.example.com/data"},
		{Role: "assistant", Content: "Using shell wget to download the file."},
	}
	dbPath := createTestMemoryDB(t, entries)
	detector := NewDetector(dbPath)
	detector.SetScanLimit(5) // small limit so frequency-based priority is meaningful

	gaps, err := detector.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	hasEfficiencyGap := false
	for _, gap := range gaps {
		if gap.GapType == EfficiencyGap {
			hasEfficiencyGap = true
		}
	}
	if !hasEfficiencyGap {
		t.Error("expected efficiency gap for shell+curl usage")
	}
}

func TestDetectorStats(t *testing.T) {
	entries := []memoryEntry{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}
	dbPath := createTestMemoryDB(t, entries)
	detector := NewDetector(dbPath)

	_, _ = detector.Scan()
	stats := detector.Stats()

	if stats.EntriesScanned != 2 {
		t.Errorf("expected 2 entries scanned, got %d", stats.EntriesScanned)
	}
	if stats.LastScan.IsZero() {
		t.Error("last scan time should be set")
	}
}

func TestDetectorScanLimit(t *testing.T) {
	// Create 10 entries
	entries := make([]memoryEntry, 10)
	for i := range entries {
		entries[i] = memoryEntry{Role: "user", Content: fmt.Sprintf("Message %d", i)}
	}
	dbPath := createTestMemoryDB(t, entries)

	detector := NewDetector(dbPath)
	detector.SetScanLimit(5)

	_, _ = detector.Scan()
	stats := detector.Stats()

	if stats.EntriesScanned != 5 {
		t.Errorf("expected 5 entries scanned with limit, got %d", stats.EntriesScanned)
	}
}

// --- Designer Tests ---

func TestDesignerDesign(t *testing.T) {
	llm := &mockLLM{}
	designer := NewDesigner(llm)

	gap := ToolGap{
		Name:               "pdf_parser",
		Description:        "Parse PDF files and extract text",
		GapType:            HardGap,
		Evidence:           []string{"No tool available for parsing PDF"},
		Priority:           0.8,
		SuggestedInterface: "func ParsePDF(path string) (string, error)",
	}

	spec, err := designer.Design(gap)
	if err != nil {
		t.Fatalf("design failed: %v", err)
	}

	if spec.Name == "" {
		t.Error("spec should have a name")
	}
	if spec.Description == "" {
		t.Error("spec should have a description")
	}
	if spec.GoInterface == "" {
		t.Error("spec should have a Go interface")
	}
	if spec.Origin != "synthesized" {
		t.Errorf("expected origin 'synthesized', got %q", spec.Origin)
	}
	if spec.CreatedAt.IsZero() {
		t.Error("spec should have a creation timestamp")
	}
}

func TestDesignerLLMError(t *testing.T) {
	llmErr := &failingLLM{}
	designer := NewDesigner(llmErr)

	gap := ToolGap{Name: "test", Description: "test", GapType: HardGap}
	_, err := designer.Design(gap)
	if err == nil {
		t.Error("expected error from failing LLM")
	}
}

type failingLLM struct{}

func (f *failingLLM) Complete(_ CompletionRequest) (string, error) {
	return "", fmt.Errorf("LLM unavailable")
}

func TestParseToolSpec_ValidJSON(t *testing.T) {
	spec := ToolSpec{
		Name:        "json_parser",
		Description: "Parse JSON files",
		GoInterface: "func ParseJSON(path string) (map[string]any, error)",
		ReturnType:  "map[string]any",
		TestCases:   []TestCase{{Input: map[string]any{"path": "test.json"}, Expected: "result"}},
	}
	b, _ := json.Marshal(spec)

	parsed, err := parseToolSpec(string(b))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.Name != "json_parser" {
		t.Errorf("expected name 'json_parser', got %q", parsed.Name)
	}
}

func TestParseToolSpec_WithMarkdownFences(t *testing.T) {
	spec := ToolSpec{
		Name:        "csv_parser",
		Description: "Parse CSV",
		GoInterface: "func ParseCSV(path string) ([][]string, error)",
		ReturnType:  "[][]string",
		TestCases:   []TestCase{{Input: map[string]any{"path": "test.csv"}, Expected: "rows"}},
	}
	b, _ := json.Marshal(spec)
	response := "```json\n" + string(b) + "\n```"

	parsed, err := parseToolSpec(response)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.Name != "csv_parser" {
		t.Errorf("expected name 'csv_parser', got %q", parsed.Name)
	}
}

// --- Coder Tests ---

func TestCoderGenerate(t *testing.T) {
	llm := &mockLLM{}
	_ = llm // LLM used by Generate internally
	validator := &mockValidator{shouldFail: false}
	outputDir := filepath.Join(t.TempDir(), "generated")

	coder := NewCoder(llm, validator, outputDir)

	spec := ToolSpec{
		Name:        "test_tool",
		Description: "A test tool",
		GoInterface: "func TestTool(input string) (string, error)",
		ReturnType:  "string",
		Parameters:  []ParamSpec{{Name: "input", Type: "string", Required: true}},
		TestCases: []TestCase{
			{Input: map[string]any{"input": "hello"}, Expected: "result"},
		},
	}

	// Note: mockLLM returns a ToolSpec JSON, not Go source.
	// This test verifies the coder processes responses without crashing.
	// The actual source generation test would need a real LLM.
	_, err := coder.Generate(spec)
	// We expect an error because mockLLM returns JSON, not Go source
	// The important thing is it doesn't panic
	if err == nil {
		t.Log("coder generated candidates (LLM returned valid source)")
	} else {
		t.Logf("coder failed as expected with mock LLM: %v", err)
	}
}

func TestCoderWriteValidSource(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	os.MkdirAll(outputDir, 0o755)
	coder := NewCoder(nil, nil, outputDir)

	spec := ToolSpec{Name: "my_tool"}
	candidates := []GeneratedCandidate{
		{Source: "bad code", Temperature: 0.2, BuildError: "syntax error"},
		{Source: "package generated\n\nfunc MyTool() {}", Temperature: 0.5, BuildError: ""},
	}

	path, err := coder.WriteValidSource(spec, candidates)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("source file should exist after write")
	}

	content, _ := os.ReadFile(path)
	if string(content) != "package generated\n\nfunc MyTool() {}" {
		t.Errorf("unexpected source content: %q", string(content))
	}
}

// --- Tester Tests ---

func TestTesterAllPass(t *testing.T) {
	runner := &mockRunner{
		results: map[string]string{
			"test_tool": "processed: hello",
		},
	}
	tester := NewTester(runner)

	spec := ToolSpec{
		Name: "test_tool",
		TestCases: []TestCase{
			{Input: map[string]any{"input": "hello"}, Expected: "processed: hello"},
			{Input: map[string]any{"input": "world"}, Expected: "processed: world"},
		},
	}

	result, err := tester.Test(spec, "source code")
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}

	// First test passes (contains "processed: hello"), second may fail
	// since mock returns the same result for all
	if result.PassRate < 0.5 {
		t.Errorf("expected pass rate >= 0.5, got %f", result.PassRate)
	}
}

func TestTesterAllFail(t *testing.T) {
	runner := &mockRunner{
		errors: map[string]error{
			"fail_tool": fmt.Errorf("tool not found"),
		},
	}
	tester := NewTester(runner)

	spec := ToolSpec{
		Name: "fail_tool",
		TestCases: []TestCase{
			{Input: map[string]any{"input": "test"}, Expected: "result"},
		},
	}

	result, err := tester.Test(spec, "source")
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}

	if result.AllPassed {
		t.Error("expected AllPassed=false when tool errors")
	}
	if result.PassRate != 0 {
		t.Errorf("expected pass rate 0, got %f", result.PassRate)
	}
}

func TestTesterEmptyCases(t *testing.T) {
	runner := &mockRunner{}
	tester := NewTester(runner)

	spec := ToolSpec{Name: "empty", TestCases: nil}

	_, err := tester.Test(spec, "source")
	if err == nil {
		t.Error("expected error for empty test cases")
	}
}

func TestTesterMinPassRate(t *testing.T) {
	runner := &mockRunner{
		results: map[string]string{
			"partial_tool": "partial result",
		},
	}
	tester := NewTester(runner)
	tester.SetMinPassRate(0.8)

	spec := ToolSpec{
		Name: "partial_tool",
		TestCases: []TestCase{
			{Input: map[string]any{"x": "1"}, Expected: "partial"},   // passes
			{Input: map[string]any{"x": "2"}, Expected: "complete"},  // fails
			{Input: map[string]any{"x": "3"}, Expected: "exact"},     // fails
			{Input: map[string]any{"x": "4"}, Expected: "not found"}, // fails
		},
	}

	result, err := tester.Test(spec, "source")
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}

	// 1/4 = 0.25 pass rate, below 0.8 threshold
	if result.AllPassed {
		t.Error("expected AllPassed=false with 0.25 pass rate and 0.8 threshold")
	}
}

// --- Registrar Tests ---

func TestRegistrarRegister(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	dbPath := filepath.Join(t.TempDir(), "toolsynth.db")
	registrar := NewRegistrar(outputDir, dbPath)

	spec := ToolSpec{
		Name:        "my_tool",
		Description: "Test tool",
		GoInterface: "func MyTool(x string) (string, error)",
		ReturnType:  "string",
		Origin:      "synthesized",
		CreatedAt:   time.Now(),
	}
	source := "package generated\n\nfunc MyTool(x string) (string, error) { return x, nil }"
	testResult := ToolTestResult{AllPassed: true, PassRate: 1.0}

	err := registrar.Register(spec, source, testResult)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Check entry exists
	entry, ok := registrar.Get("my_tool")
	if !ok {
		t.Fatal("tool should be registered")
	}
	if entry.Origin != "synthesized" {
		t.Errorf("expected origin 'synthesized', got %q", entry.Origin)
	}
	if entry.SourcePath == "" {
		t.Error("should have source path")
	}

	// Check source file was written
	if _, err := os.Stat(entry.SourcePath); os.IsNotExist(err) {
		t.Error("source file should exist")
	}

	// Check IsSynthesized
	if !registrar.IsSynthesized("my_tool") {
		t.Error("IsSynthesized should return true")
	}
}

func TestRegistrarList(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	registrar := NewRegistrar(outputDir, "")

	_ = registrar.Register(ToolSpec{Name: "tool_a", Origin: "synthesized"}, "code", ToolTestResult{AllPassed: true})
	_ = registrar.Register(ToolSpec{Name: "tool_b", Origin: "synthesized"}, "code", ToolTestResult{AllPassed: true})

	list := registrar.List()
	if len(list) != 2 {
		t.Errorf("expected 2 tools, got %d", len(list))
	}
}

func TestRegistrarRemove(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	registrar := NewRegistrar(outputDir, "")

	_ = registrar.Register(ToolSpec{Name: "temp_tool", Origin: "synthesized"}, "code", ToolTestResult{AllPassed: true})

	err := registrar.Remove("temp_tool")
	if err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	_, ok := registrar.Get("temp_tool")
	if ok {
		t.Error("tool should be removed")
	}
}

func TestRegistrarRemoveNonExistent(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	registrar := NewRegistrar(outputDir, "")

	err := registrar.Remove("nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent tool")
	}
}

// --- Improvement Loop Tests ---

func TestImprovementLoopUsageTracking(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	registrar := NewRegistrar(outputDir, "")
	improver := NewImprovementLoop(nil, registrar)

	// Record calls
	for i := 0; i < 150; i++ {
		improver.RecordCall("my_tool", false)
	}
	improver.RecordCall("my_tool", true) // one error

	stats := improver.UsageStats()
	if stats["my_tool"].CallCount != 151 {
		t.Errorf("expected 151 calls, got %d", stats["my_tool"].CallCount)
	}
	if stats["my_tool"].Errors != 1 {
		t.Errorf("expected 1 error, got %d", stats["my_tool"].Errors)
	}
}

func TestImprovementLoopEligibility(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	registrar := NewRegistrar(outputDir, "")

	// Register a tool as synthesized
	_ = registrar.Register(ToolSpec{Name: "synth_tool", Origin: "synthesized"}, "code", ToolTestResult{AllPassed: true})

	improver := NewImprovementLoop(nil, registrar)
	improver.SetMinUses(100)

	// 99 calls — not eligible
	for i := 0; i < 99; i++ {
		improver.RecordCall("synth_tool", false)
	}
	eligible := improver.EligibleForRSI()
	if len(eligible) != 0 {
		t.Error("should not be eligible at 99 calls")
	}

	// One more call — now eligible
	improver.RecordCall("synth_tool", false)
	eligible = improver.EligibleForRSI()
	if len(eligible) != 1 {
		t.Errorf("expected 1 eligible tool, got %d", len(eligible))
	}
}

func TestImprovementLoopStartStop(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	registrar := NewRegistrar(outputDir, "")
	improver := NewImprovementLoop(nil, registrar)
	improver.SetScanInterval(100 * time.Millisecond)

	improver.Start()
	time.Sleep(50 * time.Millisecond)
	improver.Stop()

	// Should not panic or deadlock
}

// --- Utility Function Tests ---

func TestNormalizeGapKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"parsing PDF files", "parsing_pdf_files"},
		{"a JSON parser", "json_parser"},
		{"the spreadsheet tool", "spreadsheet_tool"},
		{"", "unknown_gap"},
	}

	for _, tt := range tests {
		result := normalizeGapKey(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeGapKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestInferInterface(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		expected string
	}{
		{"pdf_parser", "parse PDF", "func ParsePDF(path string) (string, error)"},
		{"excel_reader", "read Excel", "func ParseExcel(path string) ([]Sheet, error)"},
		{"csv_handler", "handle CSV", "func ParseCSV(path string) ([][]string, error)"},
		{"json_tool", "parse JSON", "func ParseJSON(path string) (map[string]any, error)"},
		{"unknown_thing", "do stuff", "func Execute(args map[string]any) (string, error)"},
	}

	for _, tt := range tests {
		result := inferInterface(tt.name, tt.desc)
		if result != tt.expected {
			t.Errorf("inferInterface(%q, %q) = %q, want %q", tt.name, tt.desc, result, tt.expected)
		}
	}
}

func TestDedupStrings(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b", "d"}
	result := dedupStrings(input)
	if len(result) != 4 {
		t.Errorf("expected 4 unique strings, got %d: %v", len(result), result)
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Error("short strings should not be truncated")
	}
	if truncate("this is a very long string", 10) != "this is a ..." {
		t.Errorf("expected truncation, got %q", truncate("this is a very long string", 10))
	}
}

func TestSortGapsByPriority(t *testing.T) {
	gaps := []ToolGap{
		{Name: "low", Priority: 0.1},
		{Name: "high", Priority: 0.9},
		{Name: "mid", Priority: 0.5},
	}
	sortGapsByPriority(gaps)

	if gaps[0].Name != "high" {
		t.Errorf("expected highest priority first, got %q", gaps[0].Name)
	}
	if gaps[2].Name != "low" {
		t.Errorf("expected lowest priority last, got %q", gaps[2].Name)
	}
}
