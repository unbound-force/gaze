package quality

import (
	"go/ast"
	"go/token"
	"go/types"
	"io"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// MapAssertionsToEffects maps detected assertion sites to side effects
// using SSA data flow analysis combined with AST assignment analysis.
// It traces return values and mutations from the target call site
// through the test function to assertion sites.
//
// The bridge between SSA and AST domains works by finding the AST
// assignment statement that contains the target call, mapping each
// LHS identifier to the corresponding return value side effect, and
// then matching assertion expressions via types.Object identity.
//
// Assertions that cannot be linked to a specific side effect are
// reported as unmapped per the spec (FR-003): they are excluded from
// both Contract Coverage and Over-Specification metrics. Each unmapped
// AssertionMapping carries an UnmappedReason field classifying why the
// mapping failed: helper_param (assertion in a helper body at depth > 0),
// inline_call (return value asserted inline without assignment), or
// no_effect_match (no side effect object matched the assertion).
//
// It returns three values: mapped assertions, unmapped assertions,
// and a set of side effect IDs whose return values were explicitly
// discarded (e.g., _ = target()), making them definitively unasserted.
func MapAssertionsToEffects(
	testFunc *ssa.Function,
	targetFunc *ssa.Function,
	sites []AssertionSite,
	effects []taxonomy.SideEffect,
	testPkg *packages.Package,
	aiMapper ...AIMapperFunc,
) (mapped []taxonomy.AssertionMapping, unmapped []taxonomy.AssertionMapping, discardedIDs map[string]bool) {
	return mapAssertionsToEffectsImpl(testFunc, targetFunc, sites, effects, testPkg, nil, aiMapper...)
}

// mapAssertionsToEffectsImpl is the internal implementation that
// accepts a stderr writer for AI mapper error logging.
func mapAssertionsToEffectsImpl(
	testFunc *ssa.Function,
	targetFunc *ssa.Function,
	sites []AssertionSite,
	effects []taxonomy.SideEffect,
	testPkg *packages.Package,
	stderr io.Writer,
	aiMapper ...AIMapperFunc,
) (mapped []taxonomy.AssertionMapping, unmapped []taxonomy.AssertionMapping, discardedIDs map[string]bool) {
	// Extract the optional AI mapper (variadic to preserve backward compat).
	// At most one AIMapperFunc is used; additional values are ignored.
	var aiMapperFn AIMapperFunc
	if len(aiMapper) > 0 {
		aiMapperFn = aiMapper[0]
	}

	discardedIDs = make(map[string]bool)

	if len(sites) == 0 || len(effects) == 0 {
		// If no assertions or no effects, everything is unmapped.
		unmapped = make([]taxonomy.AssertionMapping, 0, len(sites))
		for _, s := range sites {
			unmapped = append(unmapped, taxonomy.AssertionMapping{
				AssertionLocation: s.Location,
				AssertionType:     mapKindToType(s.Kind),
				Confidence:        0,
				UnmappedReason:    classifyUnmappedReason(s, nil, effects),
			})
		}
		return nil, unmapped, discardedIDs
	}

	// Find the call to the target function in the test SSA.
	targetCall := FindTargetCall(testFunc, targetFunc)

	// Build a map from side effect ID to side effect for matching.
	effectMap := make(map[string]*taxonomy.SideEffect, len(effects))
	for i := range effects {
		effectMap[effects[i].ID] = &effects[i]
	}

	// Detect discarded returns: _ = target() patterns where SSA
	// produces no Extract referrers for return values.
	discardedIDs = detectDiscardedReturns(targetCall, effects)

	// Build a map from types.Object to effect ID by finding the
	// AST assignment that receives the target call's return values
	// and correlating LHS identifiers with side effects.
	objToEffectID := traceTargetValues(targetCall, effects, testPkg, testFunc, targetFunc)

	// Build return-value effect ID for inline call matching.
	returnEffectID := findReturnEffectID(effects)

	// Match assertion expressions to traced values.
	for _, site := range sites {
		mapping := matchAssertionToEffect(site, objToEffectID, effectMap, testPkg)
		if mapping == nil && returnEffectID != "" {
			// Fallback: check if the assertion expression contains
			// an inline call to the target function (e.g., if f() != x).
			mapping = matchInlineCall(site, targetFunc, returnEffectID, effectMap, testPkg)
		}
		if mapping == nil && returnEffectID != "" {
			// Container unwrap: trace data flow forward from the
			// return value through field access and transformation
			// calls to the assertion expression (confidence 55).
			mapping = matchContainerUnwrap(site, objToEffectID, effectMap, testPkg, returnEffectID)
		}
		if mapping == nil && aiMapperFn != nil {
			// AI fallback: for structurally disconnected assertions,
			// ask the AI to evaluate the semantic relationship.
			mapping = tryAIMapping(site, targetFunc, effects, testPkg.Fset, aiMapperFn, stderr)
		}
		if mapping != nil {
			mapped = append(mapped, *mapping)
		} else {
			// Per spec FR-003: unmapped assertions are reported
			// separately and excluded from metrics.
			unmapped = append(unmapped, taxonomy.AssertionMapping{
				AssertionLocation: site.Location,
				AssertionType:     mapKindToType(site.Kind),
				Confidence:        0,
				UnmappedReason:    classifyUnmappedReason(site, objToEffectID, effects),
			})
		}
	}

	return mapped, unmapped, discardedIDs
}

// findReturnEffectID returns the ID of the first ReturnValue effect,
// or "" if none exists.
func findReturnEffectID(effects []taxonomy.SideEffect) string {
	for _, e := range effects {
		if e.Type == taxonomy.ReturnValue {
			return e.ID
		}
	}
	return ""
}

// detectDiscardedReturns identifies return/error side effects whose
// values were explicitly discarded at the call site (e.g., _ = f()
// or f() with ignored returns). In SSA, discarded returns produce
// no Extract referrers for the corresponding tuple index.
func detectDiscardedReturns(
	targetCall *ssa.Call,
	effects []taxonomy.SideEffect,
) map[string]bool {
	discarded := make(map[string]bool)

	if targetCall == nil {
		return discarded
	}

	returnEffects := filterEffectsByType(effects,
		taxonomy.ReturnValue, taxonomy.ErrorReturn)
	if len(returnEffects) == 0 {
		return discarded
	}

	referrers := targetCall.Referrers()

	// Single-return function: if no referrers at all, the return
	// is discarded (e.g., bare f() call).
	if len(returnEffects) == 1 {
		if referrers == nil || len(*referrers) == 0 {
			discarded[returnEffects[0].ID] = true
		}
		return discarded
	}

	// Multi-return function: check which tuple indices have
	// Extract referrers. Indices without Extract are discarded
	// (assigned to blank identifier).
	extractedIndices := make(map[int]bool)
	if referrers != nil {
		for _, ref := range *referrers {
			if extract, ok := ref.(*ssa.Extract); ok {
				extractedIndices[extract.Index] = true
			}
		}
	}

	for idx, effect := range returnEffects {
		if !extractedIndices[idx] {
			discarded[effect.ID] = true
		}
	}

	return discarded
}

// FindTargetCall finds the SSA call instruction in the test function
// that calls the target function. It searches both the top-level
// function body and any closures (e.g., t.Run sub-tests) to handle
// table-driven test patterns.
func FindTargetCall(
	testFunc *ssa.Function,
	targetFunc *ssa.Function,
) *ssa.Call {
	if testFunc == nil || testFunc.Blocks == nil || targetFunc == nil {
		return nil
	}

	return findTargetCallInFunc(testFunc, targetFunc, make(map[*ssa.Function]bool))
}

// findTargetCallInFunc recursively searches an SSA function and its
// closures for a call to the target function.
func findTargetCallInFunc(
	fn *ssa.Function,
	targetFunc *ssa.Function,
	visited map[*ssa.Function]bool,
) *ssa.Call {
	if fn == nil || fn.Blocks == nil || visited[fn] {
		return nil
	}
	visited[fn] = true

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			// Check for direct calls to the target.
			if call, ok := instr.(*ssa.Call); ok {
				callee := call.Call.StaticCallee()
				if callee != nil && (callee == targetFunc || sameFunction(callee, targetFunc)) {
					return call
				}
			}
			// Follow MakeClosure instructions to search inside
			// closures (handles t.Run sub-tests and anonymous functions).
			if mc, ok := instr.(*ssa.MakeClosure); ok {
				if closureFn, ok := mc.Fn.(*ssa.Function); ok {
					if result := findTargetCallInFunc(closureFn, targetFunc, visited); result != nil {
						return result
					}
				}
			}
		}
	}
	return nil
}

