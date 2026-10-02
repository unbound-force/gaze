# Architecture Overview

Gaze is a single-binary CLI tool that performs static analysis on Go source code to detect observable side effects, classify them, compute risk scores, and assess test quality. All business logic lives under `internal/` and cannot be imported externally.

## Package Map

| Package | Purpose | Key Dependencies |
|---------|---------|-----------------|
| `cmd/gaze/` | CLI entry point. Cobra commands, Bubble Tea TUI, flag parsing. Delegates to `runXxx(params)` functions. | `analysis`, `classify`, `crap`, `quality`, `report`, `aireport`, `docscan`, `scaffold`, `config`, `loader`, `taxonomy` |
| `internal/taxonomy/` | Domain type system. Defines `SideEffectType` constants (37 types across P0-P4), `Tier`, `ClassificationLabel`, `SideEffect`, `AnalysisResult`, `QualityReport`, `AssertionMapping`, and stable ID generation. | None (leaf package) |
| `internal/loader/` | Go package loading. Wraps `go/packages` with the minimum load mode flags needed for SSA-ready analysis. Provides `Load` (single package) and `LoadModule` (all packages via `./...`). | `go/packages` |
| `internal/analysis/` | Core side effect detection engine. Uses AST and SSA analysis to detect observable side effects in Go functions. | `taxonomy`, `loader`, `go/ast`, `go/types`, `x/tools/go/ssa` |
| `internal/classify/` | Contractual classification engine. Five signal analyzers (interface, visibility, caller, naming, godoc) produce weighted confidence scores. Classifies each effect as contractual, ambiguous, or incidental. | `taxonomy`, `config`, `go/types`, `go/packages` |
| `internal/config/` | Configuration file handling. Loads and validates `.gaze.yaml` files with classification thresholds and other settings. | None (leaf package) |
| `internal/crap/` | CRAP score computation and report formatting. Combines cyclomatic complexity with line coverage (CRAP) and contract coverage (GazeCRAP). Produces text, JSON, and self-contained HTML, including baseline comparisons. | `taxonomy`, `quality`, `analysis`, `classify`, `loader`, `config`, `html/template` |
| `internal/quality/` | Test quality assessment and report formatting. Performs test-target pairing, assertion detection and mapping, contract coverage, and over-specification scoring. Produces text, JSON, and self-contained HTML. | `taxonomy`, `analysis`, `classify`, `loader`, `config`, `go/ast`, `html/template`, `x/tools/go/ssa` |
| `internal/report/` | Output formatters for analysis results. JSON, styled text, and self-contained analyze HTML formatters. Embeds the JSON Schema (Draft 2020-12) and analyze HTML template. | `taxonomy`, `lipgloss`, `html/template` |
| `internal/docscan/` | Documentation file scanner. Finds Markdown files in the repository, prioritized by proximity to the target package. | None (leaf package) |
| `internal/docscan/apidoc/` | API documentation coverage analysis. Cross-references analyzer output (`doc_coverage` + `analyze`) against documentation files; computes coverage, stale references, and code-block language validation. | `docscan`, `protocol` |
| `internal/aireport/` | Combined CI quality report pipeline. Orchestrates all four analysis operations, emits native JSON or self-contained HTML, or pipes JSON to external AI CLIs for text output. Also handles threshold enforcement and GitHub Step Summary integration. | `taxonomy`, `crap`, `quality`, `analysis`, `classify`, `docscan`, `loader`, `config`, `html/template` |
| `internal/scaffold/` | OpenCode file scaffolding. Uses `embed.FS` to scaffold agent and command files into user projects via [`gaze init`](../reference/cli/init.md). | None (uses `embed.FS`) |

## Data Flow

The following diagram shows how data flows through Gaze from CLI invocation to output:

