# Tasks: Side Effect Detection Engine

**Input**: Design documents from `specs/001-side-effect-detection/`
**Prerequisites**: plan.md (required), spec.md (required)

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Project initialization and Go module structure

- [x] T001 Initialize Go module `github.com/unbound-force/gaze` with
  `go.mod` (Go 1.24), add `golang.org/x/tools` and
  `github.com/spf13/cobra` dependencies
- [x] T002 [P] Create directory structure:
  `cmd/gaze/`, `internal/analysis/`, `internal/taxonomy/`,
  `internal/loader/`, `internal/report/`,
  `internal/analysis/testdata/src/`
- [x] T003 [P] Create `cmd/gaze/main.go` with cobra root command
  and `analyze` subcommand skeleton

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and package loading that all analyzers
depend on

- [x] T004 [US1] Define `SideEffectType` enum in
  `internal/taxonomy/types.go` — all P0 types: `ReturnValue`,
  `ErrorReturn`, `SentinelError`, `ReceiverMutation`,
  `PointerArgMutation`
- [x] T005 [P] [US1] Define `SideEffect` struct in
  `internal/taxonomy/types.go` — fields: ID, Type, Tier,
  Location, Description, Target
- [x] T006 [P] [US1] Define `FunctionTarget` struct in
  `internal/taxonomy/types.go` — fields: Package, Function,
  Receiver, Signature, Location
- [x] T007 [P] [US1] Define `AnalysisResult` struct in
  `internal/taxonomy/types.go` — fields: Target, SideEffects,
  Metadata (GazeVersion, GoVersion, Duration, Warnings)
- [x] T008 [US1] Define priority tier mapping in
  `internal/taxonomy/priority.go` — function that returns
  tier (P0-P4) for each SideEffectType
- [x] T009 [US1] Implement stable ID generation in
  `internal/taxonomy/types.go` —
  `sha256(pkg+func+type+location)` truncated to 8 hex,
  prefixed `se-`
- [x] T010 [US1] Write unit tests for taxonomy types and ID
  generation in `internal/taxonomy/types_test.go`
- [x] T011 [US1] Implement package loader in
  `internal/loader/loader.go` — wraps `go/packages.Load()`
  with correct LoadMode flags, error handling for build
  failures, function lookup by name
- [x] T012 [US1] Write tests for loader in
  `internal/loader/loader_test.go` — test loading a valid
  package, handling build errors, function filtering

**Checkpoint**: Core types defined, packages can be loaded.
Analyzer implementation can begin.

---

## Phase 3: User Story 1 — Analyze a Single Function (P1) MVP

**Goal**: Point Gaze at one Go function, get back all P0 side
effects.

**Independent Test**: Analyze known Go functions with predetermined
side effects, verify 100% detection with zero false positives.

### Tests for User Story 1

> **Write tests FIRST, ensure they FAIL before implementation**

- [x] T013 [P] [US1] Create test fixtures for return analysis in
  `internal/analysis/testdata/src/returns/` — functions with:
  single return, multiple returns, `(T, error)`, named returns,
  named returns modified in defer, pure functions (no returns),
  void functions
- [x] T014 [P] [US1] Create test fixtures for sentinel analysis in
  `internal/analysis/testdata/src/sentinel/` — package-level
  `Err*` vars with `errors.New()`, `fmt.Errorf("...%w...")`,
  functions that return/wrap these sentinels, functions with no
  sentinel usage
- [x] T015 [P] [US1] Create test fixtures for mutation analysis in
  `internal/analysis/testdata/src/mutation/` — pointer receiver
  methods that mutate fields, value receiver methods (should NOT
  detect mutation), pointer params that are written through,
  non-pointer params, deep field mutations
  (`s.config.nested.timeout`)
- [x] T016 [P] [US1] Write return analysis tests in
  `internal/analysis/analysis_test.go` using `go/packages` +
  direct function calls
- [x] T017 [P] [US1] Write sentinel analysis tests in
  `internal/analysis/analysis_test.go` using `go/packages` +
  direct function calls
- [x] T018 [P] [US1] Write mutation analysis tests in
  `internal/analysis/analysis_test.go` using `go/packages` +
  direct function calls

### Implementation for User Story 1

- [x] T019 [US1] Implement ReturnAnalyzer in
  `internal/analysis/returns.go` — walk `*ast.FuncDecl`,
  inspect `.Type.Results`, classify each return position as
  `ReturnValue` or `ErrorReturn`, detect named returns,
  detect deferred named return modification via AST walk of
  `*ast.DeferStmt` bodies
- [x] T020 [US1] Implement SentinelAnalyzer in
  `internal/analysis/sentinel.go` — scan package-level `var`
  decls for `Err*` with `errors.New`/`fmt.Errorf` + `%w`,
  walk function bodies to find returns/wraps of sentinels
- [x] T021 [US1] Implement MutationAnalyzer in
  `internal/analysis/mutation.go` — build SSA via
  `buildssa.Analyzer`, walk `*ssa.Store` instructions,
  classify stores through receiver `FieldAddr` as
  `ReceiverMutation`, stores through pointer params as
  `PointerArgMutation`, resolve field names via
  `types.Struct.Field(idx)`
- [x] T022 [US1] Implement GazeAnalyzer aggregator in
  `internal/analysis/analyzer.go` — requires
  ReturnAnalyzer + SentinelAnalyzer + MutationAnalyzer,
  merges `[]SideEffect` results per function, assigns stable
  IDs, returns `[]AnalysisResult`
- [x] T023 [US1] Verify all US1 tests pass — run full test suite,
  confirm 100% P0 detection on fixtures with zero false positives

