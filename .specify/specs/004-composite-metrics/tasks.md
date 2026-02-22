# Tasks: Composite Quality Metrics (CRAP / GazeCRAP)

**Input**: Design documents from `.specify/specs/004-composite-metrics/`  
**Prerequisites**: plan.md (required), spec.md (required)

**Note**: This task list is **retroactive**. Implementation of Phases 1–5
occurred prior to the creation of `plan.md` and `tasks.md` — directly on
`main` via commits `4b37981`, `5ea4085`, `58b2da5`, and `2d67f76`. Checkbox
states reflect actual implementation status at the time this document was
written. Phases 6–7 document work that remains to be done.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: New dependency and directory structure

- [x] T001 Add `github.com/fzipp/gocyclo` dependency via
  `go get github.com/fzipp/gocyclo`
- [x] T002 [P] Create directory `internal/crap/`

---

## Phase 2: Foundational Types

**Purpose**: Core types and pure functions that all analysis depends on

- [x] T003 [US1] Define `Score` struct in `internal/crap/crap.go` —
  fields: Package, Function, File (string), Line (int), Complexity (int),
  LineCoverage (float64), CRAP (float64), plus nullable stub fields
  ContractCoverage (`*float64, omitempty`), GazeCRAP (`*float64, omitempty`),
  Quadrant (`*Quadrant, omitempty`) for future Spec 003 activation
- [x] T004 [P] [US1] Define `Quadrant` type and four constants in
  `internal/crap/crap.go` — `Q1Safe`, `Q2ComplexButTested`,
  `Q3SimpleButUnderspecified`, `Q4Dangerous`
- [x] T005 [P] [US1] Define `Summary` struct in `internal/crap/crap.go` —
  fields: TotalFunctions, AvgComplexity, AvgLineCoverage, AvgCRAP, CRAPload,
  CRAPThreshold, plus nullable GazeCRAPload (`*int`), GazeCRAPThreshold
  (`*float64`), QuadrantCounts (`map[Quadrant]int`), WorstCRAP ([]Score)
- [x] T006 [P] [US1] Define `Report` struct in `internal/crap/crap.go` —
  fields: Scores ([]Score), Summary (Summary)
- [x] T007 [US1] Implement `Formula()` in `internal/crap/crap.go` —
  `CRAP(m) = comp^2 * (1 - cov/100)^3 + comp` using `math.Pow`
- [x] T008 [US3] Implement `ClassifyQuadrant()` in
  `internal/crap/crap.go` — switch on `crap >= crapThreshold` and
  `gazeCRAP >= gazeCRAPThreshold`, returns one of the four Quadrant
  constants; both thresholds are independent

**Checkpoint**: Core types defined and formula implemented. All other
phases can proceed.

---

## Phase 3: US1 — Coverage Profile Parsing

**Goal**: Parse `go test -coverprofile` output into per-function coverage
percentages

- [x] T009 [US1] Define `FuncCoverage` struct in
  `internal/crap/coverage.go` — fields: File, FuncName, StartLine, EndLine,
  CoveredStmts, TotalStmts, Percentage; all with JSON tags
- [x] T010 [US1] Implement `ParseCoverProfile()` in
  `internal/crap/coverage.go` — calls `cover.ParseProfiles()`, resolves
  import-path filenames to absolute paths via `resolveFilePath()`, walks
  each file's AST with `findFunctions()`, computes per-function covered/total
  statement counts via `funcCoverage()`
- [x] T011 [P] [US1] Implement `findFunctions()` in
  `internal/crap/coverage.go` — parses Go source with `go/parser`,
  returns `[]funcExtent` with name, startLine, startCol, endLine, endCol
  for each `*ast.FuncDecl` with a non-nil body; formats method names as
  `(*RecvType).MethodName` via `recvTypeString()`
- [x] T012 [P] [US1] Implement `funcCoverage()` in
  `internal/crap/coverage.go` — iterates `cover.Profile.Blocks`, accumulates
  covered and total statement counts for blocks that overlap the function
  extent; uses line+column boundary comparison for accuracy
- [x] T013 [P] [US1] Implement `resolveFilePath()` and `readModulePath()`
  in `internal/crap/coverage.go` — maps coverage-profile import paths
  (e.g., `github.com/jflowers/gaze/internal/crap/crap.go`) to absolute
  filesystem paths by stripping the module prefix read from `go.mod`

**Checkpoint**: Can parse any valid `go test -coverprofile` output and
produce per-function coverage percentages.

---

## Phase 4: US1 — Analysis Orchestration

**Goal**: Combine gocyclo complexity with coverage data and compute CRAP
scores for all functions in a package pattern

