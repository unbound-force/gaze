# Contract Coverage Investigation Report

**Date:** 2026-02-25  
**Branch:** `006-agent-quality-report-enhancements`  
**Spec**: [spec.md](spec.md)  
**Goal:** Improve contract coverage across the three identified tracks:
1. Write tests for zero-coverage packages (`internal/crap`, `buildContractCoverageFunc`)
2. Fix assertion-to-effect mapping accuracy from 73.8% to ≥ 90% (TODO #6)
3. Add GoDoc to reduce the 68% ambiguous classification rate

---

## 1. Toolchain Issue (Blocking Test Execution)

`GOENV_VERSION` in this environment is pinned to `1.11.4`, but `go.mod` requires Go `1.24.2`. The installed versions under `~/.goenv/versions/` are `1.11.4`, `1.22.4`, `1.24.0`, and `1.24.6`. Neither `1.24.0` nor `1.24.6` matches the exact `go 1.24.2` directive in `go.mod`; Go's toolchain selection logic treats any mismatch as a "updates to go.mod needed" error unless `-mod=mod` is passed explicitly.

**Workaround:** All test runs in this session require:
```bash
~/.goenv/versions/1.24.6/bin/go test -mod=mod -count=1 ...
```

**Impact:** Any automated test commands from `AGENTS.md` (e.g., `go test -race -count=1 -short ./...`) fail silently when the toolchain dispatcher routes to `1.11.4`. This affects all verification during implementation and makes it easy to accidentally think a test passed (the 1.11 binary errors out before compiling anything).

---

## 2. Mapping Accuracy: Confirmed Root Causes

Ran `TestSC003_MappingAccuracy` and `TestSC001_ContractCoverageAccuracy` with verbose output. The 11 unmapped assertions break down as follows:

### Cause A — Helper param tracing (4 unmapped, helpers fixture)

All 4 unmapped assertions come from the `helpers` fixture. `TestMultiply`, `TestSafeDivide`, and `TestSafeDivide_ZeroError` use helper functions like `assertEqual(t, got, 12)`. The assertion sites are detected at `depth=1` inside the helper body (e.g., `if got != want { t.Errorf }`). The identifiers `got` and `want` in that body are the **helper's parameter objects**, not the test function's local variables. `matchAssertionToEffect` resolves them via `TypesInfo.Uses` and looks them up in `objToEffectID`, which was built from the test function's variable assignments — so no match is found.

This is the gap explicitly named in the TODO #6 comment. The fix is clear:  
- Add a `ParamSubstitutions map[types.Object]types.Object` field to `AssertionSite`  
- In `detectHelperAssertions`, after calling `d.detect(helperDecl, depth+1)`, build a substitution map from the helper's parameter objects → the caller's argument objects (using the call's `Args` and the helper's `Params.List` against `TypesInfo.Defs`/`TypesInfo.Uses`), then attach the composed substitution to each returned site  
- In `matchAssertionToEffect`, after resolving an identifier's `types.Object`, apply the substitution chain before looking it up in `objToEffectID`

### Cause B — Inline return value assertion (3 unmapped, welltested fixture)

`TestCounter_Increment` has this assertion:
```go
if c.Value() != 5 {
    t.Errorf("Value() = %d, want 5", c.Value())
}
```
`c.Value()` is called **inline inside the condition**, not assigned to a variable first. `traceReturnValues` only traces values that appear on the LHS of an assignment statement (`:=` or `=`). It uses `findAssignLHS` which walks the AST looking for `*ast.AssignStmt` nodes. A bare call expression inside an `if` condition produces no assignment and therefore no entry in `objToEffectID`.

This assertion is unmapped for **both** the `(*Counter).Increment` target (1 unmapped) and the `(*Counter).Value` target (2 unmapped — neither assertion maps because `Value`'s return was never assigned).

**Fix:** Extend `traceReturnValues` (or add a parallel `traceInlineCallReturn`) to detect the pattern `if f() != X` and `if X != f()`, extract the call position, and map the caller's return effect to a synthetic "inline return" entry in `objToEffectID`. This is more involved than the helper param fix.

### Cause C — The remaining 4 unmapped (undertested + other fixtures)

From the `undertested` fixture: `TestStore_Set` calls `NewStore`, `Set`, and `Get`. `InferTargets` returns all three as targets. When the assertions are evaluated against `Set`'s and `NewStore`'s effects, the test's assertion variables (assigned from `Get`'s return) are in `objToEffectID` only for the `Get` target — not for `Set` or `NewStore`. This is structurally correct behaviour (the test legitimately does not assert on `Set`'s contractual outputs), but those assertion sites count as "unmapped" in the SC-003 accuracy measurement.

These are not actual mapping bugs — they are genuine coverage gaps (the fixture is designed to be undertested). The SC-003 test conflates "not mapped because the test doesn't assert on this effect" with "not mapped because of a mapping bug."

---

## 3. "Testify Field-Access Patterns" — Label is Inaccurate

The TODO #6 comment in `quality_test.go:1058` names two causes: "helper param tracing" and "testify field-access patterns." After running the tests, the multilib testify assertions are **fully mapping** today:

```
[multilib] TestNewUser_Testify -> NewUser: 100% (2/2)
[multilib] TestNewUser_Require -> NewUser: 100% (2/2)
[multilib] TestSum_Testify -> Sum: 100% (1/1)
[multilib] TestDivide_Mixed -> Divide: 100% (2/2)
```

`assert.Equal(t, "Alice", user.Name)` works because `ast.Inspect` visits the `user` identifier inside the `SelectorExpr`, and `TypesInfo.Uses[user_ident]` resolves to the same `*types.Var` as `TypesInfo.Defs[user_ident_at_assignment]`. This has either already been fixed or was never actually broken. The "testify field-access" label in the TODO comment appears to be a stale note.

**Implication:** Fixing helpers alone accounts for 4 of the 11 unmapped assertions. Fixing the inline-call pattern accounts for 3 more. The final 4 are not mapping bugs — they are the undertested fixture working as designed.

---

## 4. Testing `crap.Analyze` and `ParseCoverProfile`

The `crap_test.go` is already an internal package test (`package crap`) with 40+ test cases covering `Formula`, `ClassifyQuadrant`, `buildSummary`, `buildCoverMap`, `lookupCoverage`, `resolvePatterns`, `isGeneratedFile`, `findFunctions`, `funcCoverage`, `resolveFilePath`, `readModulePath`, `shortenPath`, `WriteJSON`, and `WriteText`.

**What is missing:**
- `ParseCoverProfile` — requires a real coverage profile file on disk and a Go module directory with source files that match the profile's import paths. These do not currently exist as testdata.
- `Analyze` — the top-level function. It calls `generateCoverProfile` which spawns `go test -coverprofile`. The `CoverProfile` option bypasses the subprocess, but the function then calls `gocyclo.Analyze(absPaths, ...)` which requires real Go source files on disk, and `ParseCoverProfile` which requires the profile to match those files.

**Approach:** Create temp modules in `t.TempDir()` for both tests:
- Write a `go.mod`, a source file, and a synthetic coverage profile that matches the source line numbers
- Pass `CoverProfile = tmpProfile` to `Analyze` to skip the subprocess
- Verify the returned `Report.Scores` contains correct `CRAP`, `LineCoverage`, and `Function` values

The line-number alignment in synthetic profiles is fragile: the profile uses `startline.col,endline.col` and `funcCoverage` uses the extent boundaries from `findFunctions`. As long as the block start and end lines are strictly within the function's `startLine..endLine` range, the overlap check passes.

---

## 5. Testing `buildContractCoverageFunc`

`buildContractCoverageFunc` in `cmd/gaze/main.go` (line 506) is an unexported function in `package main` that:
1. Calls `packages.Load` to resolve package paths
2. Runs `analysis.LoadAndAnalyze` (Spec 001)
3. Runs `classify` (Spec 002)
4. Loads the test package via `loadTestPackage`
5. Runs `quality.Assess` (Spec 003)
6. Aggregates contract coverage into a closure

**Why it's hard to test:**  
The testdata fixtures under `internal/quality/testdata/src/` are real Go packages, but they do not have their own `go.mod`. `packages.Load` requires loadable packages — it resolves import paths relative to the module root. The fixtures are imported as sub-packages of `github.com/unbound-force/gaze`, which means they are only loadable from within the gaze module itself.

`buildContractCoverageFunc` could be tested with a call like:
```go
ccFunc := buildContractCoverageFunc(
    []string{"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"},
    moduleDir,
    &bytes.Buffer{},
)
```
where `moduleDir` is the gaze module root. This would run the full pipeline on the welltested fixture, which has known contract coverage values.

**Risk:** This makes the test an integration test that depends on the full module being present and loadable. It cannot run in isolation, and running it in CI with `-short` would need a `testing.Short()` guard. It also means the test is testing the whole pipeline, not just `buildContractCoverageFunc`'s wiring logic.

**Simpler alternative:** Test only the nil-return paths (empty package list, packages.Load failure, no coverage data collected) via a writable temp dir with an empty `go.mod`, and rely on the existing `runCrap` integration tests (which call `buildContractCoverageFunc` indirectly) for the happy path.

---

## 6. GoDoc and Ambiguity Reduction

The 68% ambiguous classification rate is real and addressable. The classifier's base-50 scoring model adds weight for:
- GoDoc comments (`weight: 15`) — currently missing from most functions in `internal/crap`, several in `internal/analysis`, and most return-value-producing functions in `internal/quality`
- Caller assertion signals (from contract coverage — these would improve after fix #2)

The GoDoc changes are mechanical: add doc comments to exported functions and types that currently lack them, following the existing pattern in the codebase. The most impactful targets (highest number of ambiguous effects per the classification output) are:
- `internal/crap`: `Analyze`, `ParseCoverProfile`, `WriteText`
- `internal/analysis`: `Analyze`, `LoadAndAnalyze`, `AnalyzeP1Effects`, `AnalyzeP2Effects`
- `internal/quality`: `ComputeContractCoverage`, `InferTargets`, `BuildTestSSA`, `MapAssertionsToEffects`

---

## Summary of What Can and Cannot Be Fixed in This Session

| Issue | Fixable? | Notes |
|-------|----------|-------|
| Helper param tracing (4 unmapped) | **Yes** | Clear fix in `assertion.go` + `mapping.go` |
| Inline-call assertion (3 unmapped) | **Yes** | Requires `traceReturnValues` extension |
| "Testify field-access" gap | **N/A** | Already working; stale TODO comment |
| Undertested fixture "unmapped" (4) | **No** | These are correct behaviour, not bugs |
| Tests for `ParseCoverProfile` | **Yes** | Temp module with synthetic profile |
| Tests for `crap.Analyze` | **Yes** | Temp module + pre-supplied `CoverProfile` |
| Tests for `buildContractCoverageFunc` | **Partial** | Nil/error paths via temp dir; happy path relies on existing integration tests |
| GoDoc additions | **Yes** | Mechanical, high ambiguity-reduction value |
| Toolchain PATH issue | **No** | Environment config outside repo scope |
