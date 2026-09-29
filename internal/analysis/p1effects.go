// Package analysis provides the core side effect detection engine.
package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
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

	// Build set of signature-level variable objects (params, named
	// returns, receiver) for scope-aware effect filtering.
	sigVars := collectSignatureVars(info, fd)

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			effects = append(effects,
				detectAssignEffects(fset, info, node, pkg, funcName, seen, locals, sigVars)...)
		case *ast.IncDecStmt:
			effects = append(effects,
				detectIncDecEffects(fset, info, node, pkg, funcName, seen, locals)...)
		case *ast.SendStmt:
			effects = append(effects,
				detectSendEffects(fset, info, node, pkg, funcName, seen, sigVars)...)
		case *ast.CallExpr:
			effects = append(effects,
				detectP1CallEffects(fset, info, node, pkg, funcName, seen, sigVars)...)
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
	sigVars map[types.Object]bool,
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
		// Only emit when the variable is externally observable (parameter,
		// receiver, named return, or package-level). Body-local variables
		// are not side effects — they are internal state.
		if idx, ok := lhs.(*ast.IndexExpr); ok {
			if isMapType(info, idx.X) && isExternallyObservable(info, idx.X, sigVars) {
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
			} else if isSliceType(info, idx.X) && isExternallyObservable(info, idx.X, sigVars) {
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
// ChannelSend effects (ch <- value). Only emits when the channel
// variable is externally observable (parameter, receiver, named
// return, or package-level).
func detectSendEffects(
	fset *token.FileSet,
	info *types.Info,
	node *ast.SendStmt,
	pkg string,
	funcName string,
	seen map[string]bool,
	sigVars map[types.Object]bool,
) []taxonomy.SideEffect {
	// Skip sends on body-local channels — not externally observable.
	if !isExternallyObservable(info, node.Chan, sigVars) {
		return nil
	}
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
	sigVars map[types.Object]bool,
) []taxonomy.SideEffect {
	var effects []taxonomy.SideEffect

	// Channel close: close(ch). Only emit when the channel variable
	// is externally observable (parameter, receiver, named return,
	// or package-level).
	if isCloseCall(node, info) && len(node.Args) == 1 &&
		isExternallyObservable(info, node.Args[0], sigVars) {
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
	// Check HTTP response writer first (more specific); fall through
	// to generic io.Writer only when the receiver is not an
	// http.ResponseWriter (issue #131).
	if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
		if isHTTPResponseWriter(info, sel.X) {
			// HTTP response writes: calls to
			// ResponseWriter.Write, .WriteHeader, .Header.
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
		} else if sel.Sel.Name == "Write" && isWriterType(info, sel.X) {
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

// ioWriterInterface is a synthetically constructed io.Writer interface
// type used for types.Implements checks. Built once at init time.
//
//	interface{ Write(p []byte) (n int, err error) }
var ioWriterInterface *types.Interface

func init() {
	byteSlice := types.NewSlice(types.Typ[types.Byte])
	sig := types.NewSignatureType(
		nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "p", byteSlice)),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]),
			types.NewVar(token.NoPos, nil, "err", types.Universe.Lookup("error").Type()),
		),
		false,
	)
	writeMethod := types.NewFunc(token.NoPos, nil, "Write", sig)
	ioWriterInterface = types.NewInterfaceType([]*types.Func{writeMethod}, nil)
	ioWriterInterface.Complete()
}

// isWriterType checks if an expression implements io.Writer by
// verifying the full Write([]byte) (int, error) signature, not
// just the method name. Uses types.Implements against a synthetic
// io.Writer interface.
func isWriterType(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	tv, ok := info.Types[expr]
	if !ok {
		return false
	}
	if types.Implements(tv.Type, ioWriterInterface) {
		return true
	}
	// Also check pointer type.
	return types.Implements(types.NewPointer(tv.Type), ioWriterInterface)
}

// isHTTPResponseWriter checks if an expression implements the
// net/http.ResponseWriter interface by verifying its method set
// contains all three required methods: Header, Write, and WriteHeader.
//
// Write and WriteHeader signatures are verified exactly against
// io.Writer and func(int) respectively. Header is checked by name
// only because its return type (http.Header) is a named type that
// cannot be synthetically constructed without importing net/http.
//
// This correctly handles type aliases, custom implementations,
// and embedded interfaces — unlike the previous string comparison
// approach.
func isHTTPResponseWriter(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	tv, ok := info.Types[expr]
	if !ok {
		return false
	}
	return hasHTTPResponseWriterMethods(tv.Type) ||
		hasHTTPResponseWriterMethods(types.NewPointer(tv.Type))
}

// hasHTTPResponseWriterMethods checks if a type's method set
// contains Header(), Write([]byte)(int, error), and WriteHeader(int).
func hasHTTPResponseWriterMethods(t types.Type) bool {
	mset := types.NewMethodSet(t)
	var hasHeader, hasWrite, hasWriteHeader bool
	for i := 0; i < mset.Len(); i++ {
		obj := mset.At(i).Obj()
		sig, ok := obj.Type().(*types.Signature)
		if !ok {
			continue
		}
		switch obj.Name() {
		case "Header":
			// Accept any Header() method with zero params and one
			// result. The return type is http.Header (a named type)
			// which we cannot construct synthetically, but any type
			// with all three method names and correct Write/WriteHeader
			// signatures is effectively an http.ResponseWriter.
			if sig.Params().Len() == 0 && sig.Results().Len() == 1 {
				hasHeader = true
			}
		case "Write":
			// Must match io.Writer: Write([]byte) (int, error).
			if matchesWriteSignature(sig) {
				hasWrite = true
			}
		case "WriteHeader":
			// Must match WriteHeader(statusCode int).
			if sig.Params().Len() == 1 && sig.Results().Len() == 0 &&
				types.Identical(sig.Params().At(0).Type(), types.Typ[types.Int]) {
				hasWriteHeader = true
			}
		}
	}
	return hasHeader && hasWrite && hasWriteHeader
}