// traceTargetValues bridges the SSA and AST domains by finding the
// AST assignment statement that receives the target call's return
// values, then mapping each LHS identifier's types.Object to the
// corresponding return value side effect.
//
// For mutations (receiver/pointer args), it maps the argument
// variable's types.Object to the mutation effect.
//
// The testFunc and targetFunc parameters are used for helper return
// value tracing: when the direct assignment lookup fails (because
// the target is called inside a helper), the function searches for
// helpers that call the target and traces their return assignments.
//
// The returned map keys are types.Object instances that can be
// matched against assertion operands using TypesInfo.Uses.
func traceTargetValues(
	targetCall *ssa.Call,
	effects []taxonomy.SideEffect,
	testPkg *packages.Package,
	testFunc *ssa.Function,
	targetFunc *ssa.Function,
) map[types.Object]string {
	objToEffectID := make(map[types.Object]string)

	if testPkg == nil || testPkg.TypesInfo == nil {
		return objToEffectID
	}

	// Trace return values by finding the AST assignment.
	// When targetCall is nil (target called inside a helper),
	// traceReturnValues falls back to helper return tracing.
	traceReturnValues(targetCall, effects, objToEffectID, testPkg, testFunc, targetFunc)

	// Trace mutations (receiver and pointer arg values).
	// Mutation tracing requires a direct target call.
	if targetCall != nil {
		traceMutations(targetCall, effects, objToEffectID, testPkg)
	}

	return objToEffectID
}

// traceReturnValues finds the AST assignment statement that contains
// the target function call and maps each LHS identifier to the
// corresponding return value side effect.
//
// For `got, err := Divide(10, 2)`:
//   - LHS[0] "got" -> ReturnValue effect
//   - LHS[1] "err" -> ErrorReturn effect
//
// For `got := Add(2, 3)`:
//   - LHS[0] "got" -> ReturnValue effect
//
// When the direct assignment lookup fails (because the target is
// called inside a helper function), the function falls back to
// helper return value tracing: it searches the test function's AST
// for assignments whose RHS calls a function that (at depth 1 via
// SSA call graph) invokes the target. The helper's return variable
// is then traced as if it were the target's return value.
func traceReturnValues(
	targetCall *ssa.Call,
	effects []taxonomy.SideEffect,
	objToEffectID map[types.Object]string,
	testPkg *packages.Package,
	testFunc *ssa.Function,
	targetFunc *ssa.Function,
) {
	returnEffects := filterEffectsByType(effects,
		taxonomy.ReturnValue, taxonomy.ErrorReturn)
	if len(returnEffects) == 0 {
		return
	}

	// Try direct assignment tracing first.
	if targetCall != nil {
		callPos := targetCall.Pos()
		if callPos.IsValid() {
			assignLHS := findAssignLHS(testPkg, callPos)
			if assignLHS != nil {
				// Direct assignment found — map LHS identifiers to effects.
				mapAssignLHSToEffects(assignLHS, returnEffects, objToEffectID, testPkg)
				return
			}
		}
	}

	// Fallback: helper return value tracing.
	// When direct tracing fails (either targetCall is nil because
	// the target is called inside a helper, or findAssignLHS returns
	// nil), search the test function's SSA for calls to helpers that
	// invoke the target at depth 1.
	traceHelperReturnValues(returnEffects, objToEffectID, testPkg, testFunc, targetFunc)
}

// mapAssignLHSToEffects maps each non-blank LHS identifier of an
// assignment to the corresponding return effect by position index.
func mapAssignLHSToEffects(
	assignLHS []ast.Expr,
	returnEffects []taxonomy.SideEffect,
	objToEffectID map[types.Object]string,
	testPkg *packages.Package,
) {
	for i, lhsExpr := range assignLHS {
		if i >= len(returnEffects) {
			break
		}
		ident, ok := lhsExpr.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}
		// Look up the types.Object for this identifier.
		obj := testPkg.TypesInfo.Defs[ident]
		if obj == nil {
			// For re-assignments (=), the LHS may be in Uses.
			obj = testPkg.TypesInfo.Uses[ident]
		}
		if obj != nil {
			objToEffectID[obj] = returnEffects[i].ID
		}
	}
}

