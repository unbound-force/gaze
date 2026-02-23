# AGENTS.md

## Project Overview

Gaze is a static analysis tool for Go that detects observable side effects in functions and computes CRAP (Change Risk Anti-Patterns) scores by combining cyclomatic complexity with test coverage. It helps developers find functions that are complex and under-tested — the riskiest code to change.

- **Language**: Go 1.24+
- **Module**: `github.com/unbound-force/gaze`
- **License**: Apache 2.0

## Core Mission

- **Strategic Architecture**: Engineers shift from manual coding to directing an "infinite supply of junior developers" (AI agents).
- **Outcome Orientation**: Focus on conveying business value and user intent rather than low-level technical sub-tasks.
- **Intent-to-Context**: Treat specs and rules as the medium through which human intent is manifested into code.

## Behavioral Constraints

- **Zero-Waste Mandate**: No orphaned code, unused dependencies, or "Feature Zombie" bloat.
- **Neighborhood Rule**: Changes must be audited for negative impacts on adjacent modules or the wider ecosystem.
- **Intent Drift Detection**: Evaluation must detect when the implementation drifts away from the original human-written "Statement of Intent."
- **Automated Governance**: Primary feedback is provided via automated constraints, reserving human energy for high-level security and logic.

## Technical Guardrails

- **WORM Persistence**: Use Write-Once-Read-Many patterns where data integrity is paramount.

## Council Governance Protocol

- **The Architect**: Must verify that "Intent Driving Implementation" is maintained.
- **The Adversary**: Acts as the primary "Automated Governance" gate for security.
- **The Guard**: Detects "Intent Drift" to ensure the business value remains intact.

**Rule**: A Pull Request is only "Ready for Human" once the `/review-council` command returns an **APPROVE** status.

## Speckit Workflow (Mandatory)

All non-trivial feature work **must** go through the Speckit pipeline. The constitution (`.specify/memory/constitution.md`) is the highest-authority document in this project — all work must align with it.

### Pipeline

The workflow is a strict, sequential pipeline. Each stage has a corresponding `/speckit.*` command:

```text
constitution → specify → clarify → plan → tasks → analyze → checklist → implement
```

| Command | Purpose |
|---------|---------|
| `/speckit.constitution` | Create or update the project constitution |
| `/speckit.specify` | Create a feature specification from a description |
| `/speckit.clarify` | Reduce ambiguity in the spec before planning |
| `/speckit.plan` | Generate the technical implementation plan |
| `/speckit.tasks` | Generate actionable, dependency-ordered task list |
| `/speckit.analyze` | Non-destructive cross-artifact consistency analysis |
| `/speckit.checklist` | Generate requirement quality validation checklists |
| `/speckit.implement` | Execute the implementation plan task by task |
| `/speckit.taskstoissues` | Convert tasks.md into GitHub Issues |

### Ordering Constraints

1. Constitution must exist before specs.
2. Spec must exist before plan.
3. Plan must exist before tasks.
4. Tasks must exist before implementation and analysis.
5. Clarify should run before plan (skipping increases rework risk).
6. Analyze should run after tasks but before implementation.
7. All checklists must pass before implementation (or user must explicitly override).

### Spec Organization

Specs are numbered with 3-digit zero-padded prefixes and stored under `specs/`:

```text
.specify/
  memory/
    constitution.md              # Governance document (highest authority)
  templates/                     # Templates for all artifact types
  scripts/bash/                  # Automation scripts
specs/
  001-side-effect-detection/     # spec.md, plan.md, tasks.md
  002-contract-classification/   # spec.md
  003-test-quality-metrics/      # spec.md
  004-composite-metrics/         # spec.md
```

Branch names follow the same numbering pattern (e.g., `001-side-effect-detection`).

### Task Completion Bookkeeping

When a task from `tasks.md` is completed during implementation, its checkbox **must** be updated from `- [ ]` to `- [x]` immediately. Do not defer this — mark tasks complete as they are finished, not in a batch after all work is done. This keeps the task list an accurate, real-time view of progress and prevents drift between the codebase and the plan.

### Documentation Validation Gate

Before marking any task complete, you **must** validate whether the change requires documentation updates. Check and update as needed:

- `README.md` — new/changed commands, flags, output formats, or architecture
- `AGENTS.md` — new conventions, packages, patterns, or workflow changes
- GoDoc comments — new or modified exported functions, types, and packages
- Spec artifacts under `specs/` — if the change affects planned behavior

A task is not complete until its documentation impact has been assessed and any necessary updates have been made. Skipping this step causes documentation drift, which compounds over time and erodes project accuracy.

### Spec Commit Gate

All spec artifacts (`spec.md`, `plan.md`, `tasks.md`, and any other files under `specs/`) **must** be committed and pushed before implementation begins. This ensures the planning record is preserved in version control before code changes start, and provides a clean baseline to diff against if implementation drifts from the plan. Run `/speckit.implement` only after the spec commit is on the remote.

### Constitution Check

A mandatory gate at the planning phase. The constitution's three core principles — Accuracy, Minimal Assumptions, and Actionable Output — must each receive a PASS before proceeding. Constitution violations are automatically CRITICAL severity and non-negotiable.

## Build & Test Commands

