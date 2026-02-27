# Tasks: Test Quality Metrics & Reporting

**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) |
**Clarifications**: [clarify.md](clarify.md)
**Created**: 2026-02-22

---

## Phase 1: Types and Infrastructure

**Goal**: Define domain types and set up the `internal/quality/`
package with entry points and options.

- [x] T001 [US1] Create `internal/quality/` package with package doc
  in `quality.go`
- [x] T002 [US1] Define `AssertionMapping` struct in
  `internal/taxonomy/types.go` — fields: AssertionLocation (string,
  file:line), AssertionType (string enum: equality, error_check,
  nil_check, diff_check, custom), SideEffectID (string, stable ID
  from Spec 001), Confidence (int, 0-100)
- [x] T003 [US1] Define `ContractCoverage` struct in
  `internal/taxonomy/types.go` — fields: Percentage (float64),
  CoveredCount (int), TotalContractual (int), Gaps ([]SideEffect)
- [x] T004 [US2] Define `OverSpecificationScore` struct in
  `internal/taxonomy/types.go` — fields: Count (int), Ratio
  (float64), IncidentalAssertions ([]AssertionMapping), Suggestions
  ([]string)
- [x] T005 [US1] Define `QualityReport` struct in
  `internal/taxonomy/types.go` — fields: TestFunction (string),
  TestLocation (string), TargetFunction (FunctionTarget),
  ContractCoverage (ContractCoverage), OverSpecification
  (OverSpecificationScore), AmbiguousEffects ([]SideEffect),
  UnmappedAssertions ([]AssertionMapping),
  AssertionDetectionConfidence (int), Metadata (Metadata)
- [x] T006 [US1] Define `PackageSummary` struct in
  `internal/taxonomy/types.go` — fields: TotalTests (int),
  AverageContractCoverage (float64), TotalOverSpecifications (int),
  WorstCoverageTests ([]QualityReport, top 5 by lowest coverage),
  AssertionDetectionConfidence (int)
- [x] T007 [US1] Define `Options` struct and `DefaultOptions()` in
  `internal/quality/quality.go` — fields: Verbose (bool),
  TargetFunc (string, optional override), MaxHelperDepth (int,
  default 3), Stderr (io.Writer)
- [x] T008 [US1] Define `Assess()` function signature in
  `internal/quality/quality.go` — accepts classified
  `[]taxonomy.AnalysisResult`, test `*packages.Package`, and
  `Options`; returns `([]QualityReport, *PackageSummary, error)`
- [x] T009 [US1] Add JSON tags to all new structs in
  `internal/taxonomy/types.go`

---

## Phase 2: Test-Target Pairing

**Goal**: Implement call graph inference to determine which function
each test is exercising.

- [x] T010 [US1] Implement `FindTestFunctions()` in
  `internal/quality/pairing.go` — scan `*_test.go` files in the
  loaded package for functions matching `Test*(*testing.T)`;
  return list of `(funcName, *ast.FuncDecl)` pairs
- [x] T011 [US1] Implement `BuildTestSSA()` in
  `internal/quality/pairing.go` — build SSA for the test package
  using `ssautil.AllPackages()`; return `*ssa.Package` for the
  test package
- [x] T012 [US1] Implement `InferTarget()` in
  `internal/quality/pairing.go` — given a test `*ssa.Function`,
  walk its call graph (bounded to 3 levels), identify calls to
  non-test, non-stdlib functions in the target package; return
  the primary target `*ssa.Function` or an error if ambiguous
- [x] T013 [US1] Handle multi-target case in `InferTarget()` —
  when multiple target functions are called, return all targets
  with a warning; the caller computes metrics per target separately
- [x] T014 [US1] Handle no-target case in `InferTarget()` — when
  no target function is identified, return a descriptive warning
  and skip the test function
- [x] T015 [US1] Support `--target` flag override — when the user
  specifies a target function name, skip call graph inference and
  filter to tests that call the named function
- [x] T016 [P] [US1] Write pairing tests in
  `internal/quality/quality_test.go` — test cases: single target,
  multi-target warning, no target skip, `--target` override,
  external test package (`foo_test`)

---

## Phase 3: Assertion Detection

**Goal**: Implement AST-based assertion site detection for stdlib,
testify, and go-cmp patterns.

