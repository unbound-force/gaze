// Package tabledriven is a test fixture with table-driven tests
// using t.Run sub-tests to verify assertion detection in sub-tests.
package tabledriven

import "fmt"

// Greet returns a greeting message for the given name.
func Greet(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}
	return fmt.Sprintf("Hello, %s!", name), nil
}

// Abs returns the absolute value of an integer.
func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
