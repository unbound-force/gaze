# Tasks: Agent Context Reduction

**Input**: Design documents from `/specs/016-agent-context-reduction/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: No automated test tasks for the prompt changes — formatting fidelity is verified manually via `/gaze` invocations (see quickstart.md). Scaffold logic changes require Go test updates (included as implementation tasks).

**Organization**: Tasks are grouped by user story. US1 and US2 modify the agent prompt and create reference files. US3 is a small prompt edit. Scaffold logic (Phase 2) is a blocking prerequisite since reference files must be scaffoldable before verifying the full workflow.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

```text
.opencode/agents/gaze-reporter.md           # Agent prompt (live copy)
.opencode/references/example-report.md      # NEW: canonical example
.opencode/references/doc-scoring-model.md   # NEW: scoring model
internal/scaffold/assets/                    # Embedded scaffold copies
internal/scaffold/scaffold.go               # Scaffold logic
internal/scaffold/scaffold_test.go          # Scaffold tests
cmd/gaze/main_test.go                       # CLI tests
```

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: No project initialization needed — this feature modifies existing files and adds new ones within the existing project structure.

*Phase 1 is empty for this feature. Proceed directly to Phase 2.*

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Update scaffold logic to support overwrite-on-diff for tool-owned reference files. This must be complete before reference files can be properly scaffolded and tested.

- [x] T001 Add an `Updated` field (`[]string`) to the `Result` struct in internal/scaffold/scaffold.go. This field tracks tool-owned files that were overwritten because their content differed from the embedded version.
- [x] T002 Add a `isToolOwned(relPath string) bool` helper function in internal/scaffold/scaffold.go that returns `true` when the relative path starts with `"references/"`. This determines which files use overwrite-on-diff vs skip-if-present behavior.
- [x] T003 Update the `fs.WalkDir` callback in the `Run` function in internal/scaffold/scaffold.go to implement overwrite-on-diff: when a file exists, `Force` is false, and `isToolOwned` returns true, read the existing file content, compare with the embedded content (after marker insertion), and overwrite if different (appending to `result.Updated`), or skip if identical (appending to `result.Skipped`). The existing skip-if-present path for user-owned files remains unchanged.
- [x] T004 Update the summary output section of the `Run` function in internal/scaffold/scaffold.go to print `Updated` files (e.g., `"  updated: %s (content changed)\n"`).

**Checkpoint**: Scaffold logic supports two file categories. Reference files will be auto-updated on `gaze init` when content changes. Agent/command files retain skip-if-present behavior.

---

## Phase 3: User Story 1 — On-Demand Canonical Example (Priority: P1) MVP

**Goal**: Extract the canonical example output from the agent prompt into `.opencode/references/example-report.md` and replace inline content with a read instruction.

**Independent Test**: Run `wc -c .opencode/agents/gaze-reporter.md` and verify the file is smaller by ~3,100 bytes. Run `diff` between `.opencode/references/example-report.md` and internal/scaffold/assets/references/example-report.md to confirm sync.

### Implementation for User Story 1

- [x] T005 [US1] Create .opencode/references/example-report.md by extracting the entire `## Example Output` section (lines 382-453) from .opencode/agents/gaze-reporter.md. Include both the prose framing paragraph ("Below is a concrete example...") and the full markdown code block. The file should be self-contained — readable without context from the agent prompt.
- [x] T006 [US1] Remove the `## Example Output` section (lines 382-453) from .opencode/agents/gaze-reporter.md. In its place, add a `## Reference Files` section with the following instruction: "Before producing your first report, read the formatting reference from `.opencode/references/example-report.md` using the Read tool. This file contains the definitive example of the expected output format. If the file cannot be read, use the Quick Reference Example above as your formatting guide and include: `> ⚠️ Could not load full formatting reference.`"
- [x] T007 [P] [US1] Create internal/scaffold/assets/references/example-report.md as a byte-identical copy of .opencode/references/example-report.md.
- [x] T008 [US1] Copy .opencode/agents/gaze-reporter.md to internal/scaffold/assets/agents/gaze-reporter.md to keep the scaffold copy in sync after the example removal.

**Checkpoint**: The canonical example is externalized. The prompt is ~3,100 bytes smaller. The scaffold includes the reference file. Formatting instructions reference the external file.

---

## Phase 4: User Story 2 — On-Demand Scoring Model (Priority: P2)

**Goal**: Extract the Document-Enhanced Classification scoring model from the agent prompt into `.opencode/references/doc-scoring-model.md` and replace inline content with a conditional read instruction.

