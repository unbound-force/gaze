package welltested

import "testing"

func TestAdd(t *testing.T) {
	got := Add(2, 3)
	if got != 5 {
		t.Errorf("Add(2, 3) = %d, want 5", got)
	}
}

func TestDivide(t *testing.T) {
	got, err := Divide(10, 2)
	if err != nil {
		t.Fatalf("Divide(10, 2) unexpected error: %v", err)
	}
	if got != 5 {
		t.Errorf("Divide(10, 2) = %d, want 5", got)
	}
}

func TestDivide_ZeroError(t *testing.T) {
	_, err := Divide(10, 0)
	if err == nil {
		t.Fatal("Divide(10, 0) expected error, got nil")
	}
}

func TestCounter_Increment(t *testing.T) {
	c := &Counter{}
	got := c.Increment(5)
	if got != 5 {
		t.Errorf("Increment(5) = %d, want 5", got)
	}
	if c.Value() != 5 {
		t.Errorf("Value() = %d, want 5", c.Value())
	}
}
