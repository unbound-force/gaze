# Proposal: Language-Aware Documentation Coverage via External Analyzer Protocol

## Why

`gaze docscan` currently performs a language-agnostic Markdown file discovery pass — it finds README, API reference, and tutorial files and prioritizes them by proximity to the target package. This is useful for document-enhanced classification, but it cannot answer the question every maintainer asks: **"Which public functions are undocumented?"**

The external analyzer protocol (shipped in #95) gives Gaze a structured view of any language's API surface: function names, packages, files, side effects, and optionally test mappings. By combining the protocol's `analyze` method (and the new optional `doc_coverage` method) with the existing docscan Markdown scanner, Gaze can validate documentation against the actual code — detecting renamed symbols, missing API docs, and stale code examples — for any language with an analyzer.

This closes the gap between "here are your docs" and "here's what your docs are missing."

## What Changes

1. **New protocol method: `doc_coverage`** — An optional method that external analyzers can implement to report which public symbols have associated documentation (docstrings, type annotations, module-level docs). Announced via `Capabilities.DocCoverage`. Falls back to heuristic coverage based on `discover` + `analyze` output when the analyzer doesn't implement it.

2. **New `internal/docscan/apidoc` sub-package** — Consumes analyzer output (exported symbols from `analyze`, documentation status from optional `doc_coverage`) and cross-references against discovered Markdown files. Produces an `APICoverageReport` identifying:
   - Undocumented public symbols (functions, types, constants)
   - Stale symbol references in Markdown (renamed/removed API drift)
   - Code block language validation (do fenced code blocks specify the correct language?)

3. **Separate `apidoc.Analyze` orchestrator** — The caller (CLI or report pipeline) obtains analyzer data via `adapter.Session`, then passes it to `apidoc.Analyze(docs, data)` alongside the Markdown scan results. `docscan.Scan` itself remains unchanged — it continues to return `[]DocumentFile` without any new fields.

4. **CLI integration** — `gaze docscan --analyzer <path>` (or auto-discovered) activates analyzer-aware mode. JSON output gains an `api_coverage` section alongside the existing `[]DocumentFile` array.

5. **Report pipeline integration** — `runDocscanStep` in the report pipeline passes the analyzer session (when available) to produce richer documentation analysis in `gaze report` output.

## Capabilities

| Capability | Description |
|---|---|
| **Undocumented symbol detection** | Lists public functions/types/constants that appear in analyzer output but have no corresponding documentation (docstring or Markdown reference) |
| **Symbol-reference validation** | Scans Markdown files for backtick-quoted or link-referenced symbol names; flags references to symbols not found in the current analyzer output (API drift) |
| **Code block language validation** | Checks fenced code blocks (` ```python `, ` ```go `) against the analyzer's declared language |
| **Convention-aware doc coverage** | Computes a documentation coverage percentage: (documented symbols / total public symbols) × 100 |
| **Graceful degradation** | When no analyzer is available, falls back to existing language-agnostic Markdown scan — zero behavioral change for current users |
| **Multi-language support** | Works with any language that has a conforming external analyzer (Go via goprovider, Python via snake-eyes, etc.) |

## Impact

- **New files**: `internal/docscan/apidoc/` package (~4 files: types, coverage, validation, report)
- **Modified files**: `internal/docscan/scanner.go` (new `DocscanOutput` type), `internal/protocol/types.go` (new `doc_coverage` method types), `cmd/gaze/main.go` (`runDocscan` gains `--analyzer` flag), `internal/aireport/runner_steps.go` (`runDocscanStep` gains session parameter), `internal/aireport/compact.go` (update `compactDocscanField` for new JSON shape)
- **Breaking change**: JSON output format changes from bare `[]DocumentFile` array to structured `{"documents": [...], "api_coverage": ...}` object — requires semver MAJOR version bump per constitution policy
- **Protocol version**: Remains 1.1.0 (new optional method, no protocol-level breaking changes)
- **Dependencies**: No new external dependencies — uses existing `internal/protocol` and `internal/adapter` packages

## Constitution Alignment

### Principle I: Accuracy — PASS
Analyzer-aware documentation coverage is grounded in the actual API surface reported by the external analyzer. Symbol references are validated against real function/type names, not guessed from file paths or naming conventions. False positives (flagging a symbol as undocumented when it has a docstring) are prevented by the `doc_coverage` protocol method — the analyzer itself reports documentation status rather than Gaze guessing from Markdown proximity.

### Principle II: Minimal Assumptions — PASS
No assumptions about the host project's documentation style, file layout, or naming conventions. The feature activates only when an external analyzer is available (explicit opt-in via `--analyzer` flag or auto-discovery). When no analyzer is found, behavior is identical to today — no degradation. The `doc_coverage` method is optional; analyzers that don't implement it trigger a heuristic fallback based on `discover` + `analyze` output.

### Principle III: Actionable Output — PASS
Every output item guides toward a concrete improvement: "function `Foo` in `pkg/bar` has no documentation", "symbol `OldName` referenced in `README.md:42` no longer exists in the codebase", "code block at `docs/tutorial.md:15` is tagged `python` but the analyzer reports `go`". Machine-readable JSON output supports CI integration. Human-readable text output fits the existing report format.

### Principle IV: Testability — PASS
The `apidoc` sub-package is designed for isolation testing with synthetic analyzer output (mock `AnalyzedFunction` slices and `DocCoverageResult` data). No real analyzer binary is needed for unit tests. Integration tests use the existing fake analyzer binary in `internal/protocol/testdata/fake_analyzer/`. The `AnalyzerData` input struct enables testing the coverage computation without subprocess management.
