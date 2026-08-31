## ADDED Requirements

### Requirement: Shared SSA panic guard

The system MUST provide a single, shared panic-recovery guard for
`ssa.Program.Build()` in a dedicated leaf package `internal/ssaguard`.
The exported function `SafeSSABuild(buildFn func()) (panicVal any)` MUST
execute `buildFn` under a deferred `recover()` and MUST return the
recovered panic value, or `nil` when no panic occurred. The package MUST
import nothing beyond the Go standard library.

#### Scenario: No panic returns nil

- **GIVEN** a `buildFn` that completes without panicking
- **WHEN** `SafeSSABuild(buildFn)` is called
- **THEN** it returns `nil`

#### Scenario: String panic is recovered

- **GIVEN** a `buildFn` that calls `panic("boom")`
- **WHEN** `SafeSSABuild(buildFn)` is called
- **THEN** it returns the recovered value `"boom"` and does not
  propagate the panic to the caller

#### Scenario: Error panic is recovered

- **GIVEN** a `buildFn` that calls `panic(err)` with an `error` value
- **WHEN** `SafeSSABuild(buildFn)` is called
- **THEN** it returns the recovered `error` value and does not propagate
  the panic to the caller

These three scenarios exercise all reachable branches of `SafeSSABuild`:
the deferred `recover()` capturing a non-nil panic value (scenarios 2-3)
and the normal `return nil` path when `recover()` returns nil (scenario
1). No additional branches exist, so the three cases constitute complete
branch coverage.

### Requirement: BuildSerially invariant documentation

The `SafeSSABuild` GoDoc comment MUST document that callers MUST build
the SSA program with the `ssa.BuildSerially` mode flag set. The
documentation MUST explain that `recover()` is goroutine-scoped and that
without `ssa.BuildSerially`, `prog.Build()` spawns child goroutines whose
panics escape the guard, causing an unrecoverable process crash. This is
a documentation-only precondition; `SafeSSABuild` MUST NOT attempt
runtime validation of the SSA build mode (the mode flags are set by
callers before `ssautil.AllPackages`, outside this function's scope).

#### Scenario: Guard documents the serial-build precondition

- **GIVEN** the exported `SafeSSABuild` function
- **WHEN** a developer reads its GoDoc comment
- **THEN** the comment states the `ssa.BuildSerially` requirement and the
  goroutine-scoped `recover()` rationale

### Requirement: Both SSA build sites use the shared guard

Both `internal/analysis.BuildSSA` and `internal/quality.BuildTestSSA`
MUST call `internal/ssaguard.SafeSSABuild` for panic recovery around
`prog.Build()`. Neither package MUST retain a local copy of the guard
function. Both packages MUST continue to set the `ssa.BuildSerially` mode
flag when constructing the SSA program.

Both callers MUST retain their existing `log.Warn` and `log.Debug` calls
at the panic-recovery site. Logging MUST NOT be moved into `SafeSSABuild`
because: (a) the guard receives only a `func()` and has no visibility
into the package being built, so moving the logs would lose the
package-path diagnostic context; and (b) adding a logger dependency would
violate the standard-library-only constraint on `internal/ssaguard`. The
package-path-annotated warning is the only runtime observability signal
for degraded SSA builds and MUST be preserved in the callers.

#### Scenario: analysis package delegates to shared guard

- **GIVEN** `internal/analysis.BuildSSA` constructing an SSA program with
  `ssa.BuildSerially` set
- **WHEN** it builds the program
- **THEN** it invokes `ssaguard.SafeSSABuild(prog.Build)`, no local
  `safeSSABuild` function exists in the package, and the existing
  `log.Warn`/`log.Debug` recovery-site calls remain in `BuildSSA`

#### Scenario: quality package delegates to shared guard

- **GIVEN** `internal/quality.BuildTestSSA` constructing an SSA program
  with `ssa.BuildSerially` set
- **WHEN** it builds the program
- **THEN** it invokes `ssaguard.SafeSSABuild(prog.Build)`, no local
  `safeSSABuild` function exists in the package, and the existing
  `log.Warn`/`log.Debug` recovery-site calls remain in `BuildTestSSA`

## REMOVED Requirements

### Requirement: Duplicated per-package SSA panic guard

Removed. The previous design (spec 021, research decision R3) duplicated
the `safeSSABuild` guard in both `internal/analysis` and
`internal/quality` to keep each package dependency-light. This rationale
no longer holds: the guard has zero external dependencies, so a shared
leaf package introduces no coupling. The duplicated function definitions,
their `export_test.go` shims, and their duplicated test triads are
removed in favor of the shared `internal/ssaguard` package.
