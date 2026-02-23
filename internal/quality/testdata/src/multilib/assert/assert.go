// Package assert is a minimal stub mimicking testify/assert for
// fixture purposes. It provides no-op assertion functions whose
// call signatures match testify patterns so the AST-based assertion
// detector can exercise its testify recognition logic.
package assert

import "testing"

// Equal checks equality (stub).
func Equal(t *testing.T, expected, actual interface{}, _ ...interface{}) bool {
	return expected == actual
}

// NotEqual checks inequality (stub).
func NotEqual(t *testing.T, expected, actual interface{}, _ ...interface{}) bool {
	return expected != actual
}

// NoError checks for nil error (stub).
func NoError(t *testing.T, err error, _ ...interface{}) bool {
	return err == nil
}

// Error checks for non-nil error (stub).
func Error(t *testing.T, err error, _ ...interface{}) bool {
	return err != nil
}

// Nil checks for nil value (stub).
func Nil(t *testing.T, obj interface{}, _ ...interface{}) bool {
	return obj == nil
}

// NotNil checks for non-nil value (stub).
func NotNil(t *testing.T, obj interface{}, _ ...interface{}) bool {
	return obj != nil
}

// True checks for true value (stub).
func True(t *testing.T, value bool, _ ...interface{}) bool {
	return value
}

// Len checks length (stub).
func Len(t *testing.T, obj interface{}, length int, _ ...interface{}) bool {
	return true
}

// Contains checks containment (stub).
func Contains(t *testing.T, s, contains interface{}, _ ...interface{}) bool {
	return true
}

// Greater checks a > b (stub).
func Greater(t *testing.T, e1, e2 interface{}, _ ...interface{}) bool {
	return true
}
