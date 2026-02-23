// Package cmp is a minimal stub mimicking google/go-cmp for fixture
// purposes. It provides a Diff function whose signature matches
// go-cmp patterns so the AST-based assertion detector can exercise
// its go-cmp recognition logic.
package cmp

import "fmt"

// Diff returns a human-readable report of the differences between
// two values (stub: returns empty string if equal).
func Diff(x, y interface{}) string {
	if fmt.Sprintf("%v", x) == fmt.Sprintf("%v", y) {
		return ""
	}
	return fmt.Sprintf("(-want +got):\n  - %v\n  + %v", x, y)
}
