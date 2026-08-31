// Package analysis provides the core side effect detection engine.
package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/charmbracelet/log"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/unbound-force/gaze/internal/ssaguard"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// BuildSSA constructs the SSA representation for a loaded package.
// The result is reusable across multiple function analyses within
// the same package, avoiding the cost of rebuilding SSA per function.
// Returns nil if SSA construction fails or if prog.Build() panics
// (e.g., due to upstream x/tools bugs with certain generic types
// under Go 1.25). Panics are recovered and logged; callers already
// handle nil returns by skipping mutation analysis.
//
// BuildSerially is included in the builder mode to force prog.Build()
// to run all SSA construction on the calling goroutine. Without it,
// prog.Build() spawns child goroutines per package, and Go's
// goroutine-scoped recover() cannot catch panics across goroutine
// boundaries.
func BuildSSA(pkg *packages.Package) (ssaPkg *ssa.Package) {
	prog, ssaPkgs := ssautil.AllPackages(
		[]*packages.Package{pkg},
		ssa.InstantiateGenerics|ssa.BuildSerially,
	)

	if r := ssaguard.SafeSSABuild(prog.Build); r != nil {
		log.Warn("SSA build skipped: internal panic recovered", "pkg", pkg.PkgPath)
		log.Debug("SSA panic value", "pkg", pkg.PkgPath, "panic", r)
		return nil
	}

	if len(ssaPkgs) == 0 || ssaPkgs[0] == nil {
		return nil
	}
	return ssaPkgs[0]
}

// AnalyzeMutations detects pointer receiver and pointer argument
// mutations using SSA analysis. It walks SSA instructions to find
// Store operations through FieldAddr (receiver mutations) and
// through pointer parameters. The ssaPkg parameter should be
// obtained from BuildSSA to avoid redundant SSA construction.
func AnalyzeMutations(
	fset *token.FileSet,
	ssaPkg *ssa.Package,
	fd *ast.FuncDecl,
	fnObj *types.Func,
	pkgPath string,
	funcName string,
) []taxonomy.SideEffect {
	if ssaPkg == nil {
		return analyzeASTMutations(fset, fd, pkgPath, funcName)
	}

	// Find the SSA function matching our target.
	ssaFn := findSSAFunction(ssaPkg, fnObj, fd)
	if ssaFn == nil {
		return nil
	}

	return detectMutations(fset, ssaFn, fd, pkgPath, funcName)
}

// findSSAFunction locates the SSA function matching a types.Func
// object. It first attempts a precise lookup via
// ssaPkg.Prog.FuncValue (for package-level functions) or
// ssaPkg.Prog.MethodValue (for methods), falling back to
// name-based lookup if the types.Func is nil.
func findSSAFunction(ssaPkg *ssa.Package, fnObj *types.Func, fd *ast.FuncDecl) *ssa.Function {
	// Prefer the precise lookup via types.Func when available.
	if fnObj != nil {
		if fd.Recv == nil {
			// Package-level function: direct lookup.
			if fn := ssaPkg.Prog.FuncValue(fnObj); fn != nil {
				return fn
			}
		} else {
			// Method: look up via the method set of the
			// receiver type (both value and pointer).
			sig, ok := fnObj.Type().(*types.Signature)
			if !ok {
				return nil
			}
			recv := sig.Recv()
			if recv != nil {
				mset := types.NewMethodSet(recv.Type())
				for i := 0; i < mset.Len(); i++ {
					sel := mset.At(i)
					if sel.Obj().Id() == fnObj.Id() {
						return ssaPkg.Prog.MethodValue(sel)
					}
				}
			}
		}
	}

	// Fallback: name-based lookup for cases where fnObj is nil.
	if fd.Recv != nil {
		typeName := baseTypeName(fd.Recv.List[0].Type)
		member := ssaPkg.Members[typeName]
		if member == nil {
			return nil
		}
		if namedType, ok := member.(*ssa.Type); ok {
			mset := types.NewMethodSet(namedType.Type())
			for i := 0; i < mset.Len(); i++ {
				sel := mset.At(i)
				if sel.Obj().Name() == fd.Name.Name {
					return ssaPkg.Prog.MethodValue(sel)
				}
			}
			ptrType := types.NewPointer(namedType.Type())
			mset = types.NewMethodSet(ptrType)
			for i := 0; i < mset.Len(); i++ {
				sel := mset.At(i)
				if sel.Obj().Name() == fd.Name.Name {
					return ssaPkg.Prog.MethodValue(sel)
				}
			}
		}
		return nil
	}

	member := ssaPkg.Members[fd.Name.Name]
	if member == nil {
		return nil
	}
	fn, ok := member.(*ssa.Function)
	if !ok {
		return nil
	}
	return fn
}

