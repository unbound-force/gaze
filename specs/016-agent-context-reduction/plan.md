# Implementation Plan: Agent Context Reduction

**Branch**: `016-agent-context-reduction` | **Date**: 2026-03-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/016-agent-context-reduction/spec.md`

## Summary

Reduce the gaze-reporter agent prompt from ~17,775 bytes to ≤13,300 bytes by externalizing two large content blocks into on-demand reference files and deduplicating repeated quadrant labels. The canonical example output (3,367 bytes) and document-enhanced classification scoring model (2,395 bytes) are moved to `.opencode/references/` and read by the agent via the Read tool only when needed. The scaffold system (`internal/scaffold/scaffold.go`) is updated to support overwrite-on-diff behavior for tool-owned reference files while preserving skip-if-present behavior for user-owned agent and command files.

## Technical Context

**Language/Version**: Go 1.24+ (scaffold Go code changes); Markdown (agent prompt and reference files)
**Primary Dependencies**: `embed.FS` (Go standard library), OpenCode agent runtime
**Storage**: Filesystem only (embedded assets via `embed.FS`, `.opencode/` directory)
**Testing**: Standard library `testing` package (`go test -race -count=1 -short ./...`)
**Target Platform**: Any platform running OpenCode with gaze installed
**Project Type**: Single CLI binary — agent prompt and scaffold changes
**Performance Goals**: Agent prompt ≤13,300 bytes (≥25% reduction from 17,775 bytes)
**Constraints**: Reference files must be readable by the gaze-reporter agent's Read tool; scaffold copies must be byte-identical to live `.opencode/` copies
**Scale/Scope**: 4 files modified (agent prompt, scaffold.go, 2 test files), 4 new files created (2 reference files + 2 scaffold copies)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Research Check

| Principle | Status | Assessment |
|-----------|--------|------------|
| **I. Accuracy** | PASS | This feature does not alter Gaze's analysis engine, side effect detection, CRAP scoring, or any analytical output. It only restructures how the reporting agent's instructions are stored and loaded. No risk of introducing false positives or false negatives. |
| **II. Minimal Assumptions** | PASS | No changes to how Gaze analyzes host projects. No new user-facing annotations or restructuring required. The agent reads reference files transparently. The only assumption is that `.opencode/references/` is readable, which is guaranteed by the scaffold system. |
| **III. Actionable Output** | PASS | Report content, format, and actionability are preserved identically. The same emoji markers, tables, recommendations, and metrics are produced. Only the storage location of the agent's formatting instructions changes. |

**Gate result**: PASS — All three principles satisfied. This feature operates entirely outside the analysis domain.

### Post-Design Check

| Principle | Status | Assessment |
|-----------|--------|------------|
| **I. Accuracy** | PASS | Design confirms: zero changes to analysis engine. Agent prompt instructions are identical in content, only relocated. Reference files contain verbatim extracts from the existing prompt. |
| **II. Minimal Assumptions** | PASS | Design confirms: no new user-facing requirements. `gaze init` automatically creates reference files. Overwrite-on-diff ensures reference files stay current without user action. Agent gracefully degrades if files are missing. |
| **III. Actionable Output** | PASS | Design confirms: SC-004 explicitly requires formatting fidelity preservation. The Quick Reference Example remains inline in the prompt as a formatting anchor. The canonical example is available on-demand for additional precision. |

**Post-design gate result**: PASS — No constitution concerns.

## Project Structure

### Documentation (this feature)

```text
specs/016-agent-context-reduction/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: Research decisions
├── data-model.md        # Phase 1: File organization model
├── quickstart.md        # Phase 1: Verification guide
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
.opencode/
├── agents/
│   └── gaze-reporter.md            # Modified: reduced prompt (~12,100 bytes)
├── command/
│   └── gaze.md                     # Unchanged
└── references/                     # NEW directory
    ├── example-report.md           # NEW: canonical example (extracted)
    └── doc-scoring-model.md        # NEW: scoring model (extracted)

internal/scaffold/
├── scaffold.go                     # Modified: overwrite-on-diff for references/
├── scaffold_test.go                # Modified: updated counts, new tests
└── assets/
    ├── agents/
    │   └── gaze-reporter.md        # Modified: sync with .opencode/ copy
    ├── command/
    │   └── gaze.md                 # Unchanged
    └── references/                 # NEW directory
        ├── example-report.md       # NEW: scaffold copy
        └── doc-scoring-model.md    # NEW: scaffold copy

cmd/gaze/
└── main_test.go                    # Modified: updated expected file counts
```

**Structure Decision**: No new Go source files. The feature adds 2 markdown reference files (with scaffold copies) and modifies 4 existing files (agent prompt, scaffold.go, 2 test files). The `references/` subdirectory under `.opencode/` and `internal/scaffold/assets/` is the only new directory.

## Complexity Tracking

> No Constitution Check violations. No complexity justification needed.
