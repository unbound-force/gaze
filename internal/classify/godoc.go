// Package classify implements the contractual classification engine.
package classify

import (
	"go/ast"
	"strings"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// contractualKeywords are words in godoc comments that suggest
// the described behavior is contractual.
var contractualKeywords = []struct {
	keyword    string
	impliesFor []taxonomy.SideEffectType
}{
	{"returns", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
	{"writes", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation}},
	{"modifies", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation}},
	{"updates", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation}},
	{"sets", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation}},
	{"persists", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation}},
	{"stores", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation}},
	{"deletes", []taxonomy.SideEffectType{taxonomy.ReceiverMutation}},
	{"removes", []taxonomy.SideEffectType{taxonomy.ReceiverMutation}},
}

// incidentalKeywords are words in godoc that suggest incidental
// behavior, scoped to applicable side effect types. Only I/O-related
// effects are penalized — P0 effects like ReturnValue are never demoted
// by logging/debugging keywords. See issue #105.
var incidentalKeywords = []struct {
	keyword   string
	appliesTo []taxonomy.SideEffectType
}{
	{"logs", ioEffectTypes},
	{"prints", ioEffectTypes},
	{"traces", ioEffectTypes},
	{"debugs", ioEffectTypes},
}

// maxGodocWeight is the maximum weight for godoc comment signals.
const maxGodocWeight = 15

// reducedGodocWeight is the signal weight when a contractual keyword is
// found but the effect type does not match the keyword's impliesFor list.
// This provides a positive signal for well-documented functions without
// inflating scores as much as a direct type match.
const reducedGodocWeight = 5

// AnalyzeGodocSignal parses the function's doc comment for
// behavioral declarations and returns a signal indicating whether
// the side effect is likely contractual or incidental.
func AnalyzeGodocSignal(funcDecl *ast.FuncDecl, effectType taxonomy.SideEffectType) taxonomy.Signal {
	if funcDecl == nil || funcDecl.Doc == nil {
		return taxonomy.Signal{}
	}

	docText := strings.ToLower(funcDecl.Doc.Text())

	// Check incidental keywords first (type-guarded — see issue #105).
	for _, ik := range incidentalKeywords {
		if !strings.Contains(docText, ik.keyword) {
			continue
		}
		for _, applicable := range ik.appliesTo {
			if applicable == effectType {
				return taxonomy.Signal{
					Source:    "godoc",
					Weight:    -maxGodocWeight,
					Reasoning: "godoc contains \"" + ik.keyword + "\" suggesting incidental behavior",
				}
			}
		}
	}

	// Check contractual keywords. Track whether any contractual keyword
	// was found, even if the effect type doesn't match the keyword's
	// impliesFor list — a reduced signal is returned in that case.
	foundContractualKeyword := false
	for _, ck := range contractualKeywords {
		if !strings.Contains(docText, ck.keyword) {
			continue
		}
		for _, implied := range ck.impliesFor {
			if implied == effectType {
				return taxonomy.Signal{
					Source:    "godoc",
					Weight:    maxGodocWeight,
					Reasoning: "godoc contains \"" + ck.keyword + "\" declaring " + string(effectType),
				}
			}
		}
		// Keyword found but effect type doesn't match impliesFor.
		foundContractualKeyword = true
	}

	// A contractual keyword was found but the effect type didn't match
	// any keyword's impliesFor list. Return a reduced positive signal
	// to reflect the function's documented contractual intent.
	if foundContractualKeyword {
		return taxonomy.Signal{
			Source:    "godoc_keyword_indirect",
			Weight:    reducedGodocWeight,
			Reasoning: "godoc contains contractual keyword but effect type " + string(effectType) + " is not directly implied",
		}
	}

	return taxonomy.Signal{}
}