- [x] T017 [US1] Define `AssertionSite` struct in
  `internal/quality/assertion.go` — fields: Location (string,
  file:line), Kind (string: stdlib_comparison, stdlib_error_check,
  testify_equal, testify_noerror, gocmp_diff, unknown),
  FuncDecl (*ast.FuncDecl, containing test/helper), Depth (int,
  0=test body, 1-3=helper depth), Expr (ast.Expr, the comparison
  or call expression)
- [x] T018 [US1] Implement `DetectAssertions()` in
  `internal/quality/assertion.go` — walk the test function AST
  looking for assertion patterns; return `[]AssertionSite`
- [x] T019 [US1] Implement stdlib assertion pattern matching —
  detect `if <expr> != <expr> { t.Errorf/t.Fatalf/t.Error/... }`,
  `if err != nil { t.Fatal(err) }`, and similar patterns
- [x] T020 [US1] Implement testify assertion pattern matching —
  detect `assert.Equal(t, got, want)`, `require.Equal(t, ...)`,
  `assert.NoError(t, err)`, `require.NoError(t, err)`,
  `assert.Nil(t, ...)`, and all variants that take a `got` argument
- [x] T021 [US1] Implement go-cmp assertion pattern matching —
  detect `if diff := cmp.Diff(want, got); diff != "" { ... }`
  pattern
- [x] T022 [US1] Implement helper traversal in
  `DetectAssertions()` — when a call to a function accepting
  `*testing.T` is encountered, recurse into that function's AST
  (up to MaxHelperDepth=3); set Depth field on detected assertions
- [x] T023 [US1] Implement `t.Run` sub-test detection — detect
  `t.Run("name", func(t *testing.T) { ... })` closures and
  include their assertions at depth 0 (inlined)
- [x] T024 [US1] Compute AssertionDetectionConfidence — ratio of
  recognized assertion sites to total potential assertion sites
  (heuristic: count calls to `t.Error*`/`t.Fatal*`/`assert.*`/
  `require.*` that were successfully pattern-matched vs total)
- [x] T025 [P] [US1] Write assertion detection tests in
  `internal/quality/quality_test.go` — test each pattern: stdlib
  comparison, stdlib error check, testify equal, testify noerror,
  go-cmp diff, helper traversal at depth 1/2/3, t.Run inlining,
  unrecognized patterns

---

## Phase 4: SSA Data Flow Mapping

**Goal**: Implement the core assertion-to-side-effect mapping engine
using SSA data flow analysis.

- [x] T026 [US1] Implement `FindTargetCall()` in
  `internal/quality/mapping.go` — given the test `*ssa.Function`
  and the target `*ssa.Function`, find the `*ssa.Call` instruction
  in the test SSA that calls the target; handle both direct calls
  and method calls
- [x] T027 [US1] Implement `TraceReturnValues()` in
  `internal/quality/mapping.go` — given a `*ssa.Call` to the
  target, extract each return value (via `ssa.Extract` for
  multi-return), trace it through SSA value edges (phi, store,
  load, field addr) to all reachable instructions; return a map
  from side effect ID to the set of SSA values derived from it
- [x] T028 [US1] Implement `TraceMutations()` in
  `internal/quality/mapping.go` — for mutation side effects
  (ReceiverMutation, PointerArgMutation, etc.), identify the
  value passed as receiver/pointer arg at the target call site,
  then trace reads of that value after the call to assertion sites
- [x] T029 [US1] Implement `MapAssertionsToEffects()` in
  `internal/quality/mapping.go` — given `[]AssertionSite` and the
  traced value sets, check if any assertion expression operand
  matches a traced value; produce `[]AssertionMapping` with
  confidence scores
- [x] T030 [US1] Handle `ssa.Phi` nodes — when tracing values
  through phi nodes (from if/else, loops), follow all incoming
  edges; confidence is reduced if the value merges with non-target
  sources
- [x] T031 [US1] Handle `ssa.FieldAddr` / `ssa.IndexAddr` — trace
  struct field access and array/slice indexing through the SSA
  graph; map to the appropriate sub-effect if identifiable
- [x] T032 [US1] Handle discarded returns — detect `_ = target()`
  patterns (SSA will not produce an Extract for the blank
  identifier); mark corresponding side effects as definitively
  unasserted
