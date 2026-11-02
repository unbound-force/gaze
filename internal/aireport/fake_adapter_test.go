package aireport

import (
	"context"
	"io"
	"sync"
)

// FakeAdapterCall records one invocation of FakeAdapter.Format.
// FakeAdapter is a test double; it lives in a test file so it is never
// compiled into the shipped binary.
type FakeAdapterCall struct {
	// SystemPrompt is the system prompt passed to Format.
	SystemPrompt string
	// Payload is the full payload bytes read from the io.Reader argument.
	Payload []byte
}

// FakeAdapter is an AIAdapter for use in tests. It is safe for concurrent use.
type FakeAdapter struct {
	// Response is the string returned by Format.
	Response string
	// Err is the error returned by Format. When non-nil, Response is ignored.
	Err error
	// Calls records each invocation of Format for assertion in tests.
	// Protected by mu; read via the Calls field after all goroutines are done.
	Calls []FakeAdapterCall

	mu sync.Mutex
}

// Compile-time interface check.
var _ AIAdapter = &FakeAdapter{}

// Format implements AIAdapter. It records the call and returns the configured
// Response or Err. Safe for concurrent use.
func (f *FakeAdapter) Format(_ context.Context, systemPrompt string, payload io.Reader) (string, error) {
	var payloadBytes []byte
	if payload != nil {
		payloadBytes, _ = io.ReadAll(payload)
	}
	f.mu.Lock()
	f.Calls = append(f.Calls, FakeAdapterCall{
		SystemPrompt: systemPrompt,
		Payload:      payloadBytes,
	})
	// Read Err and Response under the lock so concurrent mutations are safe.
	err := f.Err
	resp := f.Response
	f.mu.Unlock()
	if err != nil {
		return "", err
	}
	return resp, nil
}
