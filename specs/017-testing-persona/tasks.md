# Tasks: Testing Persona Integration

**Input**: Design documents from `/specs/017-testing-persona/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, quickstart.md

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

---

## Phase 1: Foundational (Constitution Amendment)

**Purpose**: Amend the constitution with Principle IV: Testability before creating the agent that references it.

**⚠️ CRITICAL**: The reviewer-testing agent references Principle IV in its audit checklist. The constitution must be amended first.

- [x] T001 [US4] Add Principle IV: Testability to `.specify/memory/constitution.md` after Principle III: Actionable Output. Include 4 MUST statements covering: isolation testability, contract-based assertions, coverage strategy specification, and ratchet enforcement. Add rationale paragraph. Dual scope: Gaze's own internal test quality AND accuracy of test quality analysis in user codebases.
- [x] T002 [US4] Update Sync Impact Report (HTML comment at top of `.specify/memory/constitution.md`): document the amendment (1.0.0 → 1.1.0), list the new principle, verify template compatibility (all templates are generic — no changes needed), update version line at bottom to `1.1.0`, update Last Amended date to today.

**Checkpoint**: Constitution amended. `/speckit.plan` Constitution Check will now validate Principle IV for future specs.

---

## Phase 2: User Story 1 + User Story 2 — The Tester Agent and Commands (Priority: P1)

**Goal**: Create the reviewer-testing agent, the /speckit.testreview command, and update the review-council command. US1 and US2 share the same agent and are implemented together since the agent serves both the standalone command (US1) and the council integration (US2).

**Independent Test (US1)**: Invoke `/speckit.testreview` on an existing spec and verify structured testability report output.
**Independent Test (US2)**: Invoke `/review-council` and verify 4 reviewers dispatched in parallel with The Tester included.

### Reviewer-Testing Agent

- [x] T003 [P] [US1] Create `.opencode/agents/reviewer-testing.md` with YAML frontmatter: `description` (1-line role summary), `mode: subagent`, `model: google-vertex-anthropic/claude-sonnet-4-6@default`, `temperature: 0.1`, tools (`write: false`, `edit: false`, `bash: false`). Follow the exact frontmatter pattern from `.opencode/agents/reviewer-adversary.md`.
- [x] T004 [US1] Add `# Role: The Tester` section to `.opencode/agents/reviewer-testing.md`: 1-paragraph role description including gaze project summary ("a Go static analysis tool that detects observable side effects..."), "Your job is to..." mission statement focused on test quality and testability auditing, bold dual-mode notice.
- [x] T005 [US1] Add `## Source Documents` section to `.opencode/agents/reviewer-testing.md`: numbered list — (1) `AGENTS.md` — Testing Conventions, Coding Conventions, (2) `.specify/memory/constitution.md` — Core Principles (especially Principle IV), (3) relevant spec/plan/tasks under `specs/`.
- [x] T006 [US1] Add `## Code Review Mode` section to `.opencode/agents/reviewer-testing.md` with `### Review Scope` (evaluate recent changes via git diff/status) and `### Audit Checklist` with 6 numbered H4 subsections: (1) Test Architecture — table-driven tests, fixture isolation, standard `testing` package, `TestXxx_Description` naming; (2) Coverage Strategy — contract surface coverage (returns, mutations, side effects), not just line coverage; (3) Assertion Depth — specific behavioral assertions, not just `err == nil`; (4) Test Isolation — no shared mutable state, no ordering dependencies, no external network access; (5) Regression Protection — tests lock down spec-critical behavior; (6) Convention Compliance — `-race -count=1`, `testing.Short()` guards for slow tests, benchmark separation.
- [x] T007 [US1] Add `## Spec Review Mode` section to `.opencode/agents/reviewer-testing.md` with `### Review Scope` (read all files under `specs/` recursively, plus constitution and AGENTS.md; do NOT use git diff) and `### Audit Checklist` with 6 numbered H4 subsections: (1) Testability of Requirements — can every acceptance criterion be objectively verified? Flag vague language; (2) Test Strategy Coverage — unit vs. integration vs. e2e defined in plan?; (3) Fixture Feasibility — testdata packages realistic and mentioned?; (4) Coverage Expectations — ratchet targets specified for new code?; (5) Contract Surface Definition — observable side effects clear enough to write contract tests?; (6) Constitution Alignment — Principle IV compliance.
- [x] T008 [US1] Add `## Output Format` section to `.opencode/agents/reviewer-testing.md`: markdown code block showing finding template with fields: SEVERITY, File, Test Quality Dimension, Description, Recommendation. List severity levels: CRITICAL, HIGH, MEDIUM, LOW. Add severity guidance: missing coverage strategy → CRITICAL; vague acceptance criteria → HIGH; missing fixture specification → MEDIUM; minor convention deviation → LOW.
- [x] T009 [US1] Add `## Decision Criteria` section to `.opencode/agents/reviewer-testing.md`: APPROVE if tests are well-structured, coverage strategy is sound, and conventions are followed. REQUEST CHANGES if any test quality issue at MEDIUM severity or above. End with "clear APPROVE or REQUEST CHANGES verdict and a summary of findings."

