## Context

`gaze analyze` and `gaze quality` accept exactly one argument (`cobra.ExactArgs(1)`) and pass it as a single `pkgPath string` to their runner functions. When that argument is a wildcard pattern like `./...`, `loader.Load` resolves it to multiple packages but silently takes only `pkgs[0]`, discarding the rest. This produces incomplete results with no warning.

Meanwhile, `gaze crap` and `gaze report` already handle multi-package patterns correctly. They use `resolvePackagePaths` (in `internal/crap/contract.go:161-181`) to expand patterns into individual fully-qualified package paths, then iterate and call `analysis.LoadAndAnalyze` per package. This proven pattern is the foundation for this design.

## Goals / Non-Goals

### Goals
- Enable `gaze analyze` to accept one or more package patterns and analyze all matched packages
- Enable `gaze quality` to accept one or more package patterns and assess all matched packages
- Add a warning log to `loader.Load` when multiple packages are resolved (defense-in-depth for future callers)
- Remove the "Single package loading" known limitation from README
- Maintain full backward compatibility with existing single-package invocations

### Non-Goals
- Consolidating the duplicated `resolvePackagePaths` across `internal/crap/contract.go` and `internal/aireport/runner_steps.go` (tracked separately in specs/022 tasks)
- Adding parallel package analysis (sequential iteration is sufficient and matches the `crap`/`report` pattern)
- Changing the `loader.Load` return type from `*Result` to `[]*Result` (callers iterate and call `Load` per resolved path, which is the established pattern)
- Modifying the `gaze report` command (already handles multi-package correctly)

## Decisions

### D1: Reuse `resolvePackagePaths` from `internal/crap/contract.go`

Both `analyze` and `quality` will use the existing `resolvePackagePaths` function to expand pattern arguments into individual package paths. This function is already battle-tested in `crap` and `report`, handles test-variant filtering and deduplication, and uses the lightweight `packages.NeedName` mode for pattern resolution.

To avoid import cycles (`cmd/gaze` already imports `internal/crap`), the commands will call `resolvePackagePaths` through `internal/crap` (which is already imported by `cmd/gaze/main.go`). Since `resolvePackagePaths` is currently unexported, it will be exported as `ResolvePackagePaths`.

**Rationale**: Reusing proven code reduces risk and avoids introducing a third copy. Exporting the function is appropriate since it implements a general-purpose pattern expansion utility, and the existing NOTE comment already acknowledges that it should be consolidated.

### D2: Change `analyzeParams.pkgPath` to `patterns []string`

The `analyzeParams` struct will change from `pkgPath string` to `patterns []string`, matching the existing `crapParams.patterns` field. The `runAnalyze` function will iterate over resolved package paths, calling `analysis.LoadAndAnalyze` per package and appending results.

The same change applies to `qualityParams`.

**Rationale**: This mirrors the pattern already used by `crapParams` and `reportParams`, maintaining consistency across all four commands.

### D3: Change `cobra.ExactArgs(1)` to `cobra.MinimumNArgs(1)`

Both `analyze` and `quality` Cobra commands will accept one or more arguments. This is backward-compatible: existing `gaze analyze ./internal/crap` invocations continue to work because a single argument satisfies `MinimumNArgs(1)`.

**Rationale**: Matches `gaze crap` which already uses `cobra.MinimumNArgs(1)`.

### D4: Add warning log to `loader.Load` for multi-package resolution

When `loader.Load` receives a pattern that resolves to `len(pkgs) > 1`, it will log a warning naming the number of packages resolved and the fact that only the first is used. This is defense-in-depth — after this change, no existing caller should hit this path (they all resolve patterns first), but it prevents silent truncation for any future caller.

**Rationale**: Defense-in-depth. The silent truncation is the root cause of #107, and even after fixing the callers, the loader should be honest about what it does.

### D5: Merged output for multi-package results

For `analyze`: Results from all packages are merged into a single `[]taxonomy.AnalysisResult` slice. Text output renders all functions in a single stream (same as today, just more functions). JSON output is a single JSON array containing all analysis results (same schema, same structure — just more entries).

For `quality`: Reports from all packages are merged into a single `[]taxonomy.QualityReport` slice. The summary aggregates across packages. Text output renders all packages. JSON output is a single array.

**Rationale**: This matches user expectations — `./...` means "everything" and the output should be a single coherent report, not fragmented per-package files.

