## Context

The `--analyzer` and `--language` flags are registered on `gaze quality`
but hidden and rejected at runtime (D12 deferral). Phase 2 (#178)
wired external analyzers into `gaze crap` and `gaze report` using
`initExternalSession` → `wireExternalProviders` → `crap.Analyze`. The
quality command was deferred because `quality.Assess` requires
Go-specific test loading, SSA construction, and assertion detection.

The key insight is that **quality does not need to call
`quality.Assess` at all** when an external analyzer is active. The
`test_mapping` protocol method provides pre-computed assertion
mappings — the external analyzer's equivalent of the entire Go quality
pipeline. The existing `buildContractLookup` in
`internal/adapter/contract.go` already converts `test_mapping` data
into contract coverage lookups. What's missing is a path from
`test_mapping` data to `QualityReport` / `PackageSummary` output.

## Goals / Non-Goals

### Goals
- Wire `--analyzer` / `--language` into `gaze quality` so external analyzers produce quality reports
- Reuse the existing `initExternalSession` / `adapter.Session` infrastructure
- Produce text/JSON output matching the same schema as Go-native quality
- Degrade gracefully when `test_mapping` capability is `false`
- Unhide `--analyzer` and `--language` flags from `gaze quality --help`

### Non-Goals
- Modifying the `quality.Assess` function or Go-native quality pipeline
- Adding new protocol methods or modifying the JSON-RPC protocol
- Supporting `--ai-mapper` with external analyzers (AI mapper is Go-specific)
- Supporting `--target` filtering with external analyzers (external analyzers provide complete mappings; filtering would require analyzer cooperation)
- Wiring external analyzers for `gaze analyze` (remains deferred, separate issue)

## Decisions

### D1: Build quality reports from test_mapping data directly

**Decision**: Add a new function `buildQualityFromMappings` in
`internal/adapter/` that converts `test_mapping` assertion mapping
data + side effect analysis results into `[]taxonomy.QualityReport` and
`*taxonomy.PackageSummary`.

**Rationale**: The existing `buildContractLookup` computes contract
coverage per function but returns a lookup function (for CRAP scoring).
Quality reports need per-test-function reports with contract coverage,
gap hints, unmapped assertions, and over-specification data. A new
function can reuse `quality.ComputeContractCoverage` for the metric
computation while constructing `QualityReport` structs from the
mapping data.

**Constitution**: Composability First — the adapter module builds
quality data without depending on Go-specific test loading or SSA.

### D2: Reuse `initExternalSession` for session lifecycle

**Decision**: The quality external analyzer path uses the same
`initExternalSession` helper as `runCrapWithExternalAnalyzer`.

**Rationale**: The session lifecycle (discover → spawn → initialize →
close) is identical. No need to duplicate.

### D3: Side effects come from the external analyzer

**Decision**: Use `ExternalSideEffectAnalyzer.AllResults()` to get
side effect data, just as `ExternalContractCoverageProvider.Build`
does today.

**Rationale**: The external analyzer provides language-specific side
effect detection via the `analyze` protocol method. This is already
loaded as part of `Providers` during `Initialize()`.

### D4: Classification uses external analyzer data as-is

**Decision**: Do not run `classify.ComputeScore` on external analyzer
results. Use the classification data already present in the analysis
results (if any) or treat unclassified effects as contractual.

**Rationale**: `quality.ComputeContractCoverage` already handles
unclassified effects conservatively. External analyzers may provide
`classify_signals` but this is orthogonal to this change.

### D5: Graceful degradation when test_mapping unavailable

**Decision**: When `Capabilities.TestMapping` is `false`, print a
warning to stderr and produce a quality report with zero contract
coverage and a clear reason ("test_mapping not supported"). Do not
error — this matches the CRAP command's graceful degradation.

**Rationale**: Constitution Principle II (Minimal Assumptions) — don't
require external analyzers to implement optional methods.

### D6: No --target filtering with external analyzers

**Decision**: Reject `--target` when `--analyzer` is set, with a
clear error message.

**Rationale**: `--target` triggers Go-specific SSA-based target
inference in `quality.Assess`. External analyzers provide complete
test→target mappings; filtering to a single target would need to
happen post-mapping. This is a non-goal for the initial wiring and
can be revisited if users need it.

### D7: No --ai-mapper with external analyzers

**Decision**: Reject `--ai-mapper` when `--analyzer` is set, with a
clear error message.

**Rationale**: The AI mapper operates on Go AST/SSA assertion sites.
External analyzers provide their own assertion mappings. Combining
them is undefined.

### Data Flow: BuildQualityFromMappings

```
Input:
  []protocol.AssertionMappingData  (from test_mapping JSON-RPC call)
  []taxonomy.AnalysisResult        (from analyze JSON-RPC call)

Processing:
  1. Group AssertionMappingData by test_function
  2. For each test function:
     a. Match assertion mappings to side effects from AnalysisResult
     b. Compute contract coverage via quality.ComputeContractCoverage
     c. Compute over-specification (assertions on incidental effects)
     d. Generate gap hints for unasserted contractual effects
  3. Aggregate into PackageSummary

Output:
  []taxonomy.QualityReport         (one per test function)
  *taxonomy.PackageSummary         (aggregated metrics)
```

### D8: Over-specification from external mappings

**Decision**: Over-specification is computed from mappings that target
incidental side effects. If the external analyzer provides
classification data (either via the analysis results or
classify_signals), incidental assertions can be counted.

**Rationale**: Over-specification = assertions on incidental effects.
The formula is the same regardless of data source.

## Risks / Trade-offs

### Risk: External analyzer test_mapping granularity mismatch

**Risk**: External analyzers may group assertions differently than
Go-native quality (e.g., per-test-file vs per-test-function).

**Mitigation**: The protocol spec defines `test_function` as the
mapping key. Analyzers that cannot resolve individual test functions
will produce less granular reports but still valid output.

### Risk: Missing classification data

**Risk**: External analyzers that don't implement `classify_signals`
produce results without classification. All effects become
"contractual" by default, inflating contract coverage.

**Mitigation**: This is by design (`ComputeContractCoverage` treats
unclassified as contractual). The quality report text will note when
classification data is absent, consistent with the CRAP path.

### Trade-off: Assertion detection confidence

**Trade-off**: Go-native quality produces `assertion_detection_
confidence` from pattern matching heuristics. External analyzers
provide their own `confidence` field per mapping but no aggregate
detection confidence metric.

**Decision**: Set `AssertionDetectionConfidence` to 0 for external
reports. The `assertion_count` field reflects the mapping count from
the analyzer.