### Speckit.testreview Command

- [x] T010 [P] [US1] Create `.opencode/command/speckit.testreview.md` with YAML frontmatter (`description` only, no agent delegation). Add `## User Input` section with `$ARGUMENTS` block. Add `## Goal` section: read-only testability analysis of spec artifacts after task generation.
- [x] T011 [US1] Add `## Operating Constraints` section to `.opencode/command/speckit.testreview.md`: STRICTLY READ-ONLY (bold), Constitution Authority referencing Principle IV as non-negotiable within analysis scope.
- [x] T012 [US1] Add `## Execution Steps` to `.opencode/command/speckit.testreview.md` with numbered steps: (1) Initialize Analysis Context — run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`, parse FEATURE_DIR, derive SPEC/PLAN/TASKS paths, abort if missing; (2) Load Artifacts — extract relevant sections from spec.md, plan.md, tasks.md using progressive disclosure; (3) Load constitution for Principle IV validation; (4) Delegate to `reviewer-testing` agent in Spec Review Mode via Task tool, passing artifact paths and instructing Spec Review Mode explicitly; (5) Collect agent findings and format report.
- [x] T013 [US1] Add report structure and next actions to `.opencode/command/speckit.testreview.md`: (6) Produce Compact Testability Report — findings table (ID, Category, Severity, Location, Summary, Recommendation), Coverage Summary, Metrics block; (7) Provide Next Actions — conditional recommendations based on severity, command suggestions for `/speckit.clarify` or `/speckit.plan`; (8) Offer Remediation — ask user, do NOT auto-apply. Add `## Context` section with `$ARGUMENTS` reference at bottom.

### Review Council Update

- [x] T014 [US2] Update `.opencode/command/review-council.md` description (line 2): change "three-reviewer" to "four-reviewer" in the YAML frontmatter description.
- [x] T015 [US2] Update `.opencode/command/review-council.md` Description section (line 15): add "The Tester" to the reviewer list alongside The Adversary, The Architect, The Guard.
- [x] T016 [US2] Update `.opencode/command/review-council.md` Code Review Mode Step 1 (line ~32-36): add 4th bullet `- \`reviewer-testing\` — audits for test architecture, coverage strategy, assertion quality, and testing convention compliance`. Update "all three" → "all four" in Steps 1, 2, 3.
- [x] T017 [US2] Update `.opencode/command/review-council.md` Spec Review Mode Step 1 (line ~59-64): add 4th bullet `- \`reviewer-testing\` — audits specs for testability of requirements, coverage strategy definition, fixture feasibility, and contract surface clarity`. Update "three reviewers" → "four reviewers" in Steps 1, 2, 3, 4.
- [x] T018 [US2] Update `.opencode/command/review-council.md` Verdict section (line ~102-104): change "all three reviewers" to "all four reviewers". Verify the APPROVE WITH ADVISORIES wording still applies.

