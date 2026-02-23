// Package quality computes test quality metrics by mapping test
// assertions to detected side effects.
package quality

import (
	"go/ast"

	"golang.org/x/tools/go/ssa"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// MapAssertionsToEffects maps detected assertion sites to side effects
// using SSA data flow analysis. It traces return values and mutations
// from the target call site through the test function to assertion
// sites.
//
// Assertions that cannot be linked to a specific side effect are
// reported as unmapped per the spec (FR-003): they are excluded from
// both Contract Coverage and Over-Specification metrics.
//
// It returns two slices: mapped assertions (linked to a side effect)
// and unmapped assertions (could not be linked to any effect).
func MapAssertionsToEffects(
	testFunc *ssa.Function,
	targetFunc *ssa.Function,
	sites []AssertionSite,
	effects []taxonomy.SideEffect,
) ([]taxonomy.AssertionMapping, []taxonomy.AssertionMapping) {
	if len(sites) == 0 || len(effects) == 0 {
		// If no assertions or no effects, everything is unmapped.
		unmapped := make([]taxonomy.AssertionMapping, 0, len(sites))
		for _, s := range sites {
			unmapped = append(unmapped, taxonomy.AssertionMapping{
				AssertionLocation: s.Location,
				AssertionType:     mapKindToType(s.Kind),
				Confidence:        0,
			})
		}
		return nil, unmapped
	}

	// Find the call to the target function in the test SSA.
	targetCall := FindTargetCall(testFunc, targetFunc)

	// Build a map from side effect ID to side effect for matching.
	effectMap := make(map[string]*taxonomy.SideEffect, len(effects))
	for i := range effects {
		effectMap[effects[i].ID] = &effects[i]
	}

	// Trace values from the target call to assertion sites.
	// This builds a map from SSA value name to effect ID.
	valueNameToEffectID := traceTargetValues(targetCall, effects, testFunc)

	// Match assertion expressions to traced values.
	var mapped, unmapped []taxonomy.AssertionMapping

	for _, site := range sites {
		mapping := matchAssertionToEffect(site, valueNameToEffectID, effectMap)
		if mapping != nil {
			mapped = append(mapped, *mapping)
		} else {
			// Per spec FR-003: unmapped assertions are reported
			// separately and excluded from metrics.
			unmapped = append(unmapped, taxonomy.AssertionMapping{
				AssertionLocation: site.Location,
				AssertionType:     mapKindToType(site.Kind),
				Confidence:        0,
			})
		}
	}

	return mapped, unmapped
}

// FindTargetCall finds the SSA call instruction in the test function
// that calls the target function. It handles both direct calls and
// method calls.
func FindTargetCall(
	testFunc *ssa.Function,
	targetFunc *ssa.Function,
) *ssa.Call {
	if testFunc == nil || testFunc.Blocks == nil || targetFunc == nil {
		return nil
	}

	for _, block := range testFunc.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			callee := call.Call.StaticCallee()
			if callee == nil {
				continue
			}
			if callee == targetFunc || callee.Name() == targetFunc.Name() {
				return call
			}
		}
	}
	return nil
}

// traceTargetValues traces return values and mutation effects from
// the target call through the SSA graph. It returns a map from SSA
// value name to the side effect ID it originates from.
func traceTargetValues(
	targetCall *ssa.Call,
	effects []taxonomy.SideEffect,
	testFunc *ssa.Function,
) map[string]string {
	valueNameToEffectID := make(map[string]string)

	if targetCall == nil {
		return valueNameToEffectID
	}

	// Trace return values.
	traceReturnValues(targetCall, effects, valueNameToEffectID)

	// Trace mutations (receiver and pointer arg values read after
	// the call).
	traceMutations(targetCall, effects, testFunc, valueNameToEffectID)

	return valueNameToEffectID
}

// traceReturnValues traces the return values from a target call
// through Extract instructions and subsequent uses.
func traceReturnValues(
	targetCall *ssa.Call,
	effects []taxonomy.SideEffect,
	valueNameToEffectID map[string]string,
) {
	// Find return-type side effects.
	returnEffects := filterEffectsByType(effects,
		taxonomy.ReturnValue, taxonomy.ErrorReturn)

	if len(returnEffects) == 0 {
		return
	}

	// The call result is used either directly (single return) or
	// via Extract instructions (multi-return).
	callValue := ssa.Value(targetCall)

	// For single-return functions, the call value itself maps to
	// the first return effect.
	if len(returnEffects) == 1 {
		valueNameToEffectID[callValue.Name()] = returnEffects[0].ID
	}

	// Trace through Extract instructions for multi-return.
	referrers := targetCall.Referrers()
	if referrers == nil {
		return
	}
	for _, ref := range *referrers {
		extract, ok := ref.(*ssa.Extract)
		if !ok {
			continue
		}
		idx := extract.Index
		if idx < len(returnEffects) {
			valueNameToEffectID[extract.Name()] = returnEffects[idx].ID

			// Follow further uses of the extracted value.
			traceUses(extract, returnEffects[idx].ID, valueNameToEffectID, make(map[ssa.Value]bool))
		}
	}

	// Also trace direct uses of a single-return call.
	if len(returnEffects) == 1 {
		traceUses(callValue, returnEffects[0].ID, valueNameToEffectID, make(map[ssa.Value]bool))
	}
}

