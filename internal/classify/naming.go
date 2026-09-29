// Package classify implements the contractual classification engine.
package classify

import (
	"strings"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// contractualPrefixes maps function name prefixes to the side effect
// types they imply as contractual.
var contractualPrefixes = []struct {
	prefix     string
	impliesFor []taxonomy.SideEffectType
}{
	{"Get", []taxonomy.SideEffectType{taxonomy.ReturnValue}},
	{"Fetch", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
	{"Load", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
	{"Read", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
	{"Save", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation, taxonomy.ErrorReturn}},
	{"Write", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation, taxonomy.ErrorReturn}},
	{"Update", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation, taxonomy.ErrorReturn}},
	{"Set", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.PointerArgMutation}},
	{"Delete", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.ErrorReturn}},
	{"Remove", []taxonomy.SideEffectType{taxonomy.ReceiverMutation, taxonomy.ErrorReturn}},
	{"Handle", nil},  // nil means all side effects
	{"Process", nil}, // nil means all side effects
	// Computational / analytical functions: return value is the primary
	// contract by Go convention (e.g., ComputeX, AnalyzeX, ParseX).
	{"Compute", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
	{"Analyze", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
	{"Classify", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
	{"Parse", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
	{"Build", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
	{"New", []taxonomy.SideEffectType{taxonomy.ReturnValue, taxonomy.ErrorReturn}},
}

// incidentalPrefixes are function name prefixes that signal incidental
// behavior, scoped to applicable side effect types. Only I/O-related
// effects are penalized — P0 effects like ReturnValue are never demoted
// by logging/debugging name prefixes. See issue #105.
var incidentalPrefixes = []struct {
	prefix    string
	appliesTo []taxonomy.SideEffectType
}{
	{"log", ioEffectTypes},
	{"Log", ioEffectTypes},
	{"debug", ioEffectTypes},
	{"Debug", ioEffectTypes},
	{"trace", ioEffectTypes},
	{"Trace", ioEffectTypes},
	{"print", ioEffectTypes},
	{"Print", ioEffectTypes},
}

// ioEffectTypes is the set of I/O-related side effect types that
// incidental naming and godoc signals apply to.
var ioEffectTypes = []taxonomy.SideEffectType{
	taxonomy.LogWrite,
	taxonomy.StdoutWrite,
	taxonomy.StderrWrite,
}

// maxNamingWeight is the maximum weight for naming convention signals.
const maxNamingWeight = 10

// sentinelNamingWeight is the weight for Err* sentinel variable naming.
// Sentinel errors are unambiguously contractual by convention — they are
// exported, named with the Err prefix, and exist solely to be matched by
// callers. The weight is set so that a sentinel with no other signals
// (base 50 + 30 = 80) reaches the default contractual threshold.
//
// Note: this intentionally exceeds maxNamingWeight (10). Sentinels are a
// special sub-case: unlike regular functions, package-level var declarations
// cannot receive interface, visibility, or godoc signals, so a stronger
// naming weight is the only way to reach the contractual threshold.
const sentinelNamingWeight = 30

// AnalyzeNamingSignal checks the function name against Go community
// naming conventions and returns a signal indicating whether the
// side effect is likely contractual or incidental based on the name.
func AnalyzeNamingSignal(funcName string, effectType taxonomy.SideEffectType) taxonomy.Signal {
	// Check incidental prefixes first (type-guarded — see issue #105).
	for _, ip := range incidentalPrefixes {
		if !strings.HasPrefix(funcName, ip.prefix) {
			continue
		}
		for _, applicable := range ip.appliesTo {
			if applicable == effectType {
				return taxonomy.Signal{
					Source:    "naming",
					Weight:    -maxNamingWeight,
					Reasoning: "function name prefix " + ip.prefix + "* suggests incidental behavior",
				}
			}
		}
	}

	// Check contractual prefixes.
	for _, cp := range contractualPrefixes {
		if !strings.HasPrefix(funcName, cp.prefix) {
			continue
		}
		// nil impliesFor means all effect types match.
		if cp.impliesFor == nil {
			return taxonomy.Signal{
				Source:    "naming",
				Weight:    maxNamingWeight,
				Reasoning: "function name prefix " + cp.prefix + "* suggests contractual behavior",
			}
		}
		for _, implied := range cp.impliesFor {
			if implied == effectType {
				return taxonomy.Signal{
					Source:    "naming",
					Weight:    maxNamingWeight,
					Reasoning: "function name prefix " + cp.prefix + "* implies " + string(effectType) + " is contractual",
				}
			}
		}
	}

	// Check sentinel error naming.
	// Err* sentinels are unambiguously contractual — exported by convention
	// and exist solely to be matched by callers. Use a stronger weight so
	// sentinels with no other signals reach the contractual threshold.
	if strings.HasPrefix(funcName, "Err") && effectType == taxonomy.SentinelError {
		return taxonomy.Signal{
			Source:    "naming",
			Weight:    sentinelNamingWeight,
			Reasoning: "Err* sentinel variable name implies contractual error",
		}
	}

	// No naming signal detected.
	return taxonomy.Signal{}
}
