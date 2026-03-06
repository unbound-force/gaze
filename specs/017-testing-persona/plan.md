# Implementation Plan: Testing Persona Integration

**Branch**: `017-testing-persona` | **Date**: 2026-03-05 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/017-testing-persona/spec.md`

## Summary

Integrate a testing persona ("The Tester") into the Gaze project by creating a `reviewer-testing` agent, a `/speckit.testreview` command, and scaffolding the `/review-council` command for deployment via `gaze init`. Amend the constitution with Principle IV: Testability. Update the scaffold system to support mixed ownership within the `command/` directory using an explicit tool-owned file list.

## Technical Context

**Language/Version**: Go 1.24+ (scaffold Go code changes); Markdown (agent/command prompts)
**Primary Dependencies**: `embed.FS` (Go standard library), OpenCode agent runtime
**Storage**: Filesystem only (embedded assets via `embed.FS`, `.opencode/` directory)
**Testing**: Standard `testing` package, `-race -count=1`
**Target Platform**: Cross-platform (Go CLI + OpenCode integration files)
**Project Type**: Single project — existing `internal/scaffold/` package
**Performance Goals**: N/A — scaffold runs once per project, agent prompts are static markdown
**Constraints**: No existing Speckit commands modified (FR-017); scaffold must handle mixed ownership in same directory
**Scale/Scope**: 3 new markdown files (agent + command + council), Go scaffold changes in 2 files (scaffold.go + scaffold_test.go), 1 constitution edit, 1 AGENTS.md update

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Accuracy — PASS

This feature does not alter Gaze's side effect detection, CRAP score computation, or test quality analysis engines. The testing persona reviews spec artifacts and code for testability — it is an AI-generated advisory report, not an automated measurement. No accuracy claims are made by the testing persona; it provides heuristic testability feedback.

The scaffold changes (file count, ownership classification) are deterministic and verified by existing automated tests that will be updated to cover the new file set.

### II. Minimal Assumptions — PASS

The testing persona makes no assumptions about the host project's language, test framework, or coding style beyond what the existing reviewer agents already assume (Go project with `AGENTS.md` and `.specify/memory/constitution.md`). The scaffold deploys markdown files that work with any OpenCode-compatible project.

The `isToolOwned` change uses an explicit file list rather than convention-based heuristics, minimizing assumptions about future file naming patterns.

### III. Actionable Output — PASS

The `/speckit.testreview` command produces a structured report with severity-ranked findings, each containing file location, description, and recommendation — following the same format as the existing reviewer agents. Every finding guides the user toward a specific improvement in their spec or test strategy.

The review council's 4-reviewer verdict continues to use the same APPROVE/REQUEST CHANGES format with specific findings.

## Project Structure

### Documentation (this feature)

```text
specs/017-testing-persona/
├── plan.md              # This file
├── research.md          # Phase 0: existing patterns analysis
├── data-model.md        # Phase 1: file inventory and ownership model
├── quickstart.md        # Phase 1: verification guide
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
.opencode/
├── agents/
│   ├── gaze-reporter.md          # Existing (unchanged)
│   ├── reviewer-adversary.md     # Existing (unchanged)
│   ├── reviewer-architect.md     # Existing (unchanged)
│   ├── reviewer-guard.md         # Existing (unchanged)
│   └── reviewer-testing.md       # NEW: The Tester agent
├── command/
│   ├── gaze.md                   # Existing (unchanged)
│   ├── review-council.md         # MODIFIED: add 4th reviewer
│   ├── speckit.analyze.md        # Existing (unchanged)
│   ├── speckit.testreview.md     # NEW: testability review command
│   └── [other speckit.*.md]      # Existing (unchanged)
└── references/
    ├── doc-scoring-model.md      # Existing (unchanged)
    └── example-report.md         # Existing (unchanged)

internal/scaffold/
├── assets/
│   ├── agents/
│   │   ├── gaze-reporter.md      # Existing (unchanged)
│   │   └── reviewer-testing.md   # NEW: embedded copy of agent
│   ├── command/
│   │   ├── gaze.md               # Existing (unchanged)
│   │   ├── speckit.testreview.md  # NEW: embedded copy of command
│   │   └── review-council.md     # NEW: embedded copy of council
│   └── references/
│       ├── doc-scoring-model.md   # Existing (unchanged)
│       └── example-report.md      # Existing (unchanged)
├── scaffold.go                    # MODIFIED: isToolOwned explicit list
└── scaffold_test.go               # MODIFIED: 4→7 file count assertions

