<!--
  [P] marks tasks eligible for parallel execution.
  Migrations 2.1-2.4 each touch a distinct file and depend only on
  the helper (Group 1), so they are parallel-eligible relative to
  each other. Group 1 must complete first.
-->

## 1. Create the generic helper

- [x] 1.1 Add `internal/adapter/call.go` with unexported
  `func callAndUnmarshal[T any](ctx context.Context, client *protocol.Client, method string, params any) (T, error)` implementing Call → transport-error check → `resp.Error` check → `json.Unmarshal[T]`, with per-method error context per design D2 (transport: `"%s protocol call: %w"`, protocol: `"%s protocol error: %s (code %d)"`, unmarshal: `"parsing %s result: %w"`). Include a GoDoc comment on the function.

## 2. Tests for the helper (TDD — write before migrating call sites)

- [x] 2.1 Add `internal/adapter/call_test.go` using `package adapter` (internal test — `callAndUnmarshal` is unexported). Prefer a single table-driven `TestCallAndUnmarshal` with named subtests (Go pack TC-006). Drive all conditions through the fake analyzer binary built in `TestMain` (design D5 table): (a) successful unmarshal via default `--stdio`, asserting returned field values; (b) transport (Call) error via `--crash-after=<method>`, asserting the method context prefix appears and `errors.Is(err, orig)` holds where an original error is available; (c) protocol error via `--error-response`, asserting method prefix + message + code appear (formatted with `%s`, not `%w` — design D6); (d) `json.Unmarshal` failure via `--malformed-json`, asserting the method prefix appears and the error wraps with `%w`.
- [x] 2.2 Add a 5th subtest exercising a second result type to verify generic instantiation, asserting specific field values from the second type (closes the LOW-severity generic-coupling gap; design D5). Confirm the 5 cases give 100% branch coverage of `callAndUnmarshal`.

## 3. Migrate hard-error call sites (each a different file — parallel)

- [x] 3.1 [P] `internal/adapter/complexity.go`: replace the inline Call+Unmarshal in `Analyze` (41-55) with `callAndUnmarshal[protocol.ComplexityResult]`, then call `convertComplexity`. Preserve the existing error label so current tests pass unmodified (design D2).
- [x] 3.2 [P] `internal/adapter/coverage.go`: replace the inline Call+Unmarshal in `Coverage` (32-46) with `callAndUnmarshal[protocol.CoverageResult]`. Preserve existing error label.
- [x] 3.3 [P] `internal/adapter/sideeffect.go`: replace the inline Call+Unmarshal in `loadBatch` (114-128) with `callAndUnmarshal[protocol.AnalyzeResult]`, then assign the converted result to `a.cached` as today. `loadBatch` returns only `error`; adapt the `(T, error)` result accordingly. Do NOT touch `CallStream` / the streaming path.
- [x] 3.4 [P] `internal/adapter/session.go`: RESOLVED via design D2's sanctioned inline-retention option. `Initialize` retains its original inline Call → 3-branch error handling UNCHANGED because the collapsed generic helper cannot reproduce Initialize's THREE DISTINCT legacy strings (`"initialize handshake: %w"` transport / `"initialize error: %s (code %d)"` protocol / `"parsing initialize result: %w"` unmarshal) without fragile prefix translation. Design D2 (design.md 88-106) explicitly permits retaining inline formatting for Initialize. session.go is byte-identical to pre-migration state; `s.client.Close()` on each error branch preserved (design D4). Effective helper adoption is 3 sites (complexity/coverage/sideeffect), not 4 — legacy operator-facing strings prioritized per task's "PRESERVE the legacy Initialize error strings".

## 4. Confirm exclusions untouched

- [x] 4.1 Verify `fetchTestMappings` (`internal/adapter/contract.go:93-116`) is unchanged — `git diff main -- contract.go` is empty. Its `p.warn()` graceful-degradation path stays as-is (design D3).
- [x] 4.2 Verify `CallStream` and the streaming side-effect path are unchanged — `sideeffect.go` diff shows ONLY the `loadBatch` Call+Unmarshal replacement; `loadStreaming`/`CallStream`/`parseSideEffectStream` untouched.
- [x] 4.3 Verify `internal/protocol/` has no modifications — `git diff main -- internal/protocol/` empty; `git diff main -- session.go contract.go` empty. Changed files: call.go (new), call_test.go (new), complexity.go, coverage.go, sideeffect.go (loadBatch only). session.go shows NO diff (reverted per 3.4).

## 5. Verification & gates

- [x] 5.1 Run `go build ./cmd/gaze` — builds clean (CMD_GAZE_BUILD_OK).
- [x] 5.2 Run `go test -race -count=1 -short ./...` — all pass, existing adapter tests pass WITHOUT modification (adapter 2.56s incl new TestCallAndUnmarshal 5 subtests + unmodified existing tests; all packages ok).
- [x] 5.3 Run `golangci-lint run` — zero issues (exit 0).
- [x] 5.4 CI parity: test.yml (`go build ./...` + `go test -race -count=1 -short -timeout 15m -coverprofile=coverage.out ./...`) and mega-linter (`golangci-lint run`) — all satisfied by the gate run above.
- [x] 5.5 Constitution alignment verified: Composability First (empty `git diff main -- internal/protocol/`, no new deps), Observable Quality (per-method error context preserved, asserted by TestCallAndUnmarshal), Testability (helper tested in isolation). All hold as claimed in proposal.md.
- [x] 5.6 Ran `/review-council` (Code Review Mode) — all 5 divisor reviewers (adversary/architect/guard/sre/testing) returned APPROVE; single consistent LOW (test temp-dir leak) accepted as non-blocking by all reviewers.

## 6. Documentation validation gate

- [x] 6.1 Added AGENTS.md "Recent Changes" entry for `adapter-call-unmarshal-helper` (new `callAndUnmarshal` helper; 3 migrated sites complexity.Analyze/coverage.Coverage/sideeffect.loadBatch; session.Initialize retained inline per design D2 to preserve 3 distinct legacy strings; error-chain contract D6; exclusions CallStream/fetchTestMappings warn path/internal/protocol/).
- [x] 6.2 Confirmed no README, CLI-help, website, or GoDoc-on-exported-API updates needed — pure internal refactoring, no user-facing behavior change, `callAndUnmarshal` is unexported, no exported-signature change (Documentation Validation Gate).

<!-- spec-review: passed -->
<!-- code-review: passed -->
