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
	objToEffectID := traceTargetValues(targetCall, effects, testPkg)

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
// The returned map keys are types.Object instances that can be
// matched against assertion operands using TypesInfo.Uses.
func traceTargetValues(
	targetCall *ssa.Call,
	effects []taxonomy.SideEffect,
	testPkg *packages.Package,
) map[types.Object]string {
	objToEffectID := make(map[types.Object]string)

	if targetCall == nil || testPkg == nil || testPkg.TypesInfo == nil {
		return objToEffectID
	}

	// Trace return values by finding the AST assignment.
	traceReturnValues(targetCall, effects, objToEffectID, testPkg)

	// Trace mutations (receiver and pointer arg values).
	traceMutations(targetCall, effects, objToEffectID, testPkg)

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
func traceReturnValues(
	targetCall *ssa.Call,
	effects []taxonomy.SideEffect,
	objToEffectID map[types.Object]string,
	testPkg *packages.Package,
) {
	returnEffects := filterEffectsByType(effects,
		taxonomy.ReturnValue, taxonomy.ErrorReturn)
	if len(returnEffects) == 0 {
		return
	}

	// Find the AST assignment that contains the target call.
	callPos := targetCall.Pos()
	if !callPos.IsValid() {
		return
	}

	assignLHS := findAssignLHS(testPkg, callPos)
	if assignLHS == nil {
		return
	}

	// Map each non-blank LHS identifier to the corresponding
	// return effect by position index.
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

// matchAssertionToEffect attempts to match an assertion site to a
// traced side effect value using types.Object identity.
//
// It resolves identifiers in the assertion expression to their
// types.Object via the package's TypesInfo.Uses map, then checks
// for matches against the objToEffectID map built from AST tracing.
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
				Confidence:        75, // SSA-traced match
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
