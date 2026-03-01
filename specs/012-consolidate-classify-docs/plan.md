# Implementation Plan: Consolidate /classify-docs into /gaze and Fix Formatting Fidelity

**Branch**: `012-consolidate-classify-docs` | **Date**: 2026-03-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/012-consolidate-classify-docs/spec.md`

## Summary

Remove the standalone `/classify-docs` command and `doc-classifier` agent. Inline the doc-classifier's document-signal scoring model directly into the gaze-reporter agent prompt's full mode section. Fix formatting fidelity by adding assertive emoji override language and restructuring the prompt with a sandwich pattern (compact example early, full example late). Update the scaffold system from 4 embedded files to 2, and update all associated tests.

## Technical Context

**Language/Version**: Go 1.24+ (scaffold Go code); Markdown (agent/command prompts)
**Primary Dependencies**: `embed.FS` (Go standard library), OpenCode agent runtime
**Storage**: Filesystem only (embedded assets via `embed.FS`, `.opencode/` directory)
**Testing**: Standard library `testing` package; `go test -race -count=1 ./internal/scaffold/...`
**Target Platform**: Any OS where OpenCode runs (darwin, linux)
**Project Type**: Single CLI project
**Performance Goals**: N/A — prompt files and scaffold code; no runtime performance targets
**Constraints**: Gaze-reporter prompt must remain within practical LLM context window limits after absorbing doc-classifier scoring model (~150 additional lines of prompt)
**Scale/Scope**: 6 files modified/deleted, 0 new files created

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Accuracy — PASS

This change does not affect gaze's side effect detection or analysis engine. The doc-classifier's scoring model is being relocated (from a separate agent prompt to inline within gaze-reporter), not altered. The scoring rules, signal sources, weight ranges, and thresholds are preserved exactly. No false positive or false negative risk introduced.

### II. Minimal Assumptions — PASS

The change reduces assumptions. Currently, the `/classify-docs` command assumes `go build ./cmd/gaze` will succeed (portability defect). After consolidation, the gaze-reporter's existing 3-step binary resolution (build from source → check PATH → go install) applies uniformly to all CLI invocations, including classification. No new assumptions about user environment are introduced.

### III. Actionable Output — PASS

The gaze-reporter's output contract is unchanged in substance. The formatting fidelity fix improves actionable output by ensuring the emoji-based severity indicators, section markers, and grade-to-emoji mappings render correctly. These visual signals help users quickly identify critical vs. moderate vs. safe results. The document-signal scoring continues to produce the same classification enrichment, making classification labels more actionable by incorporating project documentation context.

## Project Structure

### Documentation (this feature)

```text
specs/012-consolidate-classify-docs/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
└── quickstart.md        # Phase 1 output
```

### Source Code (repository root)

```text
Files to DELETE:
  .opencode/command/classify-docs.md                      # Live command file
  .opencode/agents/doc-classifier.md                      # Live agent file
  internal/scaffold/assets/command/classify-docs.md       # Embedded copy
  internal/scaffold/assets/agents/doc-classifier.md       # Embedded copy

Files to MODIFY:
  .opencode/agents/gaze-reporter.md                       # Inline scoring model + formatting fix
  internal/scaffold/assets/agents/gaze-reporter.md        # Embedded copy (must match live)
  internal/scaffold/scaffold_test.go                      # Update file counts 4→2

Files UNCHANGED:
  .opencode/command/gaze.md                               # Command dispatcher (FR-014)
  internal/scaffold/assets/command/gaze.md                # Embedded copy
  internal/scaffold/scaffold.go                           # No code changes needed (generic walker)
```

**Structure Decision**: The scaffold system uses a generic `fs.WalkDir` walker over the `assets/` embedded FS. Removing files from `assets/` automatically reduces the file count — no code changes to `scaffold.go` are required. Only `scaffold_test.go` needs updates because it hardcodes expected file counts and file paths.

## Constitution Re-Check (Post Phase 1 Design)

### I. Accuracy — PASS (confirmed)

The scoring model is relocated verbatim. Signal sources, weight ranges, thresholds, and contradiction penalties are unchanged. The prompt restructuring (sandwich pattern, override block) affects formatting only, not analytical correctness.

### II. Minimal Assumptions — PASS (confirmed)

The consolidation eliminates the `go build ./cmd/gaze` assumption. The inlined scoring model adds no new assumptions about the host project. The sandwich prompt structure assumes standard LLM attention patterns (primacy/recency), which is well-established.

### III. Actionable Output — PASS (confirmed)

The formatting fidelity fix directly improves actionable output. Emoji severity indicators (🔴🟡🟢) help users quickly triage recommendations. Document-enhanced classification continues to provide specific classification labels.

No violations detected. No Complexity Tracking entries required.
