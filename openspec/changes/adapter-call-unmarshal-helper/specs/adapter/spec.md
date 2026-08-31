# Adapter Protocol-Call Helper — Delta Spec

## ADDED Requirements

### Requirement: Generic Call-and-Unmarshal Helper

The `internal/adapter/` package MUST provide a single generic helper
function `callAndUnmarshal[T any]` that centralizes the JSON-RPC request
sequence used by provider adapters: invoke `protocol.Client.Call`, check
the transport error, check the protocol-level `resp.Error`, and
`json.Unmarshal` the raw result into a value of type `T`. The helper MUST
reside in `internal/adapter/` and MUST NOT be added to
`internal/protocol/`, so the transport layer remains usage-agnostic. The
helper MUST be unexported.

#### Scenario: Successful call and unmarshal

- **GIVEN** a `protocol.Client` whose `Call` returns a response with a
  nil `Error` and a `Result` containing valid JSON for type `T`
- **WHEN** `callAndUnmarshal[T]` is invoked with a method name and params
- **THEN** it MUST return the unmarshalled value of type `T` and a nil
  error

#### Scenario: Transport (Call) error propagation

- **GIVEN** a `protocol.Client` whose `Call` returns a non-nil transport
  error
- **WHEN** `callAndUnmarshal[T]` is invoked
- **THEN** it MUST return the zero value of `T` and a wrapped error that
  contains the method name and wraps the original transport error with
  `%w`

#### Scenario: Protocol error propagation

- **GIVEN** a `protocol.Client` whose `Call` returns a response with a
  non-nil `resp.Error` (message and code)
- **WHEN** `callAndUnmarshal[T]` is invoked
- **THEN** it MUST return the zero value of `T` and an error that
  contains the method name, the protocol error message, and the protocol
  error code, formatted with `%s` (the protocol error is a structured
  JSON-RPC error object, NOT wrapped with `%w`) — consistent with the
  current inline behavior at all four sites

#### Scenario: Unmarshal failure propagation

- **GIVEN** a `protocol.Client` whose `Call` returns a nil `Error` but a
  `Result` containing JSON that cannot be unmarshalled into type `T`
- **WHEN** `callAndUnmarshal[T]` is invoked
- **THEN** it MUST return the zero value of `T` and a wrapped error that
  contains the method name and wraps the `json.Unmarshal` error with `%w`

#### Scenario: Generic instantiation across result types

- **GIVEN** two distinct result types are requested from the helper
  (e.g., a complexity result and a coverage result)
- **WHEN** `callAndUnmarshal` is instantiated for each type
- **THEN** each instantiation MUST correctly unmarshal into its own type
  without coupling to any single concrete protocol type

### Requirement: Per-Method Error Context Preservation

All errors returned by `callAndUnmarshal` MUST embed a per-method context
string derived from the `method` argument, so that operators can
distinguish which protocol method failed (e.g., a `complexity` failure
from a `coverage` failure) in logs and machine-parseable diagnostics.
Transport and unmarshal errors MUST wrap the underlying error with `%w`
so `errors.Is`/`errors.As` unwrapping is preserved; protocol errors are
formatted with `%s` (structured JSON-RPC error object, not a Go error
chain value).

#### Scenario: Method name appears in wrapped error

- **GIVEN** the helper is invoked with a specific method name and any
  error condition occurs (transport, protocol, or unmarshal)
- **WHEN** the returned error is formatted as a string
- **THEN** the method name (or its derived context prefix) MUST appear in
  the error string

## MODIFIED Requirements

### Requirement: Adapter Batch Provider Methods Delegate the Call Pattern

The batch provider methods `ExternalComplexityProvider.Analyze`,
`ExternalLineCoverageProvider.Coverage`,
`ExternalSideEffectAnalyzer.loadBatch`, and `Session.Initialize` MUST
obtain their protocol results via `callAndUnmarshal` rather than each
open-coding the Call → error-check → unmarshal sequence. Their exported
signatures, return values, and observable error behavior MUST remain
unchanged. `Session.Initialize` MUST retain its `s.client.Close()`
cleanup at the call site when the helper returns a non-nil error, since
the helper does not manage client lifecycle. `Session.Initialize` MUST
also preserve its legacy error strings (`"initialize handshake"`,
`"initialize error"`, `"parsing initialize result"`) rather than adopting
the generic `"<method> protocol call"`/`"<method> protocol error"`
prefixes, so existing tests and any operator log patterns remain valid
(see design D2).

Previously: each of these methods open-coded the full Call, transport
error check, protocol error check, and `json.Unmarshal` sequence inline.

#### Scenario: Migrated method preserves observable behavior

- **GIVEN** an external analyzer that responds successfully to a batch
  method
- **WHEN** the migrated method is invoked through the existing
  integration test suite (fake analyzer binary)
- **THEN** the method MUST return the same typed result and the same
  error behavior as before migration, with existing tests passing
  without modification

#### Scenario: Initialize cleans up on failed handshake

- **GIVEN** an external analyzer that returns an error during the
  initialize handshake
- **WHEN** `Session.Initialize` invokes `callAndUnmarshal` and receives a
  non-nil error
- **THEN** `Session.Initialize` MUST call `s.client.Close()` before
  returning the error

## Excluded From This Change (No Requirement Change)

- `CallStream` and the streaming side-effect path MUST NOT be modified;
  they use a different (JSONL streaming) response model.
- The `fetchTestMappings` graceful-degradation path (its `p.warn()`
  calls and nil-return-on-failure semantics) MUST remain unchanged. If
  integrating the helper would compromise these semantics,
  `fetchTestMappings` is left as-is (see design.md decision D3).
- `internal/protocol/` MUST NOT be modified.
