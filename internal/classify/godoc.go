// Package classify implements the contractual classification engine.
package classify

import (
	"go/ast"
	"strings"

	"github.com/unbound-force/gaze/internal/taxonomy"
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
// behavior.
var incidentalKeywords = []string{
	"logs", "prints", "traces", "debugs",
}

// maxGodocWeight is the maximum weight for godoc comment signals.
const maxGodocWeight = 15

// AnalyzeGodocSignal parses the function's doc comment for
// behavioral declarations and returns a signal indicating whether
// the side effect is likely contractual or incidental.
func AnalyzeGodocSignal(funcDecl *ast.FuncDecl, effectType taxonomy.SideEffectType) taxonomy.Signal {
	if funcDecl == nil || funcDecl.Doc == nil {
		return taxonomy.Signal{}
	}

	docText := strings.ToLower(funcDecl.Doc.Text())

	// Check incidental keywords first.
	for _, kw := range incidentalKeywords {
		if strings.Contains(docText, kw) {
			return taxonomy.Signal{
				Source:    "godoc",
				Weight:    -maxGodocWeight,
				Reasoning: "godoc contains \"" + kw + "\" suggesting incidental behavior",
			}
		}
	}

	// Check contractual keywords.
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
	}

	return taxonomy.Signal{}
}
