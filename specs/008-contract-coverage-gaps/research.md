# Research: Contract Coverage Gap Remediation

**Branch**: `008-contract-coverage-gaps` | **Date**: 2026-02-27

## R1: Test Infrastructure for Direct Signal Function Testing

**Decision**: Use the existing `loadTestPackages` helper from
`classify_test.go` and the `contracts`/`callers` fixtures to
construct inputs for signal function tests.

**Rationale**: The classify testdata already contains fixture packages
with the exact patterns needed — exported functions with godoc
annotations, cross-package callers, exported return types, and
receiver methods. Building minimal `*ast.FuncDecl` nodes
programmatically would be fragile and miss type-resolution details
that `go/packages` provides automatically.

**Alternatives considered**:
- Manual AST construction: Rejected — `types.Info` and `types.Object`
  are tightly coupled to the `go/packages` loader; constructing them
  manually is error-prone and produces unrealistic inputs.
- New dedicated fixtures: Rejected — the existing `contracts`,
  `callers`, and `incidental` fixtures already cover all needed
  patterns (exported/unexported functions, godoc keywords, cross-
  package references).

## R2: Test Infrastructure for Direct Analysis Function Testing

**Decision**: Reuse the existing `cachedTestPackage`, `FindFuncDecl`,
`hasEffect`, `countEffects`, and `effectWithTarget` helpers from
`analysis_test.go`. New test files (`p1effects_test.go`, etc.) use
`package analysis_test` and have access to all these helpers.

**Rationale**: The helpers are already in the `analysis_test` package
and handle fixture loading, SSA building, and package caching. The
test fixtures (`p1effects/`, `p2effects/`, `returns/`) already contain
all the function patterns specified in FR-011, FR-012, and FR-013.

**Alternatives considered**:
- Internal test package (`package analysis`): Rejected — the analysis
  functions are exported and should be tested from the external test
  package to verify the public API contract.
- Separate fixture loading: Rejected — would duplicate the existing
  cached loader and create maintenance burden.

## R3: AnalyzeGodocSignal Input Construction

**Decision**: Use a hybrid approach — load the `contracts` fixture
for functions with existing godoc annotations, and construct minimal
`*ast.FuncDecl` with `*ast.CommentGroup` programmatically for
keyword coverage testing.

**Rationale**: The `contracts` fixture has godoc-annotated functions
(`GetVersion` → "returns", `SetPrimary` → "sets", `LoadProfile` →
"returns"). However, testing all contractual keywords (9 keywords ×
multiple effectTypes) plus incidental keywords (4) plus priority
ordering requires more combinations than the fixture provides.
Constructing a minimal `*ast.FuncDecl` with just a `Doc` comment
group and a `Name` ident is simple and reliable for `AnalyzeGodocSignal`
because it only reads `funcDecl.Doc` and `funcDecl.Name` — it does
not use `types.Info` or the function body.

**Alternatives considered**:
- Fixture-only: Rejected — would require adding 15+ new functions to
  the contracts fixture just for godoc keyword coverage.
- Programmatic-only: Rejected — loses the integration validation of
  testing against real parsed Go code.

## R4: renderAnalyzeContent Testing Approach

**Decision**: Construct `[]taxonomy.AnalysisResult` structs directly
in tests. Assert on string content using `strings.Contains` for
function names, tier labels, and effect types. Do not assert on
exact ANSI escape sequences or Lipgloss styling.

**Rationale**: `renderAnalyzeContent` is a pure function that takes
structured data and returns a styled string. The styling (colors,
table formatting) uses Lipgloss which produces ANSI escape sequences
that are fragile to test against exactly. Content assertions
(`strings.Contains` for key substrings) verify the contract without
coupling to rendering implementation details.

**Alternatives considered**:
- Snapshot testing: Rejected — Go stdlib has no snapshot testing
  support, and ANSI output varies by terminal capabilities.
- Strip ANSI and assert on plain text: Considered viable but adds
  complexity; content assertions are sufficient for this function's
  contract.

## R5: buildContractCoverageFunc Strengthening

**Decision**: Modify the existing
`TestBuildContractCoverageFunc_WelltestedPackage` to assert `fn != nil`,
`ok == true`, and `pct > 0`. Keep the `testing.Short()` guard.

**Rationale**: The current test allows `fn == nil` as a passing
result, which means it never verifies the closure's behavior. The
`welltested` fixture has known coverage data — `Add` should always
have contract coverage > 0% given its test (`TestAdd`) directly
asserts on the return value.

**Alternatives considered**:
- New fixture for deterministic coverage: Rejected — `welltested` is
  already deterministic and well-understood.
- Remove `testing.Short()` guard: Rejected — the test runs the full
  analysis + classify + quality pipeline, which is too slow for the
  standard CI timeout.
