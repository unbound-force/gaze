<!--
  [P] marks tasks eligible for parallel execution.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Create the shared ssaguard package

- [x] 1.1 Create `internal/ssaguard/ssaguard.go` with a package doc
  comment and the exported `SafeSSABuild(buildFn func()) (panicVal any)`
  function. Body: `defer func() { panicVal = recover() }()`, call
  `buildFn()`, return `nil`. Import nothing beyond the standard library
  (D1, D3).
- [x] 1.2 Write the `SafeSSABuild` GoDoc comment documenting the mandatory
  `ssa.BuildSerially` caller precondition and the goroutine-scoped
  `recover()` rationale (D4; spec 033 invariant). State explicitly that
  the guard does NOT validate the build mode at runtime.
- [x] 1.3 Create `internal/ssaguard/ssaguard_test.go` with the three
  canonical test cases: `TestSafeSSABuild_NoPanic` (returns nil),
  `TestSafeSSABuild_PanicString` (returns recovered string),
  `TestSafeSSABuild_PanicError` (returns recovered error) (D5). Use the
  standard library `testing` package only; assert with `t.Errorf` /
  `t.Fatalf`. Coverage strategy: unit tests only; these three cases
  achieve 100% branch coverage of `SafeSSABuild` (deferred `recover()`
  path via the two panic cases, normal `return nil` path via the no-panic
  case) — the complete set of reachable branches.

## 2. Update callers to use the shared guard

- [x] 2.1 [P] In `internal/analysis/mutation.go`: import
  `github.com/unbound-force/gaze/internal/ssaguard`, replace the
  `safeSSABuild(prog.Build)` call in `BuildSSA` with
  `ssaguard.SafeSSABuild(prog.Build)`, and delete the local
  `safeSSABuild` function and its doc comment. Keep the
  `ssa.BuildSerially` mode flag in the `ssautil.AllPackages` call. Retain
  the existing `log.Warn`/`log.Debug` recovery-site calls in `BuildSSA` —
  do NOT move logging into `ssaguard.SafeSSABuild`.
- [x] 2.2 [P] In `internal/quality/pairing.go`: import
  `github.com/unbound-force/gaze/internal/ssaguard`, replace the
  `safeSSABuild(prog.Build)` call in `BuildTestSSA` with
  `ssaguard.SafeSSABuild(prog.Build)`, and delete the local
  `safeSSABuild` function, its doc comment, and its duplication-lineage
  comment (the R3 reference). Keep the `ssa.BuildSerially` mode flag.
  Retain the existing `log.Warn`/`log.Debug` recovery-site calls in
  `BuildTestSSA` — do NOT move logging into `ssaguard.SafeSSABuild`.

## 3. Remove duplicated tests and shims

- [x] 3.1 [P] In `internal/analysis/mutation_test.go`: remove the three
  duplicated `TestSafeSSABuild_*` tests.
- [x] 3.2 [P] In `internal/analysis/export_test.go`: remove the
  `SafeSSABuild` shim line only; retain all other shims in the file.
- [x] 3.3 [P] In `internal/quality/pairing_test.go`: remove the three
  duplicated `TestSafeSSABuild_*` tests.
- [x] 3.4 [P] In `internal/quality/export_test.go`: remove the
  `SafeSSABuild` shim line only; retain all other shims in the file.

## 4. Verify (CI Parity Gate)

- [x] 4.1 `go build ./cmd/gaze` succeeds.
- [x] 4.2 `go test -race -count=1 -short ./...` passes (all packages,
  including the new `internal/ssaguard`).
- [x] 4.3 `golangci-lint run` is clean (no new findings; confirm no
  `ssautil` shadow and correct import grouping).
- [x] 4.4 Confirm no remaining `safeSSABuild` definition exists in
  `internal/analysis` or `internal/quality` (e.g., `grep -rn
  "func safeSSABuild"` returns nothing) and that both call sites now
  reference `ssaguard.SafeSSABuild`.

## 5. Constitution alignment check

- [x] 5.1 Verify Composability (II): `internal/ssaguard` imports only the
  standard library; no new external dependency introduced.
- [x] 5.2 Verify Testability (IV): `SafeSSABuild` is directly tested in
  `internal/ssaguard` without `export_test.go` shims; behavioral coverage
  of the three cases is preserved; race tests pass.

## 6. Documentation validation gate

- [x] 6.1 Assess documentation impact. No user-facing behavior, CLI,
  flag, or output-format change — README and website issue are NOT
  required (internal refactor). Add `ssaguard/` with a one-line
  description to the AGENTS.md "Architecture" internal-package list (which
  does enumerate every internal package) at an appropriate position.

<!-- spec-review: passed -->

<!-- code-review: passed -->
