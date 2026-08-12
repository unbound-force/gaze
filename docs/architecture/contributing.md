# Contributing Guide

This guide covers everything you need to set up a development environment, build, test, and contribute to Gaze.

## Dev Environment Setup

### Prerequisites

- **Go 1.25.0 or later** (the `go.mod` directive requires `go 1.25.0`)
- **golangci-lint v2** for linting
- **Git** for version control

### Clone and Build

```bash
git clone https://github.com/unbound-force/gaze.git
cd gaze
go build ./cmd/gaze
```

Verify the build:

```bash
./gaze --help
```

### Install golangci-lint

```bash
# macOS
brew install golangci-lint

# Or via Go
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Build Commands

| Command | Purpose |
|---------|---------|
| `go build ./cmd/gaze` | Build the binary |
| `go test -race -count=1 -short ./...` | Run unit + integration tests (skips e2e) |
| `go test -race -count=1 -run TestRunSelfCheck -timeout 30m ./cmd/gaze/...` | Run e2e tests only |
| `go test -race -count=1 ./...` | Run all tests (unit + integration + e2e, ~15 min) |
| `golangci-lint run` | Run linters |

Always run tests with `-race -count=1`. CI enforces this.

### Test Suites

Tests are organized into two CI suites that run in parallel:

| Suite | Command | Timeout | What It Runs |
|-------|---------|---------|-------------|
| Unit + Integration | `go test -race -count=1 -short ./...` | 10m | All tests except those guarded by `testing.Short()` |
| E2E | `go test -race -count=1 -run TestRunSelfCheck -timeout 30m ./cmd/gaze/...` | 20m | Self-check tests that spawn `go test -coverprofile` on the full module |

Use `testing.Short()` to guard tests that spawn external `go test` processes or analyze the entire module. These are too slow for the standard CI timeout.

## Coding Conventions

### Formatting

- **`gofmt`** and **`goimports`** are enforced by golangci-lint
- No manual formatting adjustments needed — run `goimports` and the linter handles the rest

### Naming

Standard Go conventions:

- `PascalCase` for exported identifiers
- `camelCase` for unexported identifiers
- Acronyms are all-caps when exported (`SSA`, `AST`, `HTTP`), lowercase when unexported (`ssa`, `ast`)

### Comments

- **GoDoc-style comments** on all exported functions and types
- **Package-level doc comments** on every package
- Comments explain *why*, not *what* — the code explains what

```go
// ComputeScore computes the confidence score from a set of signals,
// applies a tier-based boost, contradiction detection and penalty,
// clamps to 0-100, and returns a Classification based on the
// configured thresholds.
func ComputeScore(effectType taxonomy.SideEffectType, signals []taxonomy.Signal, cfg *config.GazeConfig) taxonomy.Classification {
```

### Error Handling

- Return `error` values — never panic for expected failures
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Handle all error paths — no ignored error returns

```go
result, err := loader.Load(pattern)
if err != nil {
    return nil, fmt.Errorf("loading package %q: %w", pattern, err)
}
```

### Import Grouping

Three groups separated by blank lines:

1. Standard library
2. Third-party packages
3. Internal packages

```go
import (
    "fmt"
    "go/ast"

    "golang.org/x/tools/go/packages"

    "github.com/unbound-force/gaze/internal/taxonomy"
)
```

### No Global State

The logger is the only package-level variable. Prefer functional style — pass dependencies through function parameters or options structs.

### Constants

Use string-typed constants for enumerations:

```go
type SideEffectType string

const (
    ReturnValue  SideEffectType = "ReturnValue"
    ErrorReturn  SideEffectType = "ErrorReturn"
)
```

### JSON Tags

Required on all struct fields intended for serialization:

```go
type Score struct {
    Package    string  `json:"package"`
    Function   string  `json:"function"`
    Complexity int     `json:"complexity"`
    CRAP       float64 `json:"crap"`
}
```

## Testing Conventions

### Framework

**Standard library `testing` package only.** No testify, gomega, go-cmp, or other external assertion libraries.

### Assertions

Use `t.Errorf` / `t.Fatalf` directly:

```go
if got != want {
    t.Errorf("Formula(%d, %.1f) = %.1f, want %.1f", complexity, coverage, got, want)
}
```

### Test Naming

`TestXxx_Description` format:

```go
func TestReturns_PureFunction(t *testing.T) { ... }
func TestFormula_ZeroCoverage(t *testing.T) { ... }
func TestSC001_ComprehensiveDetection(t *testing.T) { ... }  // acceptance test
```

### Test Files

- `*_test.go` alongside source in the same directory
- Both internal (`package foo`) and external (`package foo_test`) test styles are used

### Test Fixtures

Real Go packages in `testdata/src/` directories, loaded via `go/packages`:

```
internal/analysis/testdata/src/basic/    # simple functions for testing
internal/classify/testdata/              # classification test fixtures
internal/quality/testdata/               # quality assessment fixtures
```

### Benchmarks

Separate `bench_test.go` files with `BenchmarkXxx` functions:

```go
func BenchmarkAnalyze(b *testing.B) { ... }
```

### Acceptance Tests

Named after spec success criteria:

```go
func TestSC001_ComprehensiveDetection(t *testing.T) { ... }
func TestSC004_SingleFunctionPerformance(t *testing.T) { ... }
```

### JSON Schema Validation

Tests validate JSON output against the embedded JSON Schema (Draft 2020-12) in `internal/report/schema.go`.

### Output Width

Report output is verified to fit within 80-column terminals.

## Git Workflow

### Commit Format

[Conventional Commits](https://www.conventionalcommits.org/):

```
type: description
```

Types: `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `test:`

Examples:

```
feat: add GazeCRAP quadrant classification
fix: recover from SSA builder panics in goroutines
docs: add architecture overview
refactor: extract runSubprocess helper for AI adapters
test: add contract coverage tests for docscan.Filter
```

### Branching

- Feature branches required for all changes
- No direct commits to `main` except trivial doc fixes
- Branch names follow spec numbering (e.g., `001-side-effect-detection`) or kebab-case for targeted changes (e.g., `fix-ssa-goroutine-panic`)

### Code Review

Required before merge. The project uses a four-persona automated review council (Architect, Guard, Adversary, Tester) that must all return APPROVE before a PR is ready for human review.

### Semantic Versioning

Releases follow [semver](https://semver.org/). A `workflow_dispatch` trigger with a `tag` input starts the release pipeline, which delegates to org-infra reusable workflows for preflight validation and GoReleaser execution. To release: `gh workflow run release.yml -f tag=v1.2.3`.

## Spec-First Development

All changes that modify production code, test code, or CI configuration must be preceded by a spec workflow. Two workflows are available:

### Speckit Pipeline

For numbered feature specs under `specs/NNN-name/`:

```
specify → clarify → plan → tasks → implement
```

Each spec directory contains `spec.md`, `plan.md`, `tasks.md`, and optional research/data-model artifacts. Tasks are dependency-ordered and implemented sequentially.

### OpenSpec

For targeted changes under `openspec/changes/name/`:

```
propose → design → specs → tasks → apply
```

Lighter-weight artifacts for focused changes. Archived after merge under `openspec/changes/archive/`.

### When Specs Are Required

- New features or capabilities
- Refactoring that changes function signatures or moves code between packages
- Test additions across multiple functions
- CI workflow modifications
- Data model changes

### When Specs Are Exempt

- Typo corrections, comment-only changes, single-line formatting fixes
- Emergency hotfixes for critical production bugs (retroactively documented)
- Constitution amendments (governed by the constitution's own process)

When in doubt, ask rather than proceeding without a spec.
