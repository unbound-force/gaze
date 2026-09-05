## Context

`AssertionDetectionConfidence` in `taxonomy.QualityReport` indicates what fraction
of test assertions were successfully pattern-matched (0-100). In the Go-native
path, `quality.Assess` computes this via `computeDetectionConfidence(sites)` which
counts `AssertionSite` entries where `Kind != AssertionKindUnknown`.

For external analyzers, the quality pipeline still calls `quality.Assess` for the
report step, but `DetectAssertions` (which uses Go AST patterns) finds zero
assertion sites in non-Go code. The result: `AssertionDetectionConfidence` is
always 0, even when the external analyzer provides rich mapping data through the
`test_mapping` protocol method.

The external analyzer already provides the data needed to compute this metric.
Each `protocol.AssertionMappingData` entry has an `AssertionType` field (e.g.,
"equality", "error_check") — if non-empty, the analyzer recognized the assertion
pattern.

## Goals / Non-Goals

### Goals
- Compute `AssertionDetectionConfidence` from external analyzer mapping data
- Populate per-test-function confidence in `QualityReport` entries for external
  analyzer quality reports
- Aggregate confidence into `PackageSummary.AssertionDetectionConfidence`
- No protocol changes (gaze-side computation only)

### Non-Goals
- Changing the Go-native assertion detection pipeline
- Adding new protocol methods or fields
- Modifying the `computeDetectionConfidence` function (it stays Go-AST-specific)
- Changing contract coverage scores (those are correct today)

## Decisions

### D1: Compute confidence in `buildContractLookup` return value

Compute per-function assertion detection confidence during
`ExternalContractCoverageProvider.Build` (which already iterates all mappings).
Rather than changing `buildContractLookup`'s return signature, store the
confidence on the provider struct as a `map[string]int` keyed by
`pkg + "/" + function`.

The caller (`ExternalContractCoverageProvider.Build`) already receives
the mappings — but `Build`'s interface contract (`crap.ContractCoverageProvider`)
returns only the lookup function and degraded packages. The confidence data
needs to flow through a different channel to reach `QualityReport` construction.

### D2: New `DetectionConfidenceByFunc` field on provider struct

Store the per-function detection confidence map on
`ExternalContractCoverageProvider` as a computed field populated during
`Build`. The `buildExternalQualityReports` function in `cmd/gaze/main.go`
(which has access to the provider via `providers.ContractCoverage`) reads
this map when constructing `QualityReport` entries.

This avoids changing the `crap.ContractCoverageProvider` interface (which
is shared by `goprovider` and `mockprovider`) and avoids protocol changes.

```go
type ExternalContractCoverageProvider struct {
    // ... existing fields ...
    // detectionConfidence holds per-target-function assertion detection
    // confidence computed during Build from mapping data.
    detectionConfidence map[string]int // keyed by "pkg/function"
}
```

Expose a `DetectionConfidence(pkg, function string) int` method. The method
returns 0 when the map is nil (before `Build` is called) or when the
function is not found. The caller must use the comma-ok type assertion from
`crap.ContractCoverageProvider` to `*ExternalContractCoverageProvider` to
access this method. If the type assertion fails (e.g., provider is a mock
in tests), confidence remains at the default 0 — safe because the assertion
can only fail in test scenarios, never in the `--analyzer` production path.

### D3: Confidence computation logic

```go
// computeDetectionConfidenceFromMappings computes the assertion detection
// confidence for a specific target function from external analyzer mapping data.
// It counts all mappings targeting (pkg, fn) across all test functions,
// and returns recognized * 100 / total (0 when total is 0).
func computeDetectionConfidenceFromMappings(mappings []protocol.AssertionMappingData, pkg, fn string) int {
    total := 0
    recognized := 0
    for _, m := range mappings {
        if m.TargetPackage != pkg || m.TargetFunction != fn {
            continue
        }
        total++
        if m.AssertionType != "" {
            recognized++
        }
    }
    if total == 0 {
        return 0
    }
    return recognized * 100 / total
}
```

This mirrors the Go-native `computeDetectionConfidence` — same ratio, same
integer truncation, same 0-when-empty behavior. The function computes confidence
per **target function** (aggregating across all test functions mapping to that
target), which matches how `QualityReport` entries are keyed in the external
analyzer path (one report per target function).

### D4: Integration point — `buildExternalQualityReports`

The `gaze quality --analyzer` CLI path calls `buildExternalQualityReports`
in `cmd/gaze/main.go`, which constructs `taxonomy.QualityReport` entries
directly — it does NOT call `quality.Assess`. The `AssertionDetectionConfidence`
field is currently omitted from the struct literal (defaulting to 0).

The fix populates `AssertionDetectionConfidence` directly during report
construction by calling `DetectionConfidence(pkg, function)` on the provider
(via comma-ok type assertion). Summary-level confidence uses arithmetic mean
of per-report values, matching the Go-native aggregation in `quality.Assess`.

Note: `gaze report --analyzer` skips the quality step entirely (quality errors
are set to "skipped: external analyzer mode"), so this fix applies only to
`gaze quality --analyzer`. The `runQualityStep` in `internal/aireport/runner_steps.go`
is not affected since it only uses `goprovider`.

### D5: Test strategy

1. Unit test `computeDetectionConfidenceFromMappings` — table-driven with:
   - Nil/empty mappings → 0
   - All recognized → 100
   - None recognized → 0
   - Mixed recognized/unknown → correct ratio (including integer truncation, e.g. 1/3 = 33)
   - Multiple test functions, filter by name
   - Empty `AssertionType` counts as unrecognized
2. Unit test `DetectionConfidence` method on the provider struct
   - Stored value lookup
   - Unknown function → 0
   - Before `Build` (nil map) → 0 (no panic)
3. Integration test via fake analyzer binary — verify `QualityReport` entries
   have specific expected `AssertionDetectionConfidence` values (not just != 0)

Coverage target: 100% branch coverage for `computeDetectionConfidenceFromMappings`
and `DetectionConfidence`. Integration test validates end-to-end flow through
`buildExternalQualityReports`.

## Risks / Trade-offs

### R1: Semantic parity between Go-native and external confidence

Go-native confidence measures fraction of AST assertion sites with a recognized
pattern kind. External confidence measures fraction of protocol mappings with a
non-empty `AssertionType`. Both answer "what fraction of assertions were
classified?" but use different taxonomies.

**Mitigation**: Acceptable — the values are directly comparable in meaning even
if the underlying classification methods differ. Both produce 0-100 integers with
the same semantics.

### R2: Maintenance coupling with `buildExternalQualityReports`

If `buildExternalQualityReports` changes how reports are constructed, the
confidence population must be updated in the same function.

**Mitigation**: Low risk — the confidence is populated directly in the struct
literal during report construction, so any changes to the construction site
naturally include the confidence field. A code comment documents this coupling.
