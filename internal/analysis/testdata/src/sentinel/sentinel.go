// Package sentinel is a test fixture for sentinel error detection.
package sentinel

import (
	"errors"
	"fmt"
)

// ErrNotFound is a sentinel error.
var ErrNotFound = errors.New("not found")

// ErrPermission is a sentinel error.
var ErrPermission = errors.New("permission denied")

// ErrWrapped wraps another sentinel with fmt.Errorf.
var ErrWrapped = fmt.Errorf("wrapped: %w", ErrNotFound)

// NotAnError is a regular variable — should NOT be detected.
var NotAnError = "just a string"

// errUnexported is an unexported sentinel.
var errUnexported = errors.New("internal")

// FindUser returns a sentinel error.
func FindUser(id int) (string, error) {
	if id <= 0 {
		return "", ErrNotFound
	}
	return "user", nil
}

// WrapError wraps a sentinel with fmt.Errorf.
func WrapError(id int) error {
	if id <= 0 {
		return fmt.Errorf("finding user %d: %w", id, ErrNotFound)
	}
	return nil
}