// traceHelperReturnValues searches the test function's SSA for call
// instructions to functions that (at depth 1) invoke the target
// function. When found, the corresponding AST assignment's LHS
// variables are mapped to the return effects.
//
// This handles the pattern:
//
//	result := helperFunc(t, args...)  // helperFunc calls target
//	// assertions on result.Field ...
//
// Constraints:
//   - Only depth-1 helpers (the helper must directly call the target)
//   - Fallback only — never activates when direct tracing succeeds
//   - SSA call graph verification required before tracing
func traceHelperReturnValues(
	returnEffects []taxonomy.SideEffect,
	objToEffectID map[types.Object]string,
	testPkg *packages.Package,
	testFunc *ssa.Function,
	targetFunc *ssa.Function,
) {
	if testFunc == nil || testFunc.Blocks == nil || targetFunc == nil || testPkg == nil {
		return
	}

	// Find helper calls in the test function's SSA that invoke
	// the target at depth 1.
	helperCall := findHelperCall(testFunc, targetFunc)
	if helperCall == nil {
		return
	}

	// Find the AST assignment for this helper call.
	helperCallPos := helperCall.Pos()
	if !helperCallPos.IsValid() {
		return
	}

	assignLHS := findAssignLHS(testPkg, helperCallPos)
	if assignLHS == nil {
		return
	}

	// Map the helper assignment's LHS to the return effects.
	mapAssignLHSToEffects(assignLHS, returnEffects, objToEffectID, testPkg)
}

// findHelperCall searches the test function's SSA (including closures)
// for a call to any function that directly calls the target function
// at depth 1. Returns the helper call instruction if found.
func findHelperCall(
	testFunc *ssa.Function,
	targetFunc *ssa.Function,
) *ssa.Call {
	return findHelperCallInFunc(testFunc, targetFunc, make(map[*ssa.Function]bool))
}

// maxClosureDepth bounds the recursion depth when following MakeClosure
// instructions in findHelperCallInFunc. This prevents deep call stacks
// from nested anonymous functions (e.g., closures capturing closures).
// The visited map prevents cycles, but this depth limit prevents stack
// overflow from deep linear chains.
const maxClosureDepth = 10

// findHelperCallInFunc recursively searches an SSA function and its
// closures for a call to a helper that invokes the target at depth 1.
// The depth parameter tracks closure nesting to bound recursion.
func findHelperCallInFunc(
	fn *ssa.Function,
	targetFunc *ssa.Function,
	visited map[*ssa.Function]bool,
) *ssa.Call {
	return findHelperCallInFuncDepth(fn, targetFunc, visited, 0)
}

// findHelperCallInFuncDepth is the depth-bounded implementation of
// findHelperCallInFunc. It follows MakeClosure instructions up to
// maxClosureDepth levels deep.
func findHelperCallInFuncDepth(
	fn *ssa.Function,
	targetFunc *ssa.Function,
	visited map[*ssa.Function]bool,
	depth int,
) *ssa.Call {
	if fn == nil || fn.Blocks == nil || visited[fn] || depth > maxClosureDepth {
		return nil
	}
	visited[fn] = true

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if call, ok := instr.(*ssa.Call); ok {
				callee := call.Call.StaticCallee()
				if callee == nil {
					continue
				}
				// Skip the target function itself — we're looking
				// for helpers that CALL the target, not direct calls.
				if callee == targetFunc || sameFunction(callee, targetFunc) {
					continue
				}
				// Check if this callee calls the target at depth 1.
				if helperCallsTarget(callee, targetFunc) {
					return call
				}
			}
			// Follow closures (handles t.Run sub-tests).
			if mc, ok := instr.(*ssa.MakeClosure); ok {
				if closureFn, ok := mc.Fn.(*ssa.Function); ok {
					if result := findHelperCallInFuncDepth(closureFn, targetFunc, visited, depth+1); result != nil {
						return result
					}
				}
			}
		}
	}
	return nil
}

// helperCallsTarget checks whether a helper SSA function directly
// calls the target function (depth 1 only). It iterates the helper's
// blocks and instructions looking for *ssa.Call instructions whose
// callee matches the target.
func helperCallsTarget(helper *ssa.Function, target *ssa.Function) bool {
	if helper == nil || helper.Blocks == nil || target == nil {
		return false
	}

	for _, block := range helper.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			callee := call.Call.StaticCallee()
			if callee != nil && (callee == target || sameFunction(callee, target)) {
				return true
			}
		}
	}
	return false
}

// findAssignLHS walks the test package's AST to find the assignment
// statement that contains a call at the given position, returning
// the LHS expression list. This handles both := and = assignments.
func findAssignLHS(
	testPkg *packages.Package,
	callPos token.Pos,
) []ast.Expr {
	if testPkg == nil {
		return nil
	}

	for _, file := range testPkg.Syntax {
		var lhs []ast.Expr
		ast.Inspect(file, func(n ast.Node) bool {
			if lhs != nil {
				return false
			}
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			// Check if any RHS expression contains the target call
			// at the given position.
			for _, rhs := range assign.Rhs {
				if containsPos(rhs, callPos) {
					lhs = assign.Lhs
					return false
				}
			}
			return true
		})
		if lhs != nil {
			return lhs
		}
	}
	return nil
}

// containsPos checks whether an AST expression's source range
// contains the given position. This uses range checking rather
// than exact position matching because SSA and AST may report
// slightly different positions for the same call expression
// (e.g., SSA points to the open paren, AST to the function name).
func containsPos(expr ast.Expr, pos token.Pos) bool {
	return pos >= expr.Pos() && pos < expr.End()
}

