# Data Model: Contract Coverage Gap Remediation

**Branch**: `008-contract-coverage-gaps` | **Date**: 2026-02-27

## Overview

This feature adds test-only code — no new production data types or
schema changes. The data model documents the existing types that tests
will construct, assert on, or interact with.

## Existing Types Used by Tests

### taxonomy.Signal

```text
Signal
├── Source    string   // e.g., "visibility", "godoc", "caller", "naming"
├── Weight   int      // -15 to +20 (varies by signal type)
└── Reasoning string  // human-readable explanation
```

**Validation rules**: Weight is bounded per signal type:
- Visibility: 0 to 20 (clamped)
- Godoc: -15 (incidental) or +15 (contractual) or 0
- Caller: 0, 5, 10, or 15 (tiered by count)
- Naming: -10 to +15 (pattern-based)

### taxonomy.SideEffect

```text
SideEffect
├── ID          string              // unique identifier
├── Type        SideEffectType      // e.g., GlobalMutation, ReturnValue
├── Tier        Tier                // P0, P1, P2, P3, P4
├── Description string              // human-readable description
├── Location    string              // file:line
└── Target      string              // optional: type name, variable name
```

**Validation rules**: Type determines Tier (enforced by `TierOf()`).
P0 = ReturnValue/ErrorReturn. P1 = GlobalMutation/ChannelSend/etc.
P2 = GoroutineSpawn/Panic/FileSystemWrite/etc.

### taxonomy.AnalysisResult

```text
AnalysisResult
├── Target      FunctionTarget      // package + function identification
│   ├── Package       string
│   ├── Function      string
│   ├── QualifiedName string
│   └── Location      string
├── SideEffects []SideEffect        // all detected effects
└── Metadata    Metadata            // run info (version, duration)
```

Used by `renderAnalyzeContent` tests (Group C) to construct inputs.

## Test-Specific Data Patterns

### Signal Test Inputs

| Signal Function | Input Types | Source |
|----------------|-------------|--------|
| AnalyzeVisibilitySignal | `*ast.FuncDecl`, `types.Object`, `SideEffectType` | Loaded from `contracts` fixture |
| AnalyzeGodocSignal | `*ast.FuncDecl`, `SideEffectType` | Hybrid: fixture + programmatic |
| AnalyzeCallerSignal | `types.Object`, `SideEffectType`, `[]*packages.Package` | Loaded from `contracts`+`callers` fixtures |

### Analysis Test Inputs

| Analysis Function | Input Types | Source |
|-------------------|-------------|--------|
| AnalyzeP1Effects | `*token.FileSet`, `*types.Info`, `*ast.FuncDecl`, `string`, `string` | Loaded from `p1effects` fixture |
| AnalyzeP2Effects | Same as above | Loaded from `p2effects` fixture |
| AnalyzeReturns | Same as above | Loaded from `returns` fixture |

All inputs are extracted from `*packages.Package` after loading via
`go/packages`: `pkg.Fset`, `pkg.TypesInfo`, and `FindFuncDecl(pkg, name)`.
