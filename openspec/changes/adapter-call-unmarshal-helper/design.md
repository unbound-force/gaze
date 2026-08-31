## Context

The `internal/adapter/` package adapts an external language analyzer
(spoken to via JSON-RPC 2.0 over stdin/stdout) into the provider
interfaces consumed by the CRAP scoring engine. Every batch provider
method performs the same three-step protocol dance:

1. `resp, err := client.Call(ctx, method, params)` — transport
2. `if err != nil { ... }` and `if resp.Error != nil { ... }` — errors
3. `json.Unmarshal(resp.Result, &typed)` — decode into a Go struct

This ~8-line pattern is duplicated at five sites (`Analyze`,
`Coverage`, `loadBatch`, `fetchTestMappings`, `Initialize`), differing
only by method, params type, and result type. The duplication is a
maintenance liability and violates the Zero-Waste Mandate. Go 1.24+
generics allow a single type-safe extraction. See proposal.md for the
Constitution Alignment (Composability, Observable Quality, and
Testability all PASS; Autonomous Collaboration N/A).

## Goals / Non-Goals

### Goals

- Extract one unexported generic helper `callAndUnmarshal[T any]` in
  `internal/adapter/` that owns the Call → error-check → unmarshal
  sequence.
- Migrate the four hard-error batch sites (`Analyze`, `Coverage`,
  `loadBatch`, `Initialize`) to the helper.
- Preserve per-method error context in every wrapped error so operator
  observability is unchanged (Observable Quality).
- Preserve `Session.Initialize` client cleanup (`Close()` on error).
- Add dedicated unit tests for the helper (Testability).
- No observable behavior change; existing tests pass unmodified.

### Non-Goals

- Modifying `internal/protocol/` — the transport layer stays
  usage-agnostic (Composability First). The helper lives in the adapter
  layer where result typing is a legitimate concern.
- Touching `CallStream` / the streaming side-effect path (different
  response model).
