package helperreturn

import "testing"

// processHelper wraps the target function Process. This is the
// helper indirection pattern — the test calls processHelper, which
// calls Process internally.
func processHelper(t *testing.T, input string) *Result {
	t.Helper()
	result, err := Process(input)
	if err != nil {
		t.Fatalf("processHelper: unexpected error: %v", err)
	}
	return result
}

// unrelatedHelper does NOT call Process. Used for negative testing
// to ensure non-target helpers don't produce false positive tracing.
func unrelatedHelper(t *testing.T, input string) string {
	t.Helper()
	return "unrelated-" + input
}

// TestProcess_ViaHelper calls Process through a helper function and
// asserts on fields of the returned struct. This exercises the
// helper return value tracing fallback.
func TestProcess_ViaHelper(t *testing.T) {
	result := processHelper(t, "hello")

	// These selector assertions should map via helper return tracing.
	if result.Value != "processed-hello" {
		t.Errorf("result.Value = %q, want %q", result.Value, "processed-hello")
	}
	if result.Count != 5 {
		t.Errorf("result.Count = %d, want %d", result.Count, 5)
	}
	if !result.Success {
		t.Error("result.Success = false, want true")
	}
}

// TestProcess_Direct calls Process directly (no helper). This
// verifies that direct tracing still works and takes precedence
// over helper tracing.
func TestProcess_Direct(t *testing.T) {
	result, err := Process("direct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value != "processed-direct" {
		t.Errorf("result.Value = %q, want %q", result.Value, "processed-direct")
	}
}

// TestTransform_UnrelatedHelper uses an unrelated helper that does
// NOT call the target. This must NOT produce false positive tracing.
func TestTransform_UnrelatedHelper(t *testing.T) {
	got := unrelatedHelper(t, "test")
	if got != "unrelated-test" {
		t.Errorf("got = %q, want %q", got, "unrelated-test")
	}

	// Direct call to Transform.
	transformed := Transform("test")
	if transformed != "transformed-test" {
		t.Errorf("Transform() = %q, want %q", transformed, "transformed-test")
	}
}
