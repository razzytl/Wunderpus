package builtin

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
	"strings"

	"github.com/wunderpus/wunderpus/internal/tool"
)

// Calculator evaluates mathematical expressions.
type Calculator struct{}

// NewCalculator creates a new calculator tool.
func NewCalculator() *Calculator {
	return &Calculator{}
}

func (c *Calculator) Name() string { return "calculator" }
func (c *Calculator) Description() string {
	return "Evaluate a mathematical expression. Supports +, -, *, /, parentheses, and common math functions (sqrt, pow, abs, sin, cos, tan, log, log2, log10, ceil, floor, round)."
}
func (c *Calculator) Sensitive() bool                   { return false }
func (c *Calculator) ApprovalLevel() tool.ApprovalLevel { return tool.AutoExecute }
func (c *Calculator) Version() string                   { return "1.0.0" }
func (c *Calculator) Dependencies() []string            { return nil }
func (c *Calculator) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "expression", Type: "string", Description: "The math expression to evaluate, e.g. '2 * (3 + 4)' or 'sqrt(144)'", Required: true},
	}
}

func (c *Calculator) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	expr, _ := args["expression"].(string)
	if expr == "" {
		return &tool.Result{Error: "expression is required"}, nil
	}

	// Try to evaluate using Go's expression parser for safety
	result, err := evalExpr(expr)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("evaluation error: %v", err)}, nil
	}

	// Format nicely
	if result == float64(int64(result)) {
		return &tool.Result{Output: fmt.Sprintf("%d", int64(result))}, nil
	}
	return &tool.Result{Output: fmt.Sprintf("%.6g", result)}, nil
}

// evalExpr evaluates a math expression safely using Go's AST parser.
func evalExpr(expr string) (float64, error) {
	// Pre-process: handle common math functions
	expr = preprocessFunctions(expr)

	// Replace ** with a marker for power
	expr = strings.ReplaceAll(expr, "**", "^")

	// Parse as Go expression
	node, err := parser.ParseExpr(expr)
	if err != nil {
		return 0, fmt.Errorf("parse error: %v", err)
	}

	return evalNode(node)
}

func evalNode(node ast.Expr) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		return strconv.ParseFloat(n.Value, 64)

	case *ast.UnaryExpr:
		val, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.SUB:
			return -val, nil
		case token.ADD:
			return val, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator: %s", n.Op)
		}

	case *ast.BinaryExpr:
		left, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		right, err := evalNode(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		case token.REM:
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			return math.Mod(left, right), nil
		case token.XOR: // We use ^ for power
			return math.Pow(left, right), nil
		default:
			return 0, fmt.Errorf("unsupported operator: %s", n.Op)
		}

	case *ast.ParenExpr:
		return evalNode(n.X)

	case *ast.CallExpr:
		// Handle function calls like sqrt(x)
		fnIdent, ok := n.Fun.(*ast.Ident)
		if !ok {
			return 0, fmt.Errorf("unsupported function call")
		}
		if len(n.Args) < 1 {
			return 0, fmt.Errorf("function %s requires at least 1 argument", fnIdent.Name)
		}
		arg, err := evalNode(n.Args[0])
		if err != nil {
			return 0, err
		}

		switch fnIdent.Name {
		case "sqrt":
			return math.Sqrt(arg), nil
		case "abs":
			return math.Abs(arg), nil
		case "sin":
			return math.Sin(arg), nil
		case "cos":
			return math.Cos(arg), nil
		case "tan":
			return math.Tan(arg), nil
		case "log":
			return math.Log(arg), nil
		case "log2":
			return math.Log2(arg), nil
		case "log10":
			return math.Log10(arg), nil
		case "ceil":
			return math.Ceil(arg), nil
		case "floor":
			return math.Floor(arg), nil
		case "round":
			return math.Round(arg), nil
		case "pow":
			if len(n.Args) < 2 {
				return 0, fmt.Errorf("pow requires 2 arguments")
			}
			arg2, err := evalNode(n.Args[1])
			if err != nil {
				return 0, err
			}
			return math.Pow(arg, arg2), nil
		default:
			return 0, fmt.Errorf("unknown function: %s", fnIdent.Name)
		}

	case *ast.Ident:
		// Handle named constants
		switch n.Name {
		case "pi", "PI":
			return math.Pi, nil
		case "e", "E":
			return math.E, nil
		default:
			return 0, fmt.Errorf("unknown identifier: %s", n.Name)
		}

	default:
		return 0, fmt.Errorf("unsupported expression type")
	}
}

func preprocessFunctions(expr string) string {
	// No-op for now — Go's parser handles function calls natively
	return expr
}
