package quality

import (
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

// TestHintForEffect_AllTypes verifies that hintForEffect returns a non-empty
// string for every SideEffectType constant defined in the taxonomy. This
// ensures new effect types added to the taxonomy don't silently produce
// empty hints in the quality report (SC-006).
func TestHintForEffect_AllTypes(t *testing.T) {
	allTypes := []taxonomy.SideEffectType{
		// P0
		taxonomy.ReturnValue,
		taxonomy.ErrorReturn,
		taxonomy.SentinelError,
		taxonomy.ReceiverMutation,
		taxonomy.PointerArgMutation,
		// P1
		taxonomy.SliceMutation,
		taxonomy.MapMutation,
		taxonomy.GlobalMutation,
		taxonomy.WriterOutput,
		taxonomy.HTTPResponseWrite,
		taxonomy.ChannelSend,
		taxonomy.ChannelClose,
		taxonomy.DeferredReturnMutation,
		// P2
		taxonomy.FileSystemWrite,
		taxonomy.FileSystemDelete,
		taxonomy.FileSystemMeta,
		taxonomy.DatabaseWrite,
		taxonomy.DatabaseTransaction,
		taxonomy.GoroutineSpawn,
		taxonomy.Panic,
		taxonomy.CallbackInvocation,
		taxonomy.LogWrite,
		taxonomy.ContextCancellation,
		// P3
		taxonomy.StdoutWrite,
		taxonomy.StderrWrite,
		taxonomy.EnvVarMutation,
		taxonomy.MutexOp,
		taxonomy.WaitGroupOp,
		taxonomy.AtomicOp,
		taxonomy.TimeDependency,
		taxonomy.ProcessExit,
		taxonomy.RecoverBehavior,
		// P4
		taxonomy.ReflectionMutation,
		taxonomy.UnsafeMutation,
		taxonomy.CgoCall,
		taxonomy.FinalizerRegistration,
		taxonomy.SyncPoolOp,
		taxonomy.ClosureCaptureMutation,
	}

	for _, et := range allTypes {
		t.Run(string(et), func(t *testing.T) {
			e := taxonomy.SideEffect{Type: et}
			hint := hintForEffect(e)
			if hint == "" {
				t.Errorf("hintForEffect(%s) returned empty string", et)
			}
		})
	}
}

// TestHintForEffect_SpecificValues verifies the exact hint strings for
// well-known P0 and P1 effect types. These are the most common in practice
// and agents depend on their specific content.
func TestHintForEffect_SpecificValues(t *testing.T) {
	cases := []struct {
		effectType taxonomy.SideEffectType
		target     string
		wantSubstr string
	}{
		{taxonomy.ErrorReturn, "", "t.Fatal(err)"},
		{taxonomy.ReturnValue, "", "got := target()"},
		{taxonomy.SentinelError, "", "errors.Is"},
		{taxonomy.ReceiverMutation, "Count", "receiver.Count"},
		{taxonomy.ReceiverMutation, "", "receiver state"},
		{taxonomy.PointerArgMutation, "buf", "*buf"},
		{taxonomy.PointerArgMutation, "", "pointer argument"},
		{taxonomy.SliceMutation, "", "slice contents"},
		{taxonomy.MapMutation, "", "map contents"},
		{taxonomy.WriterOutput, "w", "written to w"},
		{taxonomy.ChannelSend, "ch", "sent on ch"},
		{taxonomy.ChannelClose, "done", "done is closed"},
	}

	for _, tc := range cases {
		t.Run(string(tc.effectType)+"/"+tc.target, func(t *testing.T) {
			e := taxonomy.SideEffect{Type: tc.effectType, Target: tc.target}
			hint := hintForEffect(e)
			if !strings.Contains(hint, tc.wantSubstr) {
				t.Errorf("hintForEffect(%s, target=%q) = %q, want substring %q",
					tc.effectType, tc.target, hint, tc.wantSubstr)
			}
		})
	}
}

// TestHintForEffect_P2P4_Generic verifies that P2-P4 effect types produce
// a non-empty generic hint that includes the effect type name.
func TestHintForEffect_P2P4_Generic(t *testing.T) {
	p2p4Types := []taxonomy.SideEffectType{
		taxonomy.FileSystemWrite,
		taxonomy.DatabaseWrite,
		taxonomy.GoroutineSpawn,
		taxonomy.LogWrite,
		taxonomy.StdoutWrite,
		taxonomy.ReflectionMutation,
		taxonomy.CgoCall,
	}

	for _, et := range p2p4Types {
		t.Run(string(et), func(t *testing.T) {
			e := taxonomy.SideEffect{Type: et}
			hint := hintForEffect(e)
			if !strings.Contains(hint, string(et)) {
				t.Errorf("hintForEffect(%s) = %q, want it to contain the effect type name", et, hint)
			}
		})
	}
}

// TestUnmappedReason_HelperParam verifies that an assertion at depth > 0
// produces UnmappedReasonHelperParam regardless of the effect list.
func TestUnmappedReason_HelperParam(t *testing.T) {
	site := AssertionSite{
		Location: "helpers_test.go:15",
		Kind:     AssertionKindStdlibComparison,
		Depth:    1, // inside a helper body
	}
	effects := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.ReturnValue},
	}

	reason := classifyUnmappedReason(site, nil, effects)
	if reason != taxonomy.UnmappedReasonHelperParam {
		t.Errorf("got reason %q, want %q", reason, taxonomy.UnmappedReasonHelperParam)
	}
}

