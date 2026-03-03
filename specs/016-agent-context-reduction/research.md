# Research: Agent Context Reduction

**Feature**: 016-agent-context-reduction
**Date**: 2026-03-02

## Decision 1: Reference File Directory Structure

**Decision**: Store reference files under `.opencode/references/` with scaffold copies at `internal/scaffold/assets/references/`.

**Rationale**: The existing scaffold convention maps `internal/scaffold/assets/<subdir>/<file>` to `.opencode/<subdir>/<file>`. Using a `references/` subdirectory clearly separates tool-owned reference content (overwrite-on-diff) from user-customizable prompt files (skip-if-present) in `agents/` and `command/`. The directory name is descriptive without being overly specific — it can hold any on-demand reference content in the future.

**Alternatives considered**:

| Alternative | Status | Reason rejected |
|-------------|--------|-----------------|
| `.opencode/examples/` + `.opencode/references/` (two directories) | Viable but unnecessary | Both files serve the same role (externalized prompt content). Splitting by content type adds complexity without benefit. A single `references/` directory is simpler. |
| `.opencode/gaze/` (tool-namespaced) | Viable | Reasonable if multiple tools shared `.opencode/`, but currently only gaze uses it. Adds an extra path level with no benefit. |
| Root-level files (`.opencode/gaze-report-example.md`) | Rejected | Mixes reference files with agent/command directories. Harder to reason about overwrite behavior at the directory level. |

## Decision 2: File Naming

**Decision**: Use `example-report.md` for the canonical example and `doc-scoring-model.md` for the classification scoring model.

**Rationale**: Names are descriptive, concise, and follow lowercase-with-hyphens convention consistent with the existing `.opencode/` files. The `example-` prefix distinguishes the example from other reference material. The `doc-` prefix on the scoring model clarifies this is the document-enhanced classification model (vs. mechanical classification).

## Decision 3: Scaffold Overwrite-on-Diff Strategy

**Decision**: Classify embedded assets into two categories by subdirectory: "user-owned" files (`agents/`, `command/`) retain skip-if-present behavior, while "tool-owned" files (`references/`) use overwrite-on-diff behavior. The distinction is encoded via a helper function that checks the subdirectory prefix.

**Rationale**: The spec requires that reference files be overwritten when their content differs from the embedded version (FR-010), while agent and command files retain their existing skip-if-present behavior. The simplest approach is a per-subdirectory policy rather than a per-file registry:

- `agents/*` and `command/*` → skip-if-present (user may customize)
- `references/*` → overwrite-if-changed (tool-owned, not user-customizable)

This is implemented as a small change to the `fs.WalkDir` callback in `scaffold.go`: when `exists && !opts.Force`, check if the file is in a tool-owned directory. If yes, compare content and overwrite if different. If no, skip as before.

**Alternatives considered**:

| Alternative | Status | Reason rejected |
|-------------|--------|-----------------|
| `Force` flag only (existing behavior) | Insufficient | `--force` overwrites everything including user-customized agent prompts. The spec requires selective overwrite. |
| Explicit overwrite list in Options | Viable but rigid | Requires callers to enumerate which files to overwrite. Fragile when new reference files are added. Subdirectory convention is self-maintaining. |
| Content hash comparison for all files | Overkill | The comparison is only needed for tool-owned files. Comparing user-owned files would add confusing "updated" messages for intentionally customized files. |

## Decision 4: Prompt Instruction Wording for On-Demand Reads

**Decision**: Add a `## Reference Files` section to the agent prompt with explicit "MUST read" instructions, specifying the relative path from the project root.

**Rationale**: The agent prompt needs to tell the LLM exactly where to find the externalized content and when to read it. The instruction should:

1. Use the project root-relative path (`.opencode/references/example-report.md`) so the agent can construct the absolute path using its working directory.
2. Be unambiguous about timing: "Read before producing your first report" for the example, "Read only in full mode when docscan returns documents" for the scoring model.
3. Include a fallback instruction for missing files (use Quick Reference Example inline, warn with `> ⚠️`).

The instruction replaces the removed content at roughly the same location in the prompt, so the agent encounters it at the appropriate point in its instruction processing.

## Decision 5: Canonical Example Extraction Boundaries

**Decision**: Extract the entire `## Example Output` section (lines 382-453 of the current prompt), including the prose framing ("Below is a concrete example...") and the full markdown code block. The replacement instruction references the external file.

**Rationale**: The prose framing is tightly coupled to the example — it sets expectations about how to use it ("Adapt the data to the actual project — do not copy these specific numbers"). Moving both keeps the external file self-contained. The Quick Reference Example (lines 34-59) remains inline in the prompt as the primary formatting guide.

## Decision 6: Scoring Model Extraction Boundaries

**Decision**: Extract the `### Document-Enhanced Classification` subsection from within `## Full Mode` (lines 201-250 of the current prompt). Replace with a 3-line instruction directing the agent to read the reference file.

**Rationale**: This subsection is entirely self-contained — it defines signal sources, weights, inference patterns, contradiction penalties, and thresholds. The surrounding Full Mode instructions reference it by section name ("see the Document-Enhanced Classification section below"), so the replacement instruction maintains the same reference point. The `> ⚠️ No documentation found` fallback instruction stays inline (it's only 2 lines and provides the agent's behavior when docscan fails, which is needed even without the scoring model).

## Decision 7: Quadrant Label Deduplication Location

**Decision**: Remove the explicit quadrant label listing from the CRAP Mode section (lines 123-127). Replace with: "Use the quadrant labels shown in the Quick Reference Example above."

**Rationale**: The labels appear in three places: Quick Reference Example (lines 52-55), CRAP Mode (lines 123-127), and Emoji Vocabulary table (lines 296-299). Removing from CRAP Mode is optimal because:

- The Quick Reference Example is the first thing the agent sees — highest attention weight.
- The Emoji Vocabulary table is a formal reference with clear Role/Usage columns — removing from there would lose structured context.
- The CRAP Mode listing is the least structured — it's a sub-bullet list within a numbered list. Replacing with a cross-reference is natural and low-risk.