// baseTypeName extracts the base type name from a receiver type
// expression, stripping pointer indirection.
func baseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return baseTypeName(t.X)
	case *ast.Ident:
		return t.Name
	default:
		return ""
	}
}

// detectMutations walks SSA instructions to find Store operations
// that represent receiver or pointer argument mutations.
func detectMutations(
	fset *token.FileSet,
	ssaFn *ssa.Function,
	fd *ast.FuncDecl,
	pkgPath string,
	funcName string,
) []taxonomy.SideEffect {
	if ssaFn.Blocks == nil {
		return nil
	}

	isMethod := fd.Recv != nil
	receiverParam := receiverSSAParam(ssaFn, isMethod)
	ptrParams := pointerParams(ssaFn, isMethod)

	// Track which fields/params have already been reported to avoid
	// duplicate side effects for the same field mutated multiple times.
	seenReceiverFields := make(map[string]bool)
	seenPtrArgs := make(map[string]bool)

	var effects []taxonomy.SideEffect

	for _, block := range ssaFn.Blocks {
		for _, instr := range block.Instrs {
			store, ok := instr.(*ssa.Store)
			if !ok {
				continue
			}

			loc := instrLocation(fset, store)

			// Check for receiver field mutation.
			if isMethod && receiverParam != nil {
				if fieldName, ok := isReceiverFieldStore(store, receiverParam); ok {
					if !seenReceiverFields[fieldName] {
						seenReceiverFields[fieldName] = true
						effects = append(effects, taxonomy.SideEffect{
							ID:          taxonomy.GenerateID(pkgPath, funcName, string(taxonomy.ReceiverMutation), fieldName),
							Type:        taxonomy.ReceiverMutation,
							Tier:        taxonomy.TierP0,
							Location:    loc,
							Description: fmt.Sprintf("mutates receiver field '%s'", fieldName),
							Target:      fieldName,
						})
					}
				}
			}

			// Check for pointer argument mutation.
			if paramName, ok := isPointerArgStore(store, ptrParams); ok {
				if !seenPtrArgs[paramName] {
					seenPtrArgs[paramName] = true
					effects = append(effects, taxonomy.SideEffect{
						ID:          taxonomy.GenerateID(pkgPath, funcName, string(taxonomy.PointerArgMutation), paramName),
						Type:        taxonomy.PointerArgMutation,
						Tier:        taxonomy.TierP0,
						Location:    loc,
						Description: fmt.Sprintf("mutates pointer argument '%s'", paramName),
						Target:      paramName,
					})
				}
			}
		}
	}

	return effects
}

// receiverSSAParam returns the SSA parameter that represents the
// method receiver, or nil if not a method.
func receiverSSAParam(fn *ssa.Function, isMethod bool) *ssa.Parameter {
	if !isMethod || len(fn.Params) == 0 {
		return nil
	}
	return fn.Params[0]
}

// pointerParams returns a map of parameter name → *ssa.Parameter
// for all pointer-typed parameters (excluding the receiver).
func pointerParams(fn *ssa.Function, isMethod bool) map[string]*ssa.Parameter {
	params := make(map[string]*ssa.Parameter)
	start := 0
	if isMethod {
		start = 1 // skip receiver
	}
	for i := start; i < len(fn.Params); i++ {
		p := fn.Params[i]
		if _, ok := p.Type().(*types.Pointer); ok {
			params[p.Name()] = p
		}
	}
	return params
}

