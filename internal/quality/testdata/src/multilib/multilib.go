// Package multilib is a test fixture with functions exercised by
// tests using multiple assertion libraries: testify-style and
// go-cmp-style patterns.
package multilib

import "fmt"

// User represents a user record.
type User struct {
	Name  string
	Email string
	Age   int
}

// NewUser creates a User, returning an error if the name is empty.
func NewUser(name, email string, age int) (*User, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return &User{Name: name, Email: email, Age: age}, nil
}

// Greet returns a greeting string for the user.
func Greet(u *User) string {
	return fmt.Sprintf("Hello, %s!", u.Name)
}

// Sum returns the sum of a slice of integers.
func Sum(nums []int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// Divide divides a by b, returning an error on division by zero.
func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// Transform applies a function to each element of a slice.
func Transform(nums []int, fn func(int) int) []int {
	result := make([]int, len(nums))
	for i, n := range nums {
		result[i] = fn(n)
	}
	return result
}
