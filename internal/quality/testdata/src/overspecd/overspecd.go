// Package overspecd is a test fixture with tests that assert on
// incidental side effects (log output, internal state).
package overspecd

import (
	"fmt"
	"log"
)

// Process performs a computation and logs the result.
func Process(input int) int {
	result := input * 2
	log.Printf("processed: %d -> %d", input, result)
	return result
}

// Format formats a value with a prefix.
func Format(prefix string, value int) string {
	// Incidental: prints to stdout for debugging.
	fmt.Printf("formatting %s: %d\n", prefix, value)
	return fmt.Sprintf("%s-%d", prefix, value)
}
