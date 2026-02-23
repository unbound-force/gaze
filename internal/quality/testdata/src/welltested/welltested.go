// Package welltested is a test fixture containing functions with
// known contractual side effects and tests that assert on all of
// them (100% contract coverage target).
package welltested

import "fmt"

// Add returns the sum of two integers.
func Add(a, b int) int {
	return a + b
}

// Divide divides a by b, returning the result and an error if b is zero.
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// Counter tracks a running total.
type Counter struct {
	value int
}

// Increment adds n to the counter and returns the new value.
func (c *Counter) Increment(n int) int {
	c.value += n
	return c.value
}

// Value returns the current counter value.
func (c *Counter) Value() int {
	return c.value
}
