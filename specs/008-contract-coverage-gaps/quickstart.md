# Quickstart: Contract Coverage Gap Remediation

**Branch**: `008-contract-coverage-gaps` | **Date**: 2026-02-27

## Prerequisites

- Go 1.24+
- Gaze binary built: `go build ./cmd/gaze`

## Verification Steps

### 1. Baseline Measurement (Before Implementation)

```bash
# Measure current weighted average contract coverage (FR-020)
gaze quality --format=json ./... | jq '.summary.weighted_average_contract_coverage'

# Record baseline GazeCRAPload (FR-023)
gaze crap --format=json ./... | jq '.summary.gazecrapload'

# Run spec 007 quickstart validation (FR-021)
go test -race -count=1 -run TestSC003_MappingAccuracy ./internal/quality/...
```

### 2. Run Group A Tests (Classification Signals)

```bash
# AnalyzeVisibilitySignal (FR-001..FR-003)
go test -race -count=1 -run TestAnalyzeVisibilitySignal ./internal/classify/...

# AnalyzeGodocSignal (FR-004..FR-006)
go test -race -count=1 -run TestAnalyzeGodocSignal ./internal/classify/...

# AnalyzeCallerSignal (FR-007..FR-009)
go test -race -count=1 -run TestAnalyzeCallerSignal ./internal/classify/...
```

### 3. Run Group B Tests (Analysis Core)

```bash
# AnalyzeP1Effects direct (FR-010..FR-011)
go test -race -count=1 -run TestAnalyzeP1Effects_Direct ./internal/analysis/...

# AnalyzeP2Effects direct (FR-012)
go test -race -count=1 -run TestAnalyzeP2Effects_Direct ./internal/analysis/...

# AnalyzeReturns direct (FR-013)
go test -race -count=1 -run TestAnalyzeReturns_Direct ./internal/analysis/...
```

### 4. Run Group C Tests (CLI Layer)

```bash
# renderAnalyzeContent (FR-016..FR-017)
go test -race -count=1 -run TestRenderAnalyzeContent ./cmd/gaze/...

# buildContractCoverageFunc (FR-018..FR-019)
go test -race -count=1 -run TestBuildContractCoverageFunc ./cmd/gaze/...
```

### 5. Full Suite Verification (SC-005)

```bash
go test -race -count=1 -short ./...
```

### 6. Post-Change Measurement (After Implementation)

```bash
# Re-measure contract coverage (FR-022) — should be higher
gaze quality --format=json ./... | jq '.summary.weighted_average_contract_coverage'

# Re-measure GazeCRAPload (FR-023) — should be lower
gaze crap --format=json ./... | jq '.summary.gazecrapload'

# Verify AnalyzeP1Effects/P2Effects exit Q4 (SC-003)
gaze crap --format=json ./... | jq '.functions[] | select(.function | test("AnalyzeP[12]Effects")) | {function, quadrant, gazecrap}'
```

### Expected Outcomes

| Metric | Before | After | Direction |
|--------|--------|-------|-----------|
| Weighted avg contract coverage | baseline | higher | Up |
| GazeCRAPload | baseline | lower | Down |
| AnalyzeP1Effects quadrant | Q4 (Dangerous) | Q1 or Q2 | Improved |
| AnalyzeP2Effects quadrant | Q4 (Dangerous) | Q1 or Q2 | Improved |
| Full test suite | PASS | PASS | No regressions |