// traceMutations traces mutation side effects by identifying the
// AST identifiers used as receiver or pointer arguments at the
// target call site.
//
// For methods, the SSA calling convention places the receiver at
// args[0] and explicit parameters starting at args[1]. This function
// separates receiver mutations from pointer argument mutations to
// avoid index misalignment.
func traceMutations(
	targetCall *ssa.Call,
	effects []taxonomy.SideEffect,
	objToEffectID map[types.Object]string,
	testPkg *packages.Package,
) {
	mutationEffects := filterEffectsByType(effects,
		taxonomy.ReceiverMutation, taxonomy.PointerArgMutation)

	if len(mutationEffects) == 0 {
		return
	}

	args := targetCall.Call.Args
	isMethod := targetCall.Call.IsInvoke() ||
		(len(args) > 0 && hasReceiverMutation(mutationEffects))

	// Determine the offset for explicit parameters.
	paramOffset := 0
	if isMethod {
		paramOffset = 1
	}

	ptrArgIdx := 0
	for _, effect := range mutationEffects {
		var argValue ssa.Value

		switch effect.Type {
		case taxonomy.ReceiverMutation:
			if len(args) > 0 {
				argValue = args[0]
			}
		case taxonomy.PointerArgMutation:
			argIdx := paramOffset + ptrArgIdx
			if argIdx < len(args) {
				argValue = args[argIdx]
			}
			ptrArgIdx++
		}

		if argValue == nil {
			continue
		}

		// Resolve the SSA argument value to its source-level
		// types.Object. SSA parameters and free variables have
		// positions; for allocs, follow to the defining ident.
		resolveSSAValueToObj(argValue, effect.ID, objToEffectID, testPkg)
	}
}

// resolveSSAValueToObj maps an SSA value to the types.Object of its
// source-level variable by using the value's source position to find
// the corresponding identifier in TypesInfo.
func resolveSSAValueToObj(
	v ssa.Value,
	effectID string,
	objToEffectID map[types.Object]string,
	testPkg *packages.Package,
) {
	if v == nil || testPkg == nil || testPkg.TypesInfo == nil {
		return
	}

	pos := v.Pos()
	if !pos.IsValid() {
		// For values without position (e.g., implicit allocs),
		// try to find the underlying named value.
		if unop, ok := v.(*ssa.UnOp); ok {
			resolveSSAValueToObj(unop.X, effectID, objToEffectID, testPkg)
		}
		return
	}

	// Look up the identifier defined or used at this position.
	for ident, obj := range testPkg.TypesInfo.Defs {
		if obj != nil && ident.Pos() == pos {
			objToEffectID[obj] = effectID
			return
		}
	}
	for ident, obj := range testPkg.TypesInfo.Uses {
		if ident.Pos() == pos {
			objToEffectID[obj] = effectID
			return
		}
	}
}

// hasReceiverMutation checks if any mutation effect is a receiver mutation.
func hasReceiverMutation(effects []taxonomy.SideEffect) bool {
	for _, e := range effects {
		if e.Type == taxonomy.ReceiverMutation {
			return true
		}
	}
	return false
}

// resolveExprRoot recursively unwinds expression wrappers to find the
// root *ast.Ident. It handles selector access (result.Field), index
// access (results[0]), and value-inspecting built-in calls (len(x),
// cap(x)). Returns nil if the expression cannot be unwound to an
// identifier.
//
// Resolution rules:
//   - *ast.Ident: return directly (base case)
//   - *ast.SelectorExpr: recurse on .X (e.g., result.Field -> result)
//   - *ast.IndexExpr: recurse on .X (e.g., results[0] -> results)
//   - *ast.TypeAssertExpr: recurse on .X (e.g., x.(T) -> x)
//   - *ast.CallExpr: if Fun is a *types.Builtin with name "len" or
//     "cap" and exactly 1 argument, recurse on Args[0]
//   - All other types: return nil
//
// Stack depth is bounded by Go source expression nesting (typically <= 5).
func resolveExprRoot(expr ast.Expr, info *types.Info) *ast.Ident {
	switch e := expr.(type) {
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return resolveExprRoot(e.X, info)
	case *ast.IndexExpr:
		return resolveExprRoot(e.X, info)
	case *ast.TypeAssertExpr:
		return resolveExprRoot(e.X, info)
	case *ast.CallExpr:
		// Only unwind value-inspecting built-in calls: len, cap.
		// Side-effecting built-ins (append, delete, etc.) are rejected.
		if len(e.Args) != 1 {
			return nil
		}
		funIdent, ok := e.Fun.(*ast.Ident)
		if !ok {
			return nil
		}
		if info == nil {
			return nil
		}
		obj := info.Uses[funIdent]
		builtin, ok := obj.(*types.Builtin)
		if !ok {
			return nil
		}
		switch builtin.Name() {
		case "len", "cap":
			return resolveExprRoot(e.Args[0], info)
		default:
			return nil
		}
	default:
		return nil
	}
}

// containerUnwrapConfidence is the confidence level for assertions
// mapped via the container unwrap pass. It slots between inline call
// matching (60) and AI-assisted mapping (50), reflecting the additional
// indirection through field access and transformation calls.
const containerUnwrapConfidence = 55

// maxContainerChainDepth is the maximum number of forward-tracing
// iterations for the container unwrap pass. This covers the full MCP
// test pattern (result → field → type assert → field → unmarshal →
// assert) with margin for more complex chains.
const maxContainerChainDepth = 6

// isByteLikeParam reports whether typ represents a byte-like input type
// suitable for transformation call detection. It recognizes three patterns:
//   - []byte: a slice with a byte element type
//   - string: a basic type with String kind
//   - io.Reader: an interface whose method set includes a Read method
func isByteLikeParam(typ types.Type) bool {
	// Check for []byte: *types.Slice with byte element.
	if sl, ok := typ.(*types.Slice); ok {
		if basic, ok := sl.Elem().(*types.Basic); ok && basic.Kind() == types.Byte {
			return true
		}
	}

	// Check for string: *types.Basic with Kind() == types.String.
	if basic, ok := typ.(*types.Basic); ok && basic.Kind() == types.String {
		return true
	}

	// Check for io.Reader: interface with Read method.
	if iface, ok := typ.Underlying().(*types.Interface); ok {
		ms := types.NewMethodSet(iface)
		for i := 0; i < ms.Len(); i++ {
			if ms.At(i).Obj().Name() == "Read" {
				return true
			}
		}
	}

	return false
}

// isPointerDestParam reports whether typ represents a pointer destination
// type suitable for transformation call detection. It recognizes two patterns:
//   - *T: a pointer type
//   - interface{}/any: an empty interface (zero methods), commonly used as
//     the destination parameter in unmarshal functions
func isPointerDestParam(typ types.Type) bool {
	// Check for pointer type: *types.Pointer.
	if _, ok := typ.(*types.Pointer); ok {
		return true
	}

	// Check for empty interface (any/interface{}) via Underlying
	// to handle named types.
	if iface, ok := typ.Underlying().(*types.Interface); ok {
		return iface.NumMethods() == 0
	}

	return false
}