### D6: `--function` flag interaction with multi-package

The `--function` / `-f` flag on `analyze` filters to a specific function name. With multi-package support, if the named function exists in multiple packages, all matches will be returned. If it exists in none, the error message will list all searched packages.

**Rationale**: The function filter is applied per-package by `analysis.LoadAndAnalyze` via `Options.FunctionFilter`. No special handling is needed — the iteration naturally handles this.

### D7: `--classify` flag per-package iteration

When `--classify` is enabled, `runClassify` must be called per package because it calls `loader.Load(pkgPath)` for the target package's AST. The `loader.LoadModule(cwd)` call inside `runClassify` loads the entire module and is the same across all packages — it should be called once outside the loop and reused. Each per-package classify call receives a fully-qualified package path (not a wildcard), so the `loader.Load` truncation issue does not apply.

**Rationale**: Per-package classification is required because the target package context (AST, type info) differs per package. Module loading is constant and can be amortized.

### D8: `--include-unexported` auto-detection per package

When iterating over resolved packages, `--include-unexported` auto-detection for `package main` must be applied per package, not globally. A module may contain both library packages and `cmd/` packages. The `quality` command already has per-package `isMainPackage` auto-detection; the `analyze` command currently applies the flag globally. With multi-package support, `analyze` should apply the same per-package auto-detection that `quality` uses.

**Rationale**: If a user runs `gaze analyze ./...` on a module with both library and `cmd/` packages, unexported functions in `main` packages should be auto-included without requiring `--include-unexported`. This prevents false-negative-by-omission — the same class of bug this change fixes.

### D9: Quality summary merge strategy

Each package produces its own `[]taxonomy.QualityReport` and `*taxonomy.PackageSummary`. Reports from all packages are concatenated into a single `[]taxonomy.QualityReport` array. The summary is recomputed from the merged reports to produce aggregate contract coverage and assertion counts. Text output renders all packages in sequence. JSON output is a single `quality_reports` array with a single aggregated `quality_summary`. The JSON schema is unchanged — consumers already handle results with multiple target functions from the same package; multi-package simply adds more entries.

## Risks / Trade-offs

### Risk: Performance on large monorepos

Analyzing `./...` on a large monorepo could be slow because each package requires full type information loading (`LoadMode` includes `NeedTypes`, `NeedSyntax`, `NeedTypesInfo`). This is acceptable because:
1. `gaze crap` and `gaze report` already do this with no complaints.
2. Users can narrow the pattern (`./internal/...`) if needed.
3. Sequential iteration is simpler and debuggable; parallelism can be added later if benchmarks warrant it.

### Risk: Memory usage with many packages

Each `LoadAndAnalyze` call loads a full package with type info. For a module with 50+ packages, this could consume significant memory. However, results from each package are simple value types (`[]taxonomy.AnalysisResult`), and the loaded package is eligible for GC after each iteration.

### Trade-off: Exporting `resolvePackagePaths`

Exporting a function from `internal/crap` increases the package's API surface. However, the function is a general utility (pattern resolution) not specific to CRAP scoring, and the existing NOTE comment already acknowledges it should be consolidated. This is a step toward that consolidation.

### Trade-off: Not fixing `loader.Load` to return all packages

We chose not to change `loader.Load`'s return type because all callers already have the iterate-per-package pattern. Changing the return type would require updating every caller and the `Result` struct, with no functional benefit since callers need per-package granularity anyway. The warning log is sufficient defense-in-depth.

## Coverage Strategy

- **Unit tests**: `ResolvePackagePaths` export (pattern expansion, deduplication, test-variant filtering). Loader warning emission (behavior-based: verify first package is returned when multiple resolve, and use `fmt.Fprintf` to an injectable stderr writer for testable warning output).
- **Integration tests**: `runAnalyze` with multiple package paths (verify results from all packages appear). `runQuality` with multiple package paths (verify reports from all packages appear). `runAnalyze --classify` with multiple packages (verify classification context uses correct per-package target). Error case: `--function` filter with no matches across multiple packages.
- **No e2e tests required**: Multi-package behavior is exercised by manual verification tasks and covered by the existing e2e self-check. The JSON schema is unchanged — multi-package output is the same array structure with more entries.
- **Coverage target**: All new branches in the `runAnalyze` and `runQuality` iteration loops must be covered by tests. The `ResolvePackagePaths` function already has tests; the export rename must update those tests.
