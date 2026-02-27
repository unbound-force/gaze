// Package helperreturn is a test fixture for exercising helper return
// value tracing. The target function is called inside a helper, and
// the test function asserts on the helper's returned struct fields.
package helperreturn

import "fmt"

// Result represents a computation result with multiple fields.
type Result struct {
	Value   string
	Count   int
	Success bool
}

// Process is the target function. It returns a Result struct.
func Process(input string) (*Result, error) {
	if input == "" {
		return nil, fmt.Errorf("input is required")
	}
	return &Result{
		Value:   "processed-" + input,
		Count:   len(input),
		Success: true,
	}, nil
}

// Transform is a target function not called by the helper.
// Used for negative testing.
func Transform(input string) string {
	return "transformed-" + input
}
