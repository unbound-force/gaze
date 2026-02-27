package indirectmatch

import "testing"

// TestCompute_SelectorAccess asserts on individual fields of the
// returned struct (result.Name, result.Count). These are selector
// expression patterns that require resolveExprRoot to match.
func TestCompute_SelectorAccess(t *testing.T) {
	result, err := Compute("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Selector expressions: result.Name, result.Count
	if result.Name != "hello" {
		t.Errorf("result.Name = %q, want %q", result.Name, "hello")
	}
	if result.Count != 5 {
		t.Errorf("result.Count = %d, want %d", result.Count, 5)
	}
}

// TestCompute_DeepSelector asserts on a deeply nested field
// (result.A.B.Value). This exercises multi-level selector chain
// resolution: SelectorExpr -> SelectorExpr -> SelectorExpr -> Ident.
func TestCompute_DeepSelector(t *testing.T) {
	result, err := Compute("deep")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Deep selector chain: result.A.B.Value
	if result.A.B.Value != "nested-deep" {
		t.Errorf("result.A.B.Value = %q, want %q", result.A.B.Value, "nested-deep")
	}
}

// TestCompute_Error asserts on the error return value directly.
// This is a direct identity match (confidence 75).
func TestCompute_Error(t *testing.T) {
	_, err := Compute("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

// TestListItems_LenBuiltin asserts on len(results), exercising
// built-in call unwinding: CallExpr(len) -> Args[0] -> Ident.
func TestListItems_LenBuiltin(t *testing.T) {
	results := ListItems(3)
	if len(results) != 3 {
		t.Errorf("len(results) = %d, want 3", len(results))
	}
}

// TestListItems_CapBuiltin asserts on cap(results), exercising
// built-in call unwinding: CallExpr(cap) -> Args[0] -> Ident.
func TestListItems_CapBuiltin(t *testing.T) {
	results := ListItems(5)
	if cap(results) < 5 {
		t.Errorf("cap(results) = %d, want >= 5", cap(results))
	}
}

// TestListItems_IndexAccess asserts on results[0], exercising
// index expression resolution: IndexExpr -> Ident.
func TestListItems_IndexAccess(t *testing.T) {
	results := ListItems(2)
	if results[0].Field != "item-0" {
		t.Errorf("results[0].Field = %q, want %q", results[0].Field, "item-0")
	}
}

// TestListItems_IndexPlusSelector asserts on results[0].SubField,
// exercising combined index + selector resolution:
// SelectorExpr -> IndexExpr -> Ident.
func TestListItems_IndexPlusSelector(t *testing.T) {
	results := ListItems(2)
	if results[0].SubField != "sub-0" {
		t.Errorf("results[0].SubField = %q, want %q", results[0].SubField, "sub-0")
	}
}

// TestMakeMap_LenBuiltin asserts on len(m) for a map return value.
func TestMakeMap_LenBuiltin(t *testing.T) {
	m := MakeMap([]string{"a", "b", "c"})
	if len(m) != 3 {
		t.Errorf("len(m) = %d, want 3", len(m))
	}
}

// TestIdentity_DirectMatch asserts on the bare returned variable.
// This is a direct identity match (confidence 75) and must remain
// at confidence 75 even after the two-pass strategy is added.
func TestIdentity_DirectMatch(t *testing.T) {
	got := Identity("test")
	if got != "test" {
		t.Errorf("Identity() = %q, want %q", got, "test")
	}
}

// TestCompute_NonTracedSelector asserts on a local variable's field
// that is NOT in objToEffectID. This must NOT produce a mapping
// (false positive prevention, FR-009).
func TestCompute_NonTracedSelector(t *testing.T) {
	result, err := Compute("valid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create a local variable not traced from a target call.
	localVar := &Result{Name: "local", Count: 99}
	if localVar.Name != "local" {
		t.Errorf("localVar.Name = %q, want %q", localVar.Name, "local")
	}

	// This should still map because result IS traced.
	if result.Name != "valid" {
		t.Errorf("result.Name = %q, want %q", result.Name, "valid")
	}
}

// TestCompute_VariableShadowing tests that when a traced variable is
// reassigned, assertions on the reassigned variable do NOT map to the
// original return value's effect (spec.md edge case).
func TestCompute_VariableShadowing(t *testing.T) {
	result, err := Compute("original")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Shadow the traced variable with a local value.
	result = &Result{Name: "shadowed", Count: 0}

	// This assertion is on the shadowed value, not the original
	// return. The types.Object may differ depending on Go's SSA
	// handling of reassignment.
	if result.Name != "shadowed" {
		t.Errorf("result.Name = %q, want %q", result.Name, "shadowed")
	}
}