- [x] T014 [US1] Define `Options` struct and `DefaultOptions()` in
  `internal/crap/analyze.go` — fields: CoverProfile (string), CRAPThreshold
  (float64, default 15), GazeCRAPThreshold (float64, default 15),
  IgnoreGenerated (bool, default true). Note: `MaxCRAPload` and
  `MaxGazeCRAPload` were removed from `Options` — CI threshold
  enforcement is a CLI concern handled by `checkCIThresholds()` in
  `cmd/gaze/main.go`, not a library concern
- [x] T015 [US1] Implement `generateCoverProfile()` in
  `internal/crap/analyze.go` — runs `go test -coverprofile=<tempfile>`
  for the given patterns in `moduleDir`; writes to `os.CreateTemp` to avoid
  clobbering existing `cover.out`; defers `os.Remove` of the temp file
- [x] T016 [US1] Implement `resolvePatterns()` in
  `internal/crap/analyze.go` — converts Go package patterns (`./...`,
  `./sub`, bare paths) to absolute filesystem paths for gocyclo
- [x] T017 [US1] Implement `buildCoverMap()` and `lookupCoverage()` in
  `internal/crap/analyze.go` — constructs two lookup maps (exact
  absolute-path key, basename fallback key) from `[]FuncCoverage`; O(1)
  lookup per gocyclo `Stat`
- [x] T018 [US1] Implement `isGeneratedFile()` in
  `internal/crap/analyze.go` — reads file line by line, stops at
  `package` clause, matches `// Code generated .* DO NOT EDIT.` regexp;
  result cached in `map[string]bool` within `Analyze()` to avoid re-reading
- [x] T019 [US1] Implement `buildSummary()` in
  `internal/crap/analyze.go` — computes aggregate statistics, CRAPload
  count, worst-5 by CRAP descending; gates GazeCRAPload/GazeCRAPThreshold/
  QuadrantCounts population on `hasGazeCRAP` (whether any Score has
  non-nil `GazeCRAP` field)
- [x] T020 [US1] Implement `Analyze()` in `internal/crap/analyze.go` —
  6-step pipeline: (1) generate or validate cover profile, (2) run gocyclo,
  (3) parse coverage, (4) build cover map, (5) join and compute per-function
  CRAP scores (skipping `_test.go` and generated files), (6) build summary;
  leaves GazeCRAP/ContractCoverage/Quadrant fields nil per FR-015

**Checkpoint**: `crap.Analyze()` produces a correct `*Report` for any Go
package pattern.

---

## Phase 5: US1 — Output Formatters

**Goal**: Human-readable text (lipgloss) and machine-readable JSON output

- [x] T021 [P] [US1] Implement `WriteJSON()` in
  `internal/crap/report.go` — encodes `*Report` as indented JSON using
  `json.NewEncoder`
- [x] T022 [P] [US1] Implement `WriteText()` in
  `internal/crap/report.go` — lipgloss/table sorted by CRAP descending,
  marks above-threshold rows with `*`; Summary section (functions, avg
  complexity, avg coverage, avg CRAP, CRAPload); GazeCRAP section
  (rendered only when `GazeCRAPload != nil`); Quadrant Breakdown section
  (rendered only when `QuadrantCounts` non-empty); Worst Offenders top-5;
  uses `internal/report.DefaultStyles()` for consistent palette
- [x] T023 [P] [US1] Implement `shortenPath()` in
  `internal/crap/report.go` — strips common module-path prefixes
  (`/internal/`, `/cmd/`, `/pkg/`) for display; falls back to last 3
  path segments

**Checkpoint**: Both formatters produce correct output.

---

## Phase 5b: US1 + US5 — CLI Integration

**Purpose**: Wire `internal/crap` into the cobra CLI

- [x] T024 [US1] Implement `newCrapCmd()` in `cmd/gaze/main.go` —
  registers `gaze crap [packages...]` cobra command with flags:
  `--format`, `--coverprofile`, `--crap-threshold`, `--gaze-crap-threshold`,
  `--max-crapload`, `--max-gaze-crapload`
- [x] T025 [US1] Implement `runCrap(crapParams)` in
  `cmd/gaze/main.go` — testable function following the existing params-struct
  pattern; calls `crap.Analyze()` then `writeCrapReport()`
- [x] T026 [US5] Implement `printCISummary()` in `cmd/gaze/main.go` —
  prints one-line `CRAPload: N/M (PASS|FAIL) | GazeCRAPload: N/M (PASS|FAIL)`
  summary; skips GazeCRAPload when `rpt.Summary.GazeCRAPload == nil`