**Checkpoint**: The Tester agent is operational. `/speckit.testreview` produces testability reports. `/review-council` dispatches 4 reviewers.

---

## Phase 3: User Story 3 — Scaffold Deployment (Priority: P2)

**Goal**: Deploy the testing persona files via `gaze init` with correct ownership semantics.

**Independent Test**: Run `gaze init` in a temp directory and verify 7 files created with correct ownership.

### Scaffold Go Code Changes

- [x] T019 [US3] Update `isToolOwned` function in `internal/scaffold/scaffold.go` (line ~65-67): replace single `strings.HasPrefix(relPath, "references/")` with combined check — retain `references/` prefix, add switch statement for `"command/speckit.testreview.md"` and `"command/review-council.md"` exact matches. Update GoDoc comment to reflect explicit file list approach.
- [x] T020 [US3] Update `printSummary` hint in `internal/scaffold/scaffold.go` (line ~277): change "Run /gaze in OpenCode to generate quality reports." to mention `/speckit.testreview` for testability reviews and `/review-council` for governance reviews.

### Embedded Asset Copies

- [x] T021 [P] [US3] Copy `.opencode/agents/reviewer-testing.md` to `internal/scaffold/assets/agents/reviewer-testing.md` (byte-identical).
- [x] T022 [P] [US3] Copy `.opencode/command/speckit.testreview.md` to `internal/scaffold/assets/command/speckit.testreview.md` (byte-identical).
- [x] T023 [P] [US3] Copy `.opencode/command/review-council.md` to `internal/scaffold/assets/command/review-council.md` (byte-identical).

### Scaffold Test Updates

- [x] T024 [US3] Update `TestRun_CreatesFiles` in `internal/scaffold/scaffold_test.go`: change expected created count from 4 to 7, add 3 new paths to `expected` slice (`agents/reviewer-testing.md`, `command/speckit.testreview.md`, `command/review-council.md`).
- [x] T025 [US3] Update `TestRun_SkipsExisting` in `internal/scaffold/scaffold_test.go`: change expected skipped count from 4 to 7 on second run. Update `--force` hint assertion — user-owned skipped count is now 3 (2 agents + 1 command).
- [x] T026 [US3] Update `TestRun_ForceOverwrites` in `internal/scaffold/scaffold_test.go`: change expected overwritten count from 4 to 7.
- [x] T027 [US3] Rename `TestAssetPaths_Returns4Files` to `TestAssetPaths_Returns7Files` in `internal/scaffold/scaffold_test.go`: update expected map to include all 7 asset paths, update count assertion from 4 to 7.
- [x] T028 [US3] Update `TestRun_NoGoMod_PrintsWarning` in `internal/scaffold/scaffold_test.go`: change expected created count from 4 to 7.
- [x] T029 [US3] Update `TestEmbeddedAssetsMatchSource` in `internal/scaffold/scaffold_test.go`: change expected asset count from 4 to 7.
- [x] T030 [US3] Update `TestRun_OverwriteOnDiff_ReferencesOnly` in `internal/scaffold/scaffold_test.go`: rename to `TestRun_OverwriteOnDiff_ToolOwned`. Add scenario: modify `command/speckit.testreview.md` on disk → re-run without --force → verify it appears in `result.Updated`. Update expected skipped/updated counts to account for 7 files (3 user-owned skipped + 3 tool-owned identical skipped + 1 tool-owned modified updated). Update user-skipped hint assertion.
- [x] T031 [US3] Update `TestRun_OverwriteOnDiff_SkipsIdentical` in `internal/scaffold/scaffold_test.go`: change expected skipped count from 4 to 7. Add tool-owned command files to the verification list alongside reference files.
- [x] T032 [US3] Add `TestIsToolOwned` in `internal/scaffold/scaffold_test.go`: test that `isToolOwned` returns true for `references/doc-scoring-model.md`, `references/example-report.md`, `command/speckit.testreview.md`, `command/review-council.md`; returns false for `agents/gaze-reporter.md`, `agents/reviewer-testing.md`, `command/gaze.md`.
- [x] T033 [US3] Run `go test -race -count=1 ./internal/scaffold/...` and verify all tests pass.