**Independent Test**: Run `wc -c .opencode/agents/gaze-reporter.md` and verify the file is ≤13,300 bytes total. Run `diff` between `.opencode/references/doc-scoring-model.md` and its scaffold copy.

### Implementation for User Story 2

- [x] T009 [US2] Create .opencode/references/doc-scoring-model.md by extracting the `### Document-Enhanced Classification` subsection (lines 201-250, adjusted for T006 edits) from .opencode/agents/gaze-reporter.md. Include the Document Signal Sources table, AI Inference Signals table, Contradiction Penalty rule, and Classification Thresholds table. Keep the fallback warning instruction (`> ⚠️ No documentation found — classification uses mechanical signals only.`) inline in the prompt.
- [x] T010 [US2] Remove the `### Document-Enhanced Classification` subsection from the `## Full Mode` section in .opencode/agents/gaze-reporter.md. Replace with a 3-line instruction: "If `gaze docscan` returns documentation files, read the document-enhanced classification scoring model from `.opencode/references/doc-scoring-model.md` using the Read tool. Apply the signal weights, thresholds, and contradiction penalties defined there. If the file cannot be read, skip document-enhanced scoring and use mechanical-only classification."
- [x] T011 [P] [US2] Create internal/scaffold/assets/references/doc-scoring-model.md as a byte-identical copy of .opencode/references/doc-scoring-model.md.
- [x] T012 [US2] Copy .opencode/agents/gaze-reporter.md to internal/scaffold/assets/agents/gaze-reporter.md to keep the scaffold copy in sync after the scoring model removal.

**Checkpoint**: The scoring model is externalized. The prompt is ≤13,300 bytes. Full-mode runs will read the scoring model on demand. CRAP and quality modes never load it.

---

## Phase 5: User Story 3 — Deduplicated Quadrant Labels (Priority: P3)

**Goal**: Remove one of three redundant quadrant label listings from the agent prompt.

**Independent Test**: Verify quadrant labels still appear in the Quick Reference Example (lines 52-55) and Emoji Vocabulary table (lines 296-299, adjusted). Verify the CRAP Mode section now has a cross-reference instead of explicit labels.

### Implementation for User Story 3

- [x] T013 [US3] In .opencode/agents/gaze-reporter.md, locate the CRAP Mode section's quadrant label listing (4 lines defining Q1-Q4 with emojis, currently around lines 123-127 adjusted for prior edits). Replace these 4 lines with: "Use the quadrant labels shown in the Quick Reference Example above (🟢 Q1 — Safe, 🟡 Q2 — Complex But Tested, 🔴 Q4 — Dangerous, ⚪ Q3 — Needs Tests)."
- [x] T014 [US3] Copy .opencode/agents/gaze-reporter.md to internal/scaffold/assets/agents/gaze-reporter.md to keep the scaffold copy in sync.

**Checkpoint**: Quadrant labels appear in exactly 2 locations (Quick Reference Example + Emoji Vocabulary). CRAP Mode section uses a cross-reference. ~200 bytes saved.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Update tests, documentation, and validate the complete implementation.

- [x] T015 Update all hardcoded file count assertions in internal/scaffold/scaffold_test.go from `2` to `4` (in TestRun_CreatesFiles, TestRun_SkipsExisting, TestRun_ForceOverwrites, TestRun_NoGoMod_PrintsWarning, TestAssetPaths_Returns2Files). Update expected file lists and maps to include `references/example-report.md` and `references/doc-scoring-model.md`.
- [x] T016 Add a new test `TestRun_OverwriteOnDiff_ReferencesOnly` in internal/scaffold/scaffold_test.go that verifies: (a) first run creates all 4 files, (b) second run without `--force` skips user-owned files but also skips unchanged reference files, (c) after modifying a reference file on disk, third run without `--force` overwrites the changed reference file (appears in `result.Updated`) while still skipping user-owned files.
- [x] T017 Add a new test `TestRun_OverwriteOnDiff_SkipsIdentical` in internal/scaffold/scaffold_test.go that verifies: when reference files exist with identical content to the embedded version, they appear in `result.Skipped` (not `Updated`).
- [x] T018 Update expected file counts and file lists in cmd/gaze/main_test.go (TestRunInit_CreatesFiles and TestRunInit_ForceFlag) from 2 to 4 files, adding the two reference file paths.
- [x] T019 [P] Verify prompt size meets SC-001 by running `wc -c .opencode/agents/gaze-reporter.md` and confirming ≤13,300 bytes.
- [x] T020 [P] Verify scaffold sync meets SC-006 by running `diff` between all `.opencode/` files and their `internal/scaffold/assets/` counterparts.
- [x] T021 Run `go test -race -count=1 -short ./...` to verify all tests pass after changes.
- [x] T022 Run `go build ./cmd/gaze` to verify the binary builds with the new embedded assets.
- [x] T023 [P] Update AGENTS.md "Recent Changes" section to add an entry for 016-agent-context-reduction documenting the prompt size reduction and scaffold overwrite-on-diff behavior.
- [x] T024 [P] Update AGENTS.md "Spec Organization" section to add the 016-agent-context-reduction entry if missing.

