package quality

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"

	"github.com/unbound-force/gaze/internal/taxonomy"
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
) (mapped []taxonomy.AssertionMapping, unmapped []taxonomy.AssertionMapping, discardedIDs map[string]bool) {
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

	// Match assertion expressions to traced values.
	for _, site := range sites {
		mapping := matchAssertionToEffect(site, objToEffectID, effectMap, testPkg)
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

// findHelperCallInFunc recursively searches an SSA function and its
// closures for a call to a helper that invokes the target at depth 1.
func findHelperCallInFunc(
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
					if result := findHelperCallInFunc(closureFn, targetFunc, visited); result != nil {
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

// matchAssertionToEffect attempts to match an assertion site to a
// traced side effect value using types.Object identity.
//
// It uses a two-pass matching strategy:
//
// Pass 1 (direct): Walk the expression tree with ast.Inspect looking
// for *ast.Ident nodes whose types.Object is directly in objToEffectID.
// This is the original behavior. Matches produce confidence 75.
//
// Pass 2 (indirect): If Pass 1 found no match, walk the expression
// tree again. For each SelectorExpr, IndexExpr, or CallExpr node,
// call resolveExprRoot to unwind to the root identifier. If the
// root's types.Object is in objToEffectID, produce a match at
// confidence 65.
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

	var info *types.Info
	if testPkg != nil {
		info = testPkg.TypesInfo
	}
	if info == nil {
		return nil
	}

	// Pass 1: Direct identity matching (confidence 75).
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
		return true
	})

	if matched != nil {
		return matched
	}

	// Pass 2: Indirect root resolution (confidence 65).
	// For each composite expression node (SelectorExpr, IndexExpr,
	// CallExpr), resolve to the root identifier and check against
	// the traced object map.
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
