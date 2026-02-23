// Package require is a minimal stub mimicking testify/require for
// fixture purposes. It provides no-op assertion functions whose
// call signatures match testify patterns.
package require

import "testing"

// NoError checks for nil error (stub).
func NoError(t *testing.T, err error, _ ...interface{}) {
	if err != nil {
		t.Fatal(err)
	}
}

// Equal checks equality (stub).
func Equal(t *testing.T, expected, actual interface{}, _ ...interface{}) {
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// NotNil checks for non-nil value (stub).
func NotNil(t *testing.T, obj interface{}, _ ...interface{}) {
	if obj == nil {
		t.Fatal("expected non-nil")
	}
}
