package multilib

import (
	"testing"

	"github.com/unbound-force/gaze/internal/quality/testdata/src/multilib/assert"
	"github.com/unbound-force/gaze/internal/quality/testdata/src/multilib/cmp"
	"github.com/unbound-force/gaze/internal/quality/testdata/src/multilib/require"
)

// TestNewUser_Testify uses testify/assert patterns.
func TestNewUser_Testify(t *testing.T) {
	user, err := NewUser("Alice", "alice@test.com", 30)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, "alice@test.com", user.Email)
	assert.Equal(t, 30, user.Age)
}

// TestNewUser_Require uses testify/require patterns.
func TestNewUser_Require(t *testing.T) {
	user, err := NewUser("Bob", "bob@test.com", 25)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, "Bob", user.Name)
}

// TestNewUser_Error uses testify/assert for error checking.
func TestNewUser_Error(t *testing.T) {
	user, err := NewUser("", "noname@test.com", 20)
	assert.Error(t, err)
	assert.Nil(t, user)
}

// TestGreet_GoCmp uses go-cmp diff patterns.
func TestGreet_GoCmp(t *testing.T) {
	user := &User{Name: "Charlie"}
	got := Greet(user)
	want := "Hello, Charlie!"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Greet() mismatch (-want +got):\n%s", diff)
	}
}

// TestSum_Stdlib uses stdlib comparison patterns.
func TestSum_Stdlib(t *testing.T) {
	got := Sum([]int{1, 2, 3, 4, 5})
	if got != 15 {
		t.Errorf("Sum() = %d, want 15", got)
	}
}

// TestDivide_Mixed uses a mix of stdlib and testify patterns.
func TestDivide_Mixed(t *testing.T) {
	result, err := Divide(10, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Equal(t, 5.0, result)
}

// TestDivide_ZeroError uses testify for error assertion.
func TestDivide_ZeroError(t *testing.T) {
	_, err := Divide(10, 0)
	assert.Error(t, err)
}

// TestTransform_GoCmp uses go-cmp for slice comparison.
func TestTransform_GoCmp(t *testing.T) {
	double := func(n int) int { return n * 2 }
	got := Transform([]int{1, 2, 3}, double)
	want := []int{2, 4, 6}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Transform() mismatch:\n%s", diff)
	}
}

// TestSum_Testify uses testify assertions on Sum.
func TestSum_Testify(t *testing.T) {
	result := Sum([]int{10, 20, 30})
	assert.Equal(t, 60, result)
	assert.Greater(t, result, 0)
}
