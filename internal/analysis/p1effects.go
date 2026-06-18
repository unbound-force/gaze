// Package analysis provides the core side effect detection engine.
package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

// AnalyzeP1Effects detects P1-tier side effects in a function body
// using AST inspection. This covers:
//   - GlobalMutation: assignment to package-level variables
//   - WriterOutput: calls to io.Writer.Write or fmt.Fprint* with
//     a non-stdout/stderr writer parameter
//   - ChannelSend: send statements (ch <- v)
//   - ChannelClose: calls to close(ch)
//   - HTTPResponseWrite: calls to http.ResponseWriter methods
//   - SliceMutation: direct index assignment on slice parameters
//   - MapMutation: map index assignment on map parameters
//
// Internally, the function dispatches to per-node-type handlers:
// detectAssignEffects, detectIncDecEffects, detectSendEffects, and
// detectP1CallEffects. The shared seen map preserves deduplication
// across all handlers.
func AnalyzeP1Effects(
	fset *token.FileSet,
	info *types.Info,
	fd *ast.FuncDecl,
	pkg string,
	funcName string,
) []taxonomy.SideEffect {
	if fd.Body == nil {
		return nil
	}

	var effects []taxonomy.SideEffect
	seen := make(map[string]bool)

	// Build set of parameter and local names to distinguish globals.
	locals := collectLocals(fd)

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			effects = append(effects,
				detectAssignEffects(fset, info, node, pkg, funcName, seen, locals)...)
		case *ast.IncDecStmt:
			effects = append(effects,
				detectIncDecEffects(fset, info, node, pkg, funcName, seen, locals)...)
		case *ast.SendStmt:
			effects = append(effects,
				detectSendEffects(fset, node, pkg, funcName, seen)...)
		case *ast.CallExpr:
			effects = append(effects,
				detectP1CallEffects(fset, info, node, pkg, funcName, seen)...)
		}
		return true
	})

	return effects
}

// detectAssignEffects handles *ast.AssignStmt nodes, detecting
// GlobalMutation (assignment to package-level variables),
// MapMutation (m[key] = value), and SliceMutation (s[i] = value).
func detectAssignEffects(
	fset *token.FileSet,
	info *types.Info,
	node *ast.AssignStmt,
	pkg string,
	funcName string,
	seen map[string]bool,
	locals map[string]bool,
) []taxonomy.SideEffect {
	var effects []taxonomy.SideEffect

	for _, lhs := range node.Lhs {
		// Global mutation: assignment to a package-level var.
		if ident, ok := lhs.(*ast.Ident); ok {
			if isGlobalIdent(ident, info, locals) {
				key := "global:" + ident.Name
				if !seen[key] {
					seen[key] = true
					loc := fset.Position(ident.Pos()).String()
					effects = append(effects, taxonomy.SideEffect{
						ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.GlobalMutation), ident.Name),
						Type:        taxonomy.GlobalMutation,
						Tier:        taxonomy.TierP1,
						Location:    loc,
						Description: fmt.Sprintf("assigns to package-level variable '%s'", ident.Name),
						Target:      ident.Name,
					})
				}
			}
		}
		// Map or slice mutation: m[key] = value or s[i] = value.
		if idx, ok := lhs.(*ast.IndexExpr); ok {
			if isMapType(info, idx.X) {
				name := exprName(idx.X)
				key := "map:" + name
				if !seen[key] {
					seen[key] = true
					loc := fset.Position(idx.Pos()).String()
					effects = append(effects, taxonomy.SideEffect{
						ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.MapMutation), name),
						Type:        taxonomy.MapMutation,
						Tier:        taxonomy.TierP1,
						Location:    loc,
						Description: fmt.Sprintf("writes to map '%s'", name),
						Target:      name,
					})
				}
			} else if isSliceType(info, idx.X) {
				name := exprName(idx.X)
				key := "slice:" + name
				if !seen[key] {
					seen[key] = true
					loc := fset.Position(idx.Pos()).String()
					effects = append(effects, taxonomy.SideEffect{
						ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.SliceMutation), name),
						Type:        taxonomy.SliceMutation,
						Tier:        taxonomy.TierP1,
						Location:    loc,
						Description: fmt.Sprintf("writes to slice element '%s'", name),
						Target:      name,
					})
				}
			}
		}
	}

	return effects
}

