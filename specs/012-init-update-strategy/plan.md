# Implementation Plan: Init Update Strategy

**Branch**: `012-init-update-strategy` | **Date**: 2026-03-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/012-init-update-strategy/spec.md`

## Summary

`gaze init` currently has a binary file handling strategy: skip existing files (default) or unconditionally overwrite everything (`--force`). This feature adds version-aware update logic so that `gaze init` automatically detects outdated scaffolded files via their version marker comment and replaces them with current content — without requiring `--force`. The `Result` struct and `printSummary` output are extended to distinguish four file dispositions: created, updated, up to date, and overwritten.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: Standard library only (`embed`, `io/fs`, `os`, `strings`, `fmt`). No new dependencies required.
**Storage**: Filesystem only (`.opencode/` directory tree)
**Testing**: Standard library `testing` package. Existing test suite in `internal/scaffold/scaffold_test.go` and `cmd/gaze/main_test.go`.
**Target Platform**: Cross-platform (darwin/linux x amd64/arm64, built via GoReleaser)
**Project Type**: Single binary CLI
**Performance Goals**: N/A — operates on 4 small files; performance is not a concern.
**Constraints**: All changes confined to `internal/scaffold/` and `cmd/gaze/`. No new packages. No new dependencies.
**Scale/Scope**: 4 embedded asset files, ~200 lines of production code to modify, ~400 lines of test code to modify/add.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Accuracy — PASS

This feature does not change side effect detection, assertion mapping, or CRAP scoring. It modifies the `gaze init` scaffolding command only. The version marker comparison (string equality) is deterministic and produces no false positives or negatives. Accuracy of the update detection is verified by automated tests covering all four file states (created, updated, up to date, overwritten) plus edge cases (missing marker, dev version).

### II. Minimal Assumptions — PASS

The feature introduces one new assumption: that the first line of a scaffolded file contains a version marker in the format `<!-- scaffolded by gaze VERSION -->`. This assumption is:
- Already established by the existing `versionMarker()` function (not new).
- Explicitly documented in the spec's Assumptions section.
- Enforced at a single parse point (the new `extractVersion()` function).
- Gracefully degraded when violated (missing/unparseable marker → treat as outdated).

No new assumptions about the host project's language, test framework, or coding style are introduced.

### III. Actionable Output — PASS

The feature improves actionable output by:
- Replacing the uninformative "skipped (already exists)" message with version-aware dispositions.
- Adding version transition details to updated files (e.g., "v1.0.0 -> v2.0.0").
- Providing a summary count by category so users know exactly what changed and why.
- Guiding users toward `--force` only when appropriate (not as the default suggestion for all existing files).

**Gate result: All three principles PASS. Proceeding to Phase 0.**

## Project Structure

### Documentation (this feature)

```text
specs/012-init-update-strategy/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── checklists/
    └── requirements.md  # Spec quality checklist
```

### Source Code (repository root)

```text
internal/scaffold/
├── scaffold.go          # Production code: Run(), Result, printSummary(), extractVersion() [NEW]
└── scaffold_test.go     # Tests: existing + new update/up-to-date/dev/edge-case tests

cmd/gaze/
├── main.go              # CLI layer: runInit(), newInitCmd() (no changes expected)
└── main_test.go         # Integration tests: TestRunInit_* (update existing tests)
```

**Structure Decision**: All changes are confined to the existing `internal/scaffold/` package plus test updates in `cmd/gaze/main_test.go`. No new packages or files are needed. The scaffold package already owns all the relevant logic; the CLI layer (`runInit`) is a thin passthrough that requires no modification.