// matchesWriteSignature checks if a signature matches
// Write([]byte) (int, error) — the io.Writer.Write signature.
func matchesWriteSignature(sig *types.Signature) bool {
	if sig.Params().Len() != 1 || sig.Results().Len() != 2 {
		return false
	}
	byteSlice := types.NewSlice(types.Typ[types.Byte])
	if !types.Identical(sig.Params().At(0).Type(), byteSlice) {
		return false
	}
	if !types.Identical(sig.Results().At(0).Type(), types.Typ[types.Int]) {
		return false
	}
	return types.Identical(sig.Results().At(1).Type(), types.Universe.Lookup("error").Type())
}

// unwrapToIdent unwraps selector expressions, index expressions,
// star expressions, and parenthesized expressions to find the base
// *ast.Ident. Returns nil if the expression cannot be unwrapped to
// an identifier (e.g., a function call result or composite literal).
//
// Examples:
//   - m           → m (*ast.Ident)
//   - s.field     → s (*ast.SelectorExpr → *ast.Ident)
//   - m[key]      → m (*ast.IndexExpr → *ast.Ident)
//   - s.field[key] → s (*ast.IndexExpr → *ast.SelectorExpr → *ast.Ident)
//   - *ptr        → ptr (*ast.StarExpr → *ast.Ident)
//   - (expr)      → unwrap inner (*ast.ParenExpr → recurse)
func unwrapToIdent(expr ast.Expr) *ast.Ident {
	for {
		switch e := expr.(type) {
		case *ast.Ident:
			return e
		case *ast.SelectorExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.StarExpr:
			expr = e.X
		case *ast.ParenExpr:
			expr = e.X
		default:
			return nil
		}
	}
}

// collectSignatureVars builds a set of types.Object pointers for all
// variables declared in the function signature: parameters, named
// returns, and receivers. These are the variables that are externally
// observable — mutations to them are visible to the caller.
//
// Uses types.Info.Defs to resolve AST identifiers to their types.Object,
// then uses pointer identity to match against variables found via
// info.Uses during effect detection. This is more reliable than scope
// depth counting because the Go type checker's scope hierarchy includes
// a file scope between the package scope and function scope, making
// depth-based approaches fragile.
func collectSignatureVars(info *types.Info, fd *ast.FuncDecl) map[types.Object]bool {
	sigVars := make(map[types.Object]bool)
	if info == nil {
		return sigVars
	}

	// Parameters.
	if fd.Type.Params != nil {
		for _, p := range fd.Type.Params.List {
			for _, n := range p.Names {
				if obj := info.Defs[n]; obj != nil {
					sigVars[obj] = true
				}
			}
		}
	}

	// Named returns.
	if fd.Type.Results != nil {
		for _, r := range fd.Type.Results.List {
			for _, n := range r.Names {
				if obj := info.Defs[n]; obj != nil {
					sigVars[obj] = true
				}
			}
		}
	}

	// Receiver.
	if fd.Recv != nil {
		for _, r := range fd.Recv.List {
			for _, n := range r.Names {
				if obj := info.Defs[n]; obj != nil {
					sigVars[obj] = true
				}
			}
		}
	}

	return sigVars
}

// isExternallyObservable returns true if expr refers to a variable
// that is observable from outside the function: a parameter, receiver,
// named return, or package-level variable. Returns false for body-local
// variables (make, var, :=). Returns true (conservative) when the
// expression cannot be resolved — a false negative (missing a real
// side effect) is worse than a false positive per the constitution's
// Accuracy principle.
//
// The function uses two checks:
//  1. Package-level variable: v.Parent().Parent() == types.Universe
//     (the variable's scope is the package scope, whose parent is Universe).
//  2. Signature-level variable: the variable's types.Object pointer
//     matches one in the sigVars set (built from the function's
//     parameters, named returns, and receiver via collectSignatureVars).
//
// Known limitations:
//   - Slice aliasing: a locally-created slice that is sub-sliced and
//     returned shares the backing array. Mutations to the original are
//     observable through the returned sub-slice, but this function
//     classifies the local slice as not externally observable.
//   - Closure capture: a locally-created variable captured by a returned
//     closure is observable from outside the function, but this function
//     classifies it as not externally observable. Detecting this requires
//     escape analysis (tracking whether the variable is captured by a
//     returned closure).
func isExternallyObservable(info *types.Info, expr ast.Expr, sigVars map[types.Object]bool) bool {
	ident := unwrapToIdent(expr)
	if ident == nil {
		return true // can't resolve expression — conservative
	}
	if info == nil {
		return true // no type info — conservative
	}
	obj := info.Uses[ident]
	if obj == nil {
		return true // unresolved identifier — conservative
	}
	v, ok := obj.(*types.Var)
	if !ok {
		return true // not a variable (e.g., constant, function) — conservative
	}
	// Package-level variable: parent scope is the package scope,
	// whose parent is Universe.
	if v.Parent() != nil && v.Parent().Parent() == types.Universe {
		return true
	}
	// Signature-level variable (parameter, receiver, named return):
	// check by types.Object pointer identity against the pre-built set.
	if sigVars[obj] {
		return true
	}
	// Body-local variable: not externally observable.
	return false
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