// TestUnmappedReason_InlineCall verifies that a depth-0 assertion with no
// traced return values (nil objToEffectID) and a target that has return
// effects produces UnmappedReasonInlineCall. This simulates the pattern
// "if c.Value() != 5" where the return value is never assigned.
func TestUnmappedReason_InlineCall(t *testing.T) {
	site := AssertionSite{
		Location: "counter_test.go:22",
		Kind:     AssertionKindStdlibComparison,
		Depth:    0, // in test body
	}
	effects := []taxonomy.SideEffect{
		{ID: "se-002", Type: taxonomy.ReturnValue},
	}
	// nil objToEffectID: traceReturnValues found no LHS assignment
	// because the call was inline (not assigned to a variable).
	reason := classifyUnmappedReason(site, nil, effects)
	if reason != taxonomy.UnmappedReasonInlineCall {
		t.Errorf("got reason %q, want %q", reason, taxonomy.UnmappedReasonInlineCall)
	}
}

// TestUnmappedReason_NoEffectMatch verifies that a depth-0 assertion with
// no return/error effects (only mutation effects) produces
// UnmappedReasonNoEffectMatch even when objToEffectID is empty.
func TestUnmappedReason_NoEffectMatch(t *testing.T) {
	site := AssertionSite{
		Location: "store_test.go:30",
		Kind:     AssertionKindStdlibComparison,
		Depth:    0,
	}
	// Only mutation effects — hasReturnEffects returns false.
	effects := []taxonomy.SideEffect{
		{ID: "se-003", Type: taxonomy.ReceiverMutation},
	}

	reason := classifyUnmappedReason(site, nil, effects)
	if reason != taxonomy.UnmappedReasonNoEffectMatch {
		t.Errorf("got reason %q, want %q", reason, taxonomy.UnmappedReasonNoEffectMatch)
	}
}

// TestUnmappedReason_NoEffects verifies that when there are no effects at
// all, the reason is no_effect_match (not inline_call, since hasReturnEffects
// returns false for an empty list).
func TestUnmappedReason_NoEffects(t *testing.T) {
	site := AssertionSite{
		Location: "foo_test.go:5",
		Kind:     AssertionKindStdlibErrorCheck,
		Depth:    0,
	}
	var effects []taxonomy.SideEffect

	reason := classifyUnmappedReason(site, nil, effects)
	if reason != taxonomy.UnmappedReasonNoEffectMatch {
		t.Errorf("got reason %q, want %q", reason, taxonomy.UnmappedReasonNoEffectMatch)
	}
}
