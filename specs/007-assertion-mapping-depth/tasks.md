# Tasks: Assertion Mapping Depth

**Input**: Design documents from `specs/007-assertion-mapping-depth/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, quickstart.md

**Tests**: Tests are included because the spec requires ratchet baseline updates (FR-011, FR-012) and regression verification (SC-003, SC-004). This feature modifies a core analysis engine where correctness is critical.

**Organization**: Tasks are grouped by user story. US1 (Selector Expression Matching) is the MVP. US2 (Built-in Call Unwinding) and US5 (Helper Return Value Tracing) are co-priority P2. US3 (Index Expression Resolution) is P3. US4 (Confidence Differentiation) is P4 but is woven into the implementation of US1-US3 as a cross-cutting concern.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: All changes under `internal/quality/` in the repository root
- **Primary source file**: `internal/quality/mapping.go`
- **Primary test file**: `internal/quality/quality_test.go`
- **Test fixtures**: `internal/quality/testdata/src/`
- **Taxonomy types**: `internal/taxonomy/types.go`

---

## Phase 1: Setup

**Purpose**: Establish the test fixture needed for indirect expression matching and capture the pre-change baseline

- [x] T001 Create test fixture package `internal/quality/testdata/src/indirectmatch/indirectmatch.go` with exported functions returning structs, slices, and maps to exercise selector, index, and builtin assertion patterns
- [x] T002 Create test fixture tests `internal/quality/testdata/src/indirectmatch/indirectmatch_test.go` with assertions using `result.Field`, `result.A.B`, `len(result)`, `results[0]`, `results[0].Field`, direct identity patterns, and a variable shadowing case (`result = localValue` followed by `result.Field` assertion) — covering all expression resolution patterns from data-model.md and the shadowing edge case from spec.md
- [x] T003 Run `go test -race -count=1 -run TestSC003_MappingAccuracy ./internal/quality/...` and record the pre-change baseline accuracy percentage and mapped/total counts in a comment at the top of the PR branch for regression tracking

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement `resolveExprRoot` — the core function required by ALL user stories (US1, US2, US3)

**CRITICAL**: No user story implementation can begin until this phase is complete. The two-pass matching strategy in US1-US3 depends on this function.

- [x] T004 Implement the `resolveExprRoot(expr ast.Expr, info *types.Info) *ast.Ident` function in `internal/quality/mapping.go` — recursive descent through `SelectorExpr` (recurse on `.X`), `IndexExpr` (recurse on `.X`), `CallExpr` (if `Fun` resolves to `*types.Builtin` with name in `{len, cap}` and exactly 1 argument, recurse on `Args[0]`), `Ident` (return — base case), all other types return `nil` — per data-model.md resolution rules
- [x] T005 Add unit tests for `resolveExprRoot` in `internal/quality/quality_test.go` — test cases: bare ident returns itself, single selector (`x.Field`) returns `x`, deep selector chain (`x.A.B.C`) returns `x`, `len(x)` returns `x`, `cap(x)` returns `x`, `results[0]` returns `results`, combined `results[0].Field` returns `results`, `append(x, y)` returns nil (side-effecting builtin rejected), user-defined function `myLen(x)` returns nil (non-builtin rejected), `len(x, y)` returns nil (multi-arg rejected), non-ident root returns nil
- [x] T006 Run `go test -race -count=1 -run TestResolveExprRoot ./internal/quality/...` to verify all resolution cases pass

**Checkpoint**: `resolveExprRoot` ready — user story implementation can now begin

---

## Phase 3: User Story 1 — Selector Expression Matching (Priority: P1) MVP

**Goal**: The mapping engine recognizes `result.SideEffects` as an assertion on `result` — the traced return value — and maps it to the corresponding `ReturnValue` side effect at confidence 65.

**Independent Test**: Run `gaze quality` on the `multilib` fixture. Tests like `TestNewUser_Testify` should now show covered `ReturnValue` effects because `user.Name` resolves to `user` traced from `user, err := NewUser(...)`.

### Implementation for User Story 1

- [x] T007 [US1] Modify `matchAssertionToEffect` in `internal/quality/mapping.go` to implement two-pass matching strategy: Pass 1 (existing behavior) walks expression tree with `ast.Inspect` matching `*ast.Ident` nodes directly in `objToEffectID` at confidence 75; Pass 2 (new) if Pass 1 found no match, walks expression tree again and for each `SelectorExpr`, `IndexExpr`, or `CallExpr` node calls `resolveExprRoot` — if root ident's `types.Object` is in `objToEffectID`, match at confidence 65
- [x] T008 [US1] Add test for selector expression matching in `internal/quality/quality_test.go` — using the `indirectmatch` fixture, verify that assertions on `result.Field` produce a mapping with confidence 65 and correct `SideEffectID`, while assertions on bare `result` still produce confidence 75
- [x] T009 [US1] Add test for deep selector chain in `internal/quality/quality_test.go` — verify `result.A.B.C` resolves to `result` with confidence 65
- [x] T010 [US1] Add test for non-traced selector and variable shadowing (false positive prevention) in `internal/quality/quality_test.go` — verify that `localVar.Field` where `localVar` is NOT in `objToEffectID` does NOT produce a mapping (FR-009, SC-004); also verify that when a traced variable is reassigned (`result = localValue`), assertions on the shadowed `result` do NOT map to the original return value's effect (spec.md edge case L224-228)
- [x] T011 [US1] Run `go test -race -count=1 -run TestSC003_MappingAccuracy ./internal/quality/...` and verify mapping accuracy has increased from the Phase 1 baseline — do NOT update the ratchet floor yet (wait until all stories complete)
- [x] T012 [US1] Run `go test -race -count=1 -short ./...` to verify no regressions across the project (FR-012)

**Checkpoint**: Selector expression matching works. `multilib` fixture assertions on `user.Name`, `user.Email`, `user.Age` now map correctly. MVP is functional.

---

## Phase 4: User Story 2 — Built-in Call Unwinding (Priority: P2)

**Goal**: `len(docs)` and `cap(m)` assertions on traced return values are unwrapped to match the inner argument against the traced object map at confidence 65.

**Independent Test**: Run `gaze quality` on the `docscan` package and verify that tests using `len(docs)` have their `ReturnValue` effect marked as covered.

### Implementation for User Story 2

- [x] T013 [US2] Verify that `resolveExprRoot` already handles `len(x)` and `cap(x)` via the foundational implementation (T004) — no additional code needed in `matchAssertionToEffect` since the two-pass strategy from T007 already calls `resolveExprRoot` on `CallExpr` nodes
- [x] T014 [US2] Add test for `len()` unwinding in `internal/quality/quality_test.go` — using the `indirectmatch` fixture, verify that `len(results)` where `results` is a traced return value produces a mapping with confidence 65
- [x] T015 [US2] Add test for `cap()` unwinding in `internal/quality/quality_test.go` — verify `cap(results)` produces a mapping with confidence 65
- [x] T016 [US2] Add negative test for side-effecting builtins in `internal/quality/quality_test.go` — verify `append(results, item)` does NOT produce a mapping (FR-004)
- [x] T017 [US2] Add negative test for user-defined `len` function in `internal/quality/quality_test.go` — verify that a user-defined function named `len` is NOT unwrapped (detection via `*types.Builtin` type assertion, per research.md R4)
- [x] T018 [US2] Run `go test -race -count=1 -short ./...` to verify no regressions

**Checkpoint**: Built-in call unwinding works. `len()` and `cap()` patterns are correctly matched.

---

## Phase 5: User Story 5 — Helper Return Value Tracing (Priority: P2)

**Goal**: When `findAssignLHS` fails because the target call is inside a helper function, the mapper searches the test function's AST for assignments whose RHS calls a helper that (at depth 1 via SSA call graph) invokes the target. The helper's return variable is traced as if it were the target's return value.

**Independent Test**: Run `gaze quality` on the `analysis` package. Before: ~6.9% average contract coverage. After: test-target pairs using `result := analyzeFunc(t, ...)` followed by assertions on `result.SideEffects` report coverage for the `ReturnValue` effect.

### Implementation for User Story 5

- [x] T019 [US5] Add a helper return tracing test fixture at `internal/quality/testdata/src/helperreturn/helperreturn.go` with an exported function that returns a struct, and `internal/quality/testdata/src/helperreturn/helperreturn_test.go` with a helper function that calls the target and returns its result, and test functions that call the helper and assert on fields of the returned struct
- [x] T020 [US5] Modify `traceReturnValues` in `internal/quality/mapping.go` to add a fallback path: when `findAssignLHS(testPkg, callPos)` returns nil, search the test function's AST for all `AssignStmt` nodes whose RHS is a `CallExpr`; for each such assignment, resolve the called function via `testPkg.TypesInfo` and check if it (at depth 1) calls the target function using the SSA call graph; if so, map the LHS variables to the corresponding return effects by positional index (same as direct tracing)
- [x] T021 [US5] Implement SSA call graph verification: given a candidate helper function's `*ssa.Function`, iterate its `Blocks` and `Instrs` looking for `*ssa.Call` instructions whose callee matches the target function (using `sameFunction` in `internal/quality/mapping.go`); this is depth-1 only — no recursive search (FR-015)
- [x] T022 [US5] Update `traceTargetValues` in `internal/quality/mapping.go` to accept `testFunc *ssa.Function` and `targetFunc *ssa.Function` as additional parameters, and forward them to `traceReturnValues`; update the call site in `MapAssertionsToEffects` to pass both — these are required for helper return tracing to identify candidate helper calls in the test function's SSA and verify they invoke the target
- [x] T023 [US5] Add test for helper return tracing in `internal/quality/quality_test.go` — using the `helperreturn` fixture, verify that `result := helper(t, ...)` followed by `result.Field` assertions produces mappings with confidence 65 for the target's `ReturnValue` effect
- [x] T024 [US5] Add negative test for non-target helper in `internal/quality/quality_test.go` — verify that a helper function that does NOT call the target does NOT produce false positive tracing (FR-014)
- [x] T025 [US5] Add test verifying helper tracing is fallback-only in `internal/quality/quality_test.go` — confirm that when the target call IS directly assigned in the test function, the helper tracing path is NOT activated (direct tracing at confidence 75 takes precedence)
- [x] T026 [US5] Run `go test -race -count=1 -short ./...` to verify no regressions

**Checkpoint**: Helper return value tracing works. The `analysis` package's helper indirection pattern is now handled. Combined with US1 selector matching, assertions on `result.SideEffects` in helper-traced tests are mapped.

---

## Phase 6: User Story 3 — Index Expression Resolution (Priority: P3)

**Goal**: `results[0]` and `results[0].Field` assertions on traced return values resolve through both indexing and selector unwinding to the root identifier.

**Independent Test**: Run `gaze quality` on the `indirectmatch` fixture and verify that `results[0].Field` assertions produce mappings with confidence 65.

### Implementation for User Story 3

- [x] T027 [US3] Verify that `resolveExprRoot` already handles `IndexExpr` via the foundational implementation (T004) and that the two-pass strategy from T007 already resolves index expressions — no additional code needed in `matchAssertionToEffect`
- [x] T028 [US3] Add test for index expression resolution in `internal/quality/quality_test.go` — using the `indirectmatch` fixture, verify that `results[0]` produces a mapping with confidence 65
- [x] T029 [US3] Add test for combined index + selector in `internal/quality/quality_test.go` — verify `results[0].Field.SubField` resolves to `results` with confidence 65 (FR-006)
- [x] T030 [US3] Add negative test for index on non-traced variable in `internal/quality/quality_test.go` — verify `localSlice[0]` does NOT produce a mapping
- [x] T031 [US3] Run `go test -race -count=1 -short ./...` to verify no regressions

**Checkpoint**: Index expression resolution works. Combined patterns like `results[0].Field` resolve correctly.

---

## Phase 7: User Story 4 — Confidence Differentiation (Priority: P4)

**Goal**: Direct identity matches remain at confidence 75; indirect matches (selector, builtin, index, helper) use confidence 65. JSON output reflects distinct values.

**Independent Test**: Run `gaze quality --format=json` on a test with both direct and indirect assertions. Verify direct=75 and indirect=65 in the JSON output.

### Implementation for User Story 4

- [x] T032 [US4] Verify that the two-pass matching (T007) and helper tracing (T020) already produce confidence 65 for indirect matches and 75 for direct matches — this is implemented as part of US1 and US5; no additional code changes expected
- [x] T033 [US4] Add integration test in `internal/quality/quality_test.go` verifying confidence differentiation in full pipeline output — run `Assess` on the `indirectmatch` fixture and check that `AssertionMapping.Confidence` is 75 for direct matches and 65 for indirect matches across all mapped assertions
- [x] T034 [US4] Add JSON output test in `internal/quality/quality_test.go` verifying the `confidence` field appears on all assertion mappings and values are within range [50, 100] (SC acceptance scenario 3)
- [x] T035 [US4] Run `go test -race -count=1 -short ./...` to verify no regressions

**Checkpoint**: Confidence differentiation verified in both internal types and JSON serialization.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Raise the ratchet baseline, verify success criteria, update documentation, run full validation

- [x] T036 Update `TestSC003_MappingAccuracy` baseline floor in `internal/quality/quality_test.go` from `70.0` to `76.0` to reflect the new mapping accuracy (FR-011) — measured accuracy 78.8% (52/66), floor set to 76.0% with ~3-point margin
- [ ] T037 Verify SC-001: Run `gaze quality --format=json` across packages with tests and compute the weighted average contract coverage — target >= 80% per SC-001 [DEFERRED: requires built binary and real package analysis; tracked as post-merge validation]
- [x] T038 Verify SC-003: Run `go test -race -count=1 -run TestSC003_MappingAccuracy ./internal/quality/...` and confirm accuracy >= 76% with the new baseline floor passing
- [x] T039 Verify SC-005: Run benchmarks `go test -race -count=1 -bench BenchmarkMapAssertions -benchmem ./internal/quality/...` and confirm per-pair processing time is within 2x of pre-change baseline — verified: ~1.03ms/op, 50KB/op, 506 allocs/op on Apple M1
- [x] T040 [P] Update GoDoc comments on `matchAssertionToEffect` in `internal/quality/mapping.go` to document the two-pass matching strategy, confidence values, and the `resolveExprRoot` function
- [x] T041 [P] Update GoDoc comments on `traceReturnValues` in `internal/quality/mapping.go` to document the helper return value fallback path
- [x] T042 Run the full test suite `go test -race -count=1 -short ./...` to confirm all packages pass with no regressions (FR-012, SC-003)
- [x] T043 Run `golangci-lint run` to verify no lint violations introduced
- [ ] T044 Run quickstart.md validation steps: mapping accuracy test, quality on multilib fixture, before/after on real packages (crap, classify, docscan), confidence values in JSON output [DEFERRED: requires built binary and real package analysis; tracked as post-merge validation]

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup (T001-T002 provide test fixtures) — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Foundational (resolveExprRoot) — MVP delivery
- **US2 (Phase 4)**: Depends on Foundational (resolveExprRoot) + US1 (two-pass strategy) — can start after T007
- **US5 (Phase 5)**: Depends on Foundational (resolveExprRoot) — can start after Phase 2, independent of US2
- **US3 (Phase 6)**: Depends on Foundational (resolveExprRoot) + US1 (two-pass strategy) — can start after T007
- **US4 (Phase 7)**: Depends on US1 + US5 (confidence values are set during implementation) — verification only
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational (Phase 2) — No dependencies on other stories
- **US2 (P2)**: Can start after US1 T007 (uses the same two-pass strategy) — Independent of US5
- **US5 (P2)**: Can start after Foundational (Phase 2) — Independent of US1/US2/US3 (modifies `traceReturnValues`, not `matchAssertionToEffect`)
- **US3 (P3)**: Can start after US1 T007 (uses the same two-pass strategy) — Independent of US2/US5
- **US4 (P4)**: Verification only — depends on US1 and US5 being complete

### Within Each User Story

- Foundational `resolveExprRoot` must exist before any story's implementation
- Implementation before tests (in this case, since the function being modified is existing code)
- Story-specific tests verify the implementation
- Regression suite run at end of each story

### Parallel Opportunities

- **T001 + T002**: Fixture source and fixture test can be written in parallel (different files)
- **US2 (Phase 4) + US5 (Phase 5)**: Can run in parallel after US1 T007 — US2 modifies no code (verification + tests), US5 modifies `traceReturnValues` (different function from US2)
- **US2 (Phase 4) + US3 (Phase 6)**: Can run in parallel — both are verification + tests for patterns already handled by `resolveExprRoot`
- **T040 + T041**: GoDoc updates are independent files/functions
- **Within Phase 2**: T004 (implementation) then T005+T006 (tests + verification) — sequential

---

## Parallel Example: After Foundational Phase

```bash
# After T007 (US1 two-pass strategy) is complete, these can run in parallel:

