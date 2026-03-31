package rsi

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CodeMap is a semantic map of a Go codebase produced by the CodeMapper.
type CodeMap struct {
	Packages  map[string]*PackageNode
	Functions map[string]*FunctionNode
	CallGraph *DirectedGraph
	RootPath  string
}

// PackageNode represents a Go package in the code map.
type PackageNode struct {
	Name      string
	Path      string
	Files     []string
	Functions []string // fully-qualified names
}

// FunctionNode contains everything the RSI engine needs to reason about a function.
type FunctionNode struct {
	Name            string    `json:"name"`
	QualifiedName   string    `json:"qualified_name"`
	File            string    `json:"file"`
	Package         string    `json:"package"`
	StartLine       int       `json:"start_line"`
	EndLine         int       `json:"end_line"`
	CyclomaticComp  int       `json:"cyclomatic_complexity"`
	Dependencies    []string  `json:"dependencies"`
	SourceCode      string    `json:"source_code"`
	EmbeddingVector []float64 `json:"embedding_vector"`
}

// DirectedGraph represents call relationships between functions.
type DirectedGraph struct {
	adjacency map[string][]string // caller -> callees
	mu        sync.RWMutex
}

// NewDirectedGraph creates a new empty directed graph.
func NewDirectedGraph() *DirectedGraph {
	return &DirectedGraph{
		adjacency: make(map[string][]string),
	}
}

// AddEdge records that `from` calls `to`.
func (g *DirectedGraph) AddEdge(from, to string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.adjacency[from] = append(g.adjacency[from], to)
}

// Dependents returns all functions that call `fn` (reverse edges).
func (g *DirectedGraph) Dependents(fn string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []string
	for caller, callees := range g.adjacency {
		for _, callee := range callees {
			if callee == fn {
				result = append(result, caller)
				break
			}
		}
	}
	return result
}

// Callees returns all functions called by `fn`.
func (g *DirectedGraph) Callees(fn string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.adjacency[fn]
}

// ChangedFunction represents a function that changed between two CodeMap snapshots.
type ChangedFunction struct {
	FunctionNode
	PrevCyclomaticComp int
}

// CodeMapper builds a semantic map of Go source code using go/ast.
type CodeMapper struct {
	firewallEnabled bool
}

// NewCodeMapper creates a CodeMapper. If firewallEnabled is true, Build()
// will reject paths outside internal/.
func NewCodeMapper(firewallEnabled bool) *CodeMapper {
	return &CodeMapper{firewallEnabled: firewallEnabled}
}

// Build walks all .go files under rootPath, parses them, extracts function
// declarations, builds a call graph, and calculates cyclomatic complexity.
func (m *CodeMapper) Build(rootPath string) (*CodeMap, error) {
	// RSI Firewall: reject paths outside internal/
	if m.firewallEnabled {
		absRoot, err := filepath.Abs(rootPath)
		if err != nil {
			return nil, fmt.Errorf("rsi: cannot resolve path: %w", err)
		}
		// Must be under internal/ directory
		if !strings.Contains(absRoot, string(filepath.Separator)+"internal"+string(filepath.Separator)) &&
			!strings.HasSuffix(absRoot, string(filepath.Separator)+"internal") {
			return nil, fmt.Errorf("rsi firewall: path %s is outside internal/ — blocked", absRoot)
		}
	}

	codeMap := &CodeMap{
		Packages:  make(map[string]*PackageNode),
		Functions: make(map[string]*FunctionNode),
		CallGraph: NewDirectedGraph(),
		RootPath:  rootPath,
	}

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip test files for code analysis
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		return m.parseFile(path, codeMap)
	})

	return codeMap, err
}