// resolveCallSignature extracts the function signature from a call
// expression using type information. Returns nil if the call or info
// is nil, or if the function type cannot be resolved to a signature.
func resolveCallSignature(call *ast.CallExpr, info *types.Info) *types.Signature {
	if call == nil || info == nil {
		return nil
	}
	tv, exists := info.Types[call.Fun]
	if !exists {
		return nil
	}
	sig, ok := tv.Type.(*types.Signature)
	if !ok {
		return nil
	}
	return sig
}

// findParamIndex returns the index of the first parameter matching the
// predicate, or -1 if none match.
func findParamIndex(params *types.Tuple, predicate func(types.Type) bool) int {
	for i := 0; i < params.Len(); i++ {
		if predicate(params.At(i).Type()) {
			return i
		}
	}
	return -1
}

// isTransformationCall checks whether a call expression matches the
// structural signature pattern of a transformation function: a function
// that accepts a byte-like input ([]byte, string, or io.Reader) AND a
// pointer destination argument. Returns the positional indices of the
// byte-like and pointer parameters, or ok=false if both are not found.
//
// This enables structural detection of unmarshal-like functions without
// hardcoding specific function names (FR-001, FR-008).
func isTransformationCall(call *ast.CallExpr, info *types.Info) (byteArgIdx int, ptrDestIdx int, ok bool) {
	sig := resolveCallSignature(call, info)
	if sig == nil {
		return 0, 0, false
	}

	params := sig.Params()
	if params == nil {
		return 0, 0, false
	}

	byteArgIdx = findParamIndex(params, isByteLikeParam)
	ptrDestIdx = findParamIndex(params, isPointerDestParam)

	if byteArgIdx >= 0 && ptrDestIdx >= 0 {
		return byteArgIdx, ptrDestIdx, true
	}
	return 0, 0, false
}

// containsObject checks whether an AST expression tree contains any
// identifier that resolves to the given types.Object via pointer
// identity. It walks the expression with ast.Inspect and short-circuits
// on the first match.
func containsObject(expr ast.Expr, target types.Object, info *types.Info) bool {
	if expr == nil || target == nil || info == nil {
		return false
	}

	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		obj := info.Uses[ident]
		if obj == nil {
			obj = info.Defs[ident]
		}
		if obj == target {
			found = true
			return false
		}
		return true
	})
	return found
}

// extractPointerDest extracts the base variable from the pointer
// argument of a transformation call. Given a call like
// json.Unmarshal(body, &data), it unwraps the &data UnaryExpr to
// find the underlying identifier and returns its types.Object.
// Returns nil if the argument is not an addressable identifier.
func extractPointerDest(call *ast.CallExpr, ptrIdx int, info *types.Info) types.Object {
	if call == nil || info == nil || ptrIdx < 0 || ptrIdx >= len(call.Args) {
		return nil
	}

	arg := call.Args[ptrIdx]

	// Unwrap &data -> data.
	if unary, ok := arg.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		if ident, ok := unary.X.(*ast.Ident); ok {
			obj := info.Uses[ident]
			if obj == nil {
				obj = info.Defs[ident]
			}
			return obj
		}
	}

	// Handle bare identifier (already a pointer variable).
	if ident, ok := arg.(*ast.Ident); ok {
		obj := info.Uses[ident]
		if obj == nil {
			obj = info.Defs[ident]
		}
		return obj
	}

	return nil
}

// isDataExtraction checks whether an expression is a data-extraction
// pattern: field access (x.Field), index access (x[i]), type assertion
// (x.(T)), or type conversion (T(x)). These are the patterns that
// extract data from a container without side effects. Method calls
// and function calls are excluded to prevent false positives from
// patterns like s.Get("key").
func isDataExtraction(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// x.Field — field access (not a method call; method calls
		// appear as CallExpr with SelectorExpr as Fun).
		return true
	case *ast.IndexExpr:
		// x[i] — index access.
		return true
	case *ast.TypeAssertExpr:
		// x.(T) — type assertion.
		return true
	case *ast.CallExpr:
		// Type conversions look like function calls in the AST:
		// []byte(x), string(x), int(x). They have exactly one
		// argument and the function is a type expression.
		// Exclude method calls (SelectorExpr as Fun) and
		// multi-argument function calls.
		if len(e.Args) == 1 {
			switch e.Fun.(type) {
			case *ast.Ident:
				// Could be a type conversion like string(x) or
				// int(x). SelectorExpr is excluded because it
				// could be a method call like s.Get(x).
				return true
			case *ast.ArrayType:
				// []byte(x) — slice type conversion.
				return true
			}
		}
		return false
	case *ast.ParenExpr:
		return isDataExtraction(e.X)
	default:
		return false
	}
}

// collectTrackedVars filters objToEffectID entries whose value matches
// returnEffectID and returns a set of the matching types.Object keys.
// This is the initial "seed" set for forward data-flow tracing —
// variables that hold the return value of interest.
func collectTrackedVars(objToEffectID map[types.Object]string, returnEffectID string) map[types.Object]bool {
	tracked := make(map[types.Object]bool)
	for obj, effectID := range objToEffectID {
		if effectID == returnEffectID {
			tracked[obj] = true
		}
	}
	return tracked
}

// traceForwardDataFlow performs multi-iteration forward data-flow
// tracing through assignment statements in the test package AST.
// Starting from the tracked set of variables, it walks assignments
// looking for RHS expressions that reference tracked variables. For
// transformation calls (detected via isTransformationCall), it
// extracts the pointer destination as a new tracked variable. For
// non-transformation assignments, it gates on isDataExtraction to
// prevent false positives from method calls like s.Get("key").
// Iterations continue until convergence (no new variables discovered)
// or maxContainerChainDepth is reached. The input tracked map is
// mutated in place and returned with any newly discovered entries.
func traceForwardDataFlow(tracked map[types.Object]bool, testPkg *packages.Package) map[types.Object]bool {
	if len(tracked) == 0 {
		return tracked
	}

	info := testPkg.TypesInfo

	for iter := 0; iter < maxContainerChainDepth; iter++ {
		newTracked := make(map[types.Object]bool)

		for _, file := range testPkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				assign, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}

				for rhsIdx, rhs := range assign.Rhs {
					if !rhsReferencesAnyTracked(rhs, tracked, info) {
						continue
					}

					// Check if the RHS contains a transformation call.
					dest, handled := handleTransformationCalls(rhs, tracked, info)
					if handled {
						if dest != nil {
							newTracked[dest] = true
						}
						continue
					}

					// Non-transformation assignment: only track LHS
					// when the RHS is a data-extraction expression.
					if !isDataExtraction(rhs) {
						continue
					}
					if lhsObj := extractDataFlowLHS(assign, rhsIdx, info); lhsObj != nil {
						newTracked[lhsObj] = true
					}
				}
				return true
			})
		}

		if len(newTracked) == 0 {
			break
		}

		// Merge new tracked variables into the tracked set.
		for obj := range newTracked {
			tracked[obj] = true
		}
	}

	return tracked
}