- Consolidating `safeSSABuild` (#238, separate change).
- Changing any exported signature or the wire protocol.

## Decisions

### D1: Helper signature and location

```go
// callAndUnmarshal issues a JSON-RPC call for method with params,
// checks transport and protocol errors, and unmarshals the result
// into T. Errors are wrapped with per-method context.
func callAndUnmarshal[T any](
    ctx context.Context,
    client *protocol.Client,
    method string,
    params any,
) (T, error)
```

Placed in `internal/adapter/` (e.g., a new `call.go` file), unexported.
Rationale: the transport layer (`protocol.Client.Call`) intentionally
returns a raw `*protocol.Response` with a `json.RawMessage` result; the
Call→Unmarshal triad is an adapter-layer concern. Putting the generic in
`internal/protocol/` would couple the transport layer to result typing,
violating Composability First.

### D2: Error-context strategy

The helper derives all three wrapped-error prefixes from the `method`
argument so a single call site produces the same distinguishable
context the inline code produces today. Concretely, errors take the form:

- transport: `fmt.Errorf("%s protocol call: %w", method, err)`
- protocol:  `fmt.Errorf("%s protocol error: %s (code %d)", method, resp.Error.Message, resp.Error.Code)`
- unmarshal: `fmt.Errorf("parsing %s result: %w", method, err)`

The `method` value passed is the protocol method constant
(e.g., `protocol.MethodComplexity`). If the existing human-readable
prefixes (e.g., "complexity") differ from the raw method constant, the
call site MAY pass a short label string instead of the constant to keep
the exact legacy wording; the acceptance test only requires that the
method identity is present in the wrapped error, so either is compliant.
The implementer MUST choose whichever keeps existing error strings
closest to current output to satisfy "existing tests pass without
modification".

**`session.go` is the one site that diverges.** For `Analyze`,
`Coverage`, and `loadBatch`, the method constant (`"complexity"`,
`"coverage"`, `"analyze"`) already produces the exact legacy prefixes
(`"<method> protocol call"`, `"<method> protocol error"`,
`"parsing <method> result"`). But `Initialize` currently uses
non-standard prefixes: `"initialize handshake: %w"` (transport),
`"initialize error: %s (code %d)"` (protocol), and
`"parsing initialize result: %w"` (unmarshal). Passing
`protocol.MethodInitialize` (= `"initialize"`) to the D2 templates would
change the first two strings to `"initialize protocol call"` /
`"initialize protocol error"`. Task 2.4 therefore MUST preserve the
legacy `Initialize` error strings. Since the generic helper emits a
single templated form, the implementer keeps the exact legacy wording by
constructing the three `Initialize` error strings at the call site
(wrapping the helper's returned error, or retaining the inline error
formatting for `Initialize` while still delegating the Call+Unmarshal
mechanics). The gate is: `go test` for the adapter package passes without
modifying existing tests, and any test that asserts on `"initialize
handshake"`/`"initialize error"` still holds.

### D3: `fetchTestMappings` handling (resolved)

Inspection of `contract.go:93-116` confirms `fetchTestMappings`
interleaves `p.warn(...)` on every error branch and uses inconsistent
wrapping (bare `err` on transport/unmarshal failure, `"test_mapping: %s"`
on protocol error) as part of its graceful-degradation contract (D7).
The generic helper cannot express the `p.warn` side effects without
either (a) taking a warn callback (over-generalizing the helper for one
caller) or (b) losing the warnings (a behavior regression).

**Decision**: Leave `fetchTestMappings` unchanged. The effective
migration set is the four hard-error sites. This resolves the "migrate
5 vs 4" ambiguity noted during triage: the proposal counts five
candidate sites, but the warning/degradation path is explicitly out of
scope, so four are actually migrated. The specs and tasks reflect four.

### D4: `Session.Initialize` cleanup (resolved)

Inspection of `session.go:87-103` confirms `Initialize` calls
`_ = s.client.Close()` on each error branch (cleanup of a
half-established client after a failed handshake). The helper returns
`(T, error)` and does not manage lifecycle. `Initialize` will call
`callAndUnmarshal`, and when the returned error is non-nil, call
`s.client.Close()` before returning the wrapped error. Cleanup stays at
the call site; no behavior change.

### D5: Test placement and cases

Add helper unit tests in a new `internal/adapter/call_test.go` using
`package adapter` (internal test) — `callAndUnmarshal` is unexported, so
an external `package adapter_test` file cannot reach it. The existing
adapter package already uses internal tests (`sideeffect_test.go`,
`contract_internal_test.go`).

**Test-client construction (resolved).** `protocol.Client` has no
injectable constructor — `protocol.NewClient(binary, args...)` spawns a
subprocess via `exec.LookPath` + `cmd.Start()`, and its fields (`cmd`,
`stdin`, `stdout`, `stderr`) are unexported. Wiring a "minimal client to
controllable stdin/stdout" is therefore **not possible without modifying
`internal/protocol/`**, which is out of scope (Non-Goals). All four error
conditions MUST instead be driven through the existing fake analyzer
binary (`internal/protocol/testdata/fake_analyzer/`, built once in
`TestMain` — see `adapter_test.go`), which already provides every mode
needed:

| Case | Fake-analyzer flag | Mechanism |
|------|--------------------|-----------|
| a. success | (default) `--stdio` | normal typed response |
| b. transport error | `--crash-after=<method>` | subprocess exits; the next `Call` fails reading stdout |
| c. protocol error | `--error-response` | returns a JSON-RPC error object (`resp.Error != nil`) |
| d. unmarshal failure | `--malformed-json` | returns a response whose result cannot decode into `T` |

No new fake-analyzer modes and no `internal/protocol/` changes are
required. Because these tests spawn a subprocess, they are
isolated-behavior tests rather than pure in-process unit tests; the
existing `adapter_test.go` integration suite is the regression net for
the migrated call sites.

Prefer a single table-driven `TestCallAndUnmarshal` with named subtests
(per Go pack TC-006 and AGENTS.md `TestXxx_Description` convention).
Minimum cases (Testability):

1. successful unmarshal (happy path) — assert the returned `T` field values
2. transport (Call) error → wrapped with `%w`, `errors.Is(err, orig)` holds where an original error is available, and the method context prefix appears
3. protocol error (`resp.Error != nil`) → method context prefix + message + code all appear in the string (formatted via `%s`, NOT `%w` — see D2)
4. `json.Unmarshal` failure → wrapped with `%w`, method context prefix appears
5. second result type → verifies generic instantiation and asserts specific field values from the second type (closes the LOW-severity generic-coupling gap flagged in triage)

**Coverage target.** The five cases MUST achieve 100% branch coverage of
`callAndUnmarshal` (success path plus all three error branches).

### D6: Error-chain (`%w`) contract

Transport and unmarshal errors wrap the underlying `error` with `%w`, so
`errors.Is`/`errors.As` unwrapping is part of the observable contract for
those two cases. Protocol errors are formatted with `%s` (not `%w`)
because `resp.Error` is a structured JSON-RPC error object, not a Go
`error` chain value; this matches the current inline behavior at all four
sites (e.g., `complexity.go:49`). Tests assert `errors.Is` for the
`%w` cases and string-content (method + message + code) for the protocol
case.

## Risks / Trade-offs

- **Risk: error-string drift** breaking existing tests. Mitigation: D2
  lets the call site pass the exact legacy label; run existing adapter
  tests unmodified as the gate.
- **Risk: dropping the `resp.Error` check** during extraction (silent
  swallow of protocol errors). Mitigation: dedicated protocol-error unit
  test plus the fake-analyzer integration suite.
- **Trade-off: four sites migrated, not five.** Accepted:
  `fetchTestMappings` keeps its distinct degradation semantics rather
  than forcing an awkward callback-based generalization. This is the
  Zero-Waste-correct outcome (no speculative flexibility).
- **Trade-off: generics require Go 1.18+.** Accepted: module is Go 1.24+.