```
                          CLI Layer (cmd/gaze/)
                                  |
                          Flag parsing & routing
                                  |
                    +-------------+-------------+
                    |             |             |
                 analyze       crap/quality   report
                    |             |             |
                    v             v             v
              +---------+   +---------+   +-----------+
              | loader  |   | loader  |   | aireport  |
              | Load()  |   | Load()  |   | runner    |
              +---------+   +---------+   +-----------+
                    |             |             |
                    v             v             |
              +-----------+  +-----------+     |
              | analysis  |  | analysis  |<----+ (orchestrates all 4 steps)
              | Analyze() |  | Analyze() |     |
              +-----------+  +-----------+     |
                    |             |             |
                    v             v             |
              +-----------+  +-----------+     |
              | taxonomy  |  | classify  |<----+
              | results   |  | Classify()|     |
              +-----------+  +-----------+     |
                    |             |             |
                    v             v             |
              +-----------+  +-----------+     |
              | classify  |  | crap      |<----+
              | (optional)|  | Analyze() |     |
              +-----------+  +-----------+     |
                    |             |             |
                    v             v             |
              +-----------+  +-----------+     |
              | report    |  | quality   |<----+
              | Write*()  |  | Assess()  |     |
              +-----------+  +-----------+     |
                    |             |             |
                    v             v             v
                  stdout       stdout     native JSON/HTML or AI adapter → stdout
                                          (+ $GITHUB_STEP_SUMMARY)
```

### Step-by-step flow for [`gaze analyze`](../reference/cli/analyze.md)

1. **Load**: `loader.Load(pattern)` loads the target Go package with full type information via `go/packages`
2. **Analyze**: `analysis.Analyze(pkg, opts)` runs all detectors on each function:
   - `AnalyzeReturns` — return values and error returns (AST)
   - `AnalyzeMutations` — receiver and pointer argument mutations (SSA)
   - `AnalyzeP1Effects` — globals, writers, channels, HTTP, slices, maps (AST)
   - `AnalyzeP2Effects` — filesystem, database, goroutines, panics, callbacks, logging (AST)
3. **Classify** (optional, `--classify`): `classify.Classify(results, opts)` runs five signal analyzers on each effect
4. **Format**: `report.WriteJSON`, `report.WriteText`, or the analyze-only `report.WriteHTML` renders the output

### Step-by-step flow for [`gaze crap`](../reference/cli/crap.md)

1. **Coverage**: Generate or load a Go coverage profile (`go test -coverprofile`)
2. **Complexity**: Compute cyclomatic complexity for each function via `gocyclo`
3. **CRAP scores**: `crap.Formula(complexity, coverage)` for each function
4. **Contract coverage** (optional): `quality.Assess` computes contract coverage per function, enabling GazeCRAP and quadrant classification
5. **Fix strategies**: `assignFixStrategy` labels each CRAPload function with a remediation action
6. **Format**: Text, JSON, or self-contained HTML output

### Step-by-step flow for [`gaze report`](../reference/cli/report.md)

1. **Pipeline**: `aireport.Run` orchestrates four analysis steps in sequence:
   - CRAP step — complexity + coverage + GazeCRAP
   - Quality step — contract coverage + over-specification
   - Classify step — classification label counts
   - Docscan step — documentation file inventory
2. **Payload**: Results are assembled into a `ReportPayload` JSON structure
3. **Threshold check**: If threshold flags are set, enforce quality gates (exit non-zero on violation)
4. **Format**: JSON and self-contained HTML are rendered natively; text output is produced by the selected AI adapter (Claude/Gemini/Ollama/OpenCode)
5. **Output**: The selected format is written to stdout; AI-formatted text can also be appended to `$GITHUB_STEP_SUMMARY`

## Key Patterns

### AST + SSA Dual Analysis

Gaze uses two complementary static analysis techniques:

- **AST (Abstract Syntax Tree)**: Used for syntactic pattern matching. Detects return values, sentinel errors, P1 effects (channel sends, writer calls, global mutations), and P2 effects (filesystem, database, goroutine spawns). Fast and reliable for patterns that are syntactically visible.

- **SSA (Static Single Assignment)**: Used for data flow analysis. Detects receiver mutations and pointer argument mutations by tracking `*FieldAddr` and `*IndexAddr` instructions that flow through `Store` operations. SSA analysis requires building the SSA representation via `golang.org/x/tools/go/ssa`, which is more expensive but catches mutations that are invisible at the AST level.

