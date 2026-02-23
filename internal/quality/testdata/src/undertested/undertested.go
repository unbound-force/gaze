// Package undertested is a test fixture with functions that have
// gaps: some contractual effects have no test assertions.
package undertested

import "fmt"

// Parse converts a string to an integer, returning the value and
// an error if the string is not a valid number.
func Parse(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", s, err)
	}
	return result, nil
}

// Store holds a set of key-value pairs.
type Store struct {
	data map[string]string
}

// NewStore creates a new Store.
func NewStore() *Store {
	return &Store{data: make(map[string]string)}
}

// Set stores a key-value pair and returns the previous value.
func (s *Store) Set(key, value string) string {
	prev := s.data[key]
	s.data[key] = value
	return prev
}

// Get retrieves a value by key.
func (s *Store) Get(key string) (string, bool) {
	v, ok := s.data[key]
	return v, ok
}