// rhsReferencesAnyTracked checks whether any tracked variable appears in the
// RHS expression, either via direct containsObject or resolveExprRoot fallback
// for compound expressions like result.Content[0].Text.
func rhsReferencesAnyTracked(rhs ast.Expr, tracked map[types.Object]bool, info *types.Info) bool {
	for obj := range tracked {
		if containsObject(rhs, obj, info) {
			return true
		}
	}
	// Also check via resolveExprRoot for compound expressions.
	root := resolveExprRoot(rhs, info)
	if root != nil {
		rootObj := info.Uses[root]
		if rootObj == nil {
			rootObj = info.Defs[root]
		}
		if rootObj != nil && tracked[rootObj] {
			return true
		}
	}
	return false
}

// handleTransformationCalls inspects the RHS for transformation calls
// (e.g., json.Unmarshal(data, &target)) where a tracked variable flows
// into the call arguments. Returns the pointer destination and whether
// a transformation was fully resolved — i.e., a qualifying call was found
// AND its pointer destination was successfully extracted. When handled is
// false, the caller falls through to the isDataExtraction path.
func handleTransformationCalls(rhs ast.Expr, tracked map[types.Object]bool, info *types.Info) (types.Object, bool) {
	var dest types.Object
	handled := false
	ast.Inspect(rhs, func(cn ast.Node) bool {
		if handled {
			return false
		}
		call, ok := cn.(*ast.CallExpr)
		if !ok {
			return true
		}
		_, ptrIdx, isTransform := isTransformationCall(call, info)
		if !isTransform {
			return true
		}
		// Verify a tracked variable flows into this call.
		callHasTracked := false
		for _, arg := range call.Args {
			for obj := range tracked {
				if containsObject(arg, obj, info) {
					callHasTracked = true
					break
				}
			}
			if callHasTracked {
				break
			}
		}
		if !callHasTracked {
			return true
		}
		d := extractPointerDest(call, ptrIdx, info)
		if d != nil {
			dest = d
			handled = true
		}
		return false
	})
	return dest, handled
}

// extractDataFlowLHS extracts the LHS variable from an assignment at the
// given RHS index. Returns nil for blank identifiers, non-ident LHS, or
// out-of-range indices.
func extractDataFlowLHS(assign *ast.AssignStmt, rhsIdx int, info *types.Info) types.Object {
	if rhsIdx >= len(assign.Lhs) {
		return nil
	}
	ident, ok := assign.Lhs[rhsIdx].(*ast.Ident)
	if !ok || ident.Name == "_" {
		return nil
	}
	obj := info.Defs[ident]
	if obj == nil {
		obj = info.Uses[ident]
	}
	return obj
}

