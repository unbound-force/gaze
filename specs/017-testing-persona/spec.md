# Feature Specification: Testing Persona Integration

**Feature Branch**: `017-testing-persona`  
**Created**: 2026-03-05  
**Status**: Implemented  
**Input**: User description: "Integrate a testing persona (The Tester) into the Speckit and review council workflows, deployed via gaze init scaffold. Includes constitution amendment (Principle IV: Testability), reviewer-testing agent, /speckit.testreview command, review-council scaffolding, and scaffold system updates."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Spec Testability Review (Priority: P1)

A developer has completed the Speckit pipeline through `/speckit.tasks` and wants to validate that their spec, plan, and tasks define sufficient testability criteria before starting implementation. They invoke `/speckit.testreview` to receive a structured, read-only analysis of their spec artifacts through a testing lens — identifying vague acceptance criteria, missing coverage strategy, undefined contract surfaces, and infeasible fixture requirements.

**Why this priority**: Without testability validation at the spec stage, developers discover testing gaps only during implementation — when the cost of rework is highest. This is the core value of the testing persona.

**Independent Test**: Can be fully tested by creating a spec with known testability gaps (e.g., vague acceptance criteria, no coverage targets) and verifying that `/speckit.testreview` identifies each gap in its report with appropriate severity.

**Acceptance Scenarios**:

1. **Given** a feature directory with spec.md, plan.md, and tasks.md, **When** the user invokes `/speckit.testreview`, **Then** the system produces a structured report identifying testability findings ranked by severity (CRITICAL, HIGH, MEDIUM, LOW).
2. **Given** a spec with acceptance criteria containing vague language like "works correctly" or "handles gracefully", **When** `/speckit.testreview` runs, **Then** those criteria are flagged as HIGH-severity testability findings.
3. **Given** a plan that does not define a coverage strategy (unit vs. integration vs. e2e), **When** `/speckit.testreview` runs, **Then** the missing coverage strategy is flagged as a CRITICAL-severity finding.
4. **Given** a spec where all acceptance criteria are measurable and the plan defines a clear test strategy, **When** `/speckit.testreview` runs, **Then** the report shows no CRITICAL or HIGH findings.

---

### User Story 2 - Automated Testability Gate in PR Reviews (Priority: P1)

A developer submits a pull request and runs `/review-council` to get the governance council's verdict. The council now includes a fourth reviewer — The Tester — that audits both the code and specs for test architecture quality, assertion depth, coverage strategy, and testing convention compliance. The Tester's verdict is required alongside The Adversary, The Architect, and The Guard for the council to APPROVE.

**Why this priority**: Equal to US1 because the review council is the final quality gate before human review. Without a testing voice in the council, test quality issues pass through governance undetected.

**Independent Test**: Can be tested by submitting code with known test quality issues (e.g., tests that only check "no error", missing table-driven patterns, shared mutable state between tests) and verifying The Tester returns REQUEST CHANGES with specific findings.

**Acceptance Scenarios**:

1. **Given** a code change with tests that only assert `err == nil` without verifying return values, **When** `/review-council` runs, **Then** The Tester identifies the shallow assertion pattern and returns REQUEST CHANGES.
2. **Given** a code change with well-structured tests (table-driven, isolated, specific assertions on observable side effects), **When** `/review-council` runs, **Then** The Tester returns APPROVE.
3. **Given** a code change where The Tester returns REQUEST CHANGES, **When** the developer fixes the test quality issues and re-runs the council, **Then** The Tester returns APPROVE and the council proceeds with 4/4 approvals.
4. **Given** a spec review (Spec Review Mode), **When** `/review-council specs` runs, **Then** The Tester audits spec artifacts for testability, coverage strategy, and contract surface clarity alongside the other three reviewers.

---

### User Story 3 - Scaffold Deployment via gaze init (Priority: P2)

A developer runs `gaze init` in their Go project to set up Gaze's OpenCode integration. The scaffold now deploys 7 files instead of 4: the original gaze-reporter agent, /gaze command, and 2 reference files, plus the new reviewer-testing agent, /speckit.testreview command, and review-council command. The testing persona is immediately available for use without additional setup.

**Why this priority**: Deployment is essential for adoption but secondary to the testing persona's functionality itself. The testing persona must work correctly before it can be deployed.

**Independent Test**: Can be tested by running `gaze init` in a fresh project directory and verifying that all 7 expected files are created in the correct locations with proper ownership semantics (user-owned vs. tool-owned).

**Acceptance Scenarios**:

1. **Given** a Go project without existing `.opencode/` files, **When** the user runs `gaze init`, **Then** 7 files are created: `agents/gaze-reporter.md`, `agents/reviewer-testing.md`, `command/gaze.md`, `command/speckit.testreview.md`, `command/review-council.md`, `references/doc-scoring-model.md`, `references/example-report.md`.
2. **Given** a project where the user has customized `agents/reviewer-testing.md`, **When** the user runs `gaze init` again (without --force), **Then** the customized agent file is skipped (user-owned, skip-if-present).
3. **Given** a project where `command/speckit.testreview.md` has been modified by the user, **When** the user runs `gaze init` again (without --force), **Then** the command file is overwritten with the latest embedded version (tool-owned, overwrite-on-diff).
4. **Given** a project where `command/review-council.md` has been modified, **When** the user runs `gaze init` again (without --force), **Then** the council file is overwritten with the latest embedded version (tool-owned, overwrite-on-diff).
5. **Given** all 7 files exist with identical content to the embedded versions, **When** the user runs `gaze init`, **Then** all files are reported as skipped (no unnecessary writes).

---

### User Story 4 - Constitution Testability Principle (Priority: P2)

The project constitution is amended to include Principle IV: Testability. This principle establishes that every function Gaze analyzes and every function within Gaze itself must be testable in isolation. The constitution's existing governance mechanisms — including the Constitution Check gate in `/speckit.plan` — automatically enforce this principle for all future spec work without any changes to Speckit commands.

**Why this priority**: The constitution amendment is foundational but does not deliver user-facing functionality on its own. Its value is realized through the testing persona (US1, US2) and the Constitution Check gate that already exists in `/speckit.plan`.

**Independent Test**: Can be tested by running `/speckit.plan` on a spec that violates Principle IV (e.g., a plan with no test strategy) and verifying that the Constitution Check flags a CRITICAL violation.

**Acceptance Scenarios**:

1. **Given** the updated constitution with Principle IV, **When** `/speckit.plan` runs the Constitution Check, **Then** the check validates that the plan addresses testability requirements alongside Accuracy, Minimal Assumptions, and Actionable Output.
2. **Given** a plan that does not define coverage targets or test architecture, **When** the Constitution Check runs during `/speckit.plan`, **Then** Principle IV receives a FAIL status.
3. **Given** a plan that defines clear coverage targets, test strategy, and fixture approach, **When** the Constitution Check runs, **Then** all four principles receive PASS status.

---

### Edge Cases

- What happens when `/speckit.testreview` is invoked before `tasks.md` exists? The command uses `check-prerequisites.sh --require-tasks` and aborts with a clear error instructing the user to run `/speckit.tasks` first.
- What happens when `gaze init` is run with `--force` while the user has customized the reviewer-testing agent? The customized file is overwritten and reported as "overwritten" in the summary. The `--force` hint only counts user-owned skipped files.
- What happens when a spec has zero testability issues? The `/speckit.testreview` report shows a clean result with coverage statistics and no findings, similar to `/speckit.analyze` with zero issues.
- What happens when the review council has 3 APPROVEs and 1 REQUEST CHANGES from The Tester? The overall council verdict is REQUEST CHANGES. All four reviewers must APPROVE for the council to APPROVE.
- What happens when a tool-owned command file (`speckit.testreview.md` or `review-council.md`) has identical content to the embedded version? The file is skipped (not rewritten), same as reference files with identical content.
- How does the scaffold distinguish between user-owned and tool-owned files in the same `command/` directory? An explicit file list is used rather than directory-prefix matching, since `command/gaze.md` is user-owned while `command/speckit.testreview.md` and `command/review-council.md` are tool-owned.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `reviewer-testing` agent that audits code changes for test architecture quality, assertion depth, coverage strategy, test isolation, regression protection, and convention compliance.
- **FR-002**: System MUST provide a `reviewer-testing` agent that audits spec artifacts for testability of requirements, test strategy coverage, fixture feasibility, coverage expectations, and contract surface definition.
- **FR-003**: System MUST provide a `/speckit.testreview` command that produces a read-only, structured testability analysis report from spec, plan, and tasks artifacts.
- **FR-004**: The `/speckit.testreview` command MUST use `check-prerequisites.sh --json --require-tasks --include-tasks` to resolve artifact paths and MUST abort with an error if required artifacts are missing.
- **FR-005**: The `reviewer-testing` agent MUST operate in dual mode (Code Review Mode and Spec Review Mode), following the same pattern as the existing three reviewer agents.
- **FR-006**: The `reviewer-testing` agent MUST treat missing coverage strategy as CRITICAL severity.
- **FR-007**: The `/review-council` command MUST delegate to four reviewers in parallel: reviewer-adversary, reviewer-architect, reviewer-guard, and reviewer-testing.
- **FR-008**: The review council MUST require all four reviewers to APPROVE for the council verdict to be APPROVE.
- **FR-009**: The `gaze init` scaffold MUST deploy 7 files: 2 agents (gaze-reporter, reviewer-testing), 3 commands (gaze, speckit.testreview, review-council), and 2 references (doc-scoring-model, example-report).
- **FR-010**: The scaffold MUST classify `agents/reviewer-testing.md` as user-owned (skip-if-present behavior).
- **FR-011**: The scaffold MUST classify `command/speckit.testreview.md` and `command/review-council.md` as tool-owned (overwrite-on-diff behavior).
- **FR-012**: The scaffold MUST use an explicit file list (not directory-prefix matching) to determine tool-owned status, since the `command/` directory contains both user-owned and tool-owned files.
- **FR-013**: The project constitution MUST be amended to include Principle IV: Testability, covering both Gaze's own internal test quality and the accuracy of test quality analysis in user codebases.
- **FR-014**: The constitution version MUST be bumped from 1.0.0 to 1.1.0 (MINOR: new principle added).
- **FR-015**: The constitution's Sync Impact Report MUST document the amendment and verify template compatibility.
- **FR-016**: AGENTS.md MUST be updated to document the new agent, command, constitution principle, and scaffold changes.
- **FR-017**: No existing Speckit commands (speckit.specify, speckit.clarify, speckit.plan, speckit.tasks, speckit.analyze, speckit.checklist, speckit.implement, speckit.taskstoissues) MUST be modified.
- **FR-018**: The embedded scaffold assets MUST be identical to the corresponding files in `.opencode/`. The existing drift detection test (`TestEmbeddedAssetsMatchSource`) MUST automatically cover the new files.