// isReceiverFieldStore checks if a Store instruction writes to a
// field of the receiver. Returns the top-level field name if true.
// For nested field access like `c.Nested.Value = v`, this reports
// "Nested" (the top-level field through the receiver).
func isReceiverFieldStore(store *ssa.Store, receiver *ssa.Parameter) (string, bool) {
	addr := store.Addr

	// Walk up the FieldAddr chain to find the one whose base
	// traces to the receiver parameter. We want the top-level
	// field (closest to the receiver).
	fa, ok := addr.(*ssa.FieldAddr)
	if !ok {
		return "", false
	}

	// Walk up nested FieldAddr chain to find the one closest
	// to the receiver.
	topFA := fa
	for {
		innerFA, ok := topFA.X.(*ssa.FieldAddr)
		if !ok {
			break
		}
		topFA = innerFA
	}

	// The base of the topmost FieldAddr should trace to the receiver.
	if tracesToParam(topFA.X, receiver) {
		return fieldNameFromFieldAddr(topFA), true
	}

	return "", false
}

// isPointerArgStore checks if a Store instruction writes through a
// pointer parameter. tracesToParam handles chain walking through
// FieldAddr, IndexAddr, UnOp, and Phi instructions recursively,
// so this function only needs to delegate to it for each parameter.
// Returns the parameter name if true.
func isPointerArgStore(store *ssa.Store, ptrParams map[string]*ssa.Parameter) (string, bool) {
	addr := store.Addr
	for name, param := range ptrParams {
		if tracesToParam(addr, param) {
			return name, true
		}
	}
	return "", false
}

// tracesToParam walks up the SSA value chain to check if a value
// ultimately derives from a specific parameter.
func tracesToParam(v ssa.Value, param *ssa.Parameter) bool {
	visited := make(map[ssa.Value]bool)
	return tracesToParamVisited(v, param, visited)
}

// tracesToParamVisited is the recursive implementation with cycle
// detection via a visited set. Phi nodes in SSA can form cycles
// (e.g., loop back edges), so we must track visited values.
func tracesToParamVisited(v ssa.Value, param *ssa.Parameter, visited map[ssa.Value]bool) bool {
	if v == param {
		return true
	}
	if visited[v] {
		return false
	}
	visited[v] = true

	switch val := v.(type) {
	case *ssa.FieldAddr:
		return tracesToParamVisited(val.X, param, visited)
	case *ssa.IndexAddr:
		return tracesToParamVisited(val.X, param, visited)
	case *ssa.UnOp:
		return tracesToParamVisited(val.X, param, visited)
	case *ssa.Phi:
		for _, edge := range val.Edges {
			if tracesToParamVisited(edge, param, visited) {
				return true
			}
		}
	}
	return false
}

// fieldNameFromFieldAddr extracts the struct field name from a
// FieldAddr instruction.
func fieldNameFromFieldAddr(fa *ssa.FieldAddr) string {
	pt, ok := fa.X.Type().(*types.Pointer)
	if !ok {
		return fmt.Sprintf("field_%d", fa.Field)
	}
	st, ok := pt.Elem().Underlying().(*types.Struct)
	if !ok {
		return fmt.Sprintf("field_%d", fa.Field)
	}
	if fa.Field >= st.NumFields() {
		return fmt.Sprintf("field_%d", fa.Field)
	}
	return st.Field(fa.Field).Name()
}

// instrLocation returns the source location for an SSA instruction.
func instrLocation(fset *token.FileSet, instr ssa.Instruction) string {
	pos := instr.Pos()
	if !pos.IsValid() {
		return "<unknown>"
	}
	return fset.Position(pos).String()
}

// exprRootIdent recursively unwraps composite AST expressions to
// find the base *ast.Ident. It handles SelectorExpr (x.Field),
// IndexExpr (x[i]), StarExpr (*x), and ParenExpr ((x)). Returns
// nil if the expression chain does not resolve to an identifier.
func exprRootIdent(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return exprRootIdent(e.X)
	case *ast.IndexExpr:
		return exprRootIdent(e.X)
	case *ast.StarExpr:
		return exprRootIdent(e.X)
	case *ast.ParenExpr:
		return exprRootIdent(e.X)
	default:
		return nil
	}
}

