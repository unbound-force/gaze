// Package p1effects is a test fixture for P1-tier side effect detection.
package p1effects

import (
	"io"
	"net/http"
)

// --- Global Mutation ---

var globalCounter int
var globalName string

// MutateGlobal assigns to a package-level variable.
func MutateGlobal() {
	globalCounter++
}

// MutateTwoGlobals assigns to two package-level variables.
func MutateTwoGlobals() {
	globalCounter = 42
	globalName = "updated"
}

// ReadGlobal only reads a global — should NOT produce GlobalMutation.
func ReadGlobal() int {
	return globalCounter
}

// --- Channel Send ---

// SendOnChannel sends a value on a channel.
func SendOnChannel(ch chan<- int) {
	ch <- 42
}

// CloseChannel closes a channel.
func CloseChannel(ch chan int) {
	close(ch)
}

// SendAndClose both sends and closes.
func SendAndClose(ch chan int) {
	ch <- 1
	close(ch)
}

// --- Writer Output ---

// WriteToWriter calls Write on an io.Writer parameter.
func WriteToWriter(w io.Writer) error {
	_, err := w.Write([]byte("hello"))
	return err
}

// ReadFromWriter does not write — should NOT produce WriterOutput.
func ReadFromWriter(r io.Reader) ([]byte, error) {
	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	return buf[:n], err
}

// --- HTTP Response Writer ---

// HandleHTTP writes to an http.ResponseWriter.
func HandleHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// --- Map Mutation ---

// WriteToMap assigns to a map index.
func WriteToMap(m map[string]int) {
	m["key"] = 42
}

// ReadFromMap only reads a map — should NOT produce MapMutation.
func ReadFromMap(m map[string]int) int {
	return m["key"]
}

// --- Slice Mutation ---

// WriteToSlice assigns to a slice index.
func WriteToSlice(s []int) {
	s[0] = 99
}

// ReadFromSlice only reads — should NOT produce SliceMutation.
func ReadFromSlice(s []int) int {
	return s[0]
}

// --- Pure function (no P1 effects) ---

// PureP1 has no P1 side effects.
func PureP1(x, y int) int {
	return x + y
}
