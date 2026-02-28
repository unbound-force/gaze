# Tasks: CRAPload Reduction

**Input**: Design documents from `/specs/009-crapload-reduction/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Capture baseline metrics before any changes

- [x] T001 Capture baseline CRAP/GazeCRAP metrics by running `gaze crap --format=json ./...` and recording: CRAPload=27, GazeCRAPload=7, docscan.Filter GazeCRAP=72, LoadModule CRAP=56, runCrap CRAP=52, runSelfCheck CRAP=43, buildContractCoverageFunc CRAP=70, AnalyzeP1Effects GazeCRAP=32, AnalyzeP2Effects GazeCRAP=18

---

## Phase 2: Foundational

**Purpose**: Verify baseline and confirm test infrastructure is in place

- [x] T002 Run full test suite `go test -race -count=1 -short ./...` to confirm clean baseline — all 11 packages must pass before any changes begin
- [x] T003 Run `golangci-lint run` to confirm clean lint baseline

**Checkpoint**: Baseline captured, all tests and lint pass — user story implementation can begin

---

## Phase 3: User Story 1 — Contract Coverage for Document Filter (Priority: P1) MVP

**Goal**: Close the contract coverage gap on `docscan.Filter` (GazeCRAP 72 → <15, Q3 → Q1)

**Independent Test**: Run `gaze crap ./internal/docscan/...` and verify `Filter` is Q1 with GazeCRAP < 15

- [x] T004 [P] [US1] Create `internal/docscan/filter_test.go` with `package docscan_test` and table-driven tests for `docscan.Filter` covering: default inclusion (no config patterns), nil config fallback, exclude-pattern match, include-pattern match, include-pattern miss (FR-001)
- [x] T005 [P] [US1] Add glob-matching tests to `internal/docscan/filter_test.go` covering: patterns with path separators (full-path matching), patterns without path separators (base-name matching), backslash path normalization edge case (FR-002)
- [x] T006 [US1] Run `go test -race -count=1 ./internal/docscan/...` to verify all new and existing tests pass with zero regressions (FR-023)
- [x] T007 [US1] Run `gaze crap ./internal/docscan/...` and verify `docscan.Filter` achieves contract coverage ≥ 80% and GazeCRAP < 15 (FR-003, SC-003)

**Checkpoint**: docscan.Filter moves from Q3 to Q1. Q3 quadrant eliminated from project.

---

## Phase 4: User Story 2 — Unit Tests for Module Loader (Priority: P1)

**Goal**: Add direct tests for `LoadModule` (CRAP 56 → <15, 0% coverage → >70%)

**Independent Test**: Run `gaze crap ./internal/loader/...` and verify `LoadModule` CRAP < 15

- [x] T008 [P] [US2] Add happy-path test `TestLoadModule_ValidModule` to existing `internal/loader/loader_test.go` that calls `LoadModule` with the project's own module root directory and verifies the returned `ModuleResult` contains at least one package with resolved type information. Guard with `testing.Short()` (FR-004, FR-007)
- [x] T009 [P] [US2] Add error-path test `TestLoadModule_NonExistentDir` to `internal/loader/loader_test.go` that calls `LoadModule` with a non-existent directory and verifies a descriptive error is returned without panicking (FR-005)
- [x] T010 [US2] Add error-filtering test `TestLoadModule_ExcludesBrokenPackages` to `internal/loader/loader_test.go` that creates a temporary directory with a minimal `go.mod` + deliberately broken `.go` file alongside a valid `.go` file, calls `LoadModule`, and verifies broken packages are excluded while valid packages are retained. Guard with `testing.Short()` (FR-006, FR-007)
- [x] T011 [US2] Run `go test -race -count=1 ./internal/loader/...` to verify all new and existing tests pass with zero regressions (FR-023)
- [x] T012 [US2] Run `gaze crap ./internal/loader/...` and verify `LoadModule` CRAP < 15 (SC-004)

**Checkpoint**: LoadModule has direct tests and CRAP drops from 56 to below 15.

---

## Phase 5: User Story 3 — Testable CLI Commands (Priority: P2)

**Goal**: Make `runCrap` and `runSelfCheck` testable via dependency injection (CRAP 52/43 → <20)

**Independent Test**: Run `gaze crap ./cmd/gaze/...` and verify both functions have CRAP < 20

- [x] T013 [US3] Add `analyzeFunc` and `coverageFunc` optional function fields to `crapParams` struct in `cmd/gaze/main.go`. When nil, the function delegates to the production implementation (`crap.Analyze` and `buildContractCoverageFunc` respectively). Update `runCrap` to check for non-nil fields before calling production defaults (FR-008, R1)
- [x] T014 [US3] Add `moduleRootFunc` and `runCrapFunc` optional function fields to `selfCheckParams` struct in `cmd/gaze/main.go`. Update `runSelfCheck` to delegate to `runCrapFunc` when non-nil, or call `runCrap` directly with discovered module root. Verify external behavior is preserved by running existing tests (FR-009, FR-012, R1)
- [x] T015 [US3] Verify `newCrapCmd` and `newSelfCheckCmd` cobra command constructors still produce correct params with nil function fields (default behavior). Run `go test -race -count=1 -short ./cmd/gaze/...` to confirm zero regressions (FR-012)
- [x] T016 [P] [US3] Add fast unit tests for `runCrap` to `cmd/gaze/main_test.go`: `TestRunCrap_TextOutput` (stub analyzeFunc returning canned report, verify text output contains expected content), `TestRunCrap_JSONOutput` (same with JSON format), `TestRunCrap_NoCoverageWarning` (nil coverageFunc, verify stderr contains unavailability warning), `TestRunCrap_ThresholdPass` (under threshold, verify nil error), `TestRunCrap_ThresholdBreach` (over threshold, verify non-nil error), `TestRunCrap_EmptyPatterns` (empty patterns list, verify sensible default or descriptive error per edge case spec) (FR-010)
- [x] T017 [P] [US3] Add fast unit tests for `runSelfCheck` to `cmd/gaze/main_test.go`: `TestRunSelfCheck_HappyPath` (stub runCrapFunc, verify delegation), `TestRunSelfCheck_ModuleRootError` (stub moduleRootFunc returning error, verify error propagation), `TestRunSelfCheck_FormatValidation` (invalid format, verify error) (FR-011)
- [x] T018 [US3] Run `go test -race -count=1 -short ./cmd/gaze/...` to verify all new and existing tests pass with zero regressions (FR-023)
- [x] T019 [US3] Run `gaze crap ./cmd/gaze/...` and verify `runCrap` and `runSelfCheck` each have CRAP < 20 (SC-005)

**Checkpoint**: Both CLI command functions are testable with fast unit tests. CRAP drops from 52/43 to below 20.

---

## Phase 6: User Story 4 — Decompose Quality Pipeline Orchestrator (Priority: P2)

**Goal**: Decompose `buildContractCoverageFunc` (CRAP 70, CC=18) into independently testable functions (no function > CRAP 30)

**Independent Test**: Run `gaze crap ./cmd/gaze/...` and verify no decomposed pipeline function has CRAP > 30

- [x] T020 [US4] Extract `resolvePackagePaths(patterns []string, moduleDir string) ([]string, error)` from `buildContractCoverageFunc` in `cmd/gaze/main.go`. This function resolves patterns via `go/packages.Load` (NeedName mode), filters out `_test` suffix packages, and returns individual package paths. The original function calls the new extracted function (FR-013, R5)
- [x] T021 [US4] Extract `analyzePackageCoverage(pkgPath string, gazeConfig *config.GazeConfig, stderr io.Writer) []taxonomy.QualityReport` from the per-package loop body in `buildContractCoverageFunc` in `cmd/gaze/main.go`. This function runs the 4-step pipeline (analysis → classify → test-load → quality assess) for a single package, returning quality reports or nil on any step failure. The original function iterates and calls this for each path. Note: `moduleDir` parameter was dropped during implementation because the function uses `pkgPath` directly via `analysis.LoadAndAnalyze` and does not need the module directory (FR-014, R5)
- [x] T022 [US4] Refactor `buildContractCoverageFunc` in `cmd/gaze/main.go` to be a thin coordinator: call `resolvePackagePaths`, iterate calling `analyzePackageCoverage`, aggregate reports into coverage map, return closure. Verify the return value and nil-on-empty-map behavior is preserved (FR-015, FR-017)
- [x] T023 [P] [US4] Add tests for extracted pipeline functions in `cmd/gaze/main_test.go`: (a) `TestResolvePackagePaths_ValidPattern` (verify returns non-empty paths for `./internal/docscan/...`), `TestResolvePackagePaths_FilterTestSuffix` (verify `_test` packages are excluded), `TestResolvePackagePaths_AllTestVariants` (verify that patterns resolving to only `_test` packages return an empty list, preserving current filtering behavior per edge case spec), `TestResolvePackagePaths_InvalidPattern` (verify error for non-resolvable pattern); (b) `TestAnalyzePackageCoverage_ValidPackage` (verify returns non-nil quality reports for a well-tested package like `internal/docscan`), `TestAnalyzePackageCoverage_InvalidPackage` (verify returns nil for non-existent package). Guard slow tests with `testing.Short()` (FR-016)
- [x] T024 [US4] Run `go test -race -count=1 -short ./cmd/gaze/...` to verify all new and existing tests pass with zero regressions. Pay special attention to `TestBuildContractCoverageFunc_WelltestedPackage` and `TestBuildContractCoverageFunc_InvalidPattern` (FR-017, FR-023)
- [x] T025 [US4] Run `gaze crap ./cmd/gaze/...` and verify no decomposed pipeline function has CRAP > 30 (SC-007)

**Checkpoint**: Pipeline orchestrator decomposed. No individual function exceeds CRAP 30. Existing behavior preserved.

---

## Phase 7: User Story 5 — Decompose Effect Detection Engines (Priority: P3)

**Goal**: Decompose `AnalyzeP1Effects` (CC=32, GazeCRAP 32) and `AnalyzeP2Effects` (CC=18, GazeCRAP 18) into per-node-type handlers (all handlers GazeCRAP < 15)

**Independent Test**: Run `gaze crap ./internal/analysis/...` and verify no function has GazeCRAP > 15

### P1 Effects Decomposition

- [x] T026 [US5] Extract `detectAssignEffects(fset *token.FileSet, info *types.Info, node *ast.AssignStmt, pkg, funcName string, seen map[string]bool, locals map[string]bool) []taxonomy.SideEffect` from the `*ast.AssignStmt` arm of `AnalyzeP1Effects` in `internal/analysis/p1effects.go`. This handler detects GlobalMutation, MapMutation, and SliceMutation effects from assignment statements. Note: parameter order refined during implementation to place `node` before context params (FR-018, R3)
- [x] T027 [US5] Extract `detectIncDecEffects(fset *token.FileSet, info *types.Info, node *ast.IncDecStmt, pkg, funcName string, seen map[string]bool, locals map[string]bool) []taxonomy.SideEffect` from the `*ast.IncDecStmt` arm of `AnalyzeP1Effects` in `internal/analysis/p1effects.go`. This handler detects GlobalMutation effects from increment/decrement statements (FR-018, R3)
- [x] T028 [US5] Extract `detectSendEffects(fset *token.FileSet, node *ast.SendStmt, pkg, funcName string, seen map[string]bool) []taxonomy.SideEffect` from the `*ast.SendStmt` arm of `AnalyzeP1Effects` in `internal/analysis/p1effects.go`. This handler detects ChannelSend effects. Note: `info *types.Info` omitted from signature because send detection does not require type information (FR-018, R3)
- [x] T029 [US5] Extract `detectP1CallEffects(fset *token.FileSet, info *types.Info, node *ast.CallExpr, pkg, funcName string, seen map[string]bool) []taxonomy.SideEffect` from the `*ast.CallExpr` arm of `AnalyzeP1Effects` in `internal/analysis/p1effects.go`. This handler detects ChannelClose, WriterOutput, and HTTPResponseWrite effects. Note: `locals` omitted because call effect detection does not check for local variables (FR-018, R3)
- [x] T030 [US5] Refactor `AnalyzeP1Effects` in `internal/analysis/p1effects.go` to be a thin dispatcher: collect locals, initialize seen map, then `ast.Inspect` with type switch delegating to the four extracted handlers. Append returned effects to the shared slice. Preserve deduplication via the shared `seen` map (FR-018, FR-021)
- [x] T031 [US5] Run `go test -race -count=1 ./internal/analysis/...` to verify all 9 direct P1 tests and 13 integration tests produce identical results post-decomposition. Pay special attention to `TestAnalyzeP1Effects_Direct_*` and `TestP1_*` tests (FR-021, FR-023)

### P2 Effects Decomposition

- [x] T032 [P] [US5] Extract `detectP2CallEffects(fset *token.FileSet, info *types.Info, node *ast.CallExpr, pkg, funcName string, seen map[string]bool, funcParams map[string]bool) []taxonomy.SideEffect` from the `*ast.CallExpr` arm of `AnalyzeP2Effects` in `internal/analysis/p2effects.go`. This handler detects Panic, selector-based effects (LogWrite, FileSystemWrite, FileSystemDelete, FileSystemMeta, ContextCancellation), DatabaseWrite/DatabaseTransaction, and CallbackInvocation effects (FR-020, R4)
- [x] T033 [P] [US5] Extract `detectGoroutineEffects(fset *token.FileSet, node *ast.GoStmt, pkg, funcName string, seen map[string]bool) []taxonomy.SideEffect` from the `*ast.GoStmt` arm of `AnalyzeP2Effects` in `internal/analysis/p2effects.go`. This handler detects GoroutineSpawn effects. Note: `seen` parameter added during implementation because deduplication is needed for the shared map invariant (FR-020, R4)
- [x] T034 [US5] Refactor `AnalyzeP2Effects` in `internal/analysis/p2effects.go` to be a thin dispatcher: collect function params, initialize seen map, then `ast.Inspect` with type switch delegating to the two extracted handlers (`detectGoroutineEffects` for `*ast.GoStmt`, `detectP2CallEffects` for `*ast.CallExpr`). Preserve deduplication via the shared `seen` map (FR-020, FR-021)
- [x] T035 [US5] Run `go test -race -count=1 ./internal/analysis/...` to verify all P2 direct and integration tests produce identical results post-decomposition. Pay special attention to `TestAnalyzeP2Effects_Direct_*` and `TestP2_*` tests (FR-021, FR-023)

### US5 Verification

- [x] T036 [US5] Run `gaze crap ./internal/analysis/...` and verify no individual handler function has GazeCRAP > 15. Verify `AnalyzeP1Effects` and `AnalyzeP2Effects` are no longer in Q4 (SC-006)
- [x] T037 [US5] Verify cyclomatic complexity of each extracted handler is ≤ 15 by inspecting the `gaze crap` output for the `complexity` field of each new function (FR-022)

**Checkpoint**: Both P1 and P2 effect detection engines decomposed. All handlers have GazeCRAP < 15. All existing tests pass with identical results.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, documentation, and metrics validation

- [x] T038 Run full test suite `go test -race -count=1 -short ./...` to verify zero regressions across all 11 packages (SC-008)
- [x] T039 Run `golangci-lint run` to verify no new lint violations introduced (SC-009, FR-024)
- [x] T040 Verify GoDoc comments on all new exported or helper functions across all modified files (FR-025)
- [x] T041 Run `gaze crap --format=json ./...` and verify final metrics: GazeCRAPload ≤ 4, CRAPload ≤ 24, Q3 count = 0 (SC-001, SC-002, SC-003)
- [x] T042 Update `specs/009-crapload-reduction/tasks.md` with final baseline/after measurements summary
- [x] T043 Assess documentation impact: check if README.md, AGENTS.md, or other docs need updates for new patterns or conventions introduced

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — MUST run first to capture baseline
- **Foundational (Phase 2)**: Depends on Phase 1 — verification gate only
- **US1 (Phase 3)**: Depends on Phase 2 completion — can run in parallel with US2
- **US2 (Phase 4)**: Depends on Phase 2 completion — can run in parallel with US1
- **US3 (Phase 5)**: Depends on Phase 2 completion — can run in parallel with US1/US2 but MUST complete before US4 (US4 decomposes a function in the same file US3 modifies)
- **US4 (Phase 6)**: Depends on US3 (Phase 5) completion — both modify `cmd/gaze/main.go`
- **US5 (Phase 7)**: Depends on Phase 2 completion — can run in parallel with all other stories (different files)
- **Polish (Phase 8)**: Depends on all previous phases

### User Story Dependencies

- **US1 (docscan.Filter)**: Independent — only touches `internal/docscan/filter_test.go` (new file)
- **US2 (LoadModule)**: Independent — only touches `internal/loader/loader_test.go` (new file)
- **US3 (CLI testability)**: Independent of US1/US2 — modifies `cmd/gaze/main.go` and `cmd/gaze/main_test.go`
- **US4 (Pipeline decomposition)**: Depends on US3 — both modify `cmd/gaze/main.go` sequentially
- **US5 (Effect decomposition)**: Independent — only touches `internal/analysis/p1effects.go` and `internal/analysis/p2effects.go`

### Within Each User Story

- Production code changes before test additions (for US3, US4, US5)
- Test-only stories (US1, US2) have no production changes
- Verification step at the end of each story

### Parallel Opportunities

- **US1 + US2**: Can run fully in parallel (different packages, no file conflicts)
- **US1 + US5**: Can run in parallel (different packages)
- **US2 + US5**: Can run in parallel (different packages)
- **US3 T016 + T017**: Test tasks can run in parallel (different test functions, same file but no conflicts)
- **US5 T032 + T033**: P2 handler extractions can run in parallel (different node types, but same file — must be done sequentially in practice)

---

## Parallel Example: US1 + US2

```bash
# These two stories touch completely different packages and can run simultaneously:
# Worker A:
Task: T004 [US1] Create filter_test.go with table-driven tests
Task: T005 [US1] Add glob-matching tests to filter_test.go
Task: T006 [US1] Run tests for docscan package
Task: T007 [US1] Verify GazeCRAP metrics