- [x] T033 [US1] Handle helper calls in SSA — when an assertion
  site is in a helper function, trace the value from the test
  function through the helper's parameter (the `got` argument
  passed to the helper); bounded to 3 levels
- [x] T034 [P] [US1] Write SSA mapping tests in
  `internal/quality/quality_test.go` — test cases: single return
  mapped, multi-return mapped, error return mapped, mutation
  mapped, discarded return, value through phi node, struct field
  access, helper parameter pass-through

---

## Phase 5: Metrics Computation

**Goal**: Compute Contract Coverage and Over-Specification Score
from assertion mappings.

- [x] T035 [US1] Implement `ComputeContractCoverage()` in
  `internal/quality/coverage.go` — given classified side effects
  and assertion mappings, count contractual effects that have at
  least one mapping; compute percentage; collect gaps (contractual
  effects with no mapping)
- [x] T036 [US1] Implement ambiguous exclusion — filter side effects
  with `Classification.Label == "ambiguous"` from both numerator
  and denominator; report them in a separate `AmbiguousEffects`
  list
- [x] T037 [US2] Implement `ComputeOverSpecification()` in
  `internal/quality/overspec.go` — count assertion mappings where
  the mapped side effect has `Classification.Label == "incidental"`;
  compute ratio (`incidental / total`); generate per-assertion
  suggestions (e.g., "Consider removing — logging is an
  implementation detail")
- [x] T038 [US2] Implement suggestion generation — map side effect
  types to actionable suggestions: LogWrite → "logging is an
  implementation detail", StdoutWrite → "stdout output may change",
  GoroutineSpawn → "goroutine lifecycle is internal", etc.
- [x] T039 [US1] Implement `BuildQualityReport()` in
  `internal/quality/quality.go` — assemble ContractCoverage,
  OverSpecification, AmbiguousEffects, UnmappedAssertions, and
  AssertionDetectionConfidence into a `QualityReport` for each
  test-target pair
- [x] T040 [US1] Implement `BuildPackageSummary()` in
  `internal/quality/quality.go` — aggregate QualityReports:
  compute average contract coverage, total over-specifications,
  worst coverage tests (bottom 5), aggregate detection confidence
- [x] T041 [P] [US1] Write coverage computation tests in
  `internal/quality/quality_test.go` — test cases: 100% coverage,
  0% coverage, partial coverage (75%), ambiguous excluded from
  denominator, no contractual effects (edge case)
- [x] T042 [P] [US2] Write over-specification tests in
  `internal/quality/quality_test.go` — test cases: 0 incidental,
  2 incidental with suggestions, ratio computation, all incidental

---

## Phase 6: Output and CLI

**Goal**: Implement report formatters and the `gaze quality`
CLI subcommand.

- [x] T043 [US1] Implement `WriteJSON()` in
  `internal/quality/report.go` — serialize []QualityReport +
  PackageSummary as formatted JSON; include all fields with
  JSON tags
- [x] T044 [US1] Implement `WriteText()` in
  `internal/quality/report.go` — human-readable output with
  lipgloss styling; per-test summary line format:
  "Contract Coverage: 75% (3/4) | Over-Specified: 2 |
  Detection: 95%"; gap listing; suggestion listing
- [x] T045 [US3] Add stable metadata to output — include Gaze
  version, Go version, timestamp, and commit hash (if in git repo)
  for progress tracking across runs (FR-005, FR-009)
- [x] T046 [US1] Extend JSON Schema in `internal/report/schema.go`
  — add `quality_reports` and `quality_summary` sections; maintain
  backward compatibility (additive, FR-009)
- [x] T047 [US1] Implement `newQualityCmd()` in
  `cmd/gaze/main.go` — Cobra subcommand `quality` accepting a
  package pattern; flags: `--format` (json/text, default text),
  `--target` (optional function name), `--verbose`
- [x] T048 [US4] Add `--min-contract-coverage` flag — int, 0-100;
  when set and coverage is below threshold, return error (exit 1)
- [x] T049 [US4] Add `--max-over-specification` flag — int;
  when set and over-specification count exceeds threshold, return
  error (exit 1)
- [x] T050 [US4] Implement `runQuality()` in `cmd/gaze/main.go` —
  orchestrate: load package → analyze (Spec 001) → classify
  (Spec 002) → assess (Spec 003) → write report → check
  thresholds; use testable crapParams pattern with io.Writer for
  stdout/stderr
- [x] T051 [US4] Implement threshold checking — report CI summary
  to stderr (matching `gaze crap` pattern); return error if any
  threshold is violated
- [x] T052 [P] [US1] Write CLI integration tests in
  `cmd/gaze/main_test.go` — test `runQuality()` with testdata
  fixtures: valid JSON output, text output, threshold pass/fail,
  `--target` flag, invalid format

---

## Phase 7: Test Fixtures and Validation

**Goal**: Build comprehensive test fixtures and write acceptance
tests for all success criteria.

- [x] T053 [P] [US1] Create `internal/quality/testdata/src/welltested/`
  fixture — a Go package with functions that have known contractual
  side effects and tests that assert on all of them (100% contract
  coverage)
- [x] T054 [P] [US1] Create `internal/quality/testdata/src/undertested/`
  fixture — functions with known gaps: some contractual effects
  have no test assertions
- [x] T055 [P] [US2] Create `internal/quality/testdata/src/overspecd/`
  fixture — tests that assert on incidental side effects (log
  output, goroutine spawn)
- [x] T056 [P] [US1] Create `internal/quality/testdata/src/tabledriven/`
  fixture — table-driven tests with `t.Run` sub-tests; known
  assertion union
- [x] T057 [P] [US1] Create `internal/quality/testdata/src/helpers/`
  fixture — tests using helper functions at depth 1, 2, 3, and 4
  (4 should be unmapped)
- [x] T058 [P] [US1] Create `internal/quality/testdata/src/multilib/`
  fixture — tests using testify/assert, testify/require, and
  go-cmp assertion patterns
- [x] T059 [US1] Write `TestSC001_ContractCoverageAccuracy` — verify
  correct coverage computation for 20+ test-target pairs from
  fixtures
- [x] T060 [US2] Write `TestSC002_OverSpecificationDetection` —
  verify all incidental assertions are identified in the benchmark
  suite
- [x] T061 [US1] Write `TestSC003_MappingAccuracy` — verify >= 90%
  assertion-to-side-effect mapping accuracy for standard patterns
- [x] T062 [US3] Write `TestSC004_Determinism` — run quality
  analysis twice on identical code, verify identical metrics
  (excluding timestamps)
- [x] T063 [US4] Write `TestSC005_CIThresholds` — 10+ test
  scenarios for threshold enforcement (pass/fail combinations)
- [x] T064 [US1] Write `TestSC006_PackageSummary` — verify correct
  aggregation across multiple test functions
- [x] T065 [US1] Write `TestSC007_TableDrivenUnion` — verify
  assertion union across t.Run sub-tests
- [x] T066 [US1] Write benchmarks in
  `internal/quality/bench_test.go` — benchmark Assess() for
  single pair and package-level analysis

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Types)**: No dependencies
- **Phase 2 (Pairing)**: Depends on Phase 1 (Options, types)
- **Phase 3 (Detection)**: Depends on Phase 1 (AssertionSite type)
- **Phase 4 (Mapping)**: Depends on Phases 2 + 3 (pairing + detection)
- **Phase 5 (Metrics)**: Depends on Phase 4 (assertion mappings)
- **Phase 6 (Output/CLI)**: Depends on Phases 5 (metrics) + 1 (types)
- **Phase 7 (Validation)**: Depends on all prior phases

### Parallel Opportunities

- Phases 2 and 3 can run in parallel (pairing and detection are
  independent)
- T053-T058 (fixtures) can be created in parallel and early
- T043-T046 (output) can run in parallel with Phases 4-5
- All Phase 7 tests can run in parallel after implementation

### Cross-Spec Dependencies

- **Spec 001** (analysis): Must be complete (it is — 42/42 tasks)
- **Spec 002** (classify): Must be complete (it is — 56 tasks + T051a/T051b)
- **Spec 004** (CRAP): T050 depends on this spec's completion to
  activate GazeCRAP computation

---

## Notes

- [P] = parallelizable with other [P] tasks in the same phase
- [US*] = maps to a user story from the spec
- Total: 66 tasks
- Estimated effort: comparable to Specs 001 + 002 combined
- The SSA data flow mapping (Phase 4) is the highest-risk phase;
  consider shipping Phases 1-3 + 5-6 with a simpler heuristic
  mapper first, then upgrading to full SSA in a follow-up if
  accuracy targets are not met
