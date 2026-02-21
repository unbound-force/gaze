// Package edgecases contains test fixtures for edge case scenarios:
// unsafe.Pointer usage, generics, empty functions, variadic functions,
// and functions with complex signatures.
package edgecases

import "unsafe"

// --- Generics ---

// GenericIdentity is a generic function with a type parameter.
func GenericIdentity[T any](v T) T {
	return v
}

// GenericSwap swaps two values using type parameters.
func GenericSwap[T any](a, b T) (T, T) {
	return b, a
}

// GenericSliceMap applies a function to each element of a slice.
func GenericSliceMap[T any, U any](s []T, fn func(T) U) []U {
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = fn(v)
	}
	return result
}

// GenericContainer is a generic type with methods.
type GenericContainer[T any] struct {
	items []T
}

// Add appends an item to the container (receiver mutation).
func (c *GenericContainer[T]) Add(item T) {
	c.items = append(c.items, item)
}

// Get returns an item by index.
func (c *GenericContainer[T]) Get(idx int) T {
	return c.items[idx]
}

// Count returns the length (pure, no mutation).
func (c *GenericContainer[T]) Count() int {
	return len(c.items)
}

// --- Unsafe ---

// UnsafePointerCast uses unsafe.Pointer to cast between types.
func UnsafePointerCast(i *int) *float64 {
	return (*float64)(unsafe.Pointer(i))
}

// UnsafeSliceHeader accesses slice internals via unsafe.
func UnsafeSliceHeader(s []byte) uintptr {
	return (*(*[3]uintptr)(unsafe.Pointer(&s)))[0]
}

// UnsafeSizeof uses unsafe.Sizeof (read-only, no mutation).
func UnsafeSizeof() uintptr {
	var x int64
	return unsafe.Sizeof(x)
}

// --- Empty / No-op functions ---

// EmptyFunction has no body (no side effects).
func EmptyFunction() {}

// NoOpWithParams takes params but does nothing.
func NoOpWithParams(a int, b string, c bool) {}

// --- Variadic functions ---

// VariadicSum sums variadic int arguments.
func VariadicSum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// VariadicWithCallback invokes a variadic callback.
func VariadicWithCallback(fns ...func()) {
	for _, fn := range fns {
		fn()
	}
}

// --- Complex signatures ---

// MultiReturn returns many values of different types.
func MultiReturn() (int, string, error, bool) {
	return 0, "", nil, false
}

// FuncReturningFunc returns a closure.
func FuncReturningFunc() func(int) int {
	return func(x int) int { return x * 2 }
}

// FuncTakingInterface accepts an interface value.
func FuncTakingInterface(v interface{}) interface{} {
	return v
}

// NamedMultiReturn uses named returns.
func NamedMultiReturn() (x int, y string, err error) {
	x = 42
	y = "hello"
	return
}
