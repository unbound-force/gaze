# Design: Language-Aware Documentation Coverage

## Context

`gaze docscan` currently discovers Markdown files and returns them as prioritized `[]DocumentFile` for document-enhanced classification. It has no awareness of the actual API surface of the project being analyzed. Meanwhile, the external analyzer protocol (shipped in #95) provides structured access to any language's exported symbols via `analyze` (functions with side effects) and `discover` (source/test files). Combining these two data sources enables documentation validation against the real codebase.

The proposal establishes three capabilities: undocumented symbol detection, symbol-reference validation (API drift), and code block language validation. This design specifies how those capabilities are implemented within the existing architecture.

## Goals / Non-Goals

### Goals
- Add a `doc_coverage` optional protocol method so analyzers can natively report which symbols have documentation
- Provide heuristic documentation coverage when `doc_coverage` is not available, using `analyze` output (function names) cross-referenced against Markdown content
- Detect stale symbol references in Markdown files (symbols mentioned but no longer in the codebase)
- Validate fenced code block language tags against the analyzer's declared language
- Integrate with the existing `gaze docscan` CLI and `gaze report` pipeline without breaking current behavior
- Produce machine-readable JSON output with clear, actionable findings

### Non-Goals
- **Docstring quality assessment** — we detect presence/absence, not whether docs are "good"
- **Auto-fixing documentation** — out of scope; findings are informational
- **Cross-module documentation tracking** — coverage is per-module, matching the analyzer scope
- **Go-specific docstring parsing** — Go's own documentation coverage is handled by `goprovider`; this feature targets external analyzers
- **Real-time documentation linting** — this is a batch analysis tool, not an editor plugin

## Decisions

### D1: New `doc_coverage` protocol method (optional)

**Decision**: Add a new optional method `doc_coverage` to the external analyzer protocol, announced via `Capabilities.DocCoverage bool`.

**Rationale**: Some languages have first-class documentation systems (Python docstrings, Rust doc comments, JSDoc) where the analyzer can directly report which symbols have documentation. This is more accurate than heuristic Markdown scanning. Making it optional preserves backward compatibility — existing analyzers work without changes.

**Protocol types**:
```go
// DocCoverageParams is the params for the "doc_coverage" method.
type DocCoverageParams struct {
    RootPath string   `json:"root_path"`
    Patterns []string `json:"patterns"`
}

// DocCoverageResult is the result for the "doc_coverage" method.
type DocCoverageResult struct {
    Symbols []SymbolDocStatus `json:"symbols"`
}

// SymbolDocStatus reports documentation status for a single public symbol.
type SymbolDocStatus struct {
    Name       string `json:"name"`        // Fully qualified symbol name
    Package    string `json:"package"`     // Package/module path
    File       string `json:"file"`        // Source file path
    Line       int    `json:"line"`        // Declaration line
    Kind       string `json:"kind"`        // "function", "type", "constant", "variable", "class", "method"
    Documented bool   `json:"documented"`  // Whether the symbol has a docstring/doc comment
    DocSnippet string `json:"doc_snippet,omitempty"` // First line of documentation (for display)
}
```

**Constitution alignment**: Principle II (Minimal Assumptions) — the method is optional; Principle I (Accuracy) — analyzer-native doc detection is more accurate than text matching.

### D2: Heuristic fallback when `doc_coverage` is unavailable

**Decision**: When the analyzer does not support `doc_coverage`, derive documentation coverage heuristically from `analyze` output (function names) by scanning Markdown files for backtick-quoted references to those names.

**Rationale**: Every conforming analyzer implements `analyze`, so we can always extract a list of public function names. Cross-referencing those names against Markdown content provides a reasonable (though imperfect) proxy for documentation coverage. The heuristic is clearly labeled as such in the output.

**Implementation**: The `apidoc` package accepts either `DocCoverageResult` (from analyzer) or `[]AnalyzedFunction` (from `analyze`) and produces the same `APICoverageReport` output. A `Source` field on each finding indicates whether the data came from `doc_coverage` (native) or `analyze` (heuristic).

### D3: New `internal/docscan/apidoc` sub-package

**Decision**: Create a new sub-package `internal/docscan/apidoc` for API documentation coverage logic, rather than adding to the existing `internal/docscan` package.

**Rationale**: The existing `docscan` package is focused on Markdown file discovery and prioritization. API coverage analysis is a distinct concern that deserves its own test surface. The sub-package imports from `internal/protocol` (for types) but not from `internal/adapter` (no session management). This keeps dependencies minimal and testability high (Principle IV).

**Package structure**:
```text
internal/docscan/apidoc/
  types.go      — APICoverageReport, SymbolCoverage, StaleReference, CodeBlockIssue
  coverage.go   — ComputeCoverage (from DocCoverageResult or AnalyzedFunction list)
  validation.go — ValidateReferences, ValidateCodeBlocks
  report.go     — WriteJSON, WriteText output formatters
```

### D4: Separate orchestration via `apidoc.Analyze`

**Decision**: `docscan.ScanOptions` is NOT modified. The `Scan` function continues to return `[]DocumentFile` with no awareness of analyzer data. API coverage is computed by the caller via `apidoc.Analyze(docs []DocumentFile, data *AnalyzerData) (*APICoverageReport, error)` — a separate function in the `apidoc` sub-package.

**Rationale**: Adding an `AnalyzerData` field to `ScanOptions` would create a dependency from `internal/docscan` on `internal/docscan/apidoc` types and introduce a field with no concrete behavioral requirement (Zero-Waste violation). The caller (CLI or report pipeline) is responsible for obtaining analyzer data (via `adapter.Session`) and passing it to `apidoc.Analyze` alongside the Markdown scan results. This preserves `Scan`'s single responsibility (Markdown discovery) and keeps the `docscan` package dependency-free.

**Alternative considered**: Returning a new composite type from `Scan` — rejected because it would break all existing callers and conflate discovery with analysis.

### D5: CLI integration via `--analyzer` flag reuse

**Decision**: Reuse the existing `--analyzer` and `--language` flags (already on `crap`, `quality`, `report`) on `gaze docscan`.

**Rationale**: Consistency with existing CLI surface. The three-tier discovery mechanism (`Discover` in `internal/adapter/`) already works for these flags. The docscan command creates its own `Session`, calls `Initialize`, extracts the `analyze` and optionally `doc_coverage` data, then passes it to `apidoc.Analyze`.

### D6: Report pipeline integration

**Decision**: `runDocscanStep` in `internal/aireport/runner_steps.go` gains an optional `*adapter.Session` parameter. When non-nil, it calls `apidoc.Analyze` after the existing Markdown scan and merges the API coverage data into the docscan JSON output.

**Rationale**: The report pipeline already manages an analyzer session for CRAP/quality steps. Passing the same session to the docscan step avoids spawning a second analyzer subprocess. The session is shared sequentially — the pipeline runs steps in order, so the session is never accessed concurrently.

### D7: JSON output structure

**Decision**: The docscan JSON output becomes a structured object with two top-level keys:

```json
{
  "documents": [ /* existing []DocumentFile */ ],
  "api_coverage": {
    "total_symbols": 42,
    "documented_symbols": 35,
    "coverage_percent": 83.3,
    "source": "doc_coverage",
    "undocumented": [ /* SymbolCoverage entries */ ],
    "stale_references": [ /* StaleReference entries */ ],
    "code_block_issues": [ /* CodeBlockIssue entries */ ]
  }
}
```

When no analyzer is available, `api_coverage` is `null` (omitted). This is a **breaking change** to the docscan JSON output format (previously a bare `[]DocumentFile` array). Per the constitution's release policy ("Breaking changes to public APIs or analysis behavior require a MAJOR bump"), this change requires a semver MAJOR version increment for the gaze binary. The text output format is unaffected since docscan doesn't have a text formatter today.

**Migration**: The report pipeline's `CompactForAI` already handles docscan as `json.RawMessage`, so the shape change is transparent to AI adapters. The only consumer that parses the docscan JSON is the gaze-reporter agent prompt, which receives it as opaque context. External consumers of `gaze docscan` JSON output must update their parsers to expect the structured envelope instead of the bare array.

## Risks / Trade-offs

### R1: Heuristic coverage accuracy

The Markdown-scanning heuristic (D2) will produce false positives (symbol name appears in prose but not as documentation) and false negatives (documentation uses different terminology than the function name). Mitigation: clearly label heuristic results with `"source": "heuristic"` and recommend analyzers implement `doc_coverage` for accurate results.

### R2: Breaking change to docscan JSON output (D7)

Changing from a bare array to a structured object breaks any tooling that parses `gaze docscan` JSON output. Mitigation: this requires a semver MAJOR version bump per the constitution's release policy. Document the breaking change and migration guidance in release notes and README. The structured format is strictly more informative.

### R3: Protocol version implications

Adding `doc_coverage` as an optional method does not require a protocol version bump (1.1.0 → 1.1.0 is fine for additive optional methods). If we later decide it should be required, that would need a 1.2.0 bump.

### R4: Analyzer session lifecycle in `gaze docscan`

The `gaze docscan` command currently doesn't manage any subprocess lifecycle. Adding `--analyzer` means it must now create a `Session`, call `Initialize`, use it, and `Close` — similar to what `gaze crap` does. This adds complexity to the command but follows the established pattern.

### R5: Stale reference detection precision

Detecting `renamed` vs. `removed` symbols requires comparing against a baseline. Without a baseline, we can only report "symbol X referenced in docs but not found in current API surface." Baseline comparison is out of scope for this change.

## Coverage Strategy

Per Constitution Principle IV, the coverage strategy for this change:

- **Unit tests** (primary): All exported functions in `internal/docscan/apidoc/` (`Analyze`, `ComputeCoverage`, `ValidateReferences`, `ValidateCodeBlocks`) tested with synthetic `AnalyzerData` and `DocumentFile` structs. Target: ≥90% line coverage for the `apidoc` package given its pure-function nature.
- **Integration tests**: Fake analyzer binary in `internal/protocol/testdata/fake_analyzer/` extended to support `doc_coverage` method. Round-trip test verifies `adapter.Session` → `doc_coverage` call → unmarshal → `apidoc.Analyze` pipeline.
- **CLI integration tests**: `runDocscan` tested via the existing testable CLI pattern (`docscanParams` struct with `io.Writer` injection) — verifies JSON output structure with and without analyzer.
- **No e2e tests**: `gaze docscan` is not in the e2e self-check suite. The unit + integration coverage is sufficient.
- **Coverage ratchet**: The `apidoc` package coverage will be enforced by the existing CI `go test -race -count=1 -short ./...` gate.
