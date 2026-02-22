// Package contracts provides test fixtures with known contractual
// side effects for classification testing. It contains 30+ exported
// symbols specifically designed to exercise all five mechanical
// classification signals: interface satisfaction, API visibility,
// naming convention, Godoc annotation, and caller dependency.
package contracts

import (
	"fmt"
	"io"
	"time"
)

// ---- Interfaces ---------------------------------------------------------

// Reader is an interface that defines a contractual Read method.
type Reader interface {
	// Read reads data into p and returns the number of bytes read.
	Read(p []byte) (n int, err error)
}

// Writer is an interface that defines a contractual Write method.
type Writer interface {
	// Write writes data from p and returns the number of bytes written.
	Write(p []byte) (n int, err error)
}

// Store is an interface for persistence operations.
type Store interface {
	// Save persists the given data and returns an error if it fails.
	Save(data []byte) error
	// Delete removes the item with the given ID.
	Delete(id string) error
}

// Loader is an interface for loading configuration.
type Loader interface {
	// Load reads configuration and returns it.
	Load(path string) (map[string]string, error)
}

// Updater is an interface for in-place mutation operations.
type Updater interface {
	// Update replaces existing state with new state.
	Update(data []byte) error
}

// Fetcher is an interface for remote data retrieval.
type Fetcher interface {
	// Fetch retrieves data from a remote source.
	Fetch(key string) ([]byte, error)
}

// ---- FileStore (implements Store + Writer) --------------------------------

// FileStore implements Store and Writer.
type FileStore struct {
	Path string
	data []byte
}

// Save persists data to the file store. This is a contractual side
// effect because FileStore implements Store.
func (fs *FileStore) Save(data []byte) error {
	fs.data = data
	return nil
}

// Delete removes the file. This is a contractual side effect
// because FileStore implements Store.
func (fs *FileStore) Delete(id string) error {
	fs.data = nil
	return nil
}

// Write implements io.Writer. This is a contractual side effect
// because FileStore implements Writer.
func (fs *FileStore) Write(p []byte) (n int, err error) {
	fs.data = append(fs.data, p...)
	return len(p), nil
}

// ---- MemoryCache (implements Loader + Updater) ----------------------------

// MemoryCache is an in-memory cache implementing Loader and Updater.
type MemoryCache struct {
	entries map[string][]byte
}

// Load reads configuration from the cache. Contractual via Loader.
func (mc *MemoryCache) Load(path string) (map[string]string, error) {
	return map[string]string{}, nil
}

// Update replaces cache contents. Contractual via Updater.
func (mc *MemoryCache) Update(data []byte) error {
	mc.entries = map[string][]byte{"_": data}
	return nil
}

// ---- RemoteFetcher (implements Fetcher) -----------------------------------

// RemoteFetcher fetches data over HTTP. Contractual via Fetcher.
type RemoteFetcher struct {
	BaseURL string
}

// Fetch retrieves data from the remote source. Contractual via Fetcher.
func (rf *RemoteFetcher) Fetch(key string) ([]byte, error) {
	return []byte(rf.BaseURL + "/" + key), nil
}

// ---- Naming-signal contractual functions ---------------------------------

// GetData returns the stored data. The return value is contractual
// because the function name follows the Get* convention.
func GetData(fs *FileStore) []byte {
	return fs.data
}

// FetchConfig loads configuration. The return value is contractual
// because the function name follows the Fetch* convention.
func FetchConfig(path string) (map[string]string, error) {
	return map[string]string{"path": path}, nil
}

// SaveRecord writes a record. The mutation is contractual because
// the function name follows the Save* convention.
func SaveRecord(w io.Writer, record string) error {
	_, err := fmt.Fprintln(w, record)
	return err
}

// LoadSettings reads settings from a path. Contractual via Load*.
func LoadSettings(path string) (map[string]string, error) {
	return map[string]string{"_path": path}, nil
}

// ReadBytes reads raw bytes. Contractual via Read*.
func ReadBytes(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := r.Read(buf)
	return buf, err
}

// UpdateRecord updates a record in a store. Contractual via Update*.
func UpdateRecord(w io.Writer, id string, data []byte) error {
	_, err := fmt.Fprintf(w, "%s: %s", id, data)
	return err
}

// SetTimeout sets the timeout. Contractual via Set*.
func SetTimeout(d time.Duration) time.Duration {
	return d
}

// DeleteEntry removes an entry. Contractual via Delete*.
func DeleteEntry(w io.Writer, id string) error {
	_, err := fmt.Fprintf(w, "deleted: %s", id)
	return err
}

// RemoveExpired removes expired cache entries. Contractual via Remove*.
func RemoveExpired(entries map[string]time.Time) int {
	n := 0
	for k, v := range entries {
		if time.Now().After(v) {
			delete(entries, k)
			n++
		}
	}
	return n
}

// HandleRequest processes an HTTP request. Contractual via Handle*.
func HandleRequest(w io.Writer, req string) error {
	_, err := fmt.Fprintln(w, req)
	return err
}

// ProcessEvent handles an event. Contractual via Process*.
func ProcessEvent(event []byte) ([]byte, error) {
	return event, nil
}

// ---- Godoc-annotated contractual functions --------------------------------

// GetVersion returns the current software version.
// This function is part of the public API contract.
// Callers depend on the returned version string for compatibility checks.
func GetVersion() string {
	return "1.0.0"
}

// SetPrimary sets the primary data source.
// CONTRACT: callers rely on this for runtime configuration.
func SetPrimary(w io.Writer, source string) error {
	_, err := fmt.Fprintln(w, source)
	return err
}

// LoadProfile reads a user profile.
// This return value is part of the stable API contract.
func LoadProfile(id string) (map[string]string, error) {
	return map[string]string{"id": id}, nil
}

// ---- Exported-visibility contractual functions ----------------------------

// ExportedResult is an exported return type (visibility signal).
type ExportedResult struct {
	Value string
	OK    bool
}

// ComputeResult computes a result with an exported return type.
// High visibility: exported function + exported return type.
func ComputeResult(input string) ExportedResult {
	return ExportedResult{Value: input, OK: true}
}

// ApplyTransform applies a transformation and returns ExportedResult.
// High visibility: exported function + exported return type.
func ApplyTransform(data []byte, key string) (ExportedResult, error) {
	return ExportedResult{Value: string(data) + key, OK: true}, nil
}

// ---- Sentinel errors (contractual by naming + type) ----------------------

// ErrNotFound is a sentinel error. Its existence as a named error
// is a contractual signal.
var ErrNotFound = fmt.Errorf("not found")

// ErrPermission is a sentinel error for permission failures.
var ErrPermission = fmt.Errorf("permission denied")

// ErrTimeout is a sentinel error for timeout conditions.
var ErrTimeout = fmt.Errorf("operation timed out")

// ErrInvalid is a sentinel error for invalid input.
var ErrInvalid = fmt.Errorf("invalid input")
