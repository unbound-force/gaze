## Why

Issue #229. The `--analyzer` and `--language` flags were scoped for all
four subcommands in spec 005 (issue #95). Phase 2 (#178) wired external
analyzers to `gaze crap` and `gaze report` but explicitly deferred
`gaze quality` (D12 in `cmd/gaze/main.go`). The flags are registered
on the quality command but hidden from help and return an error if used.

Infrastructure blockers are now resolved:
- #237: `callAndUnmarshal` generic helper merged
- #238: `safeSSABuild` consolidation merged
- Protocol v1.1.0 fully specifies `test_mapping`

Without this change, users with external analyzers (e.g., `snake-eyes`
for Python) cannot run `gaze quality` at all — they must use
`gaze crap` or `gaze report` to get contract coverage data, losing the
detailed per-function quality breakdown (gap hints, assertion mappings,
over-specification, skipped tests).

## What Changes

Wire `--analyzer` / `--language` into `gaze quality` so that external
analyzers providing `test_mapping` can produce quality reports. When an
external analyzer is active, the Go-specific quality pipeline
(`quality.Assess` with SSA, test loading, assertion detection) is
bypassed entirely — the analyzer's pre-computed assertion mappings are
used to build quality reports directly.

## Capabilities

### New Capabilities
- `quality-external-analyzer`: `gaze quality --analyzer <name> --language <lang> -- ./...` produces a quality report using external analyzer test_mapping data

### Modified Capabilities
- `gaze quality --analyzer`: unhide flag from help, remove error rejection
- `gaze quality --language`: unhide flag from help

### Removed Capabilities
- None

## Impact

### Files Modified
- `cmd/gaze/main.go`: Remove `--analyzer` rejection in `runQuality`, add `runQualityWithExternalAnalyzer` function, unhide flags
- `internal/adapter/contract.go` or new adapter helper: Build quality reports from `test_mapping` assertion mapping data
- `internal/quality/`: Potential new function to construct `QualityReport` / `PackageSummary` from external assertion mappings

### Behavioral Changes
- `gaze quality --analyzer snake-eyes --language python -- ./...` produces output instead of erroring
- Quality text/JSON output from external analyzers contains the same fields as Go-native quality output, with external assertion mappings in place of Go-specific assertion detection
- When `test_mapping` capability is `false`, quality degrades gracefully: warns to stderr and produces reports with zero contract coverage

### Adjacent Module Impact
- `gaze report`: No impact (already wires external analyzers separately)
- `gaze crap`: No impact (already wires external analyzers separately)
- `internal/adapter/`: May gain a helper for building quality data from test_mapping, reusable by future commands

## Constitution Alignment

Assessed against the Unbound Force org constitution.

### I. Autonomous Collaboration

**Assessment**: PASS

This change extends an existing protocol-based interface (`test_mapping`
JSON-RPC method) that is already specified and implemented. External
analyzers communicate through well-defined artifacts (JSON-RPC
request/response) — no runtime coupling is introduced. The quality
command's output remains self-describing (JSON with schema, text with
structured sections).

### II. Composability First

**Assessment**: PASS

The external analyzer remains independently installable and optional.
When no `--analyzer` flag is provided, `gaze quality` continues to use
the Go-native pipeline with zero behavioral change. The `test_mapping`
capability is negotiated at initialization — analyzers that don't
support it degrade gracefully. No mandatory dependencies are introduced.

### III. Observable Quality

**Assessment**: PASS

Quality output from external analyzers will use the same JSON schema and
text format as Go-native output, maintaining machine-parseability. The
output will include provenance indicators (analyzer name, language,
capabilities) so consumers know the data source. Contract coverage
metrics remain comparable across runs and across analyzer
implementations.

### IV. Testability

**Assessment**: PASS

The external analyzer path will be testable in isolation using the
existing fake analyzer binary pattern (`internal/protocol/testdata/
fake_analyzer/`). The function that converts `test_mapping` assertion
data into quality reports will be a pure function testable with
synthetic inputs. No external services are required — the analyzer is a
subprocess communicating via stdin/stdout.
