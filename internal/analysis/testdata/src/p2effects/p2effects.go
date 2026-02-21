// Package p2effects provides test fixtures for P2-tier side effect
// detection.
package p2effects

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
)

// --- Goroutine Spawn ---

// SpawnGoroutine starts a goroutine.
func SpawnGoroutine(ch chan int) {
	go func() {
		ch <- 1
	}()
}

// SpawnGoroutineWithFunc starts a goroutine calling a named function.
func SpawnGoroutineWithFunc(f func()) {
	go f()
}

// NoGoroutine calls a function without spawning a goroutine.
func NoGoroutine(f func()) {
	f()
}

// --- Panic ---

// PanicWithString panics with a string message.
func PanicWithString() {
	panic("something went wrong")
}

// PanicWithError panics with an error value.
func PanicWithError(err error) {
	panic(err)
}

// NoPanic returns an error instead of panicking.
func NoPanic(err error) error {
	return err
}

// --- File System Write ---

// WriteFileOS writes a file using os.WriteFile.
func WriteFileOS() error {
	return os.WriteFile("test.txt", []byte("hello"), 0644)
}

// CreateFile creates a file using os.Create.
func CreateFile() (*os.File, error) {
	return os.Create("test.txt")
}

// MkdirCall creates a directory.
func MkdirCall() error {
	return os.Mkdir("testdir", 0755)
}

// ReadFileOnly reads a file (should NOT trigger FileSystemWrite).
func ReadFileOnly() ([]byte, error) {
	return os.ReadFile("test.txt")
}

// OpenReadOnly opens a file for reading (should NOT trigger FileSystemWrite).
func OpenReadOnly() (*os.File, error) {
	return os.Open("test.txt")
}

// --- File System Delete ---

// RemoveFile removes a file.
func RemoveFile() error {
	return os.Remove("test.txt")
}

// RemoveAllDir removes a directory tree.
func RemoveAllDir() error {
	return os.RemoveAll("testdir")
}

// --- File System Meta ---

// ChmodFile changes file permissions.
func ChmodFile() error {
	return os.Chmod("test.txt", 0755)
}

// SymlinkFile creates a symbolic link.
func SymlinkFile() error {
	return os.Symlink("old", "new")
}

// StatFile reads file metadata (should NOT trigger FileSystemMeta).
func StatFile() (os.FileInfo, error) {
	return os.Stat("test.txt")
}

// --- Log Write ---

// LogPrint writes using the standard log package.
func LogPrint() {
	log.Println("a log message")
}

// LogFatal writes a fatal log message.
func LogFatal() {
	log.Fatal("fatal error")
}

// SlogInfo writes using the slog package.
func SlogInfo() {
	slog.Info("an info message")
}

// NoLogging does string formatting without logging.
func NoLogging() string {
	return "no logging here"
}

// --- Context Cancellation ---

// CancelContext creates a cancellable context.
func CancelContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// TimeoutContext creates a context with timeout.
func TimeoutContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 0)
}

// UseContextNoCancel uses a context without creating a cancellable one.
func UseContextNoCancel(ctx context.Context) context.Context {
	return ctx
}

// --- Callback Invocation ---

// InvokeCallback calls a function parameter.
func InvokeCallback(fn func()) {
	fn()
}

// InvokeCallbackWithArgs calls a function parameter with arguments.
func InvokeCallbackWithArgs(fn func(int) error) error {
	return fn(42)
}

// NoCallback does not call any function parameter.
func NoCallback(x int) int {
	return x + 1
}

// --- Database Write ---

// DBExec executes a database write.
func DBExec(db *sql.DB) error {
	_, err := db.Exec("INSERT INTO t VALUES (?)", 1)
	return err
}

// DBQuery performs a read-only database query (should NOT trigger
// DatabaseWrite).
func DBQuery(db *sql.DB) (*sql.Rows, error) {
	return db.Query("SELECT * FROM t")
}

// --- Database Transaction ---

// BeginTx starts a database transaction.
func BeginTx(db *sql.DB) (*sql.Tx, error) {
	return db.Begin()
}

// --- Pure Function (no P2 effects) ---

// PureP2 is a pure function with no P2 side effects.
func PureP2(x, y int) int {
	return x + y
}
