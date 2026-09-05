## Why

`gaze quality --analyzer` always displays "Assertion detection confidence: 0%"
regardless of how many assertions the external analyzer successfully maps.

The root cause: the external adapter pipeline (`ExternalContractCoverageProvider.Build`
→ `buildContractLookup`) produces `crap.ContractCoverageInfo` for the CRAP scoring
path but bypasses `quality.Assess`, which is the only place that calls
`computeDetectionConfidence`. When `gaze quality` runs with an external analyzer,
the `QualityReport` entries are produced by `quality.Assess` but that function has
no access to the external analyzer's mapping data — it relies on its own Go-native
`DetectAssertions` → AST assertion detection, which returns zero sites for non-Go
analyzers.

This means `AssertionDetectionConfidence` is hardcoded to 0 for all external analyzer
quality reports, even when the analyzer successfully identifies and maps assertions.
The field is cosmetic/informational (does not affect contract coverage scores), but
0% is misleading and violates Principle I (Accuracy).

Reported in #251.

## What Changes

### Gaze-side computation from mapping data (Option A — no protocol change)

Add a `computeDetectionConfidenceFromMappings` function to `internal/adapter/contract.go`
that computes assertion detection confidence from `protocol.AssertionMappingData` entries.
The logic mirrors `computeDetectionConfidence`:

- Total = count of mappings for a given test function
- Recognized = count of mappings where `AssertionType` is non-empty (the external
  analyzer successfully classified the assertion pattern)
- Confidence = `Recognized * 100 / Total` (0 when Total is 0)

This is computed per test function, matching the Go-native path where each
`QualityReport` has its own `AssertionDetectionConfidence`.

### Wire confidence into `buildContractLookup`

Extend `buildContractLookup` in `internal/adapter/contract.go` to also compute and
return per-function assertion detection confidence alongside the contract coverage
lookup. This requires a new return type or an additional map.

### Populate confidence in `buildExternalQualityReports`

When `gaze quality` runs with `--analyzer`, the `buildExternalQualityReports` function
in `cmd/gaze/main.go` constructs `QualityReport` entries. The computed detection
confidence is set on each report entry directly, replacing the current zero value.

Note: `gaze report --analyzer` skips the quality step entirely (quality errors are set
to "skipped: external analyzer mode"), so detection confidence is not relevant there.

## Why Not Option B (Protocol Extension)

Adding `assertion_detection_confidence` to the protocol (`TestMappingResult`) would
require all external analyzer implementations to compute and return this value. Since
gaze can derive it deterministically from the mapping data the analyzer already
provides (`AssertionType` field), a protocol extension adds complexity without benefit.
The computation is trivial and should live in gaze.

## Capabilities

### New Capabilities
- `computeDetectionConfidenceFromMappings` function in `internal/adapter/contract.go`
- `DetectionConfidence(pkg, function string) int` method on `ExternalContractCoverageProvider`
- Accurate `AssertionDetectionConfidence` for external analyzer quality reports

### Modified Capabilities
- `ExternalContractCoverageProvider.Build` computes and stores detection confidence
- `buildExternalQualityReports` populates detection confidence from adapter data

### Removed Capabilities
- None

## Risks

### Low
- Semantic mismatch: Go-native `AssertionDetectionConfidence` measures how many
  assertion *patterns* were recognized by Go AST analysis. External analyzer
  confidence measures how many mappings have a non-empty `AssertionType`. The metrics
  are semantically comparable (both answer "what fraction of assertions were
  classified?") but use different classification taxonomies.

### Mitigated
- The fix is purely additive — no existing behavior changes when `--analyzer` is not
  used. The Go-native quality path is untouched.

## Scope

- `internal/adapter/contract.go` — add computation function, store/expose confidence
- `cmd/gaze/main.go` — populate confidence in `buildExternalQualityReports`
- Test files for all modified packages
- No protocol changes
- No CLI flag changes
- No JSON schema changes (field already exists, just defaulting to 0)
