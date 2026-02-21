// Package returns is a test fixture for return value analysis.
package returns

import (
	"errors"
	"io"
)

// PureFunction has no return values — no side effects expected.
func PureFunction() {}

// SingleReturn returns one value.
func SingleReturn() int {
	return 42
}

// MultipleReturns returns two values.
func MultipleReturns() (string, bool) {
	return "hello", true
}

// ErrorReturn returns (T, error) — idiomatic Go pattern.
func ErrorReturn() (int, error) {
	return 0, nil
}

// ErrorOnly returns only an error.
func ErrorOnly() error {
	return errors.New("fail")
}

// TripleReturn returns three values including an error.
func TripleReturn() (string, int, error) {
	return "", 0, nil
}

// NamedReturns has named return values.
func NamedReturns() (data []byte, err error) {
	return nil, nil
}

// NamedReturnModifiedInDefer has a named return modified in a
// deferred function.
func NamedReturnModifiedInDefer() (err error) {
	defer func() {
		if err == nil {
			err = errors.New("deferred error")
		}
	}()
	return nil
}

// InterfaceReturn returns an interface type.
func InterfaceReturn() io.Reader {
	return nil
}
