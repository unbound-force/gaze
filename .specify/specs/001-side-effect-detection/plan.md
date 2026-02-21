# Implementation Plan: Side Effect Detection Engine

**Branch**: `001-side-effect-detection` | **Date**: 2026-02-20 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/001-side-effect-detection/spec.md`

## Summary

Build a Go CLI tool (`gaze`) that statically analyzes Go functions
to detect observable side effects across P0-P2 tiers: return
values, error returns (including sentinel/wrapped errors), pointer
receiver mutations, pointer argument mutations, global mutations,
channel operations, file system effects, database operations,
and more. Uses `go/ast` + `go/types` for signature and pattern
analysis, `go/ssa` for field-level mutation tracking, and direct
function composition for analyzer orchestration. Outputs
structured JSON and human-readable text.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**:
  - `golang.org/x/tools/go/packages` (package loading)
  - `golang.org/x/tools/go/ssa` + `ssautil` (SSA construction)
  - `github.com/spf13/cobra` (CLI framework)
**Storage**: N/A (stateless analysis, output to stdout)
**Testing**: `go test` with standard library `testing` package
**Target Platform**: macOS, Linux (any platform with Go toolchain)
**Project Type**: Single CLI tool
**Performance Goals**: < 500ms single function, < 5s for 50 functions
**Constraints**: Must not modify target source code; Go 1.24+
**Scale/Scope**: P0-tier effects only for v1 (5 side effect types)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after design.*

| Principle | Status | Notes |
|---|---|---|
| I. Accuracy | PASS | P0-tier targets 100% detection with zero false positives (SC-001). Benchmark suite required. |
| II. Minimal Assumptions | PASS | Uses standard Go toolchain only. No source modification. No annotations required. |
| III. Actionable Output | PASS | JSON + text output. Each effect includes type, location, description, affected target. |

## Project Structure

### Documentation (this feature)

```text
.specify/specs/001-side-effect-detection/
├── spec.md              # Feature specification
├── plan.md              # This file
└── tasks.md             # Task breakdown (next step)
```

### Source Code (repository root)

```text
cmd/
└── gaze/
    └── main.go              # CLI entry point (cobra root command)

internal/
├── analysis/
│   ├── analyzer.go          # Top-level GazeAnalyzer (aggregator)
│   ├── returns.go           # ReturnAnalyzer: return values, error returns
│   ├── returns_test.go
│   ├── sentinel.go          # SentinelAnalyzer: sentinel errors, %w wrapping
│   ├── sentinel_test.go
│   ├── mutation.go          # MutationAnalyzer: receiver + pointer arg mutations
│   └── mutation_test.go
├── taxonomy/
│   ├── types.go             # SideEffectType enum, SideEffect struct
│   ├── priority.go          # Priority tier assignments (P0-P4)
│   └── types_test.go
├── loader/
│   ├── loader.go            # Package loading via go/packages
│   └── loader_test.go
├── report/
│   ├── json.go              # JSON output formatter
│   ├── text.go              # Human-readable text formatter
│   ├── report_test.go
│   └── schema.go            # JSON Schema definition
└── analysis/testdata/src/
    ├── returns/             # Test fixtures for return analysis
    ├── sentinel/            # Test fixtures for sentinel analysis
    ├── mutation/            # Test fixtures for mutation analysis
    ├── p1effects/           # Test fixtures for P1 effects
    ├── p2effects/           # Test fixtures for P2 effects
    └── edgecases/           # Test fixtures for edge cases

