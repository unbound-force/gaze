# Tasks: Init Update Strategy

**Input**: Design documents from `/specs/012-init-update-strategy/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: Test tasks are included because the existing test suite must be migrated (compile-breaking struct change) and new behavior requires test coverage per the project's testing conventions.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Foundational (Blocking Prerequisites)

**Purpose**: Restructure the `Result` type and add the `extractVersion()` function. These changes are required by all three user stories and must be completed first. Existing tests will break at compile time after the `Result` struct change — test migration is part of this phase.

- [x] T001 Add `extractVersion()` function to `internal/scaffold/scaffold.go` that reads the first line of a file, strips the `<!-- scaffolded by gaze ` prefix and ` -->` suffix, and returns the version string. Return empty string for missing/unparseable markers per data-model.md extractVersion() behavior table.
- [x] T002 Add unit tests for `extractVersion()` in `internal/scaffold/scaffold_test.go` covering: valid marker (`"v1.0.0"`), dev marker (`"dev"`), empty prefix/suffix (`""`), non-marker first line (`""`), and empty file (`""`). Use table-driven test style.
- [x] T003 Modify the `Result` struct in `internal/scaffold/scaffold.go`: remove `Skipped []string` field, add `Updated []string`, `UpToDate []string`, and `UpdatedFrom map[string]string` fields with GoDoc comments and JSON tags per data-model.md new schema.
- [x] T004 Migrate existing tests in `internal/scaffold/scaffold_test.go` to compile against the new `Result` struct: rename `TestRun_SkipsExisting` to `TestRun_UpToDate` and change all `result.Skipped` references to `result.UpToDate`; update assertion counts and output string checks from `"skipped:"` to `"up to date:"`. Verify tests compile and pass with the new field names (behavior still uses old skip logic at this point — will be updated in US1).

**Checkpoint**: Code compiles, `extractVersion()` is tested, existing tests pass against renamed fields. The `Run()` function still uses the old skip-on-exists logic — US1 will replace it with version-aware logic.

---

## Phase 2: User Story 1 — Seamless Upgrade After Gaze Update (Priority: P1) MVP

**Goal**: `gaze init` automatically detects outdated scaffolded files via version marker comparison and replaces them with current embedded content, without requiring `--force`.

**Independent Test**: Scaffold files with version "v1.0.0", run `gaze init` with version "v2.0.0", verify all 4 files are updated and appear in `result.Updated`. Run again with same version, verify all 4 files appear in `result.UpToDate`.

### Implementation for User Story 1

- [x] T005 [US1] Modify the `Run()` function's file-exists branch in `internal/scaffold/scaffold.go` to implement version-aware update logic per data-model.md state transition rules: when `!opts.Force` and file exists, call `extractVersion()` on the existing file; if version matches `opts.Version`, classify as `UpToDate`; if version differs (or is empty), read embedded content, write file with new marker, classify as `Updated`, and record old version in `UpdatedFrom` map. Initialize `UpdatedFrom` map at the top of `Run()` if nil.
- [x] T006 [US1] Implement dev version behavior in the `Run()` function in `internal/scaffold/scaffold.go`: when `opts.Version == "dev"` and file exists and `!opts.Force`, always treat the file as outdated regardless of on-disk marker version (per FR-008 and research R3). Classify as `Updated` with old version recorded in `UpdatedFrom`.
- [x] T007 [US1] Add `TestRun_UpdatesOutdated` in `internal/scaffold/scaffold_test.go`: scaffold with version "v1.0.0", run again with version "v2.0.0" without force; assert `len(result.Updated) == 4`, `len(result.UpToDate) == 0`, `len(result.Created) == 0`, `len(result.Overwritten) == 0`; verify each file's first line contains the v2.0.0 marker; verify `result.UpdatedFrom` maps each file to "v1.0.0".
- [x] T008 [US1] Update `TestRun_UpToDate` (renamed in T004) in `internal/scaffold/scaffold_test.go` to use the same version for both runs; assert `len(result.UpToDate) == 4`, `len(result.Updated) == 0`; verify no file content was modified (compare file contents before and after second run).
- [x] T009 [US1] Add `TestRun_DevAlwaysUpdates` in `internal/scaffold/scaffold_test.go`: scaffold with version "v1.0.0", run again with version "dev" (empty string); assert `len(result.Updated) == 4`, `len(result.UpToDate) == 0`; verify marker changes to "dev". Then run again with version "dev"; assert `len(result.Updated) == 4` (dev always updates, even dev-to-dev).
- [x] T010 [US1] Add `TestRun_MissingMarkerTreatedAsOutdated` in `internal/scaffold/scaffold_test.go`: scaffold with version "v1.0.0", then overwrite one file with content that has no version marker (plain text); run `gaze init` with version "v2.0.0"; assert that file appears in `result.Updated` with `UpdatedFrom` value of `""` (empty, indicating unknown old version); assert file now has v2.0.0 marker.

**Checkpoint**: `gaze init` correctly updates outdated files, leaves current files untouched, always updates for dev builds, and handles missing markers. All US1 acceptance scenarios pass. Run `go test -race -count=1 ./internal/scaffold/...` to confirm.

---

## Phase 3: User Story 2 — Clear Reporting of File Dispositions (Priority: P2)

**Goal**: `printSummary()` output clearly distinguishes all four file states, shows version transitions for updated files, and includes a summary count line.

**Independent Test**: Set up a mixed scenario (one file missing, one outdated, two current) and verify the output contains "created:", "updated: ... (vOLD -> vNEW)", "up to date:", and a summary count line.

### Implementation for User Story 2

- [x] T011 [US2] Update `printSummary()` in `internal/scaffold/scaffold.go` to handle the new `Result` fields: print `"  updated:     %s (%s -> %s)"` for each file in `Updated` (using `UpdatedFrom` for old version and the current version for new version); print `"  up to date:  %s"` for each file in `UpToDate`; keep existing `"  created:"` and `"  overwritten:"` formats. Remove the `"skipped:"` output path entirely.
- [x] T012 [US2] Add a summary count footer to `printSummary()` in `internal/scaffold/scaffold.go`: after listing all files, print a line like `"1 created, 2 updated, 1 up to date."` showing non-zero counts. Omit categories with zero count from the summary. Remove the old `"N files skipped (use --force to overwrite)."` hint.
- [x] T013 [US2] Update the header logic in `printSummary()` in `internal/scaffold/scaffold.go`: print `"Gaze OpenCode integration initialized:"` when any files were created, updated, or overwritten; print `"Gaze OpenCode integration already up to date:"` only when all files are `UpToDate` and nothing was created/updated/overwritten.
- [x] T014 [US2] Add `TestPrintSummary_MixedScenario` in `internal/scaffold/scaffold_test.go`: construct a `Result` with 1 created, 1 updated (with `UpdatedFrom` entry), 2 up-to-date, 0 overwritten; call `printSummary()` with a `bytes.Buffer`; assert output contains `"created:"`, `"updated:"` with version transition, `"up to date:"`, the summary count line, and the `"initialized:"` header.
- [x] T015 [US2] Add `TestPrintSummary_AllUpToDate` in `internal/scaffold/scaffold_test.go`: construct a `Result` with 0 created, 0 updated, 4 up-to-date, 0 overwritten; call `printSummary()`; assert output contains `"already up to date:"` header and does not contain `"created:"`, `"updated:"`, or `"overwritten:"`.
- [x] T016 [US2] Update all existing output assertions across `internal/scaffold/scaffold_test.go` that check for `"skipped:"` or `"use --force to overwrite"` strings — replace with the appropriate new output strings (`"up to date:"`, `"updated:"`, summary counts). Ensure no test references the removed `"skipped"` wording.

**Checkpoint**: All output scenarios produce correct, categorized output. Run `go test -race -count=1 ./internal/scaffold/...` to confirm. Visually inspect output against quickstart.md examples.

---

## Phase 4: User Story 3 — Force as an Escape Hatch (Priority: P3)

**Goal**: Verify that `--force` behavior is fully preserved after the update logic changes. Force should bypass version comparison entirely and always overwrite.

**Independent Test**: Scaffold files with current version, run `gaze init --force`, verify all files appear in `result.Overwritten` (not `Updated` or `UpToDate`).

### Implementation for User Story 3

- [x] T017 [US3] Verify the `Run()` function's force branch in `internal/scaffold/scaffold.go` is unchanged: when `opts.Force` is true, the file should be overwritten unconditionally without calling `extractVersion()`, and classified as `Overwritten`. No code changes expected — this task confirms the existing force path was not broken by US1/US2 changes.
- [x] T018 [US3] Update `TestRun_ForceOverwrites` in `internal/scaffold/scaffold_test.go` to verify that force with same version still reports `Overwritten` (not `UpToDate`). Assert `len(result.Overwritten) == 4`, `len(result.Updated) == 0`, `len(result.UpToDate) == 0`. Verify output contains `"overwritten:"`.
- [x] T019 [US3] Update `TestRunInit_ForceFlag` in `cmd/gaze/main_test.go` to cover the new four-state output: first run creates files (assert `"created:"`), second run without force shows up-to-date (assert `"up to date:"`), third run with force overwrites (assert `"overwritten:"`). Update all output string assertions to match new format.

**Checkpoint**: Force behavior is fully preserved. All existing force tests pass. Run `go test -race -count=1 ./internal/scaffold/... ./cmd/gaze/...` to confirm.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Documentation updates, full test suite validation, and GoDoc comments.

- [x] T020 [P] Update GoDoc comments on `Result` struct fields in `internal/scaffold/scaffold.go` to document the four-state model and the `UpdatedFrom` map semantics.
- [x] T021 [P] Update GoDoc comment on `Run()` function in `internal/scaffold/scaffold.go` to describe the version-aware update behavior, including dev version handling.
- [x] T022 [P] Update `README.md` if `gaze init` behavior is documented there — reflect the new auto-update behavior and remove any mention of "skipped" files.
- [x] T023 Run full test suite: `go test -race -count=1 -short ./...` to confirm no regressions across the entire project.
- [x] T024 Run linter: `golangci-lint run` to confirm no lint violations.
- [x] T025 Validate output against `specs/012-init-update-strategy/quickstart.md` examples by manually reviewing test output strings match the documented before/after examples.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No dependencies — can start immediately
- **User Story 1 (Phase 2)**: Depends on Foundational completion (T001–T004)
- **User Story 2 (Phase 3)**: Depends on User Story 1 completion (T005–T010) — needs the updated `Result` fields populated correctly before output can be formatted
- **User Story 3 (Phase 4)**: Depends on User Story 1 completion (T005–T010) — verifies force path was not broken by update logic changes
- **Polish (Phase 5)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Foundational phase only. Can start as soon as `Result` struct and `extractVersion()` are ready.
- **User Story 2 (P2)**: Depends on US1. The `printSummary()` function needs the `Updated`/`UpToDate`/`UpdatedFrom` fields to be correctly populated by `Run()` before it can format them.
- **User Story 3 (P3)**: Depends on US1. Verifies the force code path was not broken. Can run in parallel with US2 since they touch different code paths (force branch vs. output formatting).

### Within Each Phase

- Production code before tests that depend on it (within same phase)
- Tasks within a phase are sequential unless marked [P]

### Parallel Opportunities

- T001 and T002 can be done together (write function + write its tests) if using TDD approach
- T020, T021, T022 can all run in parallel (different files/sections)
- US2 (Phase 3) and US3 (Phase 4) can run in parallel after US1 completes — they modify different code paths (`printSummary` vs. force branch verification)

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Foundational (T001–T004)
2. Complete Phase 2: User Story 1 (T005–T010)
3. **STOP and VALIDATE**: Run `go test -race -count=1 ./internal/scaffold/...`
4. At this point, `gaze init` correctly auto-updates outdated files — core value delivered

### Incremental Delivery

1. Foundational → `Result` struct ready, `extractVersion()` tested
2. User Story 1 → Auto-update works → Core behavior validated (MVP)
3. User Story 2 → Output is clear and informative → UX polished
4. User Story 3 → Force path confirmed intact → Full backward compatibility
5. Polish → Docs, lint, full suite → Ship-ready

---

## Notes

- All production code changes are confined to a single file: `internal/scaffold/scaffold.go`
- Test changes span two files: `internal/scaffold/scaffold_test.go` and `cmd/gaze/main_test.go`
- The `cmd/gaze/main.go` CLI layer (`runInit`, `newInitCmd`) requires NO changes — it is a thin passthrough
- The `Skipped` field removal in T003 will cause compile errors in tests — T004 must follow immediately
- Total estimated scope: ~100 lines of production code changes, ~300 lines of test code changes/additions
