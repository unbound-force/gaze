# Tasks: Contract Coverage Gap Remediation

**Input**: Design documents from `specs/008-contract-coverage-gaps/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: This feature is entirely test-only code. All tasks create or strengthen unit tests with contract-level assertions. No new production code is written.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Test files**: `*_test.go` alongside source in the same directory (Go convention)
- **Test fixtures**: `testdata/src/<pkg>/` directories loaded via `go/packages`
- **External test package**: `classify_test`, `analysis_test` (tests exported API surface)
- **Internal test package**: `package main` for `cmd/gaze/` (tests unexported functions)

---

## Phase 1: Setup

**Purpose**: Establish baseline measurements before any code changes

- [ ] T001 Build gaze binary with `go build ./cmd/gaze`
- [ ] T002 Record baseline weighted average contract coverage by running `gaze quality --format=json ./...` and extracting `.summary.weighted_average_contract_coverage`
- [ ] T003 Record baseline GazeCRAPload by running `gaze crap --format=json ./...` and extracting `.summary.gazecrapload`
- [ ] T004 Record baseline quadrant positions for target functions (`AnalyzeP1Effects`, `AnalyzeP2Effects`, `AnalyzeReturns`) by running `gaze crap --format=json ./...`
- [ ] T005 Run spec 007 quickstart.md validation: `go test -race -count=1 -run TestSC003_MappingAccuracy ./internal/quality/...` (FR-021, completing deferred T044)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Verify existing test infrastructure and fixtures support direct invocation tests

**No new code needed**: The existing test helpers (`loadTestPackages`, `cachedTestPackage`, `FindFuncDecl`, `hasEffect`, `countEffects`, `effectWithTarget`) and fixtures (`contracts`, `callers`, `incidental`, `p1effects`, `p2effects`, `returns`) already provide all required infrastructure. This phase is a verification gate only.

- [ ] T006 Verify classify test fixtures are loadable: confirm `loadTestPackages(t)` in `internal/classify/classify_test.go` loads `contracts`, `callers`, `incidental`, and `ambiguous` fixtures without errors
- [ ] T007 Verify analysis test fixtures are loadable: confirm `loadTestPackage(t, "p1effects")`, `loadTestPackage(t, "p2effects")`, and `loadTestPackage(t, "returns")` all succeed via `cachedTestPackage` in `internal/analysis/analysis_test.go`
- [ ] T008 Verify full test suite passes before changes: run `go test -race -count=1 -short ./...`

**Checkpoint**: Foundation verified - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Classification Signal Verification (Priority: P1) MVP

**Goal**: Add direct unit tests for `AnalyzeVisibilitySignal`, `AnalyzeGodocSignal`, and `AnalyzeCallerSignal` to verify weight computation, reasoning strings, and edge cases independently from the `Classify()` pipeline.

**Independent Test**: `go test -race -count=1 -run 'TestAnalyze(Visibility|Godoc|Caller)Signal' ./internal/classify/...`

### Implementation for User Story 1

- [ ] T009 [P] [US1] Create `internal/classify/visibility_test.go` with table-driven tests for `AnalyzeVisibilitySignal` covering: exported function (+8 weight), exported return type (+6), exported receiver type (+6), weight clamp to maxVisibilityWeight (20), and reasoning string content for each dimension (FR-001, FR-002)
- [ ] T010 [US1] Add edge case tests to `internal/classify/visibility_test.go` for: unexported functions (weight 0), nil `funcDecl` (zero signal), nil `funcObj` (zero signal) (FR-003)
- [ ] T011 [P] [US1] Create `internal/classify/godoc_test.go` with table-driven tests for `AnalyzeGodocSignal` covering: each contractual keyword ("returns", "sets", "writes", "sends", "creates", "deletes", "updates", "modifies", "stores") paired with matching `effectType` (+15 weight), and non-matching effectType (0 weight) (FR-004)
- [ ] T012 [US1] Add incidental priority tests to `internal/classify/godoc_test.go` verifying: when godoc contains both "logs" and "returns", weight is -15 (incidental wins) (FR-005)
- [ ] T013 [US1] Add edge case tests to `internal/classify/godoc_test.go` for: nil `funcDecl` (zero signal), nil `Doc` comment group (zero signal) (FR-006)
- [ ] T014 [P] [US1] Create `internal/classify/callers_test.go` with tests for `AnalyzeCallerSignal` covering weight tiers: 1 caller -> 5, 2-3 callers -> 10, 4+ callers -> 15 (capped). Use `loadTestPackages(t)` to load `contracts`+`callers` fixtures and find target `types.Object` (FR-007)
- [ ] T015 [US1] Add same-package caller exclusion test to `internal/classify/callers_test.go` verifying: callers from the same package are excluded from count (weight 0 if only same-package callers) (FR-008)
- [ ] T016 [US1] Add edge case tests to `internal/classify/callers_test.go` for: nil `funcObj` (zero signal), functions with zero cross-package callers (zero signal) (FR-009)
- [ ] T017 [US1] Run US1 verification: `go test -race -count=1 -run 'TestAnalyze(Visibility|Godoc|Caller)Signal' ./internal/classify/...`

**Checkpoint**: All 3 classification signal analyzers have direct unit tests. Each signal function's weight computation and reasoning are independently verifiable.

---

## Phase 4: User Story 2 - Analysis Core Contract Assertions (Priority: P1)

**Goal**: Add direct unit tests for `AnalyzeP1Effects`, `AnalyzeP2Effects`, and `AnalyzeReturns` that call each function with fixture-loaded inputs and assert on the returned `[]taxonomy.SideEffect` slice, independent of the `AnalyzeFunctionWithSSA` pipeline.

**Independent Test**: `go test -race -count=1 -run 'TestAnalyze(P1Effects|P2Effects|Returns)_Direct' ./internal/analysis/...`

### Implementation for User Story 2

- [ ] T018 [P] [US2] Create `internal/analysis/p1effects_test.go` (package `analysis_test`) with direct tests for `AnalyzeP1Effects` covering: `GlobalMutation` (via `MutateGlobal`), `ChannelSend` (via `SendOnChannel`), `ChannelClose` (via `CloseChannel`), `WriterOutput` (via `WriteToWriter`), `HTTPResponseWrite` (via `HandleHTTP`), `MapMutation` (via `WriteToMap`), `SliceMutation` (via `WriteToSlice`), and pure function returning empty slice (via `PureP1`). Use `loadTestPackage(t, "p1effects")` and `analysis.FindFuncDecl(pkg, name)`. Assert on `Type`, `Tier` (P1), and non-empty `Description` for each effect (FR-010, FR-011, FR-014)
- [ ] T019 [US2] Add nil body edge case test to `internal/analysis/p1effects_test.go` verifying: `AnalyzeP1Effects` returns empty slice without panic when `fd.Body == nil` (edge case from spec)
- [ ] T020 [P] [US2] Create `internal/analysis/p2effects_test.go` (package `analysis_test`) with direct tests for `AnalyzeP2Effects` covering: `GoroutineSpawn`, `Panic`, `FileSystemWrite`, `FileSystemDelete`, `LogWrite`, `ContextCancellation`, `CallbackInvocation`, `DatabaseWrite`, and pure function. Use `loadTestPackage(t, "p2effects")` and `analysis.FindFuncDecl(pkg, name)`. Assert on `Type`, `Tier` (P2), and non-empty `Description` (FR-012, FR-014)
- [ ] T021 [US2] Add nil body edge case test to `internal/analysis/p2effects_test.go` verifying: `AnalyzeP2Effects` returns empty slice without panic when `fd.Body == nil`
- [ ] T022 [P] [US2] Create `internal/analysis/returns_test.go` (package `analysis_test`) with direct tests for `AnalyzeReturns` covering: single return (`SingleReturn`), multiple returns (`MultipleReturns`), error return (`ErrorReturn`), named returns (`NamedReturns`), deferred return mutation (`NamedReturnModifiedInDefer`), and no returns (`PureFunction`). Use `loadTestPackage(t, "returns")` and `analysis.FindFuncDecl(pkg, name)`. Assert on `Type` (`ReturnValue`/`ErrorReturn`), `Tier` (P0), and non-empty `Description` (FR-013, FR-014)
- [ ] T023 [US2] Verify Group B tests do not duplicate existing indirect tests: review `internal/analysis/analysis_test.go` to confirm new tests focus on direct invocation (`AnalyzeP1Effects`, `AnalyzeP2Effects`, `AnalyzeReturns`) rather than re-testing via `AnalyzeFunctionWithSSA` (FR-015)
- [ ] T024 [US2] Run US2 verification: `go test -race -count=1 -run 'TestAnalyze(P1Effects|P2Effects|Returns)_Direct' ./internal/analysis/...`

**Checkpoint**: All 3 analysis core functions have direct unit tests with contract-level assertions on returned `[]SideEffect` slices. Functions are testable independent of the `AnalyzeFunctionWithSSA` pipeline.

---

## Phase 5: User Story 3 - CLI Layer Test Hardening (Priority: P2)

**Goal**: Add unit tests for `renderAnalyzeContent` and strengthen existing `buildContractCoverageFunc` tests to assert on actual coverage values rather than just "no panic" behavior.

**Independent Test**: `go test -race -count=1 -run 'Test(RenderAnalyzeContent|BuildContractCoverageFunc)' ./cmd/gaze/...`

### Implementation for User Story 3

- [ ] T025 [P] [US3] Create `cmd/gaze/interactive_test.go` (package `main`) with unit tests for `renderAnalyzeContent` covering: empty `[]AnalysisResult` slice produces title indicating zero functions, and `AnalysisResult` with side effects produces output containing qualified function name, tier labels, and effect type descriptions. Construct `[]taxonomy.AnalysisResult` structs directly — no fixture loading needed (FR-016)
- [ ] T026 [P] [US3] Add description truncation test to `cmd/gaze/interactive_test.go` verifying: descriptions longer than 50 characters are truncated in output. Add empty results handling test (FR-017)
- [ ] T027 [US3] Strengthen `TestBuildContractCoverageFunc_WelltestedPackage` in `cmd/gaze/main_test.go`: change `fn == nil` from acceptable to `t.Fatal` (assert `fn != nil`), assert `ok == true` for known function `welltested:Add`, assert `pct > 0` (not just `pct >= 0`). Keep `testing.Short()` guard. Note: the spec edge case "zero packages returns nil" is already covered by the existing `TestBuildContractCoverageFunc_InvalidPattern` test (FR-018, FR-019)
- [ ] T028 [US3] Run US3 verification: `go test -race -count=1 -run 'Test(RenderAnalyzeContent|BuildContractCoverageFunc)' ./cmd/gaze/...`

**Checkpoint**: CLI layer functions have unit tests with contract-level assertions. `renderAnalyzeContent` is tested for the first time. `buildContractCoverageFunc` tests verify actual coverage values.

---

## Phase 6: User Story 4 - Baseline Measurement and Verification (Priority: P1)

**Goal**: After all test changes are complete, re-measure weighted average contract coverage and compare against the baseline recorded in Phase 1 to verify improvement.

**Independent Test**: `gaze quality --format=json ./...` and compare `.summary.weighted_average_contract_coverage` against Phase 1 baseline.

**Note**: Phase 1 records the "before" baseline. This phase records the "after" measurement. The ordering dependency is: Phase 1 (baseline) -> Phases 3-5 (code changes) -> Phase 6 (verification).

### Implementation for User Story 4

- [ ] T029 [US4] Rebuild gaze binary: `go build ./cmd/gaze`
- [ ] T030 [US4] Re-measure weighted average contract coverage: `gaze quality --format=json ./...` and compare against Phase 1 baseline (FR-022). Value MUST be higher.
- [ ] T031 [US4] Re-run spec 007 quickstart.md validation: `go test -race -count=1 -run TestSC003_MappingAccuracy ./internal/quality/...` to verify no regressions (FR-021)

**Checkpoint**: Contract coverage improvement is quantified. Spec 007 deferred tasks T037/T044 are complete.

---

## Phase 7: User Story 5 - GazeCRAP Score Improvement (Priority: P2)

**Goal**: Verify that the remediation has reduced GazeCRAPload and moved target functions out of the Dangerous (Q4) quadrant.

**Independent Test**: `gaze crap --format=json ./...` and compare quadrant distribution against Phase 1 baseline.

**Dependency**: Requires all test changes from Phases 3-5 to be complete.

### Implementation for User Story 5

- [ ] T032 [US5] Re-measure GazeCRAPload: `gaze crap --format=json ./...` and compare against Phase 1 baseline (FR-023). Value MUST be lower.
- [ ] T033 [US5] Verify target functions exited Q4: `AnalyzeP1Effects` and `AnalyzeP2Effects` MUST no longer be in the Dangerous (Q4) quadrant (SC-003)
- [ ] T034 [US5] Verify `AnalyzeReturns` quadrant improvement: check that `AnalyzeReturns` has moved to a better quadrant than baseline

**Checkpoint**: GazeCRAP metrics demonstrate measurable improvement from the remediation effort.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and documentation

- [ ] T035 Run full test suite: `go test -race -count=1 -short ./...` to verify zero regressions (SC-005)
- [ ] T036 Run linter: `golangci-lint run` to verify no lint violations introduced (SC-006)
- [ ] T037 Verify GoDoc comments on all new test functions follow project conventions (GoDoc-style comments on exported test helpers if any)
- [ ] T038 Update `specs/008-contract-coverage-gaps/` artifacts: record final baseline/after measurements in a summary, mark spec tasks complete
- [ ] T039 Assess documentation impact: check if README.md, AGENTS.md, or other docs need updates for any new patterns or conventions introduced

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - MUST run first to capture baseline
- **Foundational (Phase 2)**: Depends on Phase 1 - verification gate only
- **US1 (Phase 3)**: Depends on Phase 2 completion
- **US2 (Phase 4)**: Depends on Phase 2 completion - can run in parallel with Phase 3
- **US3 (Phase 5)**: Depends on Phase 2 completion - can run in parallel with Phases 3-4
- **US4 (Phase 6)**: Depends on Phases 3, 4, and 5 completion (needs all test changes)
- **US5 (Phase 7)**: Depends on Phases 3, 4, and 5 completion (needs all test changes)
- **Polish (Phase 8)**: Depends on all previous phases

### User Story Dependencies

- **US1 (Classification Signals)**: Independent - only touches `internal/classify/` test files
- **US2 (Analysis Core)**: Independent - only touches `internal/analysis/` test files
- **US3 (CLI Layer)**: Independent - only touches `cmd/gaze/` test files
- **US4 (Baseline Verification)**: Depends on US1 + US2 + US3 (needs all test changes to measure improvement)
- **US5 (GazeCRAP Verification)**: Depends on US1 + US2 + US3 (needs all test changes to measure improvement)

### Within Each User Story

- Edge case tests can run in parallel with main tests (different test functions, same file)
- Verification step (final task in each US) must run after all other tasks in that US

### Parallel Opportunities

- **Phase 3, 4, 5 can all run in parallel** (different packages, different files)
- Within Phase 3: T009, T011, T014 can run in parallel (different files: `visibility_test.go`, `godoc_test.go`, `callers_test.go`). Same-file follow-up tasks (T010, T012-T013, T015-T016) are sequential within their file group.
- Within Phase 4: T018, T020, T022 can run in parallel (different files: `p1effects_test.go`, `p2effects_test.go`, `returns_test.go`). Same-file follow-ups (T019, T021) are sequential within their file group.
- Within Phase 5: T025-T026 can run in parallel with T027 (different files: `interactive_test.go` vs `main_test.go`)
- Phase 6 and 7 can run in parallel (both are measurement tasks after code changes)

---

## Parallel Example: User Stories 1, 2, and 3

```bash
# All three user stories can be implemented simultaneously since they touch different packages:

# Agent A: User Story 1 (Classification Signals)
Task: "Create visibility_test.go in internal/classify/"
Task: "Create godoc_test.go in internal/classify/"
Task: "Create callers_test.go in internal/classify/"

# Agent B: User Story 2 (Analysis Core)
Task: "Create p1effects_test.go in internal/analysis/"
Task: "Create p2effects_test.go in internal/analysis/"
Task: "Create returns_test.go in internal/analysis/"

# Agent C: User Story 3 (CLI Layer)
Task: "Create interactive_test.go in cmd/gaze/"
Task: "Strengthen main_test.go in cmd/gaze/"
```

---

## Parallel Example: Within User Story 1

```bash
# All classify test files can be created in parallel:
Task: "T009 — visibility_test.go main tests (internal/classify/visibility_test.go)"
Task: "T011 — godoc_test.go main tests (internal/classify/godoc_test.go)"
Task: "T014 — callers_test.go main tests (internal/classify/callers_test.go)"

# Then edge cases can be added in parallel:
Task: "T010 — visibility edge cases (internal/classify/visibility_test.go)"
Task: "T012-T013 — godoc incidental + edge cases (internal/classify/godoc_test.go)"
Task: "T015-T016 — callers same-pkg + edge cases (internal/classify/callers_test.go)"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (record baseline)
2. Complete Phase 2: Foundational (verify infrastructure)
3. Complete Phase 3: US1 - Classification Signal Tests (3 files, 8 tasks)
4. Complete Phase 4: US2 - Analysis Core Tests (3 files, 7 tasks)
5. **STOP and VALIDATE**: Run `go test -race -count=1 -short ./...`
6. These two P1 stories deliver the largest GazeCRAP reduction

### Incremental Delivery

1. Phase 1-2: Setup + Foundational -> Baseline captured, infrastructure verified
2. Phase 3: US1 -> Classification signals independently testable -> Run `go test ./internal/classify/...`
3. Phase 4: US2 -> Analysis core independently testable -> Run `go test ./internal/analysis/...`
4. Phase 5: US3 -> CLI layer independently testable -> Run `go test ./cmd/gaze/...`
5. Phase 6-7: US4 + US5 -> Measure improvement, verify GazeCRAP reduction
6. Phase 8: Polish -> Full suite, lint, docs

### Key Constraints

- All test files use standard library `testing` only (no testify, no external assertion libraries)
- All tests must pass with `-race -count=1`
- Group A tests (`classify/`) use external test package (`classify_test`)
- Group B tests (`analysis/`) use external test package (`analysis_test`) with access to `FindFuncDecl` via `export_test.go`
- Group C tests (`cmd/gaze/`) use internal test package (`package main`) since `renderAnalyzeContent` is unexported
- No new production code is written; only `*_test.go` files are added or modified

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Total: 39 tasks across 8 phases covering 5 user stories
