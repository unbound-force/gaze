package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// AnalyzeReturns detects return value and error return side effects
// for a function declaration. It inspects the function signature's
// result list to find:
//   - ReturnValue for each non-error return position
//   - ErrorReturn for each error-typed return position
//   - DeferredReturnMutation for named returns modified in defer
func AnalyzeReturns(
	fset *token.FileSet,
	info *types.Info,
	fd *ast.FuncDecl,
	pkg string,
	funcName string,
) []taxonomy.SideEffect {
	if fd.Type.Results == nil || len(fd.Type.Results.List) == 0 {
		return nil
	}

	var effects []taxonomy.SideEffect

	// Collect named return variable names for defer analysis.
	var namedReturns []string

	pos := 0
	for _, field := range fd.Type.Results.List {
		typeStr := types.ExprString(field.Type)
		isError := isErrorType(info, field.Type)

		// A field can declare multiple names: (a, b int)
		names := fieldNames(field)
		count := len(names)
		if count == 0 {
			count = 1
			names = []string{""}
		}

		for i := 0; i < count; i++ {
			name := names[i]
			if name != "" {
				namedReturns = append(namedReturns, name)
			}

			loc := fset.Position(field.Pos()).String()
			desc := formatReturnDesc(typeStr, pos, name)

			if isError {
				effects = append(effects, taxonomy.SideEffect{
					ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.ErrorReturn), loc),
					Type:        taxonomy.ErrorReturn,
					Tier:        taxonomy.TierP0,
					Location:    loc,
					Description: desc,
					Target:      typeStr,
				})
			} else {
				effects = append(effects, taxonomy.SideEffect{
					ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.ReturnValue), loc),
					Type:        taxonomy.ReturnValue,
					Tier:        taxonomy.TierP0,
					Location:    loc,
					Description: desc,
					Target:      typeStr,
				})
			}
			pos++
		}
	}

	// Check for named returns modified in deferred functions.
	if len(namedReturns) > 0 && fd.Body != nil {
		deferred := findDeferredReturnMutations(fd.Body, namedReturns)
		for _, name := range deferred {
			loc := fset.Position(fd.Pos()).String()
			effects = append(effects, taxonomy.SideEffect{
				ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.DeferredReturnMutation), name),
				Type:        taxonomy.DeferredReturnMutation,
				Tier:        taxonomy.TierP1,
				Location:    loc,
				Description: fmt.Sprintf("named return '%s' modified in defer", name),
				Target:      name,
			})
		}
	}

	return effects
}

// isErrorType checks if an AST expression represents the error type.
func isErrorType(info *types.Info, expr ast.Expr) bool {
	if info != nil {
		tv, ok := info.Types[expr]
		if ok {
			return tv.Type.String() == "error"
		}
	}
	// Fallback to AST name matching.
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "error"
	}
	return false
}

// fieldNames returns the names declared in a field, or nil if unnamed.
func fieldNames(field *ast.Field) []string {
	if len(field.Names) == 0 {
		return nil
	}
	names := make([]string, len(field.Names))
	for i, n := range field.Names {
		names[i] = n.Name
	}
	return names
}

// formatReturnDesc builds a human-readable description for a return.
func formatReturnDesc(typeStr string, pos int, name string) string {
	if name != "" {
		return fmt.Sprintf("returns %s at position %d (named: %s)", typeStr, pos, name)
	}
	return fmt.Sprintf("returns %s at position %d", typeStr, pos)
}

// findDeferredReturnMutations walks a function body looking for
// defer statements that assign to any of the named return variables.
// Returns the list of named returns that are modified.
func findDeferredReturnMutations(body *ast.BlockStmt, namedReturns []string) []string {
	nameSet := make(map[string]bool, len(namedReturns))
	for _, n := range namedReturns {
		nameSet[n] = true
	}

	var modified []string
	seen := make(map[string]bool)

	ast.Inspect(body, func(n ast.Node) bool {
		ds, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}
		// Walk the deferred call for assignments to named returns.
		ast.Inspect(ds, func(inner ast.Node) bool {
			assign, ok := inner.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, lhs := range assign.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					if nameSet[ident.Name] && !seen[ident.Name] {
						modified = append(modified, ident.Name)
						seen[ident.Name] = true
					}
				}
			}
			return true
		})
		return true
	})

	return modified
}

// receiverName extracts the receiver type name from a FuncDecl.
// Returns empty string for non-method functions.
func receiverName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	return typeExprString(fd.Recv.List[0].Type)
}

// typeExprString converts a type expression to a string like "*T" or "T".
func typeExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.IndexExpr:
		return typeExprString(t.X) + "[" + typeExprString(t.Index) + "]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// funcSignature returns a readable signature string for a FuncDecl.
func funcSignature(_ *token.FileSet, fd *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func ")
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		b.WriteString("(")
		if len(fd.Recv.List[0].Names) > 0 {
			b.WriteString(fd.Recv.List[0].Names[0].Name)
			b.WriteString(" ")
		}
		b.WriteString(typeExprString(fd.Recv.List[0].Type))
		b.WriteString(") ")
	}
	b.WriteString(fd.Name.Name)
	b.WriteString("(")
	if fd.Type.Params != nil {
		for i, p := range fd.Type.Params.List {
			if i > 0 {
				b.WriteString(", ")
			}
			for j, n := range p.Names {
				if j > 0 {
					b.WriteString(", ")
				}
				b.WriteString(n.Name)
			}
			if len(p.Names) > 0 {
				b.WriteString(" ")
			}
			b.WriteString(types.ExprString(p.Type))
		}
	}
	b.WriteString(")")
	if fd.Type.Results != nil && len(fd.Type.Results.List) > 0 {
		b.WriteString(" ")
		if len(fd.Type.Results.List) > 1 || len(fd.Type.Results.List[0].Names) > 0 {
			b.WriteString("(")
		}
		for i, r := range fd.Type.Results.List {
			if i > 0 {
				b.WriteString(", ")
			}
			for j, n := range r.Names {
				if j > 0 {
					b.WriteString(", ")
				}
				b.WriteString(n.Name)
			}
			if len(r.Names) > 0 {
				b.WriteString(" ")
			}
			b.WriteString(types.ExprString(r.Type))
		}
		if len(fd.Type.Results.List) > 1 || len(fd.Type.Results.List[0].Names) > 0 {
			b.WriteString(")")
		}
	}
	return b.String()
}
