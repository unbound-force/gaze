## Context

`safeSSABuild` is a 7-line `recover()` guard around `prog.Build()` that
is byte-for-byte duplicated in two packages:

- `internal/analysis/mutation.go` — used by `BuildSSA`, which builds the
  SSA program via
  `ssautil.AllPackages([]*packages.Package{pkg},
  ssa.InstantiateGenerics|ssa.BuildSerially)` and then calls
  `safeSSABuild(prog.Build)`.
- `internal/quality/pairing.go` — used by `BuildTestSSA`, identical shape,
  plus a duplication-lineage comment citing spec 021 research decision R3.

The duplication also spans:
- `export_test.go` shims exposing the unexported guard to external-package
  tests (`internal/analysis/export_test.go`,
  `internal/quality/export_test.go`).
- Three-case test triads in both `mutation_test.go` and `pairing_test.go`
  (`TestSafeSSABuild_NoPanic` / `_PanicString` / `_PanicError`).

The guard encodes a safety invariant established by spec 033 (issue #33):
`recover()` is goroutine-scoped, so it only catches `prog.Build()` panics
when the caller sets `ssa.BuildSerially` (forcing serial construction on
the calling goroutine). Without that flag, `Build()` spawns child
goroutines whose panics escape the guard and crash the process. The
current design relies on this invariant living implicitly next to each
caller — a fragile arrangement that already required a synchronized
two-site fix during spec 033.

## Goals / Non-Goals

### Goals
- Establish a single source of truth for the SSA panic guard in a new
  `internal/ssaguard` leaf package.
- Export `SafeSSABuild(buildFn func()) (panicVal any)` so it is directly
  testable without `export_test.go` shim indirection.
- Document the `ssa.BuildSerially` precondition authoritatively in one
  GoDoc comment.
- Update both callers to delegate; remove all duplicated definitions,
  shims, and tests.
- Preserve exact runtime behavior and keep race tests green.

### Non-Goals
- Runtime validation of the SSA build mode. `SafeSSABuild` receives only a
  `func()`; it has no visibility into how the program was constructed. The
  `BuildSerially` flag is set by callers before `ssautil.AllPackages`,
  outside this function's scope. Enforcement stays documentation-based.
- Changing the callers' SSA construction (mode flags, package loading)
  beyond swapping the local guard for the shared one.
- Any change to Gaze's exported API, CLI, output formats, or config.

## Decisions

**D1 — New package name `ssaguard`, not `ssautil`.** Both `mutation.go`
and `pairing.go` already import `golang.org/x/tools/go/ssa/ssautil`.
Naming the new package `ssautil` would shadow that import at call sites
and invite confusion/misuse. `ssaguard` is descriptive (it guards SSA
builds) and follows the project's flat `internal/` naming convention
(`loader`, `taxonomy`, `classify`, `docscan`).

**D2 — Export the function.** Making `SafeSSABuild` exported removes the
need for `export_test.go` shims in both consumers. This improves
testability (Constitution IV) by allowing direct in-package unit tests in
`internal/ssaguard/ssaguard_test.go`, and eliminates two lines of shim
indirection.

**D3 — Reverse spec 021 R3.** R3 duplicated the guard to keep packages
dependency-light. Since the guard has zero external dependencies (input
`func()`, output `any`, body uses only `recover()`), a shared leaf
package adds no transitive coupling. Both callers already depend on
`x/tools/go/ssa`; the new internal edge is strictly lighter. The reversal
is recorded in the delta spec's REMOVED section.

**D4 — Documentation-only invariant.** The `BuildSerially` requirement is
captured in the `SafeSSABuild` GoDoc as a mandatory caller precondition,
with the goroutine-scoped `recover()` rationale. This centralizes the
knowledge that spec 033 previously spread across two files, reducing the
risk that a future third build site omits the flag.

**D5 — Consolidate tests, preserve behavioral coverage.** The three
canonical cases (no-panic → nil; string panic → recovered string; error
panic → recovered error) move into `internal/ssaguard/ssaguard_test.go`.
The six duplicated tests are removed. Net test count is -3, but the same
logic is now exercised once in its canonical home; behavioral coverage is
neutral-to-positive.

**Coverage strategy (Constitution IV)**: Unit tests only. The three test
cases achieve 100% branch coverage of the 7-line `SafeSSABuild` function:
the deferred `recover()` path is exercised by the two panic scenarios
(string and error), and the normal `return nil` path is exercised by the
no-panic scenario. These three branches are the complete set — the
function has no other reachable paths. No integration or e2e tests are
required because the guard is a pure recovery wrapper with zero
dependencies. Target: 100% branch coverage of the new package.

**Constitution ties** (Gaze project constitution,
`.specify/memory/constitution.md`): This is an internal refactor —
Accuracy (I) is N/A (no analysis behavior/output change). Minimal
Assumptions (II) holds: `ssaguard` is a standalone zero-dependency leaf
that adds no new host-project assumptions. Actionable Output (III) is N/A
(no human- or machine-readable output change); the documented invariant
improves internal maintainability of a safety property. Testability (IV)
improves via D2 and D5.

## Risks / Trade-offs

- **R1 — Loss of caller proximity for the invariant.** Moving the guard
  away from its callers means the `BuildSerially` requirement is no longer
  physically adjacent to `ssautil.AllPackages`. Mitigation: D4's mandatory
  GoDoc precondition makes the invariant explicit and discoverable at the
  guard's definition, which is stronger than implicit proximity.
- **R2 — Net test count decreases by 3.** Accepted trade-off: the removed
  tests were duplicates of the same 7-line logic. The acceptance criterion
  "net behavioral coverage does not decrease" is satisfied because the
  three canonical cases remain, now tested once. Race tests
  (`-race -count=1`) must still pass.
- **R3 — Mechanical churn across six files.** Low risk: the guard is pure
  (no state, no side effects) and both callers use it identically
  (`safeSSABuild(prog.Build)` → `ssaguard.SafeSSABuild(prog.Build)`).
  Verified by building and running the full `-race -count=1` suite plus
  `golangci-lint run` per the CI Parity Gate.
