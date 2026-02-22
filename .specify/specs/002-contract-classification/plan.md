# Implementation Plan: Contractual Classification & Confidence Scoring

**Branch**: `002-contract-classification` | **Date**: 2026-02-21 | **Spec**: `.specify/specs/002-contract-classification/spec.md`
**Input**: Feature specification from `.specify/specs/002-contract-classification/spec.md`

## Summary

Classify each side effect detected by Spec 001 as contractual,
incidental, or ambiguous using a weighted confidence scoring system.
Mechanical signals (interface satisfaction, API surface visibility,
same-module caller analysis, naming conventions, godoc) provide a
deterministic baseline. Document-enhanced signals are extracted via
an OpenCode agent/command that reads project documentation and
produces structured classification evidence. A `.gaze.yaml`
configuration file provides threshold and document-scanning
settings. All four user stories are delivered as a single unit.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: `golang.org/x/tools` (go/packages, go/types, SSA), `github.com/spf13/cobra` (CLI), `gopkg.in/yaml.v3` (config parsing — new dependency)
**Storage**: N/A (stateless analysis; `.gaze.yaml` is read-only config)
**Testing**: Standard library `testing` package only, `-race -count=1`
**Target Platform**: Any platform supported by Go toolchain
**Project Type**: Single binary CLI
**Performance Goals**: Mechanical classification adds zero latency beyond Spec 001 analysis (SC-004). Document scanning + AI adds < 10s for typical projects (SC-005).
**Constraints**: Same-module caller analysis only (no whole-program). No embedded LLM SDK. OpenCode agent/command for AI inference.
**Scale/Scope**: Typical Go modules with < 500 packages, < 50 .md documentation files.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Check

| Principle | Status | Rationale |
|-----------|--------|-----------|
| **I. Accuracy** | PASS | SC-001 requires >= 90% true positive rate for contractual classification on a 30+ function benchmark suite. SC-003 caps false contractual rate at < 15%. Determinism guaranteed by FR-011. All accuracy claims will be backed by automated regression tests. |
| **II. Minimal Assumptions** | PASS | Mechanical classification requires no annotation, no restructuring, no external services. Document-enhanced mode is opt-in via OpenCode agent/command (FR-010). `.gaze.yaml` is optional with sensible defaults (FR-009). No assumptions about test framework or coding style. |
| **III. Actionable Output** | PASS | Each classification includes the label, confidence score, and signal breakdown (FR-014). Users can see exactly which signals contributed and by how much. JSON output extends existing schema additively (FR-013). Confidence scores are comparable across runs (FR-011). |

**GATE RESULT: PASS** — Proceed to design.

## Project Structure

### Documentation (this feature)

```text
.specify/specs/002-contract-classification/
├── spec.md              # Feature specification (complete)
├── plan.md              # This file
└── tasks.md             # Task breakdown (next step)
```

### Source Code (repository root)

```text
cmd/gaze/
├── main.go              # Add --classify flag, --config flag, --verbose flag
└── main_test.go         # Tests for classify integration

internal/
├── analysis/            # Existing — no changes needed
├── taxonomy/
│   ├── types.go         # Add Classification, Signal, ConfidenceScore types
│   ├── priority.go      # Existing — no changes
│   └── types_test.go    # Extend with classification type tests
├── classify/            # NEW — core classification engine
│   ├── classify.go      # Classifier entry point, Options struct
│   ├── mechanical.go    # Mechanical signal analyzers
│   ├── interface.go     # Interface satisfaction detector
│   ├── callers.go       # Same-module caller analysis
│   ├── visibility.go    # Exported API surface visibility scorer
│   ├── naming.go        # Naming convention signal
│   ├── godoc.go         # Godoc comment signal
│   ├── score.go         # Confidence score computation, thresholds
│   ├── classify_test.go # Unit tests
│   ├── bench_test.go    # Benchmark tests (SC-004)
│   └── testdata/src/    # Test fixture packages
│       ├── contracts/   # Functions with known contractual effects
│       ├── incidental/  # Functions with known incidental effects
│       └── ambiguous/   # Functions with contradicting signals
├── docscan/             # NEW — document scanning
│   ├── scanner.go       # Scan repo for documentation files
│   ├── filter.go        # Exclude/include pattern matching
│   ├── scanner_test.go  # Tests
│   └── testdata/        # Test fixture repos with docs
├── config/              # NEW — .gaze.yaml loading
│   ├── config.go        # GazeConfig struct, Load(), defaults
│   ├── config_test.go   # Tests
│   └── testdata/        # Sample .gaze.yaml files
├── loader/              # Existing — extend for multi-package loading
│   ├── loader.go        # Add LoadModule() for sibling packages
│   └── loader_test.go   # Extend with module loading tests
├── report/              # Existing — extend for classified output
│   ├── json.go          # Extend JSONReport with classifications
│   ├── text.go          # Extend text output with classification column
│   ├── schema.go        # Extend JSON Schema with classification types
│   ├── styles.go        # Add classification label styling
│   └── report_test.go   # Extend tests
└── crap/                # Existing — no changes needed
```

