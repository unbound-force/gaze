// Package indirectmatch is a test fixture for exercising indirect
// expression resolution patterns: selector access (result.Field),
// deep selector chains (result.A.B), index access (results[0]),
// combined index+selector (results[0].Field), and built-in wrapping
// (len(results), cap(results)).
package indirectmatch

import "fmt"

// Inner holds a nested field for deep selector chain testing.
type Inner struct {
	Value string
}

// Nested holds a field that is itself a struct, enabling
// multi-level selector access (result.A.B.Value).
type Nested struct {
	B Inner
}

// Result represents a typical function return value with
// multiple fields that tests assert on individually.
type Result struct {
	Name  string
	Count int
	A     Nested
}

// Compute returns a Result struct. Tests assert on individual
// fields (result.Name, result.Count, result.A.B.Value) rather
// than the struct as a whole.
func Compute(input string) (*Result, error) {
	if input == "" {
		return nil, fmt.Errorf("input is required")
	}
	return &Result{
		Name:  input,
		Count: len(input),
		A:     Nested{B: Inner{Value: "nested-" + input}},
	}, nil
}

// Item represents a single element in a collection.
type Item struct {
	Field    string
	SubField string
}

// ListItems returns a slice of Items. Tests assert on individual
// elements (results[0], results[0].Field) and on the length
// (len(results)).
func ListItems(n int) []Item {
	items := make([]Item, n)
	for i := 0; i < n; i++ {
		items[i] = Item{
			Field:    fmt.Sprintf("item-%d", i),
			SubField: fmt.Sprintf("sub-%d", i),
		}
	}
	return items
}

// MakeMap returns a map for testing len() on map return values.
func MakeMap(keys []string) map[string]int {
	m := make(map[string]int, len(keys))
	for i, k := range keys {
		m[k] = i
	}
	return m
}

// Identity returns its input unchanged. Used for testing direct
// identity matching (bare variable comparison at confidence 75).
func Identity(s string) string {
	return s
}