**Checkpoint**: `gaze init` deploys 7 files with correct ownership. All scaffold tests pass.

---

## Phase 4: Polish & Documentation

**Purpose**: Update project documentation to reflect all changes.

- [x] T034 [P] Update `AGENTS.md` Council Governance Protocol section: add `- **The Tester**: Must verify that test quality, coverage strategy, and testability are maintained.` alongside The Architect, The Adversary, The Guard. Update the Rule to reference four reviewers.
- [x] T035 [P] Update `AGENTS.md` Core Principles section: add Principle IV: Testability summary alongside Accuracy, Minimal Assumptions, and Actionable Output.
- [x] T036 Update `AGENTS.md` Recent Changes section: add 017-testing-persona entry documenting — constitution Principle IV, reviewer-testing agent, /speckit.testreview command, review-council 4th reviewer, scaffold 4→7 files with mixed ownership via explicit isToolOwned file list.
- [x] T037 Run `go test -race -count=1 -short ./...` to verify no regressions across the full test suite.
- [x] T038 Run `golangci-lint run` to verify lint passes.

**Checkpoint**: All documentation updated. Full test suite passes. Lint clean.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Constitution)**: No dependencies — start immediately
- **Phase 2 (Agent + Commands)**: Depends on Phase 1 (agent references Principle IV)
- **Phase 3 (Scaffold)**: Depends on Phase 2 (scaffold embeds the agent and command files)
- **Phase 4 (Documentation)**: Depends on Phases 1-3 (documents all changes)

### User Story Dependencies

- **US4 (Constitution)**: No dependencies — foundational
- **US1 (Spec Testability Review)**: Depends on US4 (Principle IV) — agent + command files
- **US2 (PR Review Council)**: Depends on US4 (Principle IV) — council update + agent (shared with US1)
- **US3 (Scaffold Deployment)**: Depends on US1 + US2 (embeds the completed files)

### Within Each Phase

- Tasks marked [P] can run in parallel
- T003-T009 (agent sections) must be sequential (building the same file)
- T010-T013 (command sections) must be sequential (building the same file)
- T014-T018 (council updates) must be sequential (editing the same file)
- T021-T023 (asset copies) can run in parallel
- T024-T032 (test updates) can run in parallel (different test functions, same file but independent edits)

### Parallel Opportunities

```
Phase 1: T001 → T002 (sequential, same file)

Phase 2: 
  Agent (T003-T009) || Command (T010-T013)  → then Council (T014-T018)
  
Phase 3:
  Go code (T019-T020) || Asset copies (T021-T023)  → then Tests (T024-T033)

Phase 4:
  T034 || T035  → T036 → T037 || T038
```

---

## Implementation Strategy

### MVP First (US1 + US4 Only)

1. Complete Phase 1: Constitution amendment
2. Complete T003-T013: Agent + command for `/speckit.testreview`
3. **STOP and VALIDATE**: Test `/speckit.testreview` on an existing spec
4. This gives a working testability review without council integration or scaffold deployment

### Incremental Delivery

1. Phase 1 (Constitution) → Principle IV active
2. Phase 2 (Agent + Commands) → `/speckit.testreview` + council operational
3. Phase 3 (Scaffold) → deployable via `gaze init`
4. Phase 4 (Documentation) → fully documented
5. Each phase adds value without breaking previous phases

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- The agent file (T003-T009) is the largest single deliverable (~200 lines of markdown)
- The review-council.md edit (T014-T018) is surgical — changing "three" to "four" and adding 4th delegation bullets
- Scaffold test updates (T024-T032) are mechanical — updating hardcoded counts from 4 to 7
- No existing Speckit commands are modified (FR-017 constraint)
