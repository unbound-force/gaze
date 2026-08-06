// Package cliutil provides shared helper functions used across
// multiple CLI subcommands in cmd/gaze and the report pipeline
// in internal/aireport. These helpers eliminate duplication of
// common patterns such as format validation and JSON capture.
package cliutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// ValidateFormat checks that format is one of the supported output
// formats ("text" or "json"). Returns nil if valid, or a descriptive
// error containing the invalid value.
func ValidateFormat(format string) error {
	if format != "text" && format != "json" {
		return fmt.Errorf("invalid format %q: must be 'text' or 'json'", format)
	}
	return nil
}

// CaptureJSON calls fn with a buffer as the writer, then returns the
// buffer contents as a json.RawMessage. If fn returns an error, it is
// propagated and the message is nil.
func CaptureJSON(fn func(w io.Writer) error) (json.RawMessage, error) {
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		return nil, err
	}
	return json.RawMessage(buf.Bytes()), nil
}