// analyzeASTMutations detects receiver and pointer argument mutations
// using AST walking when SSA is unavailable. This is a lower-fidelity
// fallback that covers the most common mutation patterns: direct
// field assignments and method calls on receiver/parameter fields.
// Effects detected by this fallback include " (AST fallback)" in
// their Description to distinguish them from SSA-detected mutations.
//
// Design decision: AST fallback chosen over unconditional effect
// emission per SOLID Single Responsibility — the fallback does one
// thing (AST-based mutation detection) and does it conservatively.
// See research.md R1 for alternatives considered.
func analyzeASTMutations(
	fset *token.FileSet,
	fd *ast.FuncDecl,
	pkgPath string,
	funcName string,
) []taxonomy.SideEffect {
	var effects []taxonomy.SideEffect

	// Detect receiver mutations for methods.
	effects = append(effects, detectASTReceiverMutations(fset, fd, pkgPath, funcName)...)

	// Detect pointer argument mutations.
	effects = append(effects, detectASTPointerArgMutations(fset, fd, pkgPath, funcName)...)

	return effects
}

// detectASTReceiverMutations checks if a method mutates its pointer
// receiver via field assignments or method calls on receiver fields.
// Returns nil for non-methods, value receivers, or unnamed receivers
// (FR-004: value receiver mutations are not observable by the caller).
func detectASTReceiverMutations(
	fset *token.FileSet,
	fd *ast.FuncDecl,
	pkgPath string,
	funcName string,
) []taxonomy.SideEffect {
	// Must be a method.
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return nil
	}

	recv := fd.Recv.List[0]

	// Must be a pointer receiver (FR-004).
	if _, ok := recv.Type.(*ast.StarExpr); !ok {
		return nil
	}

	// Must have a name to trace assignments.
	if len(recv.Names) == 0 {
		return nil
	}
	receiverName := recv.Names[0].Name

	if fd.Body == nil {
		return nil
	}

	// Walk the function body looking for mutations.
	found := false
	var foundPos token.Pos
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if found {
			return false // short-circuit after first find
		}
		switch node := n.(type) {
		case *ast.AssignStmt:
			found, foundPos = handleReceiverAssignStmt(node, receiverName)
		case *ast.IncDecStmt:
			found, foundPos = handleReceiverIncDecStmt(node, receiverName)
		case *ast.CallExpr:
			found, foundPos = handleReceiverCallExpr(node, receiverName)
		}
		return !found
	})

	if !found {
		return nil
	}

	// Deduplicate to one effect per receiver (consistent with SSA
	// detector behavior for the fallback scope).
	return []taxonomy.SideEffect{{
		ID:          taxonomy.GenerateID(pkgPath, funcName, string(taxonomy.ReceiverMutation), receiverName),
		Type:        taxonomy.ReceiverMutation,
		Tier:        taxonomy.TierP0,
		Description: fmt.Sprintf("mutates receiver %s (AST fallback)", receiverName),
		Target:      funcName,
		Location:    fset.Position(foundPos).String(),
	}}
}

// handleReceiverAssignStmt checks whether an assignment statement mutates
// a receiver field. Returns (true, pos) if any LHS expression roots to
// receiverName via a field access (e.g., recv.Field = v). Bare receiver
// assignments (recv = v) are excluded because they rebind the local
// pointer, not the pointed-to state (FR-003).
func handleReceiverAssignStmt(node *ast.AssignStmt, receiverName string) (bool, token.Pos) {
	for _, lhs := range node.Lhs {
		if ident := exprRootIdent(lhs); ident != nil && ident.Name == receiverName {
			// Ensure this is a field access, not a bare
			// receiver assignment (e.g., `recv = something`).
			if _, ok := lhs.(*ast.Ident); !ok {
				return true, node.Pos()
			}
		}
	}
	return false, token.NoPos
}

