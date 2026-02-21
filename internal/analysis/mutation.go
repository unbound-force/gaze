package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// BuildSSA constructs the SSA representation for a loaded package.
// The result is reusable across multiple function analyses within
// the same package, avoiding the cost of rebuilding SSA per function.
// Returns nil if SSA construction fails.
func BuildSSA(pkg *packages.Package) *ssa.Package {
	prog, ssaPkgs := ssautil.AllPackages(
		[]*packages.Package{pkg},
		ssa.InstantiateGenerics,
	)
	prog.Build()

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
		return nil
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
			recv := fnObj.Type().(*types.Signature).Recv()
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
// pointer parameter. Returns the parameter name if true.
func isPointerArgStore(store *ssa.Store, ptrParams map[string]*ssa.Parameter) (string, bool) {
	addr := store.Addr

	for name, param := range ptrParams {
		if tracesToParam(addr, param) {
			return name, true
		}
		// Also check UnOp (dereference).
		if unop, ok := addr.(*ssa.UnOp); ok {
			if tracesToParam(unop.X, param) {
				return name, true
			}
		}
		// FieldAddr through dereferenced pointer param.
		if fa, ok := addr.(*ssa.FieldAddr); ok {
			if tracesToParam(fa.X, param) {
				return name, true
			}
			if unop, ok := fa.X.(*ssa.UnOp); ok {
				if tracesToParam(unop.X, param) {
					return name, true
				}
			}
		}
		// IndexAddr through pointer param (for *[]T, *[N]T).
		if ia, ok := addr.(*ssa.IndexAddr); ok {
			if tracesToParam(ia.X, param) {
				return name, true
			}
			if unop, ok := ia.X.(*ssa.UnOp); ok {
				if tracesToParam(unop.X, param) {
					return name, true
				}
			}
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
