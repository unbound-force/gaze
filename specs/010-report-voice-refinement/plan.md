# Implementation Plan: Report Voice Refinement

**Branch**: `010-report-voice-refinement` | **Date**: 2026-02-28 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/010-report-voice-refinement/spec.md`

## Summary

Rewrite the Output Format section and mode-specific formatting instructions in the gaze-reporter agent prompt to produce clinical, emoji-free reports with concise structure, word-based grades, right-aligned numerics, a prominent risk matrix, and a "Bottom line:" closing paragraph. The change surface is a single markdown file maintained in two synchronized locations. No production Go code is modified.

## Technical Context

**Language/Version**: Markdown (agent prompt file) — no compiled code changes
**Primary Dependencies**: OpenCode agent framework (reads `.opencode/agents/*.md` as agent definitions)
**Storage**: N/A — file-based prompt, no database or persistence
**Testing**: Manual verification by running `/gaze` on Go projects; visual inspection of output against spec acceptance scenarios
**Target Platform**: Any platform running OpenCode with the gaze-reporter agent
**Project Type**: Single project — prompt file modification only
**Performance Goals**: N/A — prompt file change has no runtime performance impact
**Constraints**: Both copies of the agent prompt (`.opencode/agents/gaze-reporter.md` and `internal/scaffold/assets/agents/gaze-reporter.md`) must remain byte-identical (FR-015, SC-006)
**Scale/Scope**: Single file (~170 lines), two locations

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Accuracy — PASS

The agent prompt refinement does not alter what data gaze reports — it changes how that data is formatted and presented. The factual content (CRAP scores, complexity, coverage percentages, quadrant assignments) remains derived from the same JSON output. The spec requires terse, data-packed interpretations (FR-013, FR-014), which reduces the risk of the agent fabricating or misrepresenting metrics.

### II. Minimal Assumptions — PASS

The change adds no new assumptions about the host project. The prompt continues to operate on `gaze` JSON output. The formatting instructions (right-aligned numerics, word-based grades, risk matrix structure) are presentation-layer concerns that do not require the user to annotate, restructure, or change anything about their project.

### III. Actionable Output — PASS

The spec directly enhances actionable output:
- Risk matrix with Priority/Function/Risk/Why columns (FR-010) makes the highest-priority action immediately visible.
- "Bottom line:" paragraph (FR-011) gives the developer a single-sentence directive.
- Recommendations as action sentences with concrete metrics (FR-012, US5) tell the developer exactly what to fix and what the target is.
- Omitting empty sections (FR-005) removes noise that obscures actionable content.

No violations. No complexity tracking needed.

## Project Structure

### Documentation (this feature)

```text
specs/010-report-voice-refinement/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
.opencode/agents/
└── gaze-reporter.md              # Primary agent prompt (live runtime)

internal/scaffold/assets/agents/
└── gaze-reporter.md              # Embedded scaffold copy (distributed by gaze init)
```

**Structure Decision**: No new files or directories. The change modifies a single existing file in two locations. No contracts/ directory is needed — there are no API endpoints or programmatic interfaces.

## Implementation Groups

### Group A: Rewrite Output Format and Tone Instructions

**Files**: `.opencode/agents/gaze-reporter.md`

Replace the current `## Output Format` section (lines 158-166) with a comprehensive formatting specification that includes:

1. **Tone directive** — Explicit instruction for clinical, matter-of-fact voice. No emojis, no pedagogical explanations, no filler paragraphs. Every sentence conveys data or an actionable observation.

2. **Title and metadata format** — Single-line title: `Gaze Health Report — <project-name>`. Single-line metadata: `Package: <pattern> | Date: <date>`.

3. **Table formatting rules** — Right-aligned numeric columns using `|------:|` syntax. Concise metric labels ("Functions analyzed" not "Total functions analyzed").

4. **Interpretation rules** — Single terse sentence after each data table (max 25 words). No multi-paragraph explanations.

5. **Section omission rule** — If a gaze command returns no data or fails, omit that section entirely. No placeholder headers, blockquotes, or "N/A" content. A single-line note at the end lists unavailable analyses.

### Group B: Update Mode-Specific Formatting Instructions

**Files**: `.opencode/agents/gaze-reporter.md`

Update CRAP Mode (lines 54-77), Quality Mode (lines 79-100), and Full Mode (lines 101-132) sections to align with the new voice:

1. **CRAP Mode** — Update metric labels to concise forms. Add CRAPload percentage format instruction. Remove emoji-prone quadrant label instructions. Add instruction for terse summary sentence at end.

2. **Quality Mode** — Remove the blockquote fallback instruction (line 97-99). Replace with: omit the section entirely if quality data is unavailable.

3. **Full Mode** — Restructure Overall Health Assessment to specify:
   - Risk matrix as first table (Priority/Function/Risk/Why columns)
   - Overall Grade table with word-based ratings (Poor/Fair/Good/Strong/Excellent)
   - Prioritized recommendations as numbered action sentences (no emoji prefixes)
   - "Bottom line:" closing paragraph (1-3 sentences)

### Group C: Add Example Output Snippet

**Files**: `.opencode/agents/gaze-reporter.md`

Add a new `## Example Output` section (after Output Format, before Graceful Degradation) containing a concrete example showing:

1. Title line format
2. Metadata line format
3. CRAP Summary table with right-aligned numerics
4. One-sentence interpretation
5. Risk matrix table with Priority/Function/Risk/Why columns
6. Overall Grade table with word-based ratings
7. Bottom line paragraph

This example serves as the definitive formatting reference for the agent (FR-016, SC-007).

### Group D: Synchronize Scaffold Copy

**Files**: `internal/scaffold/assets/agents/gaze-reporter.md`

After Groups A-C are complete, copy the updated `.opencode/agents/gaze-reporter.md` to `internal/scaffold/assets/agents/gaze-reporter.md` byte-for-byte. Verify byte-identity (FR-015, SC-006).

### Group E: Update Graceful Degradation

**Files**: `.opencode/agents/gaze-reporter.md`

Update the Graceful Degradation section (lines 134-142) to align with the omission-over-placeholder rule:

- Replace "Note which sections are missing and why" with: "Append a single-line note at the end of the report listing which analyses were unavailable."
- Remove any instruction that could produce blockquotes or warning banners for missing data.

## Dependency Order

```text
Group A (Output Format) → Group B (Mode Updates) → Group C (Example) → Group E (Degradation) → Group D (Sync)
```

Groups A and B could be done in parallel since they touch different sections, but sequential execution is safer since the tone directive in A informs the specific instructions in B. Group C depends on A+B (needs the final format rules to write the example). Group E depends on A (needs the omission rule). Group D is always last (synchronization gate).

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Agent ignores formatting instructions in prompt | Medium | High | Include a concrete example output (Group C) — models are better at following examples than abstract rules |
| Tone instructions are too restrictive, producing robotic output | Low | Medium | Use "clinical, matter-of-fact" rather than "no personality" — allow the agent to make observations, just tersely |
| Scaffold copy diverges from live copy | Low | High | Group D is a mandatory gate — byte-identical copy verified before task completion |
| Existing graceful degradation behavior breaks | Low | Medium | Group E explicitly preserves the partial-report behavior, only changing how missing sections are communicated |
