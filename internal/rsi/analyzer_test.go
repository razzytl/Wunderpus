package rsi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestGoFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	fullPath := filepath.Join(dir, filename)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return fullPath
}

func TestCodeMapper_Build(t *testing.T) {
	dir := t.TempDir()

	// Create a test Go file with known functions
	src := `package testpkg

import "fmt"

// SimpleFunction has complexity 1
func SimpleFunction() string {
	return "hello"
}

// ComplexFunction has complexity >= 5
func ComplexFunction(x int) int {
	if x > 10 {
		if x > 20 {
			return x * 2
		}
		for i := 0; i < x; i++ {
			if i%2 == 0 {
				fmt.Println(i)
			}
		}
	}
	switch x {
	case 1:
		return 1
	case 2:
		return 2
	default:
		return x
	}
}

// CallsOther has complexity 1 but calls SimpleFunction
func CallsOther() {
	SimpleFunction()
}
`
	createTestGoFile(t, dir, "test.go", src)

	mapper := NewCodeMapper(false)
	codeMap, err := mapper.Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Check functions were found
	if len(codeMap.Functions) != 3 {
		t.Fatalf("expected 3 functions, got %d", len(codeMap.Functions))
	}

	// Check SimpleFunction
	fn, ok := codeMap.Functions["testpkg.SimpleFunction"]
	if !ok {
		t.Fatal("SimpleFunction not found")
	}
	if fn.StartLine < 5 || fn.StartLine > 7 {
		t.Fatalf("SimpleFunction start line expected 5-7, got %d", fn.StartLine)
	}
	if fn.CyclomaticComp != 1 {
		t.Fatalf("SimpleFunction complexity expected 1, got %d", fn.CyclomaticComp)
	}

	// Check ComplexFunction — should have high complexity
	fn, ok = codeMap.Functions["testpkg.ComplexFunction"]
	if !ok {
		t.Fatal("ComplexFunction not found")
	}
	if fn.CyclomaticComp < 5 {
		t.Fatalf("ComplexFunction complexity expected >= 5, got %d", fn.CyclomaticComp)
	}

	// Check call graph: CallsOther should call SimpleFunction
	callees := codeMap.CallGraph.Callees("testpkg.CallsOther")
	found := false
	for _, c := range callees {
		if strings.Contains(c, "SimpleFunction") {
			found = true
		}
	}
	if !found {
		t.Fatalf("CallsOther should call SimpleFunction, got callees: %v", callees)
	}
}

func TestCodeMapper_ComplexityCounting(t *testing.T) {
	dir := t.TempDir()

	// File with a function that should have complexity 15
	src := `package testpkg

func VeryComplex(x, y, z int) int {
	if x > 0 {
		if y > 0 {
			if z > 0 {
				for i := 0; i < x; i++ {
					switch i {
					case 1:
					case 2:
					case 3:
					}
				}
			}
		}
	}
	if x > 0 && y > 0 {
		return 1
	}
	if x > 0 || y > 0 {
		return 2
	}
	for j := 0; j < 10; j++ {
		_ = j
	}
	return 0
}
`
	createTestGoFile(t, dir, "complex.go", src)

	mapper := NewCodeMapper(false)
	codeMap, err := mapper.Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	fn, ok := codeMap.Functions["testpkg.VeryComplex"]
	if !ok {
		t.Fatal("VeryComplex not found")
	}

	// Count expected: 1(base) + 4(if) + 1(for) + 3(case) + 1(LAND) + 1(LOR) + 1(for) = 12
	if fn.CyclomaticComp < 10 {
		t.Fatalf("VeryComplex expected complexity >= 10, got %d", fn.CyclomaticComp)
	}
	t.Logf("VeryComplex cyclomatic complexity: %d", fn.CyclomaticComp)
}

func TestCodeMapper_Firewall(t *testing.T) {
	mapper := NewCodeMapper(true)

	// Should reject path outside internal/
	_, err := mapper.Build("/tmp")
	if err == nil {
		t.Fatal("should reject path outside internal/")
	}
	if !strings.Contains(err.Error(), "firewall") {
		t.Fatalf("expected firewall error, got: %v", err)
	}
}

func TestCodeMapper_Diff(t *testing.T) {
	dir := t.TempDir()

	src := `package testpkg
func Foo() string { return "foo" }
`
	filePath := createTestGoFile(t, dir, "test.go", src)

	mapper := NewCodeMapper(false)
	before, _ := mapper.Build(dir)

	// Modify the file
	modified := `package testpkg
func Foo() string { return "modified foo" }
func Bar() int { return 42 }
`
	os.WriteFile(filePath, []byte(modified), 0644)
	after, _ := mapper.Build(dir)

	changed := mapper.Diff(before, after)

	// Should detect Foo as changed and Bar as new
	if len(changed) != 2 {
		t.Fatalf("expected 2 changed functions, got %d: %v", len(changed), changed)
	}
}