# Worker B (simultaneously):
Task: T008 [US2] Create loader_test.go with happy-path test
Task: T009 [US2] Add error-path test
Task: T010 [US2] Add error-filtering test
Task: T011 [US2] Run tests for loader package
Task: T012 [US2] Verify CRAP metrics
```

---

## Implementation Strategy

### MVP First (US1 + US2 Only)

1. Complete Phase 1: Capture baseline metrics
2. Complete Phase 2: Verify clean baseline
3. Complete Phase 3 (US1) + Phase 4 (US2) in parallel
4. **STOP and VALIDATE**: GazeCRAPload should drop from 7 to ~5, CRAPload from 27 to ~26
5. This delivers the two highest-ROI improvements with zero production code changes

### Incremental Delivery

1. US1 + US2 → Test-only MVP (GazeCRAPload 7→5, CRAPload 27→26)
2. US3 → CLI testability (CRAPload 26→24, adds testable CLI pattern)
3. US4 → Pipeline decomposition (highest CRAP function addressed)
4. US5 → Effect decomposition (structural improvement, Q4 cleanup)
5. Polish → Final metrics validation

### Sequential Single-Worker Order

For a single developer working sequentially:

1. Phase 1 + 2 (baseline)
2. US1 (test-only, fast, highest GazeCRAP impact)
3. US2 (test-only, fast, highest CRAP impact)
4. US3 (production code change, establishes DI pattern)
5. US4 (depends on US3, decomposes pipeline)
6. US5 (independent, can be done any time after Phase 2)
7. Phase 8 (polish)
