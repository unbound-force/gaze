## Why

The `internal/adapter/` package repeats the same JSON-RPC request pattern
across every provider adapter: call `protocol.Client.Call`, check the
transport error, check the protocol-level `resp.Error`, then
`json.Unmarshal` the raw result into a typed struct. This ~8-line
sequence appears at five batch call sites, differing only by method name,
params type, and result type:

- `complexity.go` — `Analyze` (lines 41-55)
- `coverage.go` — `Coverage` (lines 32-46)
- `sideeffect.go` — `loadBatch` (lines 114-128)
- `contract.go` — `fetchTestMappings` (lines 97-114)
- `session.go` — `Initialize` (lines 87-103)

The duplication is a maintenance liability: any change to protocol error
handling (e.g., adding a new error-classification field, adjusting the
wrapped-error format) must be applied five times, and drift between sites
is a real risk. This violates the Zero-Waste Mandate. Go 1.24+ generics
make a single, type-safe extraction straightforward.

This change is a split from parent issue #201; it covers only the
`callAndUnmarshal` extraction. It does not address the `safeSSABuild`
consolidation (tracked separately as #238).

## What Changes

- Add a single unexported generic helper to `internal/adapter/`:
  `callAndUnmarshal[T any](ctx, client, method, params) (T, error)`.
  The helper performs Call → transport-error check → protocol-error
  check → `json.Unmarshal`, wrapping each failure with a per-method
  error context string derived from the `method` argument.
- Migrate the four hard-error batch call sites (`Analyze`, `Coverage`,
  `loadBatch`, `Initialize`) to call the helper. `Initialize` keeps its
  `s.client.Close()` cleanup at the call site (the helper handles only
  Call + Unmarshal, not lifecycle cleanup).
- Evaluate `fetchTestMappings`: its Call+Unmarshal core can use the
  helper, but its `p.warn()` graceful-degradation branches remain at the
  call site unchanged. If integrating the helper there would compromise
  the warning/degradation semantics, `fetchTestMappings` is left as-is
  (making the effective migration count four). This decision is resolved
  in design.md.
- Preserve per-method error context in all wrapped errors so operators
  can still distinguish a `complexity` failure from a `coverage` failure
  in logs.
- Add dedicated unit tests for `callAndUnmarshal`.

Explicitly out of scope: `CallStream` (a different streaming response
model), the `fetchTestMappings` warning/degradation path, and any change
to `internal/protocol/` (the transport layer stays usage-agnostic).

## Capabilities

### New Capabilities
- `adapter.callAndUnmarshal`: internal generic helper that centralizes
  the JSON-RPC Call → error-check → unmarshal pattern for adapter
  provider methods, preserving per-method error context.

### Modified Capabilities
- `adapter.ExternalComplexityProvider.Analyze`,
  `adapter.ExternalLineCoverageProvider.Coverage`,
  `adapter.ExternalSideEffectAnalyzer.loadBatch`,
  `adapter.Session.Initialize`: internal implementation now delegates the
  Call+Unmarshal sequence to `callAndUnmarshal`. No change to their
  exported signatures, return values, or observable error behavior.

### Removed Capabilities
- None.

## Impact

- **Files changed**: `internal/adapter/complexity.go`,
  `internal/adapter/coverage.go`, `internal/adapter/sideeffect.go`,
  `internal/adapter/session.go`, possibly
  `internal/adapter/contract.go` (pending design decision), plus a new
  or existing test file for `callAndUnmarshal` unit tests.
- **Behavior**: No observable behavior change. Same errors, same
  per-method context, same return values. This is a pure DRY
  refactoring guarded by the existing integration test suite (fake
  analyzer binary) and new unit tests.
- **API surface**: Unchanged. The helper is unexported; no exported
  signatures are modified.
- **Dependencies**: None added. Pure standard-library generics.

## Constitution Alignment

Assessed against the Unbound Force org constitution (below). This change
also trivially satisfies the project constitution
(`.specify/memory/constitution.md`), the highest-authority document for
Gaze: **I. Accuracy** — no behavior change, error identification
unchanged, regression-tested; **II. Minimal Assumptions** — no new
assumptions about host projects or tooling; **III. Actionable Output** —
no user-facing output changes; **IV. Testability** — the extracted helper
is tested in isolation with 100% branch coverage (shared with the org
Testability principle below).

### I. Autonomous Collaboration

**Assessment**: N/A

This is an internal refactoring within a single package. It does not
change how heroes collaborate through artifacts, nor does it alter any
self-describing output. Adapter provider methods continue to return the
same typed results consumed by the scoring engine.

### II. Composability First

**Assessment**: PASS

The change introduces no new mandatory dependencies and adds no new
package. The helper is confined to `internal/adapter/` and does not
couple the transport layer (`internal/protocol/`) to adapter concerns —
the transport layer stays usage-agnostic. Each adapter remains
independently usable exactly as before.

### III. Observable Quality

**Assessment**: PASS

Per-method error context is explicitly preserved: every wrapped error
still names the failing method (e.g., "complexity protocol call",
"coverage protocol error"), so machine-parseable diagnostics and
operator logs retain full provenance. An acceptance test asserts the
method name appears in the wrapped error, making this observable quality
guarantee enforced rather than aspirational.

### IV. Testability

**Assessment**: PASS

The extracted helper is testable in isolation without external services:
dedicated unit tests cover successful unmarshal, transport (Call) error
propagation, protocol error propagation, and `json.Unmarshal` failure
(minimum four cases), plus a case exercising a second result type to
verify generic instantiation. Existing integration tests (against the
fake analyzer binary) provide a regression safety net and must pass
without modification.