// matchTrackedInExpr walks the expression tree via ast.Inspect and
// returns true if any identifier's types.Object is in the tracked set.
// It checks both direct identity (via info.Uses and info.Defs) and
// resolveExprRoot fallback for composite expressions like data["key"]
// where the root ident "data" is the tracked variable.
func matchTrackedInExpr(expr ast.Expr, tracked map[types.Object]bool, info *types.Info) bool {
	var matched bool
	ast.Inspect(expr, func(n ast.Node) bool {
		if matched {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		obj := info.Uses[ident]
		if obj == nil {
			obj = info.Defs[ident]
		}
		if obj != nil && tracked[obj] {
			matched = true
			return false
		}
		return true
	})

	// Also check via resolveExprRoot for compound expressions
	// like data["key"] where the root ident is "data".
	if !matched {
		root := resolveExprRoot(expr, info)
		if root != nil {
			rootObj := info.Uses[root]
			if rootObj == nil {
				rootObj = info.Defs[root]
			}
			if rootObj != nil && tracked[rootObj] {
				matched = true
			}
		}
	}

	return matched
}

// matchContainerUnwrap traces data flow forward from the return value
// variable through intermediate assignments and transformation calls
// to the assertion expression. This handles the container-unwrap-assert
// pattern where a test assigns a function's return value, accesses a
// field, passes it through a transformation (like JSON unmarshal), and
// asserts on the result.
//
// The algorithm delegates to three helpers:
//  1. collectTrackedVars — seeds the tracked set from objToEffectID.
//  2. traceForwardDataFlow — expands the set through AST assignments.
//  3. matchTrackedInExpr — checks the assertion expression for matches.
//
// If matched, returns an AssertionMapping with confidence 55.
func matchContainerUnwrap(
	site AssertionSite,
	objToEffectID map[types.Object]string,
	effectMap map[string]*taxonomy.SideEffect,
	testPkg *packages.Package,
	returnEffectID string,
) *taxonomy.AssertionMapping {
	if site.Expr == nil || testPkg == nil || testPkg.TypesInfo == nil || returnEffectID == "" {
		return nil
	}

	info := testPkg.TypesInfo

	tracked := collectTrackedVars(objToEffectID, returnEffectID)
	if len(tracked) == 0 {
		return nil
	}

	tracked = traceForwardDataFlow(tracked, testPkg)

	effect := effectMap[returnEffectID]
	if effect == nil {
		return nil
	}

	if matchTrackedInExpr(site.Expr, tracked, info) {
		return &taxonomy.AssertionMapping{
			AssertionLocation: site.Location,
			AssertionType:     mapKindToType(site.Kind),
			SideEffectID:      returnEffectID,
			Confidence:        containerUnwrapConfidence,
		}
	}

	return nil
}

// matchDirect walks the assertion expression tree via ast.Inspect looking
// for *ast.Ident nodes whose types.Object is directly in objToEffectID
// (confidence 75) or reachable through helperBridge → objToEffectID
// (confidence 70). It skips nil/true/false literal identifiers and
// returns the first match found, or nil if no match exists.
//
// Because ast.Inspect visits all descendant nodes, this handles simple
// selector expressions (e.g., result.Name) — the root ident "result"
// is visited as a child of the SelectorExpr and matched directly.
func matchDirect(
	site AssertionSite,
	objToEffectID map[types.Object]string,
	effectMap map[string]*taxonomy.SideEffect,
	info *types.Info,
	helperBridge map[types.Object]types.Object,
) *taxonomy.AssertionMapping {
	if site.Expr == nil {
		return nil
	}

	var matched *taxonomy.AssertionMapping
	ast.Inspect(site.Expr, func(n ast.Node) bool {
		if matched != nil {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		// Skip nil/true/false literals.
		if ident.Name == "nil" || ident.Name == "true" || ident.Name == "false" {
			return true
		}

		// Look up the types.Object this identifier refers to.
		obj := info.Uses[ident]
		if obj == nil {
			// Also check Defs — some assertions use the
			// defining occurrence (e.g., diff := cmp.Diff(...)).
			obj = info.Defs[ident]
		}
		if obj == nil {
			return true
		}

		// Check direct match first.
		if effectID, ok := objToEffectID[obj]; ok {
			effect := effectMap[effectID]
			if effect == nil {
				return true
			}
			matched = &taxonomy.AssertionMapping{
				AssertionLocation: site.Location,
				AssertionType:     mapKindToType(site.Kind),
				SideEffectID:      effectID,
				Confidence:        75, // SSA-traced direct match
			}
			return false
		}
		// Check helper parameter bridge — if this identifier is
		// a helper parameter, resolve it to the caller's argument
		// and check objToEffectID with the caller's object.
		if callerObj, ok := helperBridge[obj]; ok {
			if effectID, ok := objToEffectID[callerObj]; ok {
				effect := effectMap[effectID]
				if effect == nil {
					return true
				}
				matched = &taxonomy.AssertionMapping{
					AssertionLocation: site.Location,
					AssertionType:     mapKindToType(site.Kind),
					SideEffectID:      effectID,
					Confidence:        70, // Helper parameter bridge
				}
				return false
			}
		}
		return true
	})

	return matched
}

// matchIndirectRoot walks the assertion expression tree looking for
// composite expression nodes (SelectorExpr, IndexExpr, CallExpr) and
// resolves each to its root identifier via resolveExprRoot. If the
// root's types.Object is in objToEffectID, it returns a match at
// confidence 65. Returns nil if no composite expression resolves to
// a tracked object.
//
// This handles cases where the root ident is not directly reachable by
// ast.Inspect as a bare *ast.Ident — e.g., index expressions
// (results[0]) or nested composites where the root is buried inside
// a complex expression structure.
func matchIndirectRoot(
	site AssertionSite,
	objToEffectID map[types.Object]string,
	effectMap map[string]*taxonomy.SideEffect,
	info *types.Info,
) *taxonomy.AssertionMapping {
	if site.Expr == nil {
		return nil
	}

	var matched *taxonomy.AssertionMapping
	ast.Inspect(site.Expr, func(n ast.Node) bool {
		if matched != nil {
			return false
		}

		// Only process composite expression nodes that wrap an
		// identifier — skip bare Idents (already handled in Pass 1).
		expr, ok := n.(ast.Expr)
		if !ok {
			return true
		}
		switch expr.(type) {
		case *ast.SelectorExpr, *ast.IndexExpr, *ast.CallExpr:
			// Proceed with resolution.
		default:
			return true
		}

		root := resolveExprRoot(expr, info)
		if root == nil {
			return true
		}

		obj := info.Uses[root]
		if obj == nil {
			obj = info.Defs[root]
		}
		if obj == nil {
			return true
		}

		if effectID, ok := objToEffectID[obj]; ok {
			effect := effectMap[effectID]
			if effect == nil {
				return true
			}
			matched = &taxonomy.AssertionMapping{
				AssertionLocation: site.Location,
				AssertionType:     mapKindToType(site.Kind),
				SideEffectID:      effectID,
				Confidence:        65, // Indirect root resolution match
			}
			return false
		}
		return true
	})

	return matched
}

// matchAssertionToEffect attempts to match an assertion site to a
// traced side effect value using types.Object identity.
//
// It uses a two-pass matching strategy:
//
// Pass 1 (direct via matchDirect): Walk the expression tree looking
// for *ast.Ident nodes whose types.Object is directly in objToEffectID
// (confidence 75) or reachable through the helper bridge (confidence 70).
//
// Pass 2 (indirect via matchIndirectRoot): If Pass 1 found no match,
// walk the expression tree for composite nodes (SelectorExpr, IndexExpr,
// CallExpr), resolve to root identifiers, and check against objToEffectID
// (confidence 65).
//
// Pass 1 always executes first so direct identity matches are never
// degraded by indirect resolution.
func matchAssertionToEffect(
	site AssertionSite,
	objToEffectID map[types.Object]string,
	effectMap map[string]*taxonomy.SideEffect,
	testPkg *packages.Package,
) *taxonomy.AssertionMapping {
	if site.Expr == nil {
		return nil
	}

	if testPkg == nil {
		return nil
	}
	info := testPkg.TypesInfo
	if info == nil {
		return nil
	}

	// Build supplemental param→arg bridging for helper assertions.
	helperBridge := buildHelperBridge(site, objToEffectID, info)

	if m := matchDirect(site, objToEffectID, effectMap, info, helperBridge); m != nil {
		return m
	}
	return matchIndirectRoot(site, objToEffectID, effectMap, info)
}

// matchInlineCall checks if the assertion expression contains a direct
// call to the target function (e.g., `if c.Value() != 5`). When the
// return value is used inline without assignment, the normal tracing
// pipeline can't map it because there's no types.Object in objToEffectID.
// This fallback detects the call by comparing the callee's name and
// package with the target function.
func matchInlineCall(
	site AssertionSite,
	targetFunc *ssa.Function,
	returnEffectID string,
	effectMap map[string]*taxonomy.SideEffect,
	testPkg *packages.Package,
) *taxonomy.AssertionMapping {
	if site.Expr == nil || targetFunc == nil {
		return nil
	}

	info := testPkg.TypesInfo
	if info == nil {
		return nil
	}

	targetName := targetFunc.Name()
	targetPkg := ""
	if targetFunc.Package() != nil {
		targetPkg = targetFunc.Package().Pkg.Path()
	}

	var found bool
	ast.Inspect(site.Expr, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Resolve the callee name.
		var calleeName, calleePkg string
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			calleeName = fn.Name
			if obj := info.Uses[fn]; obj != nil && obj.Pkg() != nil {
				calleePkg = obj.Pkg().Path()
			}
		case *ast.SelectorExpr:
			calleeName = fn.Sel.Name
			if sel, ok := info.Selections[fn]; ok && sel.Obj().Pkg() != nil {
				calleePkg = sel.Obj().Pkg().Path()
			}
		}

		if calleeName == targetName && calleePkg == targetPkg {
			found = true
		}
		return !found
	})

	if !found {
		return nil
	}

	effect := effectMap[returnEffectID]
	if effect == nil {
		return nil
	}

	return &taxonomy.AssertionMapping{
		AssertionLocation: site.Location,
		AssertionType:     mapKindToType(site.Kind),
		SideEffectID:      returnEffectID,
		Confidence:        60, // Inline call match (lower confidence)
	}
}

