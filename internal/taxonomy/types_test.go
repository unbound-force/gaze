package taxonomy

import (
	"testing"
)

func TestGenerateID_Deterministic(t *testing.T) {
	id1 := GenerateID("pkg/foo", "Save", "ReceiverMutation", "foo.go:10:2")
	id2 := GenerateID("pkg/foo", "Save", "ReceiverMutation", "foo.go:10:2")

	if id1 != id2 {
		t.Errorf("GenerateID not deterministic: %q != %q", id1, id2)
	}
}

func TestGenerateID_Format(t *testing.T) {
	id := GenerateID("pkg/foo", "Save", "ReceiverMutation", "foo.go:10:2")

	if len(id) != 11 { // "se-" + 8 hex chars
		t.Errorf("expected ID length 11, got %d: %q", len(id), id)
	}
	if id[:3] != "se-" {
		t.Errorf("expected ID to start with 'se-', got %q", id)
	}
}

func TestGenerateID_UniqueForDifferentInputs(t *testing.T) {
	id1 := GenerateID("pkg/foo", "Save", "ReceiverMutation", "foo.go:10:2")
	id2 := GenerateID("pkg/foo", "Save", "ReturnValue", "foo.go:10:2")
	id3 := GenerateID("pkg/foo", "Load", "ReceiverMutation", "foo.go:20:2")

	if id1 == id2 {
		t.Errorf("different effect types should produce different IDs")
	}
	if id1 == id3 {
		t.Errorf("different functions should produce different IDs")
	}
}

func TestTierOf_P0Types(t *testing.T) {
	p0Types := []SideEffectType{
		ReturnValue, ErrorReturn, SentinelError,
		ReceiverMutation, PointerArgMutation,
	}
	for _, st := range p0Types {
		if got := TierOf(st); got != TierP0 {
			t.Errorf("TierOf(%s) = %s, want P0", st, got)
		}
	}
}

func TestTierOf_AllTypesHaveTiers(t *testing.T) {
	allTypes := []SideEffectType{
		// P0
		ReturnValue, ErrorReturn, SentinelError,
		ReceiverMutation, PointerArgMutation,
		// P1
		SliceMutation, MapMutation, GlobalMutation,
		WriterOutput, HTTPResponseWrite, ChannelSend,
		ChannelClose, DeferredReturnMutation,
		// P2
		FileSystemWrite, FileSystemDelete, FileSystemMeta,
		DatabaseWrite, DatabaseTransaction, GoroutineSpawn,
		Panic, CallbackInvocation, LogWrite, ContextCancellation,
		// P3
		StdoutWrite, StderrWrite, EnvVarMutation,
		MutexOp, WaitGroupOp, AtomicOp, TimeDependency,
		ProcessExit, RecoverBehavior,
		// P4
		ReflectionMutation, UnsafeMutation, CgoCall,
		FinalizerRegistration, SyncPoolOp,
		ClosureCaptureMutation,
	}

	for _, st := range allTypes {
		tier := TierOf(st)
		if tier == "" {
			t.Errorf("TierOf(%s) returned empty tier", st)
		}
	}
}

func TestTierOf_UnknownDefaultsToP4(t *testing.T) {
	if got := TierOf("UnknownType"); got != TierP4 {
		t.Errorf("TierOf(unknown) = %s, want P4", got)
	}
}

func TestFunctionTarget_QualifiedName(t *testing.T) {
	tests := []struct {
		name     string
		target   FunctionTarget
		expected string
	}{
		{
			name:     "package function",
			target:   FunctionTarget{Function: "ParseConfig"},
			expected: "ParseConfig",
		},
		{
			name:     "pointer receiver method",
			target:   FunctionTarget{Function: "Save", Receiver: "*Store"},
			expected: "(*Store).Save",
		},
		{
			name:     "value receiver method",
			target:   FunctionTarget{Function: "String", Receiver: "Config"},
			expected: "(Config).String",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.target.QualifiedName()
			if got != tt.expected {
				t.Errorf("QualifiedName() = %q, want %q", got, tt.expected)
			}
		})
	}
}