```bash
# Build
go build ./cmd/gaze

# Run unit + integration tests (use -short to skip e2e)
go test -race -count=1 -short ./...

# Run e2e tests only (self-check: spawns go test -coverprofile)
go test -race -count=1 -run 'TestRunSelfCheck' -timeout 30m ./cmd/gaze/...

# Run all tests (no -short, requires ~15min)
go test -race -count=1 ./...

# Lint
golangci-lint run
```

Always run tests with `-race -count=1`. CI enforces this.

### Test Suites

Tests are organized into two CI suites that run in parallel:

| Suite | Command | Timeout | What it runs |
|-------|---------|---------|-------------|
| Unit + Integration | `go test -race -count=1 -short ./...` | 10m (default) | All tests except those guarded by `testing.Short()` |
| E2E | `go test -race -count=1 -run TestRunSelfCheck -timeout 30m ./cmd/gaze/...` | 20m | Self-check tests that spawn `go test -coverprofile` on the full module |

Use `testing.Short()` to guard tests that spawn external `go test` processes or analyze the entire module. These are too slow for the standard CI timeout.

## Architecture

Single binary CLI with layered internal packages:

```text
cmd/gaze/              CLI layer (Cobra commands, Bubble Tea TUI)
internal/
  analysis/            Core side effect detection engine (AST + SSA)
  taxonomy/            Domain types: SideEffect, AnalysisResult, Tier, etc.
  loader/              Go package loading (go/packages wrapper)
  report/              Output formatters (JSON, text, HTML stub)
  crap/                CRAP score computation and reporting
```

All business logic lives under `internal/` and cannot be imported externally.

### Key Patterns

- **AST + SSA dual analysis**: Returns, sentinels, and P1/P2 effects use Go AST. Mutation tracking uses SSA via `golang.org/x/tools`.
- **Testable CLI pattern**: Commands delegate to `runXxx(params)` functions. Params structs include `io.Writer` for stdout/stderr, enabling unit testing without subprocess execution.
- **Options structs**: Configurable behavior uses options/params structs rather than long parameter lists.
- **Tiered effect taxonomy**: Side effects are organized into priority tiers P0-P4.

## Coding Conventions

- **Formatting**: `gofmt` and `goimports` (enforced by golangci-lint).
- **Naming**: Standard Go conventions. PascalCase for exported, camelCase for unexported.
- **Comments**: GoDoc-style comments on all exported functions and types. Package-level doc comments on every package.
- **Error handling**: Return `error` values. Wrap with `fmt.Errorf("context: %w", err)`.
- **Import grouping**: Standard library, then third-party, then internal packages (separated by blank lines).
- **No global state**: The logger is the only package-level variable. Prefer functional style.
- **Constants**: Use string-typed constants for enumerations (`SideEffectType`, `Tier`, `Quadrant`).
- **JSON tags**: Required on all struct fields intended for serialization.

## Testing Conventions

- **Framework**: Standard library `testing` package only. No testify, gomega, or other external assertion libraries.
- **Assertions**: Use `t.Errorf` / `t.Fatalf` directly. No assertion helpers from third-party packages.
- **Test naming**: `TestXxx_Description` (e.g., `TestReturns_PureFunction`, `TestFormula_ZeroCoverage`).
- **Test files**: `*_test.go` alongside source in the same directory. Both internal and external package test styles are used.
- **Test fixtures**: Real Go packages in `testdata/src/` directories, loaded via `go/packages`.
- **Benchmarks**: Separate `bench_test.go` files with `BenchmarkXxx` functions.
- **Acceptance tests**: Named after spec success criteria (e.g., `TestSC001_ComprehensiveDetection`, `TestSC004_SingleFunctionPerformance`).
- **JSON Schema validation**: Tests validate JSON output against the embedded JSON Schema (Draft 2020-12).
- **Output width**: Report output is verified to fit within 80-column terminals.

## Core Principles

These principles (from the project constitution) guide all development:

1. **Accuracy**: Gaze MUST correctly identify all observable side effects. False positives erode trust and MUST be treated as bugs. False negatives MUST be tracked, measured, and driven toward zero. Accuracy claims MUST be backed by automated regression tests.
2. **Minimal Assumptions**: Gaze MUST operate with the fewest possible assumptions about the host project's language, test framework, or coding style. No source annotation or restructuring required. When assumptions are unavoidable, they MUST be explicit and enforced.
3. **Actionable Output**: Every piece of output MUST guide the user toward a concrete improvement. Reports MUST identify specific test, target, and unasserted change. Output formats MUST support human-readable and machine-readable (JSON). Metrics MUST be comparable across runs.

## Git & Workflow

- **Commit format**: Conventional Commits — `type: description` (e.g., `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`).
- **Branching**: Feature branches required. No direct commits to `main` except trivial doc fixes.
- **Code review**: Required before merge.
- **Semantic versioning**: For releases.

## CI/CD

Two GitHub Actions workflows on push/PR to `main`:

1. **Test** (`.github/workflows/test.yml`): Build + test with `-race -count=1`.
2. **MegaLinter** (`.github/workflows/mega-linter.yml`): Runs golangci-lint, markdownlint, yamllint, and gitleaks. Auto-commits lint fixes to PR branches.

## Linting

golangci-lint v2 is configured in `.golangci.yml` with these linters enabled:

- errcheck, govet, staticcheck, ineffassign, unused, misspell

Formatters: gofmt, goimports.
