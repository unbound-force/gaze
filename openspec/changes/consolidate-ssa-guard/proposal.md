## Why

The `safeSSABuild` helper — a `recover()` guard wrapped around
`prog.Build()` — is byte-for-byte duplicated across two packages:

- `internal/analysis/mutation.go` (called by `BuildSSA`)
- `internal/quality/pairing.go` (called by `BuildTestSSA`)

The duplication extends beyond the 7-line function body into parallel
`export_test.go` shims and three near-identical test triads
(`TestSafeSSABuild_NoPanic` / `_PanicString` / `_PanicError`) in each
package. This is a direct violation of the **Zero-Waste Mandate** (no
duplicated code, no maintenance drift).

The original decision to duplicate (spec 021, research decision R3) chose
to keep both packages "dependency-light." That rationale is now stale:
`safeSSABuild` takes a `func()` and returns `any` — it has **zero
external dependencies**. A shared leaf package introduces no transitive
coupling into either consumer.

More importantly, `safeSSABuild` encodes a **safety-critical invariant**:
`recover()` is goroutine-scoped and only catches panics from
`prog.Build()` because the callers set `ssa.BuildSerially` (established
by spec 033, issue #33). This invariant currently lives implicitly via
caller proximity in two separate files. Spec 033 already demonstrated
the dual-maintenance hazard — both copies had to be corrected in lock
step. Centralizing the helper and documenting the `BuildSerially`
requirement in one authoritative place removes the drift risk and the
chance a future third SSA build site silently omits `BuildSerially`
(which would cause an unrecoverable panic → process crash).

This change was triaged (issue #238, split from #201) and received a
unanimous VALID / enhancement verdict from all five Divisor reviewers.

## What Changes

- Create a new leaf package `internal/ssaguard/` exporting
  `SafeSSABuild(buildFn func()) (panicVal any)`.
- Document the `ssa.BuildSerially` requirement in the exported function's
  GoDoc comment as a mandatory caller precondition (documentation only —
  not runtime validation, since the mode flags are set by callers before
  `ssautil.AllPackages`, outside this function's scope).
- Update `internal/analysis/mutation.go` (`BuildSSA`) to call
  `ssaguard.SafeSSABuild(prog.Build)`; remove the local `safeSSABuild`.
- Update `internal/quality/pairing.go` (`BuildTestSSA`) to call
  `ssaguard.SafeSSABuild(prog.Build)`; remove the local `safeSSABuild`
  and its duplication-lineage comment.
- Consolidate the three test cases into `internal/ssaguard/`; remove the
  six duplicated tests from `mutation_test.go` and `pairing_test.go`.
- Remove the `SafeSSABuild` shim line from both
  `internal/analysis/export_test.go` and `internal/quality/export_test.go`
  (retaining all other shims in each file).

The package is named `ssaguard` (not `ssautil`) deliberately, to avoid
shadowing `golang.org/x/tools/go/ssa/ssautil`, which is already imported
in both `mutation.go` and `pairing.go`.

## Capabilities

### New Capabilities
- `internal/ssaguard.SafeSSABuild`: Single-source-of-truth panic guard
  for `ssa.Program.Build()`, with the `BuildSerially` invariant
  documented in its GoDoc.

### Modified Capabilities
- `internal/analysis.BuildSSA`: now delegates panic recovery to
  `ssaguard.SafeSSABuild`; behavior unchanged.
- `internal/quality.BuildTestSSA`: now delegates panic recovery to
  `ssaguard.SafeSSABuild`; behavior unchanged.

### Removed Capabilities
- `internal/analysis.safeSSABuild` (unexported): removed; folded into
  `internal/ssaguard`.
- `internal/quality.safeSSABuild` (unexported): removed; folded into
  `internal/ssaguard`.

## Impact

- **Files changed**: `internal/analysis/mutation.go`,
  `internal/analysis/mutation_test.go`,
  `internal/analysis/export_test.go`,
  `internal/quality/pairing.go`, `internal/quality/pairing_test.go`,
  `internal/quality/export_test.go`.
- **Files added**: `internal/ssaguard/ssaguard.go`,
  `internal/ssaguard/ssaguard_test.go`.
- **Behavior**: No behavioral change. Pure internal refactor. No exported
  API surface (all symbols under `internal/`), no CLI change, no config
  change, no release-pipeline change.
- **Tests**: Net test count decreases by 3 (six duplicates → three
  shared), but behavioral coverage is neutral-to-positive — the same
  logic is now tested once in its canonical home instead of twice.
- **Dependency graph**: adds two leaf edges
  (`analysis → ssaguard`, `quality → ssaguard`); `ssaguard` itself
  imports nothing beyond the standard library.

## Constitution Alignment

Assessed against the Gaze project constitution
(`.specify/memory/constitution.md`), the highest-authority document for
this repository.

### I. Accuracy

**Assessment**: N/A

This is an internal implementation refactor with no behavioral change.
Side-effect detection, CRAP scoring, and every analysis result are
byte-for-byte identical before and after. No false positives or false
negatives are introduced or removed.

### II. Minimal Assumptions

**Assessment**: PASS

No new assumptions about the host project's language, test framework, or
coding style are introduced. The new `internal/ssaguard` package is a
self-contained, zero-dependency leaf. Both consumers already depend on
`golang.org/x/tools/go/ssa`; the new internal edge is strictly lighter
than their existing dependency graph.

### III. Actionable Output

**Assessment**: N/A

No change to Gaze's human-readable or machine-parseable output. The
change improves internal maintainability of a safety invariant by
documenting the `ssa.BuildSerially` precondition in a single
authoritative GoDoc comment, reducing the chance of silent panic-escape
regressions.

### IV. Testability

**Assessment**: PASS

Testability improves. Exporting `SafeSSABuild` as a first-class function
eliminates the `export_test.go` shim indirection required to test the
unexported copies — the function becomes directly testable in isolation.
The consolidated test triad verifies the observable behavior (returned
panic value for no-panic, string-panic, and error-panic cases) rather
than implementation details.

**Coverage strategy**: Unit tests only. The three canonical test cases
achieve 100% branch coverage of `SafeSSABuild` — the deferred `recover()`
path (the two panic scenarios) and the normal `return nil` path (the
no-panic scenario). No integration or e2e tests are required; the guard
is a pure recovery wrapper with no dependencies. The net test count
decreases by 3 (six duplicates removed, three canonical tests added), but
behavioral coverage is neutral-to-positive: the same logic is now tested
once in its canonical home instead of twice. Race tests
(`-race -count=1`) continue to pass. The measurable acceptance criterion
is: all three canonical test cases exist in
`internal/ssaguard/ssaguard_test.go` and pass under `-race -count=1`.