SSA construction includes `BuildSerially` mode to ensure panics from upstream `x/tools` bugs are recoverable. When SSA fails, Gaze degrades gracefully — mutation analysis is skipped but AST-based detection continues.

### Testable CLI Pattern

Commands delegate to `runXxx(params)` functions that accept a params struct including `io.Writer` for stdout/stderr. This enables unit testing without subprocess execution:

```go
type crapParams struct {
    stdout    io.Writer
    stderr    io.Writer
    patterns  []string
    format    string
    threshold float64
    // ...
}

func runCrap(p crapParams) error {
    // All business logic here, writing to p.stdout
}
```

Tests call `runCrap` directly with a `bytes.Buffer` as the writer, avoiding the overhead and flakiness of spawning subprocesses.

### Options Structs

Configurable behavior uses options structs rather than long parameter lists:

```go
type Options struct {
    IncludeUnexported bool
    FunctionFilter    string
    Version           string
}
```

This pattern appears in `analysis.Options`, `classify.Options`, `crap.Options`, `quality.Options`, and `report.TextOptions`. It makes function signatures stable — new options are added as struct fields without breaking existing callers.

### Tiered Effect Taxonomy

Side effects are organized into five priority tiers defined in `internal/taxonomy/types.go`:

| Tier | Name | Detection | Examples |
|------|------|-----------|----------|
| P0 | Must Detect | Implemented | `ReturnValue`, `ErrorReturn`, `SentinelError`, `ReceiverMutation`, `PointerArgMutation` |
| P1 | High Value | Implemented | `GlobalMutation`, `WriterOutput`, `ChannelSend`, `HTTPResponseWrite`, `SliceMutation`, `MapMutation` |
| P2 | Important | Implemented | `FileSystemWrite`, `DatabaseWrite`, `GoroutineSpawn`, `Panic`, `LogWrite` |
| P3 | Nice to Have | Defined only | `StdoutWrite`, `StderrWrite`, `EnvVarMutation`, `MutexOp`, `TimeDependency` |
| P4 | Exotic | Defined only | `ReflectionMutation`, `UnsafeMutation`, `CgoCall`, `ClosureCaptureMutation` |

Each effect type is a string constant. The tier determines the confidence boost during classification: P0 effects start at confidence 75 (base 50 + 25 boost), P1 at 60 (base 50 + 10 boost), and P2-P4 at the base of 50.

## Package Dependency Graph

Arrows indicate "imports" relationships. Only `internal/` packages are shown.

```
taxonomy  (leaf — no internal deps)
    ^
    |
config    (leaf — no internal deps)
    ^
    |
loader    (leaf — depends on go/packages only)
    ^
    |
analysis  ──> taxonomy, loader
    ^
    |
classify  ──> taxonomy, config
    ^
    |
quality   ──> taxonomy, analysis, classify, loader, config
    ^
    |
crap      ──> taxonomy, quality, analysis, classify, loader, config
    |
report    ──> taxonomy (leaf for output formatting)
    |
docscan   (leaf — no internal deps)
    |
apidoc    ──> docscan, protocol
    |
scaffold  (leaf — uses embed.FS only)
    |
aireport  ──> taxonomy, crap, quality, analysis, classify, docscan, apidoc, loader, config
    ^
    |
cmd/gaze/ ──> all internal packages
```

Key observations:

- **`taxonomy`** is the foundational leaf package — every analysis package depends on it
- **`config`** and **`loader`** are also leaf packages with no internal dependencies
- **`analysis`** depends only on `taxonomy` and `loader`
- **`classify`** depends only on `taxonomy` and `config` (no dependency on `analysis`)
- **`quality`** and **`crap`** are higher-level packages that compose `analysis` + `classify`
- **`report`** depends only on `taxonomy` — it formats results without knowing how they were produced
- **`aireport`** is the highest-level internal package, orchestrating all others
- **`scaffold`** is an isolated utility with no internal dependencies; **`docscan`** is a leaf package whose `apidoc` sub-package depends on it and on `protocol`
