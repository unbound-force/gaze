# Implementation Plan: CRAPload Reduction

**Branch**: `009-crapload-reduction` | **Date**: 2026-02-27 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/009-crapload-reduction/spec.md`

## Summary

Reduce the project's CRAPload and GazeCRAPload by addressing the top 5 priority functions identified in the Gaze quality report. The work consists of three strategies: (1) adding contract-level tests for functions with zero coverage (`docscan.Filter`, `LoadModule`), (2) making CLI command functions testable via dependency injection (`runCrap`, `runSelfCheck`), and (3) decomposing high-complexity monolithic functions (`buildContractCoverageFunc`, `AnalyzeP1Effects`, `AnalyzeP2Effects`). The target is GazeCRAPload 7→4 and CRAPload 27→24.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: `golang.org/x/tools` (go/packages, go/ssa), Cobra (CLI), Bubble Tea/Lipgloss (TUI)
**Storage**: Filesystem only (embedded assets via `embed.FS`)
**Testing**: Standard library `testing` package only; no external assertion libraries
**Target Platform**: darwin/linux (amd64, arm64)
**Project Type**: Single binary CLI with layered internal packages
**Performance Goals**: Standard test suite (`-short`) completes within 10-minute CI timeout
**Constraints**: All tests must pass with `-race -count=1`; no new external dependencies
**Scale/Scope**: ~208 functions across 11 packages; changes affect 5 packages

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Accuracy — PASS

- **Decomposition (US4, US5)**: FR-017 and FR-021 explicitly require that decomposition produces identical analysis output — no false positives or negatives introduced.
- **Testing (US1, US2, US3)**: All new tests assert on observable contractual outputs (return values, error conditions, written output). No tests verify implementation internals.
- **Regression guard**: SC-008 requires all existing tests pass with zero regressions, backed by FR-023.

### II. Minimal Assumptions — PASS

- **No annotation changes**: No production code changes require users to modify their test code or project structure.
- **CLI behavior preserved**: FR-012 explicitly requires that dependency injection changes do not alter external CLI behavior. Users see identical output.
- **Explicit assumptions**: 6 assumptions documented in spec, all specific and auditable.

### III. Actionable Output — PASS

- **Metrics remain comparable**: Decomposition and testing changes do not alter the CRAP/GazeCRAP scoring formulas or report format. Users can compare runs before and after this feature.
- **Report output unchanged**: FR-017 (pipeline orchestrator) and FR-021 (effect detection) both require output identity pre- and post-change.
- **JSON/text formats preserved**: No changes to output schemas or formatters.

## Project Structure

### Documentation (this feature)

```text
specs/009-crapload-reduction/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: technical decisions
├── data-model.md        # Phase 1: entity definitions
├── quickstart.md        # Phase 1: integration scenarios
├── checklists/
│   └── requirements.md  # Specification quality checklist
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/gaze/
├── main.go              # US3: runCrap/runSelfCheck dependency injection
│                        # US4: buildContractCoverageFunc decomposition
└── main_test.go         # US3: fast unit tests for runCrap/runSelfCheck
                         # US4: tests for extracted pipeline functions

internal/
├── analysis/
│   ├── p1effects.go     # US5: decompose AnalyzeP1Effects into handlers
│   ├── p1effects_test.go # US5: verify handler decomposition
│   ├── p2effects.go     # US5: decompose AnalyzeP2Effects into handlers
│   └── p2effects_test.go # US5: verify handler decomposition
├── docscan/
│   ├── filter.go        # US1: (no changes — test-only)
│   └── filter_test.go   # US1: new contract-level tests for Filter
├── loader/
│   ├── loader.go        # US2: (no changes — test-only)
│   └── loader_test.go   # US2: new unit tests for LoadModule
└── crap/
    └── analyze.go       # (no changes — reference only)
```

**Structure Decision**: This feature modifies existing files within the established `cmd/gaze/` and `internal/` package structure. No new packages or top-level directories are introduced. New test files are created alongside their source files following Go conventions. Production code changes are limited to `cmd/gaze/main.go` (US3 dependency injection + US4 decomposition) and `internal/analysis/p1effects.go`/`p2effects.go` (US5 decomposition).

### Group Organization

The implementation is organized into three groups based on the type of change:

**Group A — Test-Only (US1, US2)**: New test files with zero production code changes. These are independent and can be developed in parallel.

- `internal/docscan/filter_test.go` — contract-level tests for `docscan.Filter`
- `internal/loader/loader_test.go` — unit tests for `LoadModule`

**Group B — Production Refactoring + Tests (US3, US4)**: Changes to `cmd/gaze/main.go` that modify function signatures and internal structure, plus corresponding test updates.

- `cmd/gaze/main.go` — dependency injection for `runCrap`/`runSelfCheck` params structs + `buildContractCoverageFunc` decomposition
- `cmd/gaze/main_test.go` — fast unit tests for all modified functions

**Group C — Production Decomposition + Test Verification (US5)**: Structural refactoring of analysis functions with existing test verification.

- `internal/analysis/p1effects.go` — decompose into per-node-type handlers
- `internal/analysis/p2effects.go` — decompose into per-node-type handlers
- `internal/analysis/p1effects_test.go` — verify identical output after decomposition
- `internal/analysis/p2effects_test.go` — verify identical output after decomposition