// detectIncDecEffects handles *ast.IncDecStmt nodes, detecting
// GlobalMutation via increment (++) or decrement (--) operators
// on package-level variables.
func detectIncDecEffects(
	fset *token.FileSet,
	info *types.Info,
	node *ast.IncDecStmt,
	pkg string,
	funcName string,
	seen map[string]bool,
	locals map[string]bool,
) []taxonomy.SideEffect {
	ident, ok := node.X.(*ast.Ident)
	if !ok {
		return nil
	}
	if !isGlobalIdent(ident, info, locals) {
		return nil
	}
	key := "global:" + ident.Name
	if seen[key] {
		return nil
	}
	seen[key] = true
	loc := fset.Position(ident.Pos()).String()
	return []taxonomy.SideEffect{{
		ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.GlobalMutation), ident.Name),
		Type:        taxonomy.GlobalMutation,
		Tier:        taxonomy.TierP1,
		Location:    loc,
		Description: fmt.Sprintf("modifies package-level variable '%s'", ident.Name),
		Target:      ident.Name,
	}}
}

// detectSendEffects handles *ast.SendStmt nodes, detecting
// ChannelSend effects (ch <- value).
func detectSendEffects(
	fset *token.FileSet,
	node *ast.SendStmt,
	pkg string,
	funcName string,
	seen map[string]bool,
) []taxonomy.SideEffect {
	name := exprName(node.Chan)
	key := "chsend:" + name
	if seen[key] {
		return nil
	}
	seen[key] = true
	loc := fset.Position(node.Pos()).String()
	return []taxonomy.SideEffect{{
		ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.ChannelSend), name),
		Type:        taxonomy.ChannelSend,
		Tier:        taxonomy.TierP1,
		Location:    loc,
		Description: fmt.Sprintf("sends on channel '%s'", name),
		Target:      name,
	}}
}

// detectP1CallEffects handles *ast.CallExpr nodes, detecting
// ChannelClose (close(ch)), WriterOutput (w.Write(...) where w
// implements io.Writer), and HTTPResponseWrite (calls to
// ResponseWriter.Write, .WriteHeader, .Header).
func detectP1CallEffects(
	fset *token.FileSet,
	info *types.Info,
	node *ast.CallExpr,
	pkg string,
	funcName string,
	seen map[string]bool,
) []taxonomy.SideEffect {
	var effects []taxonomy.SideEffect

	// Channel close: close(ch).
	if isCloseCall(node, info) && len(node.Args) == 1 {
		name := exprName(node.Args[0])
		key := "chclose:" + name
		if !seen[key] {
			seen[key] = true
			loc := fset.Position(node.Pos()).String()
			effects = append(effects, taxonomy.SideEffect{
				ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.ChannelClose), name),
				Type:        taxonomy.ChannelClose,
				Tier:        taxonomy.TierP1,
				Location:    loc,
				Description: fmt.Sprintf("closes channel '%s'", name),
				Target:      name,
			})
		}
	}

	// Writer output and HTTP response writes via selector expressions.
	if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "Write" && isWriterType(info, sel.X) {
			name := exprName(sel.X)
			key := "writer:" + name
			if !seen[key] {
				seen[key] = true
				loc := fset.Position(node.Pos()).String()
				effects = append(effects, taxonomy.SideEffect{
					ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.WriterOutput), name),
					Type:        taxonomy.WriterOutput,
					Tier:        taxonomy.TierP1,
					Location:    loc,
					Description: fmt.Sprintf("writes to io.Writer '%s'", name),
					Target:      name,
				})
			}
		}

		// HTTP response writes: calls to
		// ResponseWriter.Write, .WriteHeader, .Header.
		if isHTTPResponseWriter(info, sel.X) {
			method := sel.Sel.Name
			if method == "Write" || method == "WriteHeader" || method == "Header" {
				name := exprName(sel.X)
				key := "http:" + name + ":" + method
				if !seen[key] {
					seen[key] = true
					loc := fset.Position(node.Pos()).String()
					effects = append(effects, taxonomy.SideEffect{
						ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.HTTPResponseWrite), name+"."+method),
						Type:        taxonomy.HTTPResponseWrite,
						Tier:        taxonomy.TierP1,
						Location:    loc,
						Description: fmt.Sprintf("calls %s.%s()", name, method),
						Target:      name + "." + method,
					})
				}
			}
		}
	}

	return effects
}