- [x] T027 [US5] Implement `checkCIThresholds()` in `cmd/gaze/main.go` —
  returns error when `rpt.Summary.CRAPload > maxCrapload` (if limit > 0)
  or `*rpt.Summary.GazeCRAPload > maxGazeCrapload` (if limit > 0 and
  GazeCRAPload non-nil); zero limits mean report-only (always exit 0)
- [x] T028 [US1] Register `newCrapCmd()` in root command's
  `root.AddCommand()` call in `cmd/gaze/main.go`

**Checkpoint**: `gaze crap ./...` works end-to-end from the command line.

---

## Phase 5c: US1 — Unit Tests

**Purpose**: Validate all components against spec success criteria

- [x] T029 [P] [US1] Write `Formula` tests in
  `internal/crap/crap_test.go` — 7 cases: 0% coverage, 100% coverage,
  50% coverage, comp=1+0%, comp=1+100%, high complexity, 75% coverage;
  verify to ±0.01; plus 21-case `TestFormula_BenchmarkSuite` table-driven
  test covering boundary, full-coverage, 25%, 90%, and mixed coverage
  levels for a total of 28 hand-computed pairs (SC-001)
- [x] T030 [P] [US3] Write `ClassifyQuadrant` tests in
  `internal/crap/crap_test.go` — 6 cases: all four quadrants, at-threshold
  boundary (CRAP == threshold is "at or above"), independent thresholds
  (SC-004)
- [x] T031 [P] [US1] Write `buildSummary` tests in
  `internal/crap/crap_test.go` — 3 cases: CRAPload counting, worst-5
  ordering (SC-003), empty input
- [x] T032 [P] [US1] Write `WriteJSON` test in
  `internal/crap/crap_test.go` — validates output is valid JSON
- [x] T033 [P] [US1] Write `WriteText` tests in
  `internal/crap/crap_test.go` — 2 cases: summary sections present, `*`
  marker for above-threshold functions
- [x] T034 [P] [US1] Write `isGeneratedFile` tests in
  `internal/crap/crap_test.go` — 4 cases
- [x] T035 [P] [US1] Write `resolvePatterns` tests in
  `internal/crap/crap_test.go` — 4 cases
- [x] T036 [P] [US1] Write `buildCoverMap` and `lookupCoverage` tests in
  `internal/crap/crap_test.go` — 5 cases total
- [x] T037 [P] [US1] Write `findFunctions`, `recvTypeString`,
  `funcCoverage` tests in `internal/crap/crap_test.go` — 5 cases total
- [x] T038 [P] [US1] Write `resolveFilePath` and `readModulePath` tests
  in `internal/crap/crap_test.go` — 5 cases total (SC-008)
- [x] T039 [P] [US1] Write `shortenPath` tests in
  `internal/crap/crap_test.go` — 5 cases
- [x] T040 [P] [US5] Write `printCISummary` tests in
  `cmd/gaze/main_test.go` — 6 cases (no thresholds, CRAPload PASS/FAIL,
  GazeCRAPload PASS/FAIL, combined, nil GazeCRAPload skipped) (SC-006)
- [x] T041 [P] [US5] Write `checkCIThresholds` tests in
  `cmd/gaze/main_test.go` — 6 cases (all pass, no limits, CRAPload
  exceeded, GazeCRAPload exceeded, nil GazeCRAPload, both exceeded)
- [x] T042 [P] [US1] Write `writeCrapReport` tests in
  `cmd/gaze/main_test.go` — 2 cases (JSON, text)
- [x] T043 [P] [US1] Write `runCrap` format validation test in
  `cmd/gaze/main_test.go` — invalid format `"xml"` rejected
- [x] T044 [P] [US1] Write benchmarks in
  `internal/crap/bench_test.go` — 5 benchmarks: Formula,
  ClassifyQuadrant, buildSummary, buildCoverMap, isGeneratedFile

**Checkpoint**: All tests pass (`go test -race -count=1 ./...`). SC-001,
SC-003, SC-004, SC-006, SC-007, SC-008 verified.

---

## Phase 6: US2 Stub Wire (GazeCRAP fields)

**Purpose**: Add nullable GazeCRAP fields to `Score` and `Summary` so the
activation path is ready when Spec 003 contract coverage becomes available

- [x] T045 [US2] Add `ContractCoverage *float64` field to `Score` in
  `internal/crap/crap.go` with `json:"contract_coverage,omitempty"` tag;
  remains nil until Spec 003 populates it
- [x] T046 [US2] Add `GazeCRAP *float64` field to `Score` in
  `internal/crap/crap.go` with `json:"gaze_crap,omitempty"` tag; remains nil
- [x] T047 [US2] Add `Quadrant *Quadrant` field to `Score` in
  `internal/crap/crap.go` with `json:"quadrant,omitempty"` tag; remains nil
