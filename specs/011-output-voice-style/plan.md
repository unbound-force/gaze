# Implementation Plan: Output Voice & Style Standardization

**Branch**: `011-output-voice-style` | **Date**: 2026-03-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/011-output-voice-style/spec.md`

## Summary

Rewrite the gaze-reporter agent prompt to replace the clinical, emoji-free voice (spec 010) with a fun, emoji-rich voice standard. The voice standard defines a closed 10-emoji vocabulary, grade-to-emoji severity mappings, section marker assignments, tone anti-pattern bans, and a canonical example output. No Go source code changes are required — the implementation is entirely a markdown prompt rewrite plus documentation cleanup.

## Technical Context

**Language/Version**: Go 1.24+ (no Go code changes; prompt is markdown)
**Primary Dependencies**: OpenCode agent runtime (renders markdown prompt), embed.FS (scaffolds prompt copy)
**Storage**: Filesystem only (markdown files)
**Testing**: Manual verification via `/gaze` command in OpenCode; scaffold tests verify file existence (not content)
**Target Platform**: OpenCode agent environment (any OS)
**Project Type**: Single project (CLI tool)
**Performance Goals**: N/A (prompt rewrite, no runtime impact)
**Constraints**: Both copies of gaze-reporter.md must remain byte-identical (active + scaffolded)
**Scale/Scope**: 2 files rewritten, 1 file updated, 7 files deleted

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Research Check

| Principle | Status | Rationale |
|-----------|--------|-----------|
| I. Accuracy | PASS | This feature changes report presentation, not analysis accuracy. Side effect detection, CRAP scoring, and coverage measurement are untouched. The voice standard requires every interpretive sentence to convey factual data (FR-004, FR-005), preventing inaccurate commentary. |
| II. Minimal Assumptions | PASS | The voice standard assumes only that the rendering context supports Unicode emoji characters. This is documented in the Assumptions section. No new assumptions about host project language, test framework, or coding style are introduced. |
| III. Actionable Output | PASS | The voice standard explicitly preserves actionability: recommendations are numbered with severity markers (FR-007), interpretive sentences are limited to actionable observations (FR-004), and every success criterion requires concrete, data-backed output (SC-005). Emojis serve as visual navigation aids, not decoration. |

**Gate result**: PASS — all three principles satisfied. Proceeding to Phase 0.

### Post-Design Check

| Principle | Status | Rationale |
|-----------|--------|-----------|
| I. Accuracy | PASS | No changes to analysis logic. The closed emoji vocabulary (research R3) ensures emojis carry precise semantic meaning, not vague decoration. Grade-to-emoji mapping is deterministic (research R5). |
| II. Minimal Assumptions | PASS | Unicode emoji support is the only assumption. The prompt works with any Go project — no framework-specific behavior. |
| III. Actionable Output | PASS | The canonical example demonstrates actionable output: specific function names, concrete metrics, prioritized recommendations with severity indicators. The redesigned report structure (research R6) improves scannability without reducing information density. |

**Gate result**: PASS — design phase complete with no constitution violations.

## Project Structure

### Documentation (this feature)

```text
specs/011-output-voice-style/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: voice standard design decisions
├── data-model.md        # Phase 1: emoji vocabulary and mapping tables
├── quickstart.md        # Phase 1: verification guide
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
.opencode/agents/
└── gaze-reporter.md          # REWRITE: active agent prompt

internal/scaffold/assets/
└── agents/
    └── gaze-reporter.md      # REWRITE: scaffolded copy (byte-identical)

AGENTS.md                     # UPDATE: Recent Changes + spec listing

specs/010-report-voice-refinement/  # DELETE: entire directory (7 files)
├── spec.md
├── plan.md
├── tasks.md
├── research.md
├── data-model.md
├── quickstart.md
└── checklists/
    └── requirements.md
```

**Structure Decision**: No new directories or files created in the source tree. Two existing files are rewritten in-place, one is updated, and one directory is deleted. The project structure is unchanged.

## Implementation Approach

### Phase 1: Rebase and Clean Up Spec 010

Before any prompt changes, rebase the branch onto current `main` to pick up the spec 010 artifacts, then delete them.

1. `git fetch origin && git rebase origin/main`
2. `rm -rf specs/010-report-voice-refinement/`
3. Update `AGENTS.md`:
   - Remove `010-report-voice-refinement/` from spec listing (line 90)
   - Replace spec 010 entry in Recent Changes (line 240) with spec 011 entry

### Phase 2: Rewrite Gaze-Reporter Prompt

Rewrite the following sections of `.opencode/agents/gaze-reporter.md`:

1. **Line 20-21** (agent description): Replace "clinical diagnostic summaries — factual, terse, and emoji-free" with "fun, approachable quality summaries with emoji section markers and severity indicators"

2. **CRAP Mode section** (lines 81-91): Replace plain-text quadrant labels with emoji-prefixed labels (🟢 Q1 — Safe, 🟡 Q2 — Complex But Tested, etc.)

3. **Full Mode section** (lines 127-191):
   - Replace Overall Health Assessment structure: remove Risk Matrix and word-based Overall Grade; replace with Summary Scorecard using letter grades + emoji indicators
   - Replace "Bottom line:" paragraph with conversational closing
   - Add severity emoji prefixes to Prioritized Recommendations

4. **Output Format section** (lines 192-235): Complete rewrite:
   - Replace "clinical, matter-of-fact" with "fun, approachable, and conversational"
   - Add emoji vocabulary table (10 emojis with semantic roles)
   - Add section marker assignments
   - Add grade-to-emoji severity mapping
   - Add tone anti-pattern bans (exclamation marks, slang, puns, first-person)
   - Update title format: `🔍 Gaze Full Quality Report` (or mode variant)
   - Update metadata format: `Project: <path> · Branch: <branch>` + `Gaze Version: <ver> · Go: <ver> · Date: <date>`
   - Add warning callout format: `> ⚠️ ...`
   - Preserve interpretation limit (one sentence, max 25 words) and section omission rule

5. **Example Output section** (lines 239-306): Replace clinical example with canonical example from the spec

### Phase 3: Sync Scaffolded Copy

Copy the rewritten `.opencode/agents/gaze-reporter.md` to `internal/scaffold/assets/agents/gaze-reporter.md` to maintain byte-identity.

### Phase 4: Verification

1. Run `/gaze` in OpenCode and verify output matches canonical example structure
2. Run `/gaze crap ./...` and verify CRAP-only mode uses same voice
3. Run `/gaze quality ./...` and verify quality-only mode uses same voice
4. Verify no spec 010 artifacts remain in the repository
5. Verify AGENTS.md references spec 011
