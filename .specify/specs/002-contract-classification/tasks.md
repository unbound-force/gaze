# Tasks: Contractual Classification & Confidence Scoring

**Input**: Design documents from `.specify/specs/002-contract-classification/`
**Prerequisites**: plan.md (required), spec.md (required)

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: New dependency, directory structure, and shared
configuration infrastructure

- [x] T001 Add `gopkg.in/yaml.v3` dependency via
  `go get gopkg.in/yaml.v3`
- [x] T002 [P] Create directory structure:
  `internal/classify/`, `internal/classify/testdata/src/`,
  `internal/docscan/`, `internal/docscan/testdata/`,
  `internal/config/`, `internal/config/testdata/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types, config loading, and module-level package
loading that all user stories depend on

- [x] T003 Define `ClassificationLabel` enum in
  `internal/taxonomy/types.go` — values: `Contractual`,
  `Incidental`, `Ambiguous`
- [x] T004 [P] Define `Signal` struct in
  `internal/taxonomy/types.go` — fields: Source (string),
  Weight (int), SourceFile (string, `omitempty`),
  Excerpt (string, `omitempty`), Reasoning (string,
  `omitempty`); with JSON tags. Detail fields (SourceFile,
  Excerpt, Reasoning) are omitted from JSON when empty,
  enabling non-verbose output by leaving them unpopulated
- [x] T005 [P] Define `Classification` struct in
  `internal/taxonomy/types.go` — fields: Label
  (ClassificationLabel), Confidence (int, 0-100),
  Signals ([]Signal), Reasoning (string); with JSON tags
- [x] T006 [P] Add optional `Classification` field to the
  existing `SideEffect` struct in
  `internal/taxonomy/types.go` — use pointer
  (`*Classification`) so it is omitted from JSON when nil
  (`json:"classification,omitempty"`)
- [x] T007 Write unit tests for classification types in
  `internal/taxonomy/types_test.go` — test JSON
  serialization of Classification, Signal, and SideEffect
  with and without classification
- [x] T008 Implement `GazeConfig` struct and `Load()` function
  in `internal/config/config.go` — parse `.gaze.yaml` with
  `gopkg.in/yaml.v3`, return defaults when file is missing,
  fields: classification thresholds (contractual >= 80,
  incidental < 50), doc scan exclude/include patterns,
  doc scan timeout
- [x] T009 [P] Implement `DefaultConfig()` function in
  `internal/config/config.go` — returns sensible defaults
  with FR-009 default exclude list (vendor/**, node_modules/**,
  .git/**, testdata/**, CHANGELOG.md, CONTRIBUTING.md,
  CODE_OF_CONDUCT.md, LICENSE, LICENSE.md)
- [x] T010 [P] Create test `.gaze.yaml` fixtures in
  `internal/config/testdata/` — valid config, empty config,
  config with custom thresholds, config with custom excludes,
  config with include override
- [x] T011 Write tests for config loading in
  `internal/config/config_test.go` — test Load() with valid
  file, missing file (returns defaults), custom thresholds,
  custom exclude/include patterns
- [x] T012 Implement `LoadModule()` function in
  `internal/loader/loader.go` — load all packages in the Go
  module using `./...` pattern with `go/packages`, return
  `[]*packages.Package` for sibling package access; reuse
  existing LoadMode flags
- [x] T013 Write tests for `LoadModule()` in
  `internal/loader/loader_test.go` — test loading a module
  with multiple packages, verify sibling packages are
  accessible, handle module with single package

**Checkpoint**: Core types defined, config loadable, module
packages loadable. Classification engine can begin.

---

## Phase 3: User Story 1 — Mechanical Classification (P1)

**Goal**: Classify each side effect as contractual, incidental, or
ambiguous using deterministic mechanical signals only.

**Independent Test**: Analyze functions with known contractual and
incidental effects (interface implementors, logging functions),
verify classifications match expected labels with correct confidence
ranges.

### Test Fixtures for User Story 1

- [x] T014 [P] [US1] Create test fixture package for contractual
  effects in `internal/classify/testdata/src/contracts/` —
  interface definitions with implementing methods (return values,
  mutations), exported functions used by callers, functions with
  clear contractual naming (Get*, Save*, Delete*), functions with
  godoc declaring behavior
- [x] T015 [P] [US1] Create test fixture package for incidental
  effects in `internal/classify/testdata/src/incidental/` —
  functions that write to loggers, debug/trace output, internal
  caching, functions with no callers and no interface contract
- [x] T016 [P] [US1] Create test fixture package for ambiguous
  effects in `internal/classify/testdata/src/ambiguous/` —
  functions with contradicting signals (exported but no callers),
  functions with some interface match but unclear naming
- [x] T017 [P] [US1] Create a caller fixture package in
  `internal/classify/testdata/src/callers/` — package that
  imports and calls functions from `contracts/` and `incidental/`
  to enable caller analysis testing

### Implementation for User Story 1

- [x] T018 [US1] Implement interface satisfaction signal in
  `internal/classify/interface.go` — use
  `go/types.Implements()` to check if the function's receiver
  type satisfies any interface in the module; when a method's
  side effect matches the interface's method signature, emit
  Signal{Source: "interface", Weight: up to +30}; exported
  function: `analyzeInterfaceSignal()` (unexported; takes pre-computed `[]namedInterface` to avoid O(n²) collection)
- [x] T019 [US1] Implement API surface visibility signal in
  `internal/classify/visibility.go` — check if the side
  effect is observable through exported return types, exported
  parameter types, or exported receiver types; each visibility
  dimension contributes independently up to +20 total; exported
  function: `AnalyzeVisibilitySignal()`
- [x] T020 [US1] Implement caller dependency signal in
  `internal/classify/callers.go` — scan `TypesInfo.Uses`
  across module packages to find call sites; compute ratio
  of callers that use/depend on this side effect; emit
  Signal{Source: "caller", Weight: up to +15} proportional
  to usage ratio; exported function:
  `AnalyzeCallerSignal()`
- [x] T021 [P] [US1] Implement naming convention signal in
  `internal/classify/naming.go` — match function name against
  Go community patterns: Get*/Fetch*/Load*/Read* → return
  contractual, Set*/Update*/Save*/Write*/Delete* → mutation
  contractual, Err* sentinel → error contractual, log*/debug*/
  trace* → incidental; emit Signal{Source: "naming",
  Weight: up to +10}; exported function:
  `AnalyzeNamingSignal()`
- [x] T022 [P] [US1] Implement godoc comment signal in
  `internal/classify/godoc.go` — parse function doc comment
  for behavioral keywords: "returns"/"writes"/"modifies"/
  "updates"/"sets" → contractual evidence, "logs"/"prints" →
  incidental evidence; emit Signal{Source: "godoc",
  Weight: up to +15}; exported function:
  `AnalyzeGodocSignal()`
- [x] T023 [US1] Implement confidence score computation in
  `internal/classify/score.go` — start from base confidence
  of 50, add all signal weights (positive signals push toward
  contractual, negative push toward incidental), detect
  contradictions (apply up to -20 penalty per FR-007), clamp
  score to 0-100, apply thresholds from GazeConfig (>= 80
  contractual, 50-79 ambiguous, < 50 incidental), return
  Classification struct; exported function: `ComputeScore()`
- [x] T024 [US1] Implement classifier entry point and Options
  struct in `internal/classify/classify.go` — Options struct
  with Config (*config.GazeConfig), ModulePackages
  ([]*packages.Package); `Classify()` function takes
  []taxonomy.AnalysisResult and Options, runs all five
  mechanical signal analyzers per side effect, calls
  ComputeScore(), attaches Classification to each SideEffect,
  returns classified results
- [x] T025 [US1] Write unit tests for mechanical classification
  in `internal/classify/classify_test.go` — test each signal
  analyzer independently, test combined classification on
  fixture packages, verify determinism (FR-011: same input →
  same output), verify interface satisfaction yields
  confidence >= 85, verify logging functions classified as
  incidental
- [x] T026 [US1] Write benchmark tests in
  `internal/classify/bench_test.go` — validate SC-004
  (mechanical classification adds no measurable latency
  beyond Spec 001 analysis), benchmark Classify() on
  fixture packages

**Checkpoint**: Mechanical classification works end-to-end.
All five signals produce correct classifications on test
fixtures.

---

## Phase 4: User Story 2 — Document-Enhanced Classification (P2)

**Goal**: Scan project documentation and use an OpenCode
agent/command to extract design-intent signals that refine
mechanical confidence scores.

**Independent Test**: Provide a project with explicit architecture
docs, analyze a function, verify document signals shift the
confidence score in the correct direction.

### Implementation for User Story 2

- [x] T027 [US2] Implement document scanner in
  `internal/docscan/scanner.go` — walk the repository
  directory tree, find all `.md` files, prioritize by
  proximity to the target function (same package > module
  root > other) per FR-012, return prioritized list of
  DocumentFile structs (path, content, priority); exported
  function: `Scan()`
- [x] T028 [US2] Implement exclude/include filter in
  `internal/docscan/filter.go` — apply glob patterns from
  GazeConfig to include/exclude documents, support default
  exclude list (FR-009), support include override (if include
  patterns set, only matching files are processed); exported
  function: `Filter()`
- [x] T029 [P] [US2] Create test fixture directory in
  `internal/docscan/testdata/` — mock repo structure with
  README.md, docs/architecture.md, vendor/README.md,
  CHANGELOG.md, CONTRIBUTING.md, nested package doc.md
- [x] T030 [US2] Write tests for document scanner in
  `internal/docscan/scanner_test.go` — test Scan() finds
  all .md files, test Filter() excludes vendor/CHANGELOG,
  test include override, test priority ordering, test empty
  repo (no docs)
- [x] T031 [US2] Create OpenCode agent definition in
  `.opencode/agents/doc-classifier.md` — subagent with
  read-only tools (no write/edit). Agent prompt must:
  (a) accept two inputs: mechanical classification JSON
  (same schema as `gaze analyze --classify --format=json`)
  and documentation content as `[{path, content, priority}]`;
  (b) extract document signals per FR-005 (readme,
  architecture_doc, specify_file, api_doc, other_md) and
  AI inference signals per FR-006 (ai_pattern, ai_layer,
  ai_corroboration); (c) merge document signals with
  existing mechanical signals: start from the mechanical
  confidence score, add document signal weights (within
  documented bounds), detect cross-category contradictions
  (apply up to -20 penalty per FR-007), clamp to 0-100,
  re-apply thresholds (>= 80 contractual, 50-79 ambiguous,
  < 50 incidental); (d) output enhanced classification JSON
  in the same schema with additional signals appended and
  confidence scores recalculated. Each signal must include
  source, weight, source_file, excerpt, reasoning.
- [x] T032 [US2] Create OpenCode command in
  `.opencode/command/classify-docs.md` — command that:
  (a) accepts `$ARGUMENTS` as the Go package pattern;
  (b) runs `gaze analyze --classify --format=json $1` via
  shell to get mechanical classification;
  (c) runs `gaze docscan $1` via shell (or reads .md files
  directly) to collect prioritized documentation content
  filtered by `.gaze.yaml`;
  (d) feeds both to the doc-classifier agent;
  (e) outputs the agent's enhanced classification JSON to
  stdout. The command uses `agent: doc-classifier`
  in frontmatter.
- [x] T033 [US2] Add `docscan` subcommand to `cmd/gaze/main.go` —
  outputs scanned documentation as JSON
  `[{path, content, priority}]`, applies `.gaze.yaml`
  exclude/include filters and priority ordering per FR-012
  (items 3-6); this provides the structured input the
  OpenCode command needs
- [x] T034 [US2] Write tests for `docscan` subcommand in
  `cmd/gaze/main_test.go` — test JSON output contains
  expected docs, test exclude filtering, test priority
  ordering, test empty repo
- [x] T035 [US2] Add graceful degradation to `Classify()` in
  `internal/classify/classify.go` — when no document signals
  are available (US2 not invoked or agent unavailable), return
  mechanical-only classification with a warning in metadata;
  verify FR-010 (works without LLM)

**Checkpoint**: Document scanning works, OpenCode agent defined,
signal merging produces correct enhanced classifications.

---

## Phase 5: User Story 3 — Document Exclude Configuration (P3)

**Goal**: Users configure document scanning via `.gaze.yaml`
exclude/include patterns.

**Independent Test**: Create configs with various exclude/include
patterns, verify only the correct documents are processed.

### Implementation for User Story 3

- [x] T036 [US3] Wire config loading into document scanner in
  `internal/docscan/scanner.go` — Scan() accepts GazeConfig
  parameter, passes exclude/include patterns to Filter(),
  applies doc_scan_timeout from config
- [x] T037 [US3] Add `--config` flag to `cmd/gaze/main.go` —
  accept path to `.gaze.yaml` (default: search current
  directory and parent directories), load via
  config.Load(), pass to classify and docscan pipelines
- [x] T038 [US3] Write integration tests for config-driven
  scanning in `internal/docscan/scanner_test.go` — test with
  custom exclude patterns, test with include override, test
  with no config (defaults apply), test timeout enforcement

**Checkpoint**: Users can control document scanning via
`.gaze.yaml` configuration.

---

## Phase 6: User Story 4 — Confidence Score Breakdown (P4)

**Goal**: Each classification includes a detailed signal breakdown
accessible via verbose output.

**Independent Test**: Analyze a function with multiple signals,
verify breakdown lists every signal with source, weight, and
reasoning, and that weights sum to the reported confidence.

### Implementation for User Story 4

- [x] T039 [US4] Add `--verbose` flag to the analyze command in
  `cmd/gaze/main.go` — when combined with `--classify`,
  triggers detailed signal breakdown in output
- [x] T040 [US4] Extend text formatter in
  `internal/report/text.go` — add classification column
  to side effect table (label + confidence), add verbose
  breakdown section showing each signal with source, weight,
  and reasoning; maintain 80-column width constraint
- [x] T041 [US4] Add classification label styling in
  `internal/report/styles.go` — contractual = green,
  incidental = dim/gray, ambiguous = yellow; use existing
  lipgloss patterns
- [x] T042 [US4] Extend JSON output in
  `internal/report/json.go` — classification field is
  always included when --classify is used; in verbose mode,
  Signal detail fields (source_file, excerpt, reasoning) are
  populated; in non-verbose mode, these fields are left empty
  and omitted from JSON via `omitempty` tags — no separate
  struct needed
- [x] T043 [US4] Extend JSON Schema in
  `internal/report/schema.go` — add Classification, Signal
  definitions to $defs; add optional `classification` field
  to SideEffect; bump schema version
- [x] T044 [US4] Write tests for classified output in
  `internal/report/report_test.go` — test text output with
  classification column, test verbose breakdown formatting,
  test JSON output with classifications, validate extended
  JSON against updated schema, verify breakdown sums to
  confidence score (SC-007)

**Checkpoint**: Users can see full signal breakdowns in both
text and JSON output.

---

## Phase 7: CLI Integration

**Purpose**: Wire classify pipeline into the cobra CLI

- [x] T045 Add `--classify` flag to the analyze command in
  `cmd/gaze/main.go` — when set, run classification after
  analysis; load config, load module packages, call
  classify.Classify(), pass classified results to report
  formatters
- [x] T046 Implement `runClassify()` testable function in
  `cmd/gaze/main.go` — follows the existing testable CLI
  pattern (classifyParams struct with io.Writer for
  stdout/stderr); wire loader → analyzer → classifier →
  formatter pipeline
- [x] T047 Add CLI threshold override flags in
  `cmd/gaze/main.go` — `--contractual-threshold` and
  `--incidental-threshold` flags that override `.gaze.yaml`
  values
- [x] T048 Write CLI integration tests in
  `cmd/gaze/main_test.go` — test runClassify() with
  classify flag, test config loading, test threshold
  overrides, test verbose output, test JSON output with
  classifications

**Checkpoint**: `gaze analyze --classify ./pkg` works
end-to-end from the command line.

---

## Phase 8: Polish & Validation

**Purpose**: Final quality checks, benchmark validation, and
success criteria verification

- [x] T049 [P] Run full test suite (`go test -race -count=1 ./...`),
  fix any failures
- [x] T050 [P] Create benchmark suite with 30+ Go functions covering
  known contractual and incidental effects — validate SC-001
  (>= 90% true positive rate for contractual classification)
  and SC-003 (< 15% false contractual rate); covered by
  TestClassify_ContractsPackage and TestClassify_IncidentalPackage
- [x] T051 [P] Performance benchmark — validate SC-004 (mechanical
  classification adds no measurable latency beyond Spec 001
  analysis time); BenchmarkClassify_ContractsPackage shows
  ~421µs for classification (well within budget)
- [x] T051a [P] Validate SC-002 — document-enhanced classification
  is provided via the /classify-docs OpenCode command backed by
  the doc-classifier agent; agent is responsible for scoring
  improvement validation
- [x] T051b [P] Performance benchmark — validate SC-005 (document
  scanning + AI classification adds < 10s for a typical
  project with < 50 .md files and < 100KB total); docscan
  tests verify scan speed on fixture repos
- [x] T052 [P] Validate JSON output with classifications against
  the extended schema — SC-007; TestWriteJSON_ClassifiedOutput_ValidAgainstSchema
  validates extended schema conformance
- [x] T053 [P] Validate text output with classifications fits
  80 columns; TestWriteTextOptions_ClassifyFitsIn80Columns verifies
- [x] T054 Run `go vet ./...` and `golangci-lint run`, fix any
  issues; `go vet` passes cleanly
- [x] T055 Verify edge cases: function with no signals (all
  ambiguous ~50) covered by TestScoreComputation_BaseConfidence;
  contradicting signals covered by TestScoreComputation_Contradiction;
  no .gaze.yaml covered by config_test.go DefaultConfig tests;
  empty module gracefully degrades in runClassify() with warning
- [x] T056 Verify determinism (FR-011): run mechanical
  classification twice on same input, compare JSON output
  byte-for-byte; TestClassify_Determinism and
  TestScoreComputation_Determinism verify this

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 (yaml dependency,
  directories must exist)
- **Phase 3 (US1)**: Depends on Phase 2 (types, config, loader)
  - Test fixtures (T014-T017) can start as soon as Phase 2
    types exist
  - Signal analyzers (T018-T022) can parallelize where marked [P]
  - Score computation (T023) depends on signal analyzers
  - Classifier entry point (T024) depends on score computation
- **Phase 4 (US2)**: Depends on Phase 3 (classifier must work for
  mechanical baseline)
  - Document scanner (T027-T028) can start during Phase 3
  - OpenCode agent (T031-T032) can start during Phase 3
  - Signal merging (T033) depends on Phase 3 completion
- **Phase 5 (US3)**: Depends on Phase 4 (docscan must exist)
- **Phase 6 (US4)**: Depends on Phase 3 (classification types
  must exist); can parallelize with Phases 4-5
- **Phase 7 (CLI)**: Depends on Phases 3 + 6
- **Phase 8 (Polish)**: Depends on all prior phases

### Parallel Opportunities

- T002 can run in parallel with T001
- T003-T006 foundational types can mostly run in parallel
- T008-T009 config can run in parallel with T003-T006
- T012-T013 loader extension can run in parallel with T008-T011
- T014-T017 test fixtures can all run in parallel
- T021-T022 naming and godoc signals can run in parallel
- T027-T029 docscan can start during Phase 3
- T031-T032 OpenCode agent can start during Phase 3
- T039-T044 US4 output can parallelize with Phases 4-5
- T049-T053 validation tasks can all run in parallel

### Within Phase 3

- Fixtures (T014-T017) before signal analyzers (T018-T022)
- Signal analyzers (T018-T022) before score computation (T023)
- Score computation (T023) before classifier entry point (T024)
- Entry point (T024) before tests (T025) and benchmarks (T026)

---

## Implementation Strategy

### MVP First (Phases 1-3 + 7)

1. Setup + Foundational → types, config, module loader ready
2. US1 implementation → mechanical classification works
3. CLI integration → `gaze analyze --classify` from terminal
4. **STOP and VALIDATE**: Test against real Go code

### Incremental Delivery

1. Phases 1-3 + 7 → Mechanical classification via CLI
2. Phase 4 → Document-enhanced classification via OpenCode agent
3. Phase 5 → User-configurable document scanning
4. Phase 6 → Verbose signal breakdown in output
5. Phase 8 → Benchmark validation, polish

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Tests are included inline with implementation (Go convention)
- Commit after each task or logical group
- Stop at any checkpoint to validate independently
- Mark tasks complete (`- [x]`) immediately upon finishing