func (m *CodeMapper) parseFile(filePath string, codeMap *CodeMap) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("rsi: parsing %s: %w", filePath, err)
	}

	pkgName := file.Name.Name
	pkgPath := filepath.Dir(filePath)

	// Ensure package node exists
	pkgKey := pkgPath
	if _, exists := codeMap.Packages[pkgKey]; !exists {
		codeMap.Packages[pkgKey] = &PackageNode{
			Name: pkgName,
			Path: pkgPath,
		}
	}
	codeMap.Packages[pkgKey].Files = append(codeMap.Packages[pkgKey].Files, filePath)

	// Read source file for extracting function source
	srcBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("rsi: reading %s: %w", filePath, err)
	}
	srcLines := strings.Split(string(srcBytes), "\n")

	// Track current receiver type for method qualification
	var currentReceiver string

	// Extract all function declarations
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			fn := m.extractFunction(node, fset, filePath, pkgName, srcLines)
			codeMap.Functions[fn.QualifiedName] = fn
			codeMap.Packages[pkgKey].Functions = append(codeMap.Packages[pkgKey].Functions, fn.QualifiedName)

			// Track receiver for method calls
			if node.Recv != nil && len(node.Recv.List) > 0 {
				currentReceiver = extractReceiverType(node.Recv.List[0].Type)
			}

			// Build call graph: find all function calls this function makes
			ast.Inspect(node.Body, func(callNode ast.Node) bool {
				if call, ok := callNode.(*ast.CallExpr); ok {
					callee := resolveCallName(call, pkgName, currentReceiver)
					if callee != "" {
						fn.Dependencies = append(fn.Dependencies, callee)
						codeMap.CallGraph.AddEdge(fn.QualifiedName, callee)
					}
				}
				return true
			})

			currentReceiver = ""
		}
		return true
	})

	return nil
}

func (m *CodeMapper) extractFunction(node *ast.FuncDecl, fset *token.FileSet, filePath, pkgName string, srcLines []string) *FunctionNode {
	startPos := fset.Position(node.Pos())
	endPos := fset.Position(node.End())

	// Compute qualified name
	var qualifiedName string
	if node.Recv != nil && len(node.Recv.List) > 0 {
		recv := extractReceiverType(node.Recv.List[0].Type)
		qualifiedName = fmt.Sprintf("%s.%s.%s", pkgName, recv, node.Name.Name)
	} else {
		qualifiedName = fmt.Sprintf("%s.%s", pkgName, node.Name.Name)
	}

	// Extract source code
	var source string
	startLine := startPos.Line - 1
	endLine := endPos.Line - 1
	if startLine >= 0 && endLine < len(srcLines) {
		source = strings.Join(srcLines[startLine:endLine+1], "\n")
	}

	// Calculate cyclomatic complexity
	complexity := calculateCyclomaticComplexity(node)

	return &FunctionNode{
		Name:           node.Name.Name,
		QualifiedName:  qualifiedName,
		File:           filePath,
		Package:        pkgName,
		StartLine:      startPos.Line,
		EndLine:        endPos.Line,
		CyclomaticComp: complexity,
		SourceCode:     source,
	}
}

// calculateCyclomaticComplexity counts decision points in a function.
// Complexity = 1 + number of (if, for, switch, case, &&, ||, range)
func calculateCyclomaticComplexity(node *ast.FuncDecl) int {
	complexity := 1
	if node.Body == nil {
		return complexity
	}

	ast.Inspect(node.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		case *ast.BinaryExpr:
			if n.Op == token.LAND || n.Op == token.LOR {
				complexity++
			}
		case *ast.TypeSwitchStmt:
			complexity++
		}
		return true
	})

	return complexity
}

// resolveCallName attempts to determine the fully-qualified name of a function call.
func resolveCallName(call *ast.CallExpr, currentPkg, currentReceiver string) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		// Local function call: myFunc()
		return fmt.Sprintf("%s.%s", currentPkg, fn.Name)
	case *ast.SelectorExpr:
		if ident, ok := fn.X.(*ast.Ident); ok {
			// Package-level call: pkg.Func() or method call: obj.Method()
			return fmt.Sprintf("%s.%s", ident.Name, fn.Sel.Name)
		}
		// Method on complex expression: (expr).Method()
		return fn.Sel.Name
	}
	return ""
}

func extractReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return "unknown"
}

// Diff detects which functions changed between two CodeMap snapshots.
func (m *CodeMapper) Diff(before, after *CodeMap) []ChangedFunction {
	var changed []ChangedFunction

	for qname, afterFn := range after.Functions {
		beforeFn, exists := before.Functions[qname]
		if !exists {
			// New function
			changed = append(changed, ChangedFunction{
				FunctionNode: *afterFn,
			})
			continue
		}

		// Check if function changed
		if beforeFn.SourceCode != afterFn.SourceCode ||
			beforeFn.CyclomaticComp != afterFn.CyclomaticComp ||
			beforeFn.StartLine != afterFn.StartLine ||
			beforeFn.EndLine != afterFn.EndLine {
			changed = append(changed, ChangedFunction{
				FunctionNode:       *afterFn,
				PrevCyclomaticComp: beforeFn.CyclomaticComp,
			})
		}
	}

	return changed
}
