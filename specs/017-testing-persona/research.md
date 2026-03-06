# Research: Testing Persona Integration

**Feature**: 017-testing-persona
**Date**: 2026-03-05

## Decision 1: Reviewer Agent Structure

**Decision**: Follow the exact structural pattern of `reviewer-adversary.md` for the new `reviewer-testing.md`.

**Rationale**: All three existing reviewer agents share a consistent structure that the review council and OpenCode runtime expect. Diverging from this pattern would create unnecessary inconsistency and potential integration issues.

**Structure to replicate**:
1. YAML frontmatter (`mode: subagent`, `model: claude-sonnet-4-6`, `temperature: 0.1`, read-only tools)
2. `# Role: The Tester` with gaze project description and dual-mode notice
3. `## Source Documents` with numbered file list
4. `## Code Review Mode` with `### Review Scope` and `### Audit Checklist` (numbered H4 sections)
5. `## Spec Review Mode` with same subsection pattern
6. `## Output Format` with finding template
7. `## Decision Criteria` with APPROVE/REQUEST CHANGES conditions

**Non-obvious details**:
- Output Format field names are NOT uniform across reviewers: adversary uses "Constraint", architect uses "Convention", guard uses "Spec Reference" + "Constraint" (5 fields). The Tester should use field names appropriate to its domain (e.g., "Test Quality Dimension").
- The architect has an extra "Alignment Score" metric. The Tester could similarly have a "Test Adequacy Score" but this is optional — the spec does not require it.
- Source Documents section lists different AGENTS.md subsections per reviewer. The Tester should reference "Testing Conventions" specifically.

**Alternatives considered**:
- Creating a lighter-weight agent without dual-mode support: rejected because the review council delegates in both modes and expects all reviewers to handle both.
- Using a different model or temperature: rejected for consistency with the existing council.

## Decision 2: Review Council Integration Points

**Decision**: Add `reviewer-testing` at the same level as the existing three reviewers in both Code Review Mode and Spec Review Mode Step 1.

**Rationale**: The review council's loop structure (Steps 2-5/6) is agent-count-agnostic — it references "all three" which becomes "all four", but the logic of "collect → fix → re-run" works identically regardless of count.

**Specific text changes required**:
- Update description and Description section: "three-reviewer" → "four-reviewer"
- Code Review Mode Step 1: add 4th bullet with `reviewer-testing` and test-focused audit summary
- Spec Review Mode Step 1: add 4th bullet with `reviewer-testing` and testability audit summary
- Steps 2/3 in both modes: "three reviewers" → "four reviewers" (multiple occurrences)
- Verdict section: "all three" → "all four"

**Non-obvious details**:
- The Spec Review Mode has a **hybrid fix policy** (auto-fix LOW/MEDIUM, report-only HIGH/CRITICAL). The Tester's findings need to fit this framework. Testability findings like "vague acceptance criteria" would be MEDIUM (auto-fixable by clarifying language), while "missing coverage strategy" would be CRITICAL (report-only, requires human decision).
- The review-council.md is NOT currently scaffolded. Adding it to scaffold (as FR-009 requires) means it becomes a distributed file, so the version users get via `gaze init` must be self-contained and not reference project-specific paths.

## Decision 3: `isToolOwned` Extension Strategy

**Decision**: Use an explicit file list combining the existing prefix check for `references/` with exact-match checks for specific command files.

**Rationale**: The `command/` directory now contains both user-owned (`gaze.md`) and tool-owned (`speckit.testreview.md`, `review-council.md`) files. A simple directory-prefix approach cannot express this mixed ownership. An explicit list is more precise and self-documenting.

**Implementation**:
```go
func isToolOwned(relPath string) bool {
    if strings.HasPrefix(relPath, "references/") {
        return true
    }
    switch relPath {
    case "command/speckit.testreview.md", "command/review-council.md":
        return true
    }
    return false
}
```

**Alternatives considered**:
- Pure map-based lookup: slightly more allocation but functionally equivalent. The switch statement is idiomatic Go for a small fixed set and avoids allocating a map on every call.
- Naming convention (e.g., tool-owned files start with "speckit."): too fragile and creates invisible coupling between file naming and ownership semantics.
- Metadata in YAML frontmatter: would require parsing YAML at scaffold time, adding complexity and a new dependency.

**Non-obvious details**:
- The `references/` prefix check is retained (not converted to exact paths) because the spec 016 research specifically chose directory-based ownership for this subdirectory, and all files in `references/` are tool-owned by convention.
- Adding new tool-owned command files in the future requires adding a case to the switch statement. This is intentional — it forces an explicit decision about ownership for each new file.