**Structure Decision**: New packages `classify/`, `docscan/`, and
`config/` follow the existing layered architecture under `internal/`.
The `classify/` package is the core engine, analogous to `analysis/`
for Spec 001. The `docscan/` package handles file discovery and
filtering. The `config/` package handles `.gaze.yaml` parsing.
The existing `loader/` package is extended to support loading
sibling packages within a module for caller analysis.

## Research Decisions

### R-001: Interface Satisfaction Detection

**Decision**: Use `go/types.Implements()` to check if a receiver
type satisfies any interface defined in the same module.

**Rationale**: The `go/types` package provides `types.Implements(T, I)`
which returns true if type T implements interface I. By loading
sibling packages via `go/packages`, we can collect all interface
types defined in the module, then check each receiver type against
them. The interface's method set defines the "contract" — any side
effect that matches a method in the interface signature is
contractual evidence.

**Alternatives considered**: SSA-based call graph analysis (too slow
for the performance budget), manual AST walking for interface
declarations (less reliable than type-checker).

### R-002: Same-Module Caller Analysis

**Decision**: Use `go/packages` with the `./...` pattern to load all
packages in the module, then scan `types.Info.Uses` to find call
sites of the target function.

**Rationale**: Loading the module's packages via `go/packages.Load`
with `./...` gives access to all sibling packages. We then scan
each package's `TypesInfo.Uses` map to find references to the target
function's `types.Func` object. This tells us which callers exist
and whether they use the return value. This stays within the
same-module scope per the clarification decision.

**Alternatives considered**: `callgraph/cha` (Class Hierarchy
Analysis) provides precise call graphs but adds latency.
`TypesInfo.Uses` scanning is faster and sufficient for caller
counting. Start with `Uses` scanning; upgrade to CHA only if
precision issues arise.

### R-003: Naming Convention Signals

**Decision**: Pattern-matching on function and variable names to
detect contractual intent.

**Rationale**: Certain naming patterns strongly signal contractual
behavior:
- `Get*`, `Fetch*`, `Load*`, `Read*` → return value is contractual
- `Set*`, `Update*`, `Save*`, `Write*`, `Delete*` → mutation is contractual
- `Err*` sentinel variables → error return is contractual
- `Handle*`, `Process*` → side effects are contractual
- `log*`, `debug*`, `trace*` → side effects are incidental

These patterns are language-community conventions in Go and can be
applied deterministically.

### R-004: Godoc Comment Signal

**Decision**: Parse the function's doc comment for behavioral
declarations using keyword matching.

