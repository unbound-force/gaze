package undertested

import "testing"

func TestParse_Valid(t *testing.T) {
	// Tests the return value but NOT the error case.
	got, _ := Parse("42")
	if got != 42 {
		t.Errorf("Parse(\"42\") = %d, want 42", got)
	}
	// Gap: error return is discarded (not asserted on).
}

func TestStore_Set(t *testing.T) {
	s := NewStore()
	// Only checks side effect: does not verify the return value.
	s.Set("key", "value")

	got, ok := s.Get("key")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != "value" {
		t.Errorf("Get(\"key\") = %q, want \"value\"", got)
	}
	// Gap: previous value (return of Set) is not asserted on.
}