## Decision 4: Scaffold File Inventory and Ownership

**Decision**: Scaffold 7 files total with the following ownership classification:

| File | Ownership | Behavior |
|------|-----------|----------|
| `agents/gaze-reporter.md` | User-owned | skip-if-present |
| `agents/reviewer-testing.md` | User-owned | skip-if-present |
| `command/gaze.md` | User-owned | skip-if-present |
| `command/speckit.testreview.md` | Tool-owned | overwrite-on-diff |
| `command/review-council.md` | Tool-owned | overwrite-on-diff |
| `references/doc-scoring-model.md` | Tool-owned | overwrite-on-diff |
| `references/example-report.md` | Tool-owned | overwrite-on-diff |

**Rationale**: The agent file is user-owned because projects may want to customize The Tester's audit checklist for their specific testing conventions (different frameworks, coverage thresholds). The command files are tool-owned because their invocation protocol must remain standardized — the review council's orchestration logic and the testreview's prerequisites check should not be user-modified.

**Precedent note**: The existing 3 reviewer agents (`reviewer-adversary.md`, `reviewer-architect.md`, `reviewer-guard.md`) are NOT scaffolded — they exist only in the Gaze project's `.opencode/` directory. Adding `reviewer-testing.md` to the scaffold is a deliberate choice to make the testing persona available to all `gaze init` consumers, unlike the project-specific review council agents.

**Non-obvious details**:
- The `--force` hint in `printSummary` counts only user-owned skipped files. With 3 user-owned files (2 agents + 1 command), the maximum user-skipped count is 3.
- `TestEmbeddedAssetsMatchSource` enforces byte-identical content between `internal/scaffold/assets/X` and `.opencode/X` for all embedded files. The review-council.md in scaffold must match the `.opencode/command/review-council.md` exactly.

## Decision 5: `/speckit.testreview` Command Structure

**Decision**: Model the command after `/speckit.analyze` — read-only analysis with structured report output — but delegate to the `reviewer-testing` agent rather than performing inline analysis.

**Rationale**: `/speckit.analyze` performs its own inline analysis across 6 detection categories. `/speckit.testreview` has a narrower focus (testability only) and the analysis logic is already defined in the `reviewer-testing` agent's Spec Review Mode checklist. Delegation avoids duplicating the audit logic between the command and agent.

**Key differences from `/speckit.analyze`**:
- Uses `check-prerequisites.sh --json --require-tasks --include-tasks` (same flags)
- Delegates to `reviewer-testing` agent via Task tool rather than performing inline analysis
- Report format follows the reviewer's output format (findings with severity/file/description/recommendation)
- Includes the same "Next Actions" and remediation offer pattern
- Does NOT include the "Build Semantic Models" or "Detection Passes" stages — those are agent-internal

**Non-obvious details**:
- The command file needs to instruct the delegate agent to operate in Spec Review Mode explicitly
- The command should load and pass artifact paths to the agent, not expect the agent to discover them independently
- The `$ARGUMENTS` placeholder appears at top and bottom of `/speckit.analyze` — replicate this for consistency

## Decision 6: Constitution Amendment Scope

**Decision**: Principle IV: Testability covers both Gaze's own internal test quality AND the accuracy of test quality analysis in user codebases.

**Rationale**: Gaze is a test quality tool. If Gaze's own tests are poorly structured, it undermines the credibility of Gaze's test quality assessments on user code. Conversely, if Gaze accurately assesses user test quality, that accuracy must be maintained as a constitutional principle — it's a direct extension of Principle I (Accuracy) applied specifically to the testing domain.

**MUST statements for Principle IV**:
1. Every function Gaze analyzes, and every function within Gaze itself, MUST be testable in isolation without requiring external services or shared mutable state.
2. Test contracts MUST verify observable side effects (return values, state mutations, I/O operations), not implementation details.
3. Coverage strategy (unit vs. integration vs. e2e, with targets) MUST be specified in the plan for all new code.
4. Coverage ratchets MUST be enforced by automated tests; coverage regression MUST be treated as a test failure.

**Version bump**: 1.0.0 → 1.1.0 (MINOR per constitution versioning rules: new principle added, no existing principles altered or removed).

**Sync Impact Report**: All existing templates are generic and do not reference specific principles by number — no template updates needed. This matches the precedent set by the initial ratification.