// handleReceiverIncDecStmt checks whether an increment/decrement statement
// mutates a receiver field. Returns (true, pos) if the operand roots to
// receiverName via a field access (e.g., recv.count++). Bare receiver
// increments (recv++) are excluded.
func handleReceiverIncDecStmt(node *ast.IncDecStmt, receiverName string) (bool, token.Pos) {
	if ident := exprRootIdent(node.X); ident != nil && ident.Name == receiverName {
		if _, ok := node.X.(*ast.Ident); !ok {
			return true, node.Pos()
		}
	}
	return false, token.NoPos
}

// handleReceiverCallExpr checks whether a call expression invokes a method
// on a receiver field (e.g., recv.field.Delete(key)). Direct method calls
// on the receiver itself (e.g., recv.Method()) are excluded because they
// are not field-mutation evidence — the receiver method may or may not
// mutate state (FR-008).
func handleReceiverCallExpr(node *ast.CallExpr, receiverName string) (bool, token.Pos) {
	sel, ok := node.Fun.(*ast.SelectorExpr)
	if !ok {
		return false, token.NoPos
	}
	if ident := exprRootIdent(sel.X); ident != nil && ident.Name == receiverName {
		// Must be a method call on a field, not a direct method call
		// on the receiver itself (e.g., si.Method() vs si.field.Method()).
		if innerSel, ok := sel.X.(*ast.SelectorExpr); ok {
			if root := exprRootIdent(innerSel.X); root != nil && root.Name == receiverName {
				return true, node.Pos()
			}
		}
	}
	return false, token.NoPos
}

// detectASTPointerArgMutations checks if a function mutates any of
// its pointer-typed parameters via field assignments. Returns nil
// if no pointer parameters are mutated.
func detectASTPointerArgMutations(
	fset *token.FileSet,
	fd *ast.FuncDecl,
	pkgPath string,
	funcName string,
) []taxonomy.SideEffect {
	if fd.Type.Params == nil || fd.Body == nil {
		return nil
	}

	// Collect pointer parameter names.
	var ptrParams []string
	for _, field := range fd.Type.Params.List {
		if _, ok := field.Type.(*ast.StarExpr); !ok {
			continue
		}
		for _, name := range field.Names {
			ptrParams = append(ptrParams, name.Name)
		}
	}

	if len(ptrParams) == 0 {
		return nil
	}

	// Build a set for fast lookup.
	ptrParamSet := make(map[string]bool, len(ptrParams))
	for _, name := range ptrParams {
		ptrParamSet[name] = true
	}

	// Walk the function body looking for mutations to pointer params.
	// Track position of first mutation per param for Location field.
	mutatedParams := make(map[string]token.Pos)
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				// exprRootIdent unwraps SelectorExpr, IndexExpr,
				// StarExpr, and ParenExpr to find the base ident.
				// This handles param.field, param[i], *param, and
				// (param).field patterns in a single check.
				if ident := exprRootIdent(lhs); ident != nil && ptrParamSet[ident.Name] {
					// Must be a field/index/deref access, not a
					// bare reassignment of the parameter variable.
					if _, ok := lhs.(*ast.Ident); !ok {
						if _, exists := mutatedParams[ident.Name]; !exists {
							mutatedParams[ident.Name] = node.Pos()
						}
					}
				}
			}
		case *ast.IncDecStmt:
			// Handle param.field++ / param.field--
			if ident := exprRootIdent(node.X); ident != nil && ptrParamSet[ident.Name] {
				if _, ok := node.X.(*ast.Ident); !ok {
					if _, exists := mutatedParams[ident.Name]; !exists {
						mutatedParams[ident.Name] = node.Pos()
					}
				}
			}
		}
		return true
	})

	var effects []taxonomy.SideEffect
	for _, name := range ptrParams {
		if pos, ok := mutatedParams[name]; ok {
			effects = append(effects, taxonomy.SideEffect{
				ID:          taxonomy.GenerateID(pkgPath, funcName, string(taxonomy.PointerArgMutation), name),
				Type:        taxonomy.PointerArgMutation,
				Tier:        taxonomy.TierP0,
				Description: fmt.Sprintf("mutates pointer argument '%s' (AST fallback)", name),
				Target:      name,
				Location:    fset.Position(pos).String(),
			})
		}
	}

	return effects
}