// collectLocals returns a set of names that are unambiguously local
// to the function signature (parameters, named returns, and
// receiver). This is used as a fast-path in isGlobalIdent to skip
// the more expensive type-info lookup for obvious locals.
//
// Note: Body-level declarations (:=, var, range) are intentionally
// excluded because they can shadow package-level variables in inner
// scopes. Including them caused false negatives where a global
// mutation was missed because a same-named variable was declared
// in a different scope. The type-based check in isGlobalIdent
// handles scoping correctly.
func collectLocals(fd *ast.FuncDecl) map[string]bool {
	locals := make(map[string]bool)

	// Parameters.
	if fd.Type.Params != nil {
		for _, p := range fd.Type.Params.List {
			for _, n := range p.Names {
				locals[n.Name] = true
			}
		}
	}

	// Named returns.
	if fd.Type.Results != nil {
		for _, r := range fd.Type.Results.List {
			for _, n := range r.Names {
				locals[n.Name] = true
			}
		}
	}

	// Receiver.
	if fd.Recv != nil {
		for _, r := range fd.Recv.List {
			for _, n := range r.Names {
				locals[n.Name] = true
			}
		}
	}

	return locals
}

// isGlobalIdent checks if an identifier refers to a package-level
// variable (not a local or parameter) using type resolution.
func isGlobalIdent(ident *ast.Ident, info *types.Info, locals map[string]bool) bool {
	// Fast-path: signature-level locals (params, named returns,
	// receiver) can never be globals.
	if locals[ident.Name] {
		return false
	}
	if info == nil {
		return false
	}
	obj := info.Uses[ident]
	if obj == nil {
		return false
	}
	// Package-level variables have a parent scope that is the
	// package scope (not a function scope).
	if v, ok := obj.(*types.Var); ok {
		return v.Parent() != nil && v.Parent().Parent() == types.Universe
	}
	return false
}

// isMapType checks if an expression has a map type.
func isMapType(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	tv, ok := info.Types[expr]
	if !ok {
		return false
	}
	_, isMap := tv.Type.Underlying().(*types.Map)
	return isMap
}

// isSliceType checks if an expression has a slice type.
func isSliceType(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	tv, ok := info.Types[expr]
	if !ok {
		return false
	}
	_, isSlice := tv.Type.Underlying().(*types.Slice)
	return isSlice
}

// isCloseCall checks if a call expression is a call to the
// builtin close() function using type resolution to avoid false
// positives from user-defined functions named "close".
func isCloseCall(call *ast.CallExpr, info *types.Info) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	if ident.Name != "close" {
		return false
	}
	// Verify this is the builtin close, not a user-defined function.
	if info != nil {
		if obj := info.Uses[ident]; obj != nil {
			_, isBuiltin := obj.(*types.Builtin)
			return isBuiltin
		}
	}
	// Fallback: accept name match when type info is unavailable.
	return true
}

// isWriterType checks if an expression implements io.Writer.
func isWriterType(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	tv, ok := info.Types[expr]
	if !ok {
		return false
	}
	// Check if the type has a Write([]byte) (int, error) method.
	mset := types.NewMethodSet(tv.Type)
	for i := 0; i < mset.Len(); i++ {
		if mset.At(i).Obj().Name() == "Write" {
			return true
		}
	}
	// Also check pointer type.
	ptrType := types.NewPointer(tv.Type)
	mset = types.NewMethodSet(ptrType)
	for i := 0; i < mset.Len(); i++ {
		if mset.At(i).Obj().Name() == "Write" {
			return true
		}
	}
	return false
}

// isHTTPResponseWriter checks if an expression has the
// net/http.ResponseWriter interface type.
func isHTTPResponseWriter(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	tv, ok := info.Types[expr]
	if !ok {
		return false
	}
	return tv.Type.String() == "net/http.ResponseWriter"
}

// exprName returns a short readable name for an expression.
func exprName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprName(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprName(e.X)
	case *ast.IndexExpr:
		return exprName(e.X)
	default:
		return "<expr>"
	}
}