### Key Entities

- **Reviewer-Testing Agent**: A subagent definition file that audits code and spec artifacts for test quality. Operates in dual mode (Code Review / Spec Review). User-owned in the scaffold (customizable per project).
- **Speckit.testreview Command**: A slash command definition file that delegates to the reviewer-testing agent in Spec Review Mode. Tool-owned in the scaffold (standardized invocation).
- **Review-Council Command**: An orchestration command that delegates to all four reviewer agents in parallel and collects their verdicts. Tool-owned in the scaffold.
- **Constitution Principle IV**: A governance principle establishing testability requirements for both Gaze's internals and the projects Gaze analyzes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `/speckit.testreview` identifies 100% of intentionally planted testability gaps (vague acceptance criteria, missing coverage strategy, undefined contract surfaces) in a test spec with zero false negatives.
- **SC-002**: The review council runs 4 reviewers in parallel and produces a unified verdict within the same iteration/retry framework as the existing 3-reviewer council.
- **SC-003**: `gaze init` creates exactly 7 files in a fresh project with correct ownership classification (2 user-owned agents, 3 commands with mixed ownership, 2 tool-owned references).
- **SC-004**: All existing scaffold tests pass after updating expected file counts and asset paths (4 to 7 files).
- **SC-005**: The drift detection test (`TestEmbeddedAssetsMatchSource`) passes for all 7 files, confirming no drift between `.opencode/` sources and `internal/scaffold/assets/` copies.
- **SC-006**: The Constitution Check in `/speckit.plan` validates Principle IV alongside the existing three principles without any changes to the `/speckit.plan` command.
- **SC-007**: Tool-owned command files (`speckit.testreview.md`, `review-council.md`) are overwritten when content differs and skipped when identical, matching the existing reference file behavior.
- **SC-008**: User-owned agent file (`reviewer-testing.md`) is skipped when present and not overwritten, matching the existing agent file behavior.

### Assumptions

- The existing `/speckit.plan` Constitution Check is generic enough to evaluate a new principle without command modification. The check reads `.specify/memory/constitution.md` and validates the plan against all MUST statements — adding a new principle with MUST statements is sufficient.
- The existing `check-prerequisites.sh` script works for `/speckit.testreview` without modification, since it uses the same `--json --require-tasks --include-tasks` flags as `/speckit.analyze`.
- The OpenCode runtime discovers command and agent files by convention (filename-based), so adding new `.md` files to `.opencode/command/` and `.opencode/agents/` is sufficient for them to become available.
- The `gaze init` summary hint message can be updated to mention the new commands without breaking existing user expectations.