// traceMutations traces mutation side effects by identifying values
// passed as receiver or pointer arguments at the target call site,
// then finding reads of those values after the call.
//
// For methods, the SSA calling convention places the receiver at
// args[0] and explicit parameters starting at args[1]. This function
// separates receiver mutations from pointer argument mutations to
// avoid index misalignment.
func traceMutations(
	targetCall *ssa.Call,
	effects []taxonomy.SideEffect,
	_ *ssa.Function, // testFunc reserved for future use
	valueNameToEffectID map[string]string,
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
	// For methods, args[0] is the receiver; explicit params start at 1.
	// For functions, all args are explicit params starting at 0.
	paramOffset := 0
	if isMethod {
		paramOffset = 1
	}

	ptrArgIdx := 0 // tracks position within pointer arg mutations
	for _, effect := range mutationEffects {
		var argValue ssa.Value

		switch effect.Type {
		case taxonomy.ReceiverMutation:
			// Receiver is always args[0] for methods.
			if len(args) > 0 {
				argValue = args[0]
			}
		case taxonomy.PointerArgMutation:
			// Pointer args start after the receiver (if method).
			argIdx := paramOffset + ptrArgIdx
			if argIdx < len(args) {
				argValue = args[argIdx]
			}
			ptrArgIdx++
		}

		if argValue == nil {
			continue
		}

		valueNameToEffectID[argValue.Name()] = effect.ID

		// Trace uses of the argument value after the call.
		traceUses(argValue, effect.ID, valueNameToEffectID, make(map[ssa.Value]bool))
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

// traceUses follows SSA value edges (phi, store, load, field addr)
// to find all reachable uses of a value. It populates the
// valueNameToEffectID map with each derived value's name.
func traceUses(
	v ssa.Value,
	effectID string,
	valueNameToEffectID map[string]string,
	visited map[ssa.Value]bool,
) {
	if v == nil || visited[v] {
		return
	}
	visited[v] = true

	referrers := v.Referrers()
	if referrers == nil {
		return
	}

	for _, ref := range *referrers {
		switch instr := ref.(type) {
		case *ssa.FieldAddr:
			valueNameToEffectID[instr.Name()] = effectID
			traceUses(instr, effectID, valueNameToEffectID, visited)

		case *ssa.IndexAddr:
			valueNameToEffectID[instr.Name()] = effectID
			traceUses(instr, effectID, valueNameToEffectID, visited)

		case *ssa.UnOp:
			valueNameToEffectID[instr.Name()] = effectID
			traceUses(instr, effectID, valueNameToEffectID, visited)

		case *ssa.Phi:
			valueNameToEffectID[instr.Name()] = effectID
			traceUses(instr, effectID, valueNameToEffectID, visited)

		case *ssa.Store:
			// The stored value may be read later.
			valueNameToEffectID[instr.Addr.Name()] = effectID

		case ssa.Value:
			// Generic value — trace its uses.
			valueNameToEffectID[instr.Name()] = effectID
			traceUses(instr, effectID, valueNameToEffectID, visited)
		}
	}
}

// matchAssertionToEffect attempts to match an assertion site to a
// traced side effect value using exact SSA value name matching.
//
// It extracts variable names from the assertion expression and
// checks for exact matches against traced SSA value names in the
// valueNameToEffectID map.
func matchAssertionToEffect(
	site AssertionSite,
	valueNameToEffectID map[string]string,
	effectMap map[string]*taxonomy.SideEffect,
) *taxonomy.AssertionMapping {
	if site.Expr == nil {
		return nil
	}

	// Extract variable names from the assertion expression.
	varNames := extractVarNames(site.Expr)

	// Check if any variable name exactly matches a traced SSA value.
	for _, name := range varNames {
		if effectID, ok := valueNameToEffectID[name]; ok {
			effect := effectMap[effectID]
			if effect == nil {
				continue
			}
			return &taxonomy.AssertionMapping{
				AssertionLocation: site.Location,
				AssertionType:     mapKindToType(site.Kind),
				SideEffectID:      effectID,
				Confidence:        75, // SSA-traced match
			}
		}
	}

	return nil
}

// extractVarNames extracts identifier names from an AST expression.
func extractVarNames(expr ast.Expr) []string {
	var names []string
	ast.Inspect(expr, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			if ident.Name != "nil" && ident.Name != "true" && ident.Name != "false" {
				names = append(names, ident.Name)
			}
		}
		return true
	})
	return names
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
	case AssertionKindGoCmpDiff:
		return taxonomy.AssertionDiffCheck
	default:
		return taxonomy.AssertionCustom
	}
}
