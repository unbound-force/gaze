package helpers

import "testing"

func TestMultiply(t *testing.T) {
	got := Multiply(3, 4)
	assertEqual(t, got, 12)
}

func TestSafeDivide(t *testing.T) {
	got, err := SafeDivide(10, 2)
	assertNoError(t, err)
	assertEqual(t, got, 5)
}

func TestSafeDivide_ZeroError(t *testing.T) {
	_, err := SafeDivide(10, 0)
	assertError(t, err)
}

// assertEqual is a depth-1 helper that asserts two values are equal.
func assertEqual(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

// assertNoError is a depth-1 helper that asserts err is nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// assertError is a depth-1 helper that asserts err is non-nil.
func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