.specify/memory/
└── constitution.md                # MODIFIED: add Principle IV

AGENTS.md                          # MODIFIED: document new components
```

**Structure Decision**: This feature extends the existing `internal/scaffold/` package and `.opencode/` directory structure. No new packages or directories are created beyond the embedded asset copies. The approach mirrors specs 005 (initial scaffold), 012 (scaffold reduction), and 016 (scaffold expansion with overwrite-on-diff).

## Phased Implementation

### Phase 1: Constitution Amendment

Amend `.specify/memory/constitution.md`:

1. Add Principle IV: Testability after Principle III: Actionable Output
2. Dual scope: Gaze's own internal test quality AND accuracy of test quality analysis in user codebases
3. MUST statements covering: isolation testability, contract-based assertions, coverage strategy specification, ratchet enforcement
4. Rationale explaining why testability is a first-class governance concern
5. Update Sync Impact Report (HTML comment) documenting the amendment
6. Bump version: 1.0.0 → 1.1.0 (MINOR: new principle, no existing principles altered)
7. Update Last Amended date
8. Verify template compatibility (all existing templates are generic — no template changes needed)

### Phase 2: Agent and Command Files

**reviewer-testing.md** (`.opencode/agents/`):

1. YAML frontmatter: `mode: subagent`, `model: claude-sonnet-4-6`, `temperature: 0.1`, read-only tools
2. Role description: "The Tester" — test quality and testability auditor for the Gaze project
3. Source Documents section: AGENTS.md, constitution (including Principle IV), relevant spec artifacts
4. Code Review Mode with audit checklist:
   - Test Architecture (table-driven, fixture isolation, standard testing package, naming)
   - Coverage Strategy (contract surface coverage, not just line coverage)
   - Assertion Depth (specific behavioral assertions, not just "no error")
   - Test Isolation (no shared mutable state, no ordering dependencies, no external network)
   - Regression Protection (tests lock down spec-critical behavior)
   - Convention Compliance (`-race -count=1`, `testing.Short()` guards, `TestXxx_Description`)
5. Spec Review Mode with audit checklist:
   - Testability of Requirements (measurable acceptance criteria, no vague language)
   - Test Strategy Coverage (unit vs. integration vs. e2e defined in plan)
   - Fixture Feasibility (testdata packages realistic and mentioned)
   - Coverage Expectations (ratchet targets specified for new code)
   - Contract Surface Definition (observable side effects clear enough for contract tests)
   - Constitution Alignment (Principle IV compliance)
6. Severity guidance: missing coverage strategy → CRITICAL; vague acceptance criteria → HIGH
7. Output format matching existing reviewers (SEVERITY, File, Constraint, Description, Recommendation)
8. Decision criteria: APPROVE/REQUEST CHANGES with same thresholds as other reviewers

**speckit.testreview.md** (`.opencode/command/`):

1. YAML frontmatter: `description` only (no `agent:` delegation — command orchestrates directly)
2. Read-only operating constraint (same as `/speckit.analyze`)
3. Execution steps:
   - Run `check-prerequisites.sh --json --require-tasks --include-tasks`
   - Load spec.md, plan.md, tasks.md from FEATURE_DIR
   - Load constitution for Principle IV validation
   - Delegate to `reviewer-testing` agent in Spec Review Mode via Task tool
   - Collect and format findings report
4. Report structure: findings table (ID, Category, Severity, Location, Summary, Recommendation)
5. Coverage summary table and metrics
6. Next actions block with command suggestions

**review-council.md** (`.opencode/command/` — modify existing):

1. Add `reviewer-testing` as 4th parallel delegation in Code Review Mode (line ~33)
2. Add `reviewer-testing` as 4th parallel delegation in Spec Review Mode (line ~59)
3. Update description text to mention "four-reviewer" (was "three-reviewer")
4. Update verdict section to reference "all four reviewers"

### Phase 3: Scaffold System Changes

**scaffold.go** — `isToolOwned` function:

1. Replace `strings.HasPrefix(relPath, "references/")` with an explicit set/map lookup
2. Tool-owned paths: `"references/"` (prefix match retained for directory), `"command/speckit.testreview.md"` (exact match), `"command/review-council.md"` (exact match)
3. Implementation: check prefix for `references/`, then check exact match for specific command files
4. No changes to `Run()`, `printSummary()`, or other functions — the ownership logic is fully encapsulated in `isToolOwned`

**scaffold.go** — `printSummary` hint:

1. Update the hint message to mention both `/gaze` and `/speckit.testreview` commands
2. Mention `/review-council` for governance reviews

**Embedded assets** (`internal/scaffold/assets/`):

1. Copy `.opencode/agents/reviewer-testing.md` → `assets/agents/reviewer-testing.md`
2. Copy `.opencode/command/speckit.testreview.md` → `assets/command/speckit.testreview.md`
3. Copy `.opencode/command/review-council.md` → `assets/command/review-council.md`

**scaffold_test.go** — update all test assertions:

1. `TestRun_CreatesFiles`: 4 → 7 created files, update expected paths list
2. `TestRun_SkipsExisting`: 4 → 7 skipped files (second run with same version)
3. `TestRun_ForceOverwrites`: 4 → 7 overwritten files
4. `TestAssetPaths_Returns4Files`: Rename to `TestAssetPaths_Returns7Files`, update expected map
5. `TestRun_VersionMarker`: No changes needed (iterates all asset paths dynamically)
6. `TestRun_VersionMarker_Dev`: No changes needed (iterates dynamically)
7. `TestRun_NoGoMod_PrintsWarning`: 4 → 7 created files
8. `TestEmbeddedAssetsMatchSource`: 4 → 7 expected assets
9. `TestRun_OverwriteOnDiff_ReferencesOnly`: Expand to cover tool-owned command files:
   - Add scenario: modify `command/speckit.testreview.md` on disk → re-run → verify Updated
   - Update expected skipped/updated counts
   - Possibly rename to `TestRun_OverwriteOnDiff_ToolOwned`
10. `TestRun_OverwriteOnDiff_SkipsIdentical`: 4 → 7 skipped, verify tool-owned command files in skipped list
11. Add `TestIsToolOwned` to verify the explicit file list covers all expected paths

**printSummary user-skipped count**:

The `printSummary` function counts user-owned skipped files for the `--force` hint. With the new mixed ownership model, this logic works correctly because `isToolOwned` already governs the count. User-owned files: `agents/gaze-reporter.md`, `agents/reviewer-testing.md`, `command/gaze.md` (3 total). Tool-owned files: `references/doc-scoring-model.md`, `references/example-report.md`, `command/speckit.testreview.md`, `command/review-council.md` (4 total). The `--force` hint appears only when user-owned files are skipped.

### Phase 4: Documentation

**AGENTS.md** updates:

1. Add `reviewer-testing` to Council Governance Protocol section
2. Add `/speckit.testreview` to the command table or description
3. Update Core Principles summary to include Principle IV: Testability
4. Add scaffold file count note in Recent Changes (4 → 7 files)
5. Document the `isToolOwned` expansion and mixed ownership model
6. Update the agent descriptions in the AGENTS.md header to include `reviewer-testing`

## Testing Strategy

### Unit Tests (Phase 3)

All scaffold changes are covered by existing test patterns:

- **File count assertions**: Every test that checks `len(result.Created)`, `len(result.Skipped)`, etc. updated from 4 to 7
- **Asset path manifest**: `TestAssetPaths_Returns7Files` verifies all 7 expected paths
- **Drift detection**: `TestEmbeddedAssetsMatchSource` automatically covers new files (iterates `assetPaths()`)
- **Ownership behavior**: `TestRun_OverwriteOnDiff_ToolOwned` verifies tool-owned command files get overwrite-on-diff treatment
- **isToolOwned unit test**: New `TestIsToolOwned` verifies explicit file list correctness

### Integration Tests (implicit)

- Running `gaze init` in a temp directory verifies end-to-end scaffold behavior
- The existing `TestRun_*` tests ARE integration tests (they create temp dirs, run the full scaffold, verify output)

### Manual Verification

- Invoke `/speckit.testreview` on an existing spec to verify report output
- Invoke `/review-council` to verify 4-reviewer parallel execution
- Run `gaze init` in a fresh project to verify all 7 files deploy correctly

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| `isToolOwned` explicit list becomes stale when new files added | LOW | Drift detection test catches mismatches; adding a file to assets/ without updating isToolOwned causes test failure |
| Review council latency increases with 4th reviewer | LOW | Reviewers run in parallel; wall-clock time bounded by slowest reviewer, not sum |
| Constitution Principle IV enforcement too strict for some specs | MEDIUM | Severity is CRITICAL only for missing coverage strategy; other testability gaps are HIGH/MEDIUM and don't auto-block |
| Users customize review-council.md then lose changes on gaze init | LOW | Expected behavior for tool-owned files; documented in spec edge cases; --force hint guides users |
