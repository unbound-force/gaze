// Package analysis provides the core side effect detection engine.
package analysis

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

// AnalyzeSentinels detects package-level sentinel error variables.
// A sentinel error is a package-level var whose name starts with
// "Err" (or "err" for unexported) and is initialized with
// errors.New(...) or fmt.Errorf("...%w...").
func AnalyzeSentinels(
	fset *token.FileSet,
	file *ast.File,
	pkg string,
) []taxonomy.SideEffect {
	var effects []taxonomy.SideEffect

	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}

		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, name := range vs.Names {
				if !isSentinelName(name.Name) {
					continue
				}
				if i >= len(vs.Values) {
					continue
				}

				call, ok := vs.Values[i].(*ast.CallExpr)
				if !ok {
					continue
				}

				wraps := false
				isSentinel := false

				switch {
				case isErrorsNewCall(call):
					isSentinel = true
				case isFmtErrorfCall(call):
					isSentinel = true
					wraps = hasWrapVerb(call)
				}

				if !isSentinel {
					continue
				}

				loc := fset.Position(name.Pos()).String()
				desc := "package-level sentinel error '" + name.Name + "'"
				if wraps {
					desc += " (wraps via %w)"
				}

				effects = append(effects, taxonomy.SideEffect{
					ID:          taxonomy.GenerateID(pkg, "", string(taxonomy.SentinelError), loc),
					Type:        taxonomy.SentinelError,
					Tier:        taxonomy.TierP0,
					Location:    loc,
					Description: desc,
					Target:      name.Name,
				})
			}
		}
	}

	return effects
}

// isSentinelName returns true if the name follows Go sentinel error
// naming conventions: starts with "Err" (exported) or "err" (unexported).
func isSentinelName(name string) bool {
	return strings.HasPrefix(name, "Err") || strings.HasPrefix(name, "err")
}

// isErrorsNewCall checks if a call is errors.New(...).
func isErrorsNewCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkg.Name == "errors" && sel.Sel.Name == "New"
}

// isFmtErrorfCall checks if a call is fmt.Errorf(...).
func isFmtErrorfCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkg.Name == "fmt" && sel.Sel.Name == "Errorf"
}

// hasWrapVerb checks if a fmt.Errorf call contains %w in its format
// string.
func hasWrapVerb(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok {
		return false
	}
	return strings.Contains(lit.Value, "%w")
}