**Checkpoint**: Single-function analysis works end-to-end.
`GazeAnalyzer` produces correct `AnalysisResult` for any Go
function.

---

## Phase 4: User Story 2 — Analyze an Entire Package (P2)

**Goal**: Analyze all exported functions in a package in one
invocation.

**Independent Test**: Analyze a package with multiple exported
functions, verify each is reported correctly.

### Implementation for User Story 2

- [x] T024 [US2] Implement package-level scanning in
  `internal/analysis/analyzer.go` — iterate all functions in
  the loaded package, filter exported by default, support
  `--include-unexported` flag via options struct
- [x] T025 [US2] Write tests for package-level scanning —
  test with multi-function package fixture, verify exported
  filtering and `--include-unexported` behavior
- [x] T026 [US2] Handle methods on types — ensure methods with
  pointer receivers on exported types are included in
  package-level results

**Checkpoint**: Package-level analysis works. All exported
functions analyzed in a single invocation.

---

## Phase 5: User Story 3 — Structured Output Formats (P3)

**Goal**: JSON and human-readable text output.

**Independent Test**: Same analysis, two formats, both correct.

### Implementation for User Story 3

- [x] T027 [P] [US3] Implement JSON formatter in
  `internal/report/json.go` — serialize `AnalysisResult` to
  JSON matching the schema in plan.md, include metadata
- [x] T028 [P] [US3] Define JSON Schema in
  `internal/report/schema.go` — embed as Go constant,
  expose via `gaze schema` subcommand for tooling
- [x] T029 [P] [US3] Implement text formatter in
  `internal/report/text.go` — tabular output, 80-column
  friendly, summary line with effect counts per tier
- [x] T030 [P] [US3] Write tests for JSON formatter in
  `internal/report/json_test.go` — validate output is valid
  JSON, parseable, matches schema, contains all fields
- [x] T031 [P] [US3] Write tests for text formatter in
  `internal/report/text_test.go` — validate output readability,
  column alignment, no truncation for typical results

**Checkpoint**: Both output formats work correctly.

---

## Phase 6: CLI Integration

**Purpose**: Wire analyzers + formatters into the cobra CLI.

- [x] T032 [US1] Implement `analyze` command in
  `cmd/gaze/main.go` — accept package path positional arg,
  `--function` flag, `--format` flag (`json`|`text`, default
  `text`), `--include-unexported` flag
- [x] T033 [US1] Wire loader → analyzer → formatter pipeline —
  load package, run GazeAnalyzer, format results, write to
  stdout
- [x] T034 [US1] Handle errors — build failures, function not
  found, invalid flags — with clear error messages to stderr
  and non-zero exit codes
- [x] T035 [US1] Add `--version` flag — print Gaze version

**Checkpoint**: `gaze analyze ./pkg --function Foo` works
end-to-end from the command line.

---

## Phase 7: Polish & Validation

**Purpose**: Final quality checks and benchmark validation.

- [x] T036 [P] Run full test suite, fix any failures
- [x] T037 [P] Create benchmark test fixture with 50+ functions
  covering all P0 side effect types — validate SC-001 (100%
  detection, zero false positives)
- [x] T038 [P] Performance benchmark — validate SC-004
  (< 500ms single function) and SC-005 (< 5s for 50 functions)
- [x] T039 [P] Validate JSON output against schema — SC-006
- [x] T040 [P] Validate text output fits 80 columns — SC-007
- [x] T041 Run `go vet`, `golangci-lint` (if available), fix
  any issues
- [x] T042 Verify edge cases: non-existent function, build
  errors, CGo (UnsafeMutation flag), generics

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 (go.mod must exist)
- **Phase 3 (US1 - MVP)**: Depends on Phase 2 (types + loader)
  - Tests (T013-T018) can start as soon as Phase 2 types exist
  - Implementation (T019-T022) depends on tests being written
- **Phase 4 (US2)**: Depends on Phase 3 (analyzer must work)
- **Phase 5 (US3)**: Depends on Phase 2 (types only), can
  parallelize with Phase 3/4
- **Phase 6 (CLI)**: Depends on Phases 3 + 5
- **Phase 7 (Polish)**: Depends on all prior phases

### Parallel Opportunities

- T002, T003 can run in parallel (different directories)
- T004-T009 models can mostly run in parallel (same file but
  independent types)
- T013, T014, T015 test fixtures can run in parallel
- T016, T017, T018 test files can run in parallel
- T027, T028, T029, T030, T031 formatters can run in parallel
- T036-T040 validation tasks can run in parallel

### Within Phase 3

- Tests MUST be written and FAIL before implementation
- T013-T015 (fixtures) before T016-T018 (tests)
- T016-T018 (tests) before T019-T021 (implementation)
- T019-T021 (sub-analyzers) before T022 (aggregator)
- T022 (aggregator) before T023 (verification)

---

## Implementation Strategy

### MVP First (Phase 1-3 + 6)

1. Setup + Foundational → types and loader ready
2. US1 tests → fixtures and test cases written, all fail
3. US1 implementation → analyzers built, tests pass
4. CLI integration → `gaze analyze` works from terminal
5. **STOP and VALIDATE**: Test against real Go code

### Incremental Delivery

1. Phases 1-3 + 6 → MVP: single function analysis via CLI
2. Phase 4 → Package scanning
3. Phase 5 → JSON + text output
4. Phase 7 → Benchmark validation, polish

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Tests MUST fail before implementation begins
- Commit after each task or logical group
- Stop at any checkpoint to validate independently
- v1 scope: P0 effects only, direct function body only
