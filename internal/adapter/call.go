package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unbound-force/gaze/internal/protocol"
)

// callAndUnmarshal issues a JSON-RPC call for method with params on
// client, checks the transport and protocol errors, and unmarshals the
// raw result into a value of type T.
//
// It centralizes the Call -> transport-error check -> protocol-error
// check -> json.Unmarshal sequence that every batch provider adapter
// would otherwise open-code. All three failure paths wrap the error
// with per-method context derived from method so operators can
// distinguish, for example, a "complexity" failure from a "coverage"
// failure in logs.
//
// Error-chain contract (design D6): transport and unmarshal errors wrap
// the underlying error with %w, so errors.Is / errors.As unwrapping is
// preserved. Protocol errors are formatted with %s because resp.Error
// is a structured JSON-RPC error object carrying a message and code,
// not a Go error chain value.
//
// On any error the zero value of T is returned alongside the wrapped
// error.
func callAndUnmarshal[T any](
	ctx context.Context,
	client *protocol.Client,
	method string,
	params any,
) (T, error) {
	var result T

	// method is the wire method constant (e.g. protocol.MethodComplexity
	// == "complexity"). Reusing it as the error prefix is intentional: it
	// reproduces the exact legacy per-method log strings the migrated call
	// sites emitted before this helper existed.
	resp, err := client.Call(ctx, method, params)
	if err != nil {
		return result, fmt.Errorf("%s protocol call: %w", method, err)
	}
	if resp.Error != nil {
		return result, fmt.Errorf("%s protocol error: %s (code %d)", method, resp.Error.Message, resp.Error.Code)
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return result, fmt.Errorf("parsing %s result: %w", method, err)
	}

	return result, nil
}
