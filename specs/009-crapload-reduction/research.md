# Research: CRAPload Reduction

**Feature**: 009-crapload-reduction
**Date**: 2026-02-27

## R1: Dependency Injection Strategy for CLI Commands

**Decision**: Use optional function fields on existing params structs with default-to-production-function semantics.

**Rationale**: The project already uses the testable CLI pattern where commands delegate to `runXxx(params)` functions with params structs containing `io.Writer` for stdout/stderr. Adding function fields (e.g., `analyzeFunc func(...) (*crap.Report, error)`) to these structs is the minimal extension of this existing pattern. When the field is nil, the production function is called; tests set the field to a stub.

**Alternatives considered**:

- **Interface-based injection**: Would require defining new interfaces (`CrapAnalyzer`, `CoverageProvider`) and wrapping production functions in adapter types. More boilerplate, introduces new types into the package namespace, and conflicts with the project's "No global state" and functional style preferences.
- **Package-level variables**: Setting `var analyzeFunc = crap.Analyze` at package level and overriding in tests. Violates the project's "No global state" convention and introduces race conditions in parallel tests.
- **Constructor/factory functions**: `newCrapRunner(opts)` returning a struct with methods. Over-engineered for two CLI commands; adds unnecessary abstraction layers.

## R2: LoadModule Test Fixture Strategy

**Decision**: Use the project's own `go.mod` for the happy-path test and a temporary directory with a minimal `go.mod` + broken `.go` file for the error-filtering test. Guard with `testing.Short()`.

**Rationale**: `LoadModule` wraps `go/packages.Load` which requires real Go source to parse and type-check. Using the project's own module root is the simplest valid input and tests a realistic scenario. For the error-filtering test, a temporary directory with a deliberately broken Go file verifies the exclusion logic. Both tests are inherently slow (they invoke the Go toolchain), so guarding with `testing.Short()` keeps the standard CI suite fast.

**Alternatives considered**:

- **Testdata fixtures only**: Would require maintaining a valid `go.mod` + Go source tree under `internal/loader/testdata/`. Feasible but adds maintenance burden for fixtures that mirror what the project root already provides.
- **Mocking `go/packages.Load`**: Not feasible without making `LoadModule` accept a loader function, which changes the production API for a function that is purely infrastructure. The cost exceeds the benefit.
- **Using the existing analysis testdata fixtures**: These are in `internal/analysis/testdata/src/` but lack standalone `go.mod` files — they rely on the parent module. Not suitable for `LoadModule` which expects a module root.

## R3: AnalyzeP1Effects Decomposition Approach

**Decision**: Extract four handler functions, one per `ast.Node` type switch arm: `detectAssignEffects`, `detectIncDecEffects`, `detectSendEffects`, `detectCallEffects`. The main `ast.Inspect` callback becomes a thin dispatcher.

**Rationale**: The current CC=32 comes from a single `ast.Inspect` callback with a 4-arm type switch (`*ast.AssignStmt`, `*ast.IncDecStmt`, `*ast.SendStmt`, `*ast.CallExpr`), each arm containing nested logic. Extracting one handler per arm creates functions with CC of 5-10 each. The `seen` map and `locals` set are passed as parameters (or captured via a shared analysis context struct) to preserve deduplication semantics. The `ast.Inspect` callback becomes approximately: `switch n := node.(type) { case *ast.AssignStmt: ... case *ast.IncDecStmt: ... }` with each case delegating to a handler that appends to a shared `effects` slice.

**Alternatives considered**:

- **Visitor pattern with interface**: Define a `NodeHandler` interface and register handlers. Over-engineered for 4 handlers; Go's type switch is the idiomatic approach.
- **Separate `ast.Inspect` passes per effect type**: Each handler does its own tree walk. Correct but O(n*k) instead of O(n), and changes the architecture unnecessarily.
- **Analysis context struct with methods**: Wrap `fset`, `info`, `pkg`, `funcName`, `locals`, `seen`, and `effects` in a struct and make handlers methods. Viable and would avoid parameter passing, but introduces a new exported type into the analysis package. The project currently uses plain functions. The decision is to start with plain functions and only introduce a struct if parameter count becomes unwieldy.

## R4: AnalyzeP2Effects Decomposition Approach

**Decision**: Same pattern as P1 — extract per-node-type handlers. The P2 function has a 2-arm type switch (`*ast.GoStmt`, `*ast.CallExpr`), with inline detection for panic, selector-based effects (logging, OS operations, context cancellation), database operations, and callback invocation within the `*ast.CallExpr` arm.

**Rationale**: The P2 function (CC=18) follows the same monolithic callback pattern as P1. The `*ast.CallExpr` arm is the most complex, containing sub-checks for logging, OS exit, panic, exec, database, and HTTP calls. Extracting `detectP2CallEffects` from the call expression arm and `detectGoroutineEffects` from the goroutine arm reduces the per-function CC to approximately 6-8 each.

**Alternatives considered**: Same as R3. The symmetry between P1 and P2 decomposition means the same pattern applies.

## R5: buildContractCoverageFunc Decomposition Strategy

**Decision**: Extract three functions: (1) `resolvePackagePaths` for pattern expansion + test-variant filtering, (2) `analyzePackageCoverage` for the per-package pipeline (analysis → classify → test-load → quality assess), and (3) keep `buildContractCoverageFunc` as a thin coordinator that calls the extracted functions and builds the final closure.

**Rationale**: The current function (CC=18, 100 lines) has three distinct responsibilities crammed together. The package resolution step (lines 516-538 of main.go) filters out `_test` packages and resolves patterns to individual paths. The per-package pipeline (lines 561-594) runs analysis, classification, test loading, and quality assessment with 5 separate `continue`-on-error branches. The aggregation step builds a map from reports. Extracting these makes each step independently testable. The coordinator retains the nil-return-on-empty-map logic.

**Alternatives considered**:

- **Pipeline struct with methods**: Wrap the entire pipeline in a struct. Would make testing cleaner via method injection, but introduces a type for a single call site. The project's codebase currently has no pipeline structs in `cmd/gaze/`.
- **Functional pipeline composition**: Chain functions like `resolvePackagePaths |> map(analyzePackageCoverage) |> buildCoverageMap`. Elegant but Go lacks first-class pipeline operators; the boilerplate would be worse than explicit calls.
- **No decomposition, just more tests**: Would improve line coverage but not CC. With CC=18, even 100% coverage yields CRAP=18, still above the threshold. Decomposition is required to bring individual functions below 15.

## R6: docscan.Filter Contract Coverage Gap

**Decision**: Create a dedicated `internal/docscan/filter_test.go` with table-driven tests using `package docscan_test` (external test package) that directly call `docscan.Filter` and assert on the boolean return value.

**Rationale**: The existing tests in `scanner_test.go` test `Filter` indirectly through the `Scan` function, but the quality pipeline does not map these assertions back to `Filter`'s contract because the assertions target `Scan`'s output (a list of documents), not `Filter`'s output (a boolean). Direct tests that call `Filter(path, cfg)` and assert `== true` / `== false` will be recognized as contract-level assertions on the return value.

**Alternatives considered**:

- **Modify existing scanner_test.go tests to also assert Filter directly**: Would mix concerns (integration tests testing both Scan and Filter). Cleaner to have a dedicated file.
- **Internal test package (`package docscan`)**: Would allow testing unexported helpers like `matchGlob` directly. However, the spec only requires contract coverage on the exported `Filter` function. Using the external test package is cleaner and tests the public API surface.