**Rationale**: Godoc comments that explicitly state behavior ("returns
an error if...", "writes to the database", "modifies the receiver")
are strong contractual signals. We use keyword extraction rather than
NLP to keep this deterministic and fast:
- "returns" + error/value keywords → return is contractual
- "writes", "modifies", "updates", "sets" → mutation is contractual
- "logs", "prints" → side effect is incidental

### R-005: OpenCode Agent for Document Analysis

**Decision**: Implement document-enhanced classification as an
OpenCode command (`.opencode/commands/classify-docs.md`) that
orchestrates the full pipeline, backed by an OpenCode agent
(`.opencode/agents/doc-classifier.md`) that does the scoring.

**Rationale**: Per clarification, Gaze is exclusive to OpenCode.
The agent handles both signal extraction and score merging so
that no additional `gaze` subcommand is needed for merging.

**End-to-end pipeline**:

1. User runs `/classify-docs <package>` in OpenCode.
2. The command shells out to `gaze analyze --classify --format=json <package>`
   to produce mechanical-only classification JSON.
3. The command shells out to `gaze docscan <package>` (or reads
   the file system directly) to collect prioritized documentation
   content, filtered by `.gaze.yaml` exclude/include patterns.
4. The command feeds both inputs to the `doc-classifier` agent.
5. The agent:
   a. Reads the mechanical classification JSON (function targets,
      side effects, current classifications with signals).
   b. Reads the documentation content.
   c. Extracts document signals (FR-005: README, architecture
      docs, .specify files, API docs, other .md) and AI inference
      signals (FR-006: design pattern recognition, layer/boundary
      analysis, cross-document corroboration).
   d. Merges document signals with existing mechanical signals
      using the base-50 scoring model: starts from the mechanical
      confidence score, adds document signal weights, detects
      cross-category contradictions (applies up to -20 penalty),
      clamps to 0-100, re-applies classification thresholds.
   e. Outputs the final enhanced classification JSON in the same
      schema as `gaze analyze --classify --format=json`, with
      additional document/AI signals in the signals array.
6. The command outputs the enhanced JSON to stdout.

**Agent input contract**:
- Mechanical classification JSON (same schema as `gaze analyze --classify --format=json`)
- Documentation content as a structured list: `[{path, content, priority}]`
- Scoring rules: base-50 model, signal weight bounds from FR-005/FR-006, contradiction penalty from FR-007, thresholds from FR-003

**Agent output contract**:
- Enhanced classification JSON (same schema, with additional signals appended and confidence scores recalculated)
- Each document signal must include: source, weight (within documented bounds), source_file, excerpt, reasoning

**Graceful degradation**: If the user runs `gaze analyze --classify`
directly (without the OpenCode command), they get mechanical-only
results. The OpenCode command is opt-in.

**Testing**: The agent's output can be validated by schema
comparison — the output must be valid against the extended JSON
schema. The scoring math can be spot-checked by verifying that
signal weights sum correctly and are within bounds (SC-007).

### R-006: Configuration File Format

**Decision**: Use YAML (`.gaze.yaml`) with `gopkg.in/yaml.v3`.

**Rationale**: YAML is the standard for Go tool configuration files
(see `.golangci.yml`, `.goreleaser.yml`). The `gopkg.in/yaml.v3`
library is the de facto standard, well-maintained, and has no
transitive dependencies. Config schema:

```yaml
# .gaze.yaml
classification:
  thresholds:
    contractual: 80    # >= this = contractual
    incidental: 50     # < this = incidental
                       # between = ambiguous
  doc_scan:
    exclude:
      - "vendor/**"
      - "node_modules/**"
      - ".git/**"
      - "testdata/**"
      - "CHANGELOG.md"
      - "CONTRIBUTING.md"
      - "CODE_OF_CONDUCT.md"
      - "LICENSE"
      - "LICENSE.md"
    include: []          # if set, overrides default full-repo scan
    timeout: "30s"
```

### R-007: JSON Schema Extension Strategy

**Decision**: Add an optional `classification` field to the
`SideEffect` schema definition. Add a `classifications_metadata`
field to the top-level report.

**Rationale**: FR-013 requires additive, non-breaking extension.
By making `classification` optional on each `SideEffect`, existing
consumers that don't understand classifications simply ignore the
field. The schema version bumps from the current value to indicate
the addition.

Extended SideEffect JSON example:

```json
{
  "id": "se-a1b2c3d4",
  "type": "ReturnValue",
  "tier": "P0",
  "location": "store.go:42:1",
  "description": "returns (bool, error)",
  "target": "bool",
  "classification": {
    "label": "contractual",
    "confidence": 87,
    "signals": [
      {
        "source": "interface",
        "weight": 30,
        "reasoning": "implements io.Reader"
      },
      {
        "source": "caller",
        "weight": 12,
        "reasoning": "8/8 callers use return value"
      }
    ]
  }
}
```

## Architecture

### Data Flow

```
┌──────────────┐    ┌──────────────┐    ┌───────────────┐
│  loader/     │───>│  analysis/   │───>│  classify/    │
│  Load()      │    │  Analyze()   │    │  Classify()   │
│  LoadModule()│    │              │    │  (mechanical) │
└──────────────┘    └──────────────┘    └───────┬───────┘
                                                │
                    ┌──────────────┐             │
                    │  config/     │─────────────┤
                    │  Load()      │             │
                    └──────────────┘             │
                                                │
                    ┌──────────────┐             │
                    │  docscan/    │─────────────┤
                    │  Scan()      │             │
                    └──────────────┘             │
                                                v
                                        ┌───────────────┐
                                        │  report/      │
                                        │  WriteJSON()  │
                                        │  WriteText()  │
                                        └───────────────┘
```

### Mechanical Classification Pipeline

For each `AnalysisResult` from Spec 001:

1. **Load context**: Load module packages (for caller analysis),
   collect interfaces (for satisfaction check).
2. **Per side effect**: Run each mechanical signal analyzer:
   - `interface.go`: Check if the function's receiver type
     satisfies an interface whose method signature implies this
     side effect → +0 to +30
   - `visibility.go`: Check if the side effect is observable
     through exported types → +0 to +20
   - `callers.go`: Count callers that use/depend on this side
     effect → +0 to +15
   - `naming.go`: Match function/variable naming patterns
     → +0 to +10
   - `godoc.go`: Extract behavioral keywords from doc comment
     → +0 to +15
3. **Compute score**: Start from base confidence of 50, add
   signal weights (positive → contractual, negative →
   incidental), apply contradiction penalty if applicable,
   clamp to 0-100.
4. **Apply thresholds**: >= 80 = contractual, 50-79 = ambiguous,
   < 50 = incidental (configurable via `.gaze.yaml`).

### Document-Enhanced Pipeline (OpenCode Agent)

1. User runs `/classify-docs <package>` in OpenCode.
2. The command runs `gaze analyze --classify --format=json <package>`
   to get mechanical-only results.
3. `docscan/` scans the repo for `.md` files, applies
   exclude/include filters from `.gaze.yaml`, prioritizes by
   proximity (same package > module root > other).
4. The command feeds mechanical JSON + prioritized docs to the
   `doc-classifier` agent.
5. The agent extracts document signals (FR-005) and AI inference
   signals (FR-006), merges them with mechanical signals using
   the base-50 scoring model, and outputs final enhanced
   classification JSON.
6. No Go-side `MergeSignals()` function is needed — the agent
   handles all scoring.

### Key Design Decisions

- **No new dependencies for mechanical classification**: Interface
  satisfaction, caller analysis, naming, and godoc all use
  `go/types` and `go/ast` already in the dependency tree.
- **Single new dependency**: `gopkg.in/yaml.v3` for `.gaze.yaml`
  parsing.
- **Separation of concerns**: `classify/` handles scoring logic,
  `docscan/` handles file discovery, `config/` handles
  configuration. Each is independently testable.
- **Backward compatibility**: Classification is optional in the
  JSON schema. Existing `gaze analyze` output is unchanged unless
  `--classify` is passed.

## Post-Design Constitution Check

| Principle | Status | Rationale |
|-----------|--------|-----------|
| **I. Accuracy** | PASS | Benchmark suite of 30+ functions with known contracts validates SC-001 (>= 90% true positive). Contradiction detection (FR-007) prevents overconfident misclassification. Signal breakdown (FR-014) makes every classification auditable. |
| **II. Minimal Assumptions** | PASS | Zero annotation required. `.gaze.yaml` is optional. Document scanning is opt-in. Naming conventions use established Go community patterns, not project-specific rules. |
| **III. Actionable Output** | PASS | Each classification tells the user: this side effect is contractual/incidental/ambiguous, with confidence N%, because of signals X, Y, Z. Users know exactly which effects their tests should assert on. |

**GATE RESULT: PASS** — Proceed to task breakdown.

## Complexity Tracking

No constitution violations requiring justification.