**Checkpoint**: All tests pass, documentation updated, prompt size verified. Feature is ready for manual formatting verification via `/gaze` commands.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Empty — no setup tasks
- **Foundational (Phase 2)**: T001-T004 are sequential — each builds on the previous within scaffold.go. MUST complete before US1 work to ensure reference files can be scaffolded correctly.
- **User Story 1 (Phase 3)**: Depends on Phase 2 (scaffold must support references/). T005-T008: T005 creates the reference file, T006 updates the prompt (can follow T005), T007 can run in parallel with T006 (different files), T008 depends on T006.
- **User Story 2 (Phase 4)**: Depends on US1 (T006 adds the Reference Files section that T010 extends). T009-T012 follow the same pattern as US1.
- **User Story 3 (Phase 5)**: Depends on US2 (T013 modifies the prompt after US2 edits are complete). T013-T014 are sequential.
- **Polish (Phase 6)**: T015-T018 depend on all prompt and scaffold changes being complete. T019-T024 are validation and can run after T015-T018.

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2 (scaffold overwrite-on-diff) — core extraction
- **User Story 2 (P2)**: Depends on US1 (Reference Files section must exist before adding scoring model instruction)
- **User Story 3 (P3)**: Depends on US2 (prompt must be in final form before dedup to get correct line references)

### Parallel Opportunities

- T007 can run in parallel with T006 (different files: scaffold copy vs prompt edit)
- T011 can run in parallel with T010 (different files: scaffold copy vs prompt edit)
- T019, T020, T023, T024 can all run in parallel (independent validation and documentation tasks)

---

## Parallel Example: User Story 1

```bash
# After T005 (reference file created), launch in parallel:
Task: "Remove Example Output section from agent prompt (T006)"
Task: "Create scaffold copy of example-report.md (T007)"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 2: Scaffold overwrite-on-diff (T001-T004)
2. Complete Phase 3: Extract canonical example (T005-T008)
3. **STOP and VALIDATE**: Verify prompt size reduced by ~3,100 bytes, scaffold sync passes
4. If formatting fidelity confirmed via manual test, proceed to US2

### Incremental Delivery

1. T001-T004 → Scaffold supports tool-owned files
2. T005-T008 → Canonical example externalized (~3,100 bytes saved)
3. T009-T012 → Scoring model externalized (~2,100 bytes saved)
4. T013-T014 → Quadrant labels deduplicated (~200 bytes saved)
5. T015-T024 → Tests, validation, documentation

### Full Verification

1. Run `wc -c .opencode/agents/gaze-reporter.md` → ≤13,300 bytes
2. Run `diff` on all 4 file pairs → byte-identical
3. Run `go test -race -count=1 -short ./...` → all pass
4. Run `go build ./cmd/gaze` → builds cleanly
5. Run `/gaze crap ./...` in OpenCode → correct formatting
6. Run `/gaze` (full mode) in OpenCode → correct classification

---

## Notes

- This feature modifies 1 Go source file (`scaffold.go`), 2 Go test files, and 1 markdown prompt file. It creates 4 new markdown files (2 reference files + 2 scaffold copies).
- US2 depends on US1 because the `## Reference Files` section created by T006 is where T010 adds the scoring model instruction. Building the section incrementally avoids conflicts.
- US3 depends on US2 to avoid line-number confusion — the prompt is edited from bottom to top (example at line 382 first, scoring model at line 201 second, quadrant labels at line 123 third).
- The scaffold `TestEmbeddedAssetsMatchSource` drift detection test automatically validates sync for any new files added to `assets/` — no additional drift test is needed.
- SC-002 (no scoring model load in crap/quality modes), SC-003 (correct classification in full mode), and SC-004 (formatting fidelity) are verified manually via `/gaze` invocations per quickstart.md — these are LLM behavioral properties that cannot be validated by automated Go tests.