// buildHelperBridge constructs a map from helper function parameter
// objects to the caller's argument objects. This enables bridging
// assertions inside helper functions back to the test's variables.
//
// For example, given:
//
//	func assertEqual(t *testing.T, got, want int) {
//	    if got != want { t.Errorf(...) }
//	}
//	// in test body:
//	assertEqual(t, result, 42)
//
// The bridge maps: helper's "got" param → test's "result" arg.
// Only argument positions that have identifiers resolvable via
// TypesInfo are included.
func buildHelperBridge(
	site AssertionSite,
	objToEffectID map[types.Object]string,
	info *types.Info,
) map[types.Object]types.Object {
	if site.Depth == 0 || len(site.CallerArgs) == 0 || site.FuncDecl == nil {
		return nil
	}

	params := site.FuncDecl.Type.Params
	if params == nil {
		return nil
	}

	bridge := make(map[types.Object]types.Object)
	argIdx := 0

	for _, field := range params.List {
		for _, name := range field.Names {
			if argIdx >= len(site.CallerArgs) {
				return bridge
			}
			arg := site.CallerArgs[argIdx]
			argIdx++

			// Resolve the param's types.Object.
			paramObj := info.Defs[name]
			if paramObj == nil {
				continue
			}

			// Resolve the caller argument's types.Object.
			argIdent, ok := arg.(*ast.Ident)
			if !ok {
				continue // only handle simple identifiers
			}
			argObj := info.Uses[argIdent]
			if argObj == nil {
				argObj = info.Defs[argIdent]
			}
			if argObj == nil {
				continue
			}

			// Only bridge if the caller's arg object is in
			// objToEffectID — otherwise the bridge is useless.
			if _, ok := objToEffectID[argObj]; ok {
				bridge[paramObj] = argObj
			}
		}
	}

	return bridge
}

// filterEffectsByType returns effects matching any of the given types.
func filterEffectsByType(
	effects []taxonomy.SideEffect,
	types ...taxonomy.SideEffectType,
) []taxonomy.SideEffect {
	typeSet := make(map[taxonomy.SideEffectType]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}

	var filtered []taxonomy.SideEffect
	for _, e := range effects {
		if typeSet[e.Type] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// sameFunction checks whether two SSA functions refer to the same
// source function by comparing both name and package path. This
// avoids false matches when different packages have functions with
// the same name (e.g., mypackage.Parse vs strconv.Parse).
func sameFunction(a, b *ssa.Function) bool {
	if a.Name() != b.Name() {
		return false
	}
	aPkg := a.Package()
	bPkg := b.Package()
	if aPkg == nil || bPkg == nil {
		return false
	}
	return aPkg.Pkg.Path() == bPkg.Pkg.Path()
}

// classifyUnmappedReason determines why an assertion site could not be
// mapped to a side effect. It uses three signals:
//
//  1. site.Depth > 0 → the assertion is inside a helper body;
//     helper parameters cannot be traced back to the test's variables.
//
//  2. depth == 0, objToEffectID is empty, AND the target has at least
//     one ReturnValue or ErrorReturn effect → the target was likely
//     called inline without assigning the return value
//     (e.g., "if f() != x"). traceReturnValues only handles assignments.
//
//  3. All other cases → no side effect object matched the assertion
//     identifiers. Typically a cross-target assertion or an unsupported
//     assertion pattern.
func classifyUnmappedReason(
	site AssertionSite,
	objToEffectID map[types.Object]string,
	effects []taxonomy.SideEffect,
) taxonomy.UnmappedReasonType {
	// Cause A: assertion is inside a helper body.
	if site.Depth > 0 {
		return taxonomy.UnmappedReasonHelperParam
	}

	// Cause B: return values were not traced because the call was inline.
	// Heuristic: no traced objects AND target has return/error effects.
	if len(objToEffectID) == 0 && hasReturnEffects(effects) {
		return taxonomy.UnmappedReasonInlineCall
	}

	// Cause C: general no-match case.
	return taxonomy.UnmappedReasonNoEffectMatch
}

// hasReturnEffects reports whether the effect list contains at least one
// ReturnValue or ErrorReturn effect. Used by classifyUnmappedReason to
// distinguish inline-call unmapping from other no-match cases.
func hasReturnEffects(effects []taxonomy.SideEffect) bool {
	for _, e := range effects {
		if e.Type == taxonomy.ReturnValue || e.Type == taxonomy.ErrorReturn {
			return true
		}
	}
	return false
}

// mapKindToType converts an AssertionKind to an AssertionType for
// the taxonomy mapping struct.
func mapKindToType(kind AssertionKind) taxonomy.AssertionType {
	switch kind {
	case AssertionKindStdlibComparison:
		return taxonomy.AssertionEquality
	case AssertionKindStdlibErrorCheck:
		return taxonomy.AssertionErrorCheck
	case AssertionKindTestifyEqual:
		return taxonomy.AssertionEquality
	case AssertionKindTestifyNoError:
		return taxonomy.AssertionErrorCheck
	case AssertionKindTestifyNilCheck:
		return taxonomy.AssertionNilCheck
	case AssertionKindGoCmpDiff:
		return taxonomy.AssertionDiffCheck
	default:
		return taxonomy.AssertionCustom
	}
}