go.mod
go.sum
```

**Structure Decision**: Go-idiomatic `cmd/` + `internal/` layout.
The `internal/analysis/` package contains one analyzer per side
effect category, composed via direct function calls. Packages are
loaded via `go/packages` with manual AST/SSA traversal.
Test fixtures live in `internal/analysis/testdata/src/` as real
Go packages loaded during tests.

## Design Decisions

### Analyzer Architecture

Each P0 side effect type gets its own `analysis.Analyzer`:

1. **ReturnAnalyzer** — AST-based. Inspects `*ast.FuncDecl.Type.Results`
   for return value count, types, positions, and named returns.
   Detects error positions. Reports `ReturnValue` and `ErrorReturn`.

2. **SentinelAnalyzer** — AST-based. Scans package-level `var`
   declarations for `Err*` variables initialized with
   `errors.New()` or `fmt.Errorf("...%w...")`. Cross-references
   function bodies to find which sentinels are returned or
   wrapped. Reports `SentinelError`.

3. **MutationAnalyzer** — SSA-based. Requires `buildssa.Analyzer`.
   Walks SSA instructions looking for:
   - `*ssa.Store` where the address is a `*ssa.FieldAddr` through
     the receiver parameter → `ReceiverMutation`
   - `*ssa.Store` where the address is through a pointer parameter
     → `PointerArgMutation`
   - Resolves field names via `types.Struct.Field(idx).Name()`

4. **GazeAnalyzer** — Aggregator. Requires all sub-analyzers.
   Merges results into `[]SideEffect` per function. This is the
   entry point for both CLI and programmatic use.

### SSA Construction

Use `golang.org/x/tools/go/ssa` with `ssa.InstantiateGenerics`
flag to handle generic functions correctly. Build SSA from
`go/packages` loaded packages using `ssautil.Packages()`.

### Package Loading

Use `go/packages` with LoadMode:
```go
packages.NeedName | packages.NeedFiles |
packages.NeedCompiledGoFiles | packages.NeedImports |
packages.NeedDeps | packages.NeedTypes |
packages.NeedSyntax | packages.NeedTypesInfo |
packages.NeedTypesSizes
```

Set `Tests: false` (we analyze production code, not test files).

### Output Schema

JSON output follows a flat structure:

```json
{
  "version": "0.1.0",
  "target": {
    "package": "github.com/example/pkg",
    "function": "Save",
    "receiver": "*Store",
    "signature": "func (s *Store) Save(ctx context.Context, item Item) (int64, error)",
    "location": "store.go:42:1"
  },
  "side_effects": [
    {
      "id": "se-001",
      "type": "ReturnValue",
      "tier": "P0",
      "location": "store.go:42:54",
      "description": "Returns int64 at position 0",
      "target": "int64"
    },
    {
      "id": "se-002",
      "type": "ErrorReturn",
      "tier": "P0",
      "location": "store.go:42:60",
      "description": "Returns error at position 1",
      "target": "error"
    },
    {
      "id": "se-003",
      "type": "ReceiverMutation",
      "tier": "P0",
      "location": "store.go:55:2",
      "description": "Mutates receiver field 'lastSaved'",
      "target": "Store.lastSaved"
    }
  ],
  "metadata": {
    "gaze_version": "0.1.0",
    "go_version": "1.24",
    "duration_ms": 145,
    "warnings": []
  }
}
```

### Stable Side Effect IDs

IDs are generated deterministically from:
`sha256(package_path + function_name + effect_type + location)`
truncated to 8 hex chars, prefixed with `se-`. This ensures
stable IDs across runs for diff/tracking (FR-005 of Spec 003).

### CLI Interface

```
gaze analyze [package] [--function name] [--format json|text]
             [--include-unexported]
```

Examples:
```bash
# Analyze all exported functions in a package
gaze analyze ./internal/store

# Analyze a specific function
gaze analyze ./internal/store --function Save

# JSON output
gaze analyze ./internal/store --format json

# Include unexported functions
gaze analyze ./internal/store --include-unexported
```

## Complexity Tracking

No constitution violations. The design follows standard Go
analysis patterns and stays within P0 scope.

| Decision | Justification |
|---|---|
| Multiple analyzer functions | Composability via direct function calls; each is independently testable |
| SSA for mutations only | AST is sufficient for return values and sentinel errors; SSA only needed for field-level tracking |
| Cobra for CLI | Industry standard, minimal overhead, good subcommand support for future commands |
