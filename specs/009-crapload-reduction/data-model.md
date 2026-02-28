# Data Model: CRAPload Reduction

**Feature**: 009-crapload-reduction
**Date**: 2026-02-27

## Overview

This feature does not introduce new persistent data entities. All changes are structural (code decomposition) or test additions. The key entities below describe the existing domain concepts that the feature's success criteria are measured against, and the new internal structures introduced by decomposition.

## Existing Entities (unchanged)

### CRAPScore

Represents the computed CRAP score for a single function.

| Field | Description |
|-------|-------------|
| Package | Go package path |
| Function | Qualified function name (including receiver) |
| Complexity | Cyclomatic complexity (integer) |
| LineCoverage | Line coverage percentage (0-100) |
| CRAP | Computed CRAP score: `CC² × (1-cov)³ + CC` |
| ContractCoverage | Contract coverage percentage (0-100), optional |
| GazeCRAP | GazeCRAP score using contract coverage, optional |
| Quadrant | Q1/Q2/Q3/Q4 classification based on CRAP and GazeCRAP thresholds |

### CRAPSummary

Aggregate metrics across all analyzed functions.

| Field | Description |
|-------|-------------|
| TotalFunctions | Count of functions analyzed |
| CRAPload | Count of functions with CRAP ≥ threshold |
| GazeCRAPload | Count of functions with GazeCRAP ≥ threshold |
| AvgContractCoverage | Mean contract coverage across scored functions |
| QuadrantCounts | Map of quadrant label → count |

## New Internal Structures

### crapParams (modified — US3)

The existing params struct for `runCrap` gains optional function fields for dependency injection.

| Field | Type | Description |
|-------|------|-------------|
| patterns | list of strings | Package patterns to analyze |
| format | string | Output format: "text" or "json" |
| opts | analysis options | CRAP analysis configuration |
| maxCrapload | integer | CI threshold for CRAPload |
| maxGazeCrapload | integer | CI threshold for GazeCRAPload |
| moduleDir | string | Module root directory |
| stdout | writer | Output destination |
| stderr | writer | Diagnostic output destination |
| analyzeFunc | function (new) | Optional override for CRAP analysis; nil = production default |
| coverageFunc | function (new) | Optional override for contract coverage lookup; nil = production default |

### selfCheckParams (modified — US3)

The existing params struct for `runSelfCheck` gains optional function fields.

| Field | Type | Description |
|-------|------|-------------|
| format | string | Output format |
| maxCrapload | integer | CI threshold |
| maxGazeCrapload | integer | CI threshold |
| stdout | writer | Output destination |
| stderr | writer | Diagnostic output destination |
| moduleRootFunc | function (new) | Optional override for module root discovery; nil = production default |
| runCrapFunc | function (new) | Optional override to delegate to runCrap; nil = production default |

## Relationships

```text
CRAPScore ──belongs to──▸ CRAPSummary
crapParams ──uses──▸ analyzeFunc ──produces──▸ CRAPScore[]
crapParams ──uses──▸ coverageFunc ──provides──▸ ContractCoverage
selfCheckParams ──delegates to──▸ crapParams (via runCrapFunc)
```

## Validation Rules

- `analyzeFunc` and `coverageFunc` fields are optional (nil = use production default). When set, they must match the production function signatures exactly.
- `moduleRootFunc` must return a directory path and an error. Nil means use production `findModuleRoot`.
- Decomposed handler functions must preserve the deduplication invariant: no duplicate effects for the same target within a single function analysis.