# Stream 1: US2 verification + tests (built-in unwinding)
Task: T013 - Verify resolveExprRoot handles len/cap
Task: T014 - Test for len() unwinding
Task: T015 - Test for cap() unwinding
Task: T016 - Negative test for side-effecting builtins
Task: T017 - Negative test for user-defined len
Task: T018 - Regression check

# Stream 2: US5 implementation (helper return tracing - different function)
Task: T019 - Create helperreturn fixture
Task: T020 - Modify traceReturnValues fallback
Task: T021 - Implement SSA call graph verification
Task: T022 - Update traceTargetValues signature
Task: T023 - Test helper return tracing
Task: T024 - Negative test for non-target helper
Task: T025 - Test fallback-only behavior
Task: T026 - Regression check

# Stream 3: US3 verification + tests (index resolution - after T007)
Task: T027 - Verify resolveExprRoot handles IndexExpr
Task: T028 - Test for index expression resolution
Task: T029 - Test for combined index + selector
Task: T030 - Negative test for non-traced index
Task: T031 - Regression check
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (create `indirectmatch` fixture)
2. Complete Phase 2: Foundational (`resolveExprRoot` function)
3. Complete Phase 3: User Story 1 (two-pass matching in `matchAssertionToEffect`)
4. **STOP and VALIDATE**: Run `gaze quality` on `multilib` — `user.Name`, `user.Email`, `user.Age` should now map
5. Selector matching alone delivers value for `multilib`, `crap`, `classify`, `config` packages

