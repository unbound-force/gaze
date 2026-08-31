package ssaguard_test

import (
	"errors"
	"testing"

	"github.com/unbound-force/gaze/internal/ssaguard"
)

// Coverage strategy: unit tests only. These three cases achieve 100%
// branch coverage of SafeSSABuild — the deferred recover() path via the
// two panic cases, and the normal return nil path via the no-panic
// case. This is the complete set of reachable branches.

// TestSafeSSABuild_NoPanic verifies that SafeSSABuild returns nil
// when the build function completes without panicking.
func TestSafeSSABuild_NoPanic(t *testing.T) {
	result := ssaguard.SafeSSABuild(func() {
		// no panic
	})
	if result != nil {
		t.Errorf("SafeSSABuild returned %v, want nil for non-panicking function", result)
	}
}

// TestSafeSSABuild_PanicString verifies that SafeSSABuild recovers
// a panic with a string value and returns it.
func TestSafeSSABuild_PanicString(t *testing.T) {
	result := ssaguard.SafeSSABuild(func() {
		panic("test panic message")
	})
	s, ok := result.(string)
	if !ok {
		t.Fatalf("SafeSSABuild returned %T, want string", result)
	}
	if s != "test panic message" {
		t.Errorf("SafeSSABuild returned %q, want %q", s, "test panic message")
	}
}

// TestSafeSSABuild_PanicError verifies that SafeSSABuild recovers
// a panic with an error value and returns it.
func TestSafeSSABuild_PanicError(t *testing.T) {
	errPanic := errors.New("SSA builder error")
	result := ssaguard.SafeSSABuild(func() {
		panic(errPanic)
	})
	e, ok := result.(error)
	if !ok {
		t.Fatalf("SafeSSABuild returned %T, want error", result)
	}
	if e != errPanic {
		t.Errorf("SafeSSABuild returned error %v, want %v", e, errPanic)
	}
}
