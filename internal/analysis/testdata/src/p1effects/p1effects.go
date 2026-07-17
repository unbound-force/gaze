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

// --- Local Variable False Positive Tests ---

// LocalMapWrite creates a local map, writes to it, and returns it.
// Should NOT produce any P1 effects — the map is body-local.
func LocalMapWrite() map[string]int {
	m := make(map[string]int)
	m["key"] = 42
	return m
}

// LocalSliceWrite creates a local slice, writes to it, and returns it.
// Should NOT produce any P1 effects — the slice is body-local.
func LocalSliceWrite() []int {
	s := make([]int, 3)
	s[0] = 42
	return s
}

// LocalChannelSend creates a local buffered channel, sends on it,
// and receives from it. Should NOT produce any P1 effects — the
// channel is body-local.
func LocalChannelSend() int {
	ch := make(chan int, 1)
	ch <- 42
	return <-ch
}

// LocalChannelClose creates a local channel and closes it.
// Should NOT produce any P1 effects — the channel is body-local.
func LocalChannelClose() {
	ch := make(chan int)
	close(ch)
}

// NamedReturnMapWrite uses a named return value of map type, creates
// it with make, writes to it, and returns. SHOULD produce MapMutation
// because named returns are externally observable (part of the function
// signature).
func NamedReturnMapWrite() (result map[string]int) {
	result = make(map[string]int)
	result["key"] = 42
	return result
}

// --- Struct with Map Field (SelectorExpr test) ---

// Container is a struct with a map field, used to test SelectorExpr
// unwrapping in isExternallyObservable.
type Container struct {
	M map[string]int
}

// WriteToStructMap writes to a map field on the receiver. SHOULD
// produce MapMutation because the receiver is externally observable
// and unwrapToIdent resolves c.M to c (the receiver).
func (c *Container) WriteToStructMap() {
	c.M["key"] = 42
}

// --- Non-io.Writer Write method (issue #109) ---

// Notifier has a Write method but does NOT implement io.Writer
// because its signature is Write(string), not Write([]byte) (int, error).
type Notifier struct{}

// Write sends a notification. This is NOT io.Writer.Write.
func (n *Notifier) Write(msg string) {}

// SendNote calls Notifier.Write. Should NOT produce WriterOutput
// because Notifier does not implement io.Writer.
func SendNote(n *Notifier) {
	n.Write("hello")
}

// --- Custom http.ResponseWriter implementation (issue #132) ---

// CustomResponseWriter implements http.ResponseWriter with a custom type.
type CustomResponseWriter struct {
	code int
	body []byte
}

// Header returns the response headers.
func (c *CustomResponseWriter) Header() http.Header { return http.Header{} }

// Write writes the response body.
func (c *CustomResponseWriter) Write(b []byte) (int, error) {
	c.body = append(c.body, b...)
	return len(b), nil
}

// WriteHeader sets the status code.
func (c *CustomResponseWriter) WriteHeader(code int) { c.code = code }

// HandleCustomRW writes to a custom http.ResponseWriter implementation.
// Should produce HTTPResponseWrite, NOT WriterOutput.
func HandleCustomRW(w *CustomResponseWriter) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

// --- Partial http.ResponseWriter (issue #162) ---

// PartialResponseWriter has Write + Header but NOT WriteHeader.
// Should produce WriterOutput, NOT HTTPResponseWrite.
type PartialResponseWriter struct{}

func (p *PartialResponseWriter) Header() http.Header { return nil }
func (p *PartialResponseWriter) Write(b []byte) (int, error) { return len(b), nil }

// HandlePartialRW calls Write on a type with only 2 of 3 ResponseWriter methods.
func HandlePartialRW(w *PartialResponseWriter) {
	w.Write([]byte("ok"))
}

// --- Near-miss Write signatures (issue #162) ---

// WrongWriteReturn has Write([]byte) int — missing error return.
type WrongWriteReturn struct{}

func (wr *WrongWriteReturn) Header() http.Header        { return nil }
func (wr *WrongWriteReturn) Write(b []byte) int         { return len(b) }
func (wr *WrongWriteReturn) WriteHeader(statusCode int) {}

// HandleWrongWriteReturn calls Write on a type whose Write signature
// doesn't match io.Writer. Should NOT produce HTTPResponseWrite or WriterOutput.
func HandleWrongWriteReturn(w *WrongWriteReturn) {
	w.Write([]byte("ok"))
}

// WrongParamType has Write(string) (int, error) — wrong param type.
type WrongParamType struct{}

func (wp *WrongParamType) Header() http.Header        { return nil }
func (wp *WrongParamType) Write(s string) (int, error) { return len(s), nil }
func (wp *WrongParamType) WriteHeader(statusCode int)  {}

// HandleWrongParamType calls Write on a type whose Write accepts string, not []byte.
func HandleWrongParamType(w *WrongParamType) {
	w.Write("ok")
}

// WrongIntReturn64 has Write([]byte) (int64, error) — wrong int type.
type WrongIntReturn64 struct{}

func (wr *WrongIntReturn64) Header() http.Header           { return nil }
func (wr *WrongIntReturn64) Write(b []byte) (int64, error) { return int64(len(b)), nil }
func (wr *WrongIntReturn64) WriteHeader(statusCode int)    {}

// HandleWrongIntReturn64 calls Write on a type whose Write returns int64, not int.
func HandleWrongIntReturn64(w *WrongIntReturn64) {
	w.Write([]byte("ok"))
}

// ExtraParamWrite has Write([]byte, int) (int, error) — extra param.
type ExtraParamWrite struct{}

func (ep *ExtraParamWrite) Header() http.Header                { return nil }
func (ep *ExtraParamWrite) Write(b []byte, n int) (int, error) { return n, nil }
func (ep *ExtraParamWrite) WriteHeader(statusCode int)         {}

// HandleExtraParamWrite calls Write on a type with an extra parameter.
func HandleExtraParamWrite(w *ExtraParamWrite) {
	w.Write([]byte("ok"), 2)
}

// --- Pure function (no P1 effects) ---

// PureP1 has no P1 side effects.
func PureP1(x, y int) int {
	return x + y
}