### Incremental Delivery

1. Setup + Foundational → `resolveExprRoot` ready
2. Add US1 (Selector) → Test independently → Selector patterns map at confidence 65 (MVP)
3. Add US2 (Built-in) + US5 (Helper) in parallel → `len()` and helper indirection patterns map
4. Add US3 (Index) → `results[0].Field` combined patterns map
5. Verify US4 (Confidence) → JSON output shows 75 vs 65 differentiation
6. Polish → Ratchet raised, benchmarks verified, docs updated
7. Each story adds coverage without breaking previous mappings (FR-012)

### Critical Path

```
T001-T002 → T004-T006 → T007 → { T013-T018 | T019-T026 | T027-T031 } → T032-T035 → T036-T044
  Setup       Foundation    US1      US2 | US5 | US3 (parallel)            US4          Polish
```

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- US4 (Confidence Differentiation) is implemented as part of US1 and US5 — Phase 7 is verification only
- US2 and US3 are largely verification that the foundational `resolveExprRoot` + US1's two-pass strategy already handles their patterns — minimal new code expected
- All changes confined to `internal/quality/mapping.go` and `internal/quality/quality_test.go` plus test fixtures
- T022 is required: `traceTargetValues` does not currently receive `testFunc` or `targetFunc`, which are needed for helper return tracing (SSA call graph verification)
- The `indirectmatch` fixture serves US1, US2, US3, and US4 — shared test infrastructure
- The `helperreturn` fixture serves US5 exclusively
- Commit after each phase checkpoint for clean git history