- [x] T048 [US2] Add `GazeCRAPload *int`, `GazeCRAPThreshold *float64`,
  `QuadrantCounts map[Quadrant]int` to `Summary` in `internal/crap/crap.go`;
  all `omitempty` and nil when GazeCRAP is unavailable
- [x] T049 [US2] Add comment in `Analyze()` in `internal/crap/analyze.go`
  documenting that GazeCRAP/ContractCoverage/Quadrant remain nil per FR-015
  until Spec 003 is complete

**Note**: Tasks T045–T049 were implemented as part of the initial CRAP
implementation rather than a separate phase, but are tracked separately here
to reflect their US2 ownership.

---

## Phase 7: Deferred — Not Yet Implemented

**Purpose**: Track work that remains to be done in future iterations

- [ ] T050 [US2] Activate GazeCRAP computation in `Analyze()` in
  `internal/crap/analyze.go` — accept contract coverage input from Spec 003
  `classify.Classify()` results, compute
  `GazeCRAP(m) = comp^2 * (1 - contractCov)^3 + comp` per function,
  populate `Score.GazeCRAP`, `Score.ContractCoverage`, and
  `Score.Quadrant` via `ClassifyQuadrant()`
- [x] T051 [US2] Emit FR-015 stderr warning in `runCrap()` in
  `cmd/gaze/main.go` — when `GazeCRAPload == nil`, write
  `"note: GazeCRAP unavailable — contract coverage not yet implemented (Spec 003)"`
  to stderr so users understand the omission
- [ ] T052 [US2] Write GazeCRAP formula accuracy tests in
  `internal/crap/crap_test.go` — validate SC-002 with hand-computed
  (complexity, contractCoverage) pairs after T050 is implemented
- [ ] T053 [US3] Activate quadrant breakdown in output — verify
  `WriteText()` renders Quadrant Breakdown section correctly once
  `QuadrantCounts` is non-empty (logic exists; no regression test for
  populated case)
- [x] T053a [US2] Populate `AvgGazeCRAP`, `AvgContractCoverage`, and
  `WorstGazeCRAP` in `buildSummary()` — fields declared in `Summary`
  as nullable/omitempty stubs alongside `GazeCRAPload`; population
  logic is implemented and tested (`TestBuildSummary_WithGazeCRAP`);
  activates only when `hasGazeCRAP` is true (requires T050 to produce
  non-nil `GazeCRAP` values)
- [ ] T054 [US4] Implement `gaze self-check` command in
  `cmd/gaze/main.go` — cobra command that runs the full CRAP + GazeCRAP
  pipeline on `github.com/jflowers/gaze/...`, reports: total CRAPload,
  total GazeCRAPload, average contract coverage, worst-5 offenders by
  GazeCRAP (SC-005); requires T050 to be complete
- [ ] T055 [US4] Write `self-check` tests in `cmd/gaze/main_test.go` —
  test that it completes without error, produces valid JSON, covers all
  exported functions in Gaze source packages

---

## Dependencies & Execution Order

### Phase Dependencies (retroactive)

- **Phase 1 (Setup)**: No dependencies
- **Phase 2 (Types)**: Depends on Phase 1
- **Phase 3 (Coverage)**: Depends on Phase 2 (FuncCoverage type)
- **Phase 4 (Analysis)**: Depends on Phases 2 + 3
- **Phase 5 (Output)**: Depends on Phase 2 (Report type); can parallelize
  with Phases 3–4
- **Phase 5b (CLI)**: Depends on Phases 4 + 5
- **Phase 5c (Tests)**: Depends on Phases 2–5b
- **Phase 6 (Stub wire)**: Merged into Phase 2–4 implementation in practice
- **Phase 7 (Deferred)**: T050 depends on Spec 003 completion; T051 is
  COMPLETE; T053a activates when T050 is done; T054–T055 depend on T050

### Parallel Opportunities

- T003–T008 types can largely run in parallel (same file, independent)
- T009–T013 coverage parsing can run in parallel with Phase 4
- T021–T023 formatters can run in parallel with Phases 3–4
- T029–T044 tests can all run in parallel
- T051 (FR-015 warning) is COMPLETE — done independently of T050

---

## Notes

- [P] tasks = different files or independent concerns, no dependencies
- [Story] label maps task to specific user story (US1–US5)
- All Phase 1–6 tasks are complete (`[x]`) — retroactive bookkeeping only
- Phase 7 tasks are the genuine remaining work for this spec
- Mark tasks complete (`- [x]`) immediately upon finishing
- Spec 003 (test quality metrics) must be implemented before T050 can proceed
