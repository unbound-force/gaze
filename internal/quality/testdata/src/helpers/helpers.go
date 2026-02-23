// Package helpers is a test fixture with tests using helper
// functions at various call depths to verify helper traversal.
package helpers

import "fmt"

// Multiply returns the product of two integers.
func Multiply(a, b int) int {
	return a * b
}

// SafeDivide divides a by b, returning an error if b is zero.
func SafeDivide(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("cannot divide by zero")
	}
	return a / b, nil
}
