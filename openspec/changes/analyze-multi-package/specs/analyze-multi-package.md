## ADDED Requirements

### Requirement: Multi-package pattern support for analyze

The `gaze analyze` command MUST accept one or more package patterns as positional arguments. When a pattern (e.g., `./...`) resolves to multiple packages, the command MUST analyze all matched packages and merge results into a single output.

#### Scenario: Wildcard pattern analyzes all packages

- **GIVEN** a Go module with packages `./a` and `./b`, each containing exported functions
- **WHEN** the user runs `gaze analyze ./...`
- **THEN** the output MUST include analysis results for functions from both `./a` and `./b`

#### Scenario: Multiple explicit patterns

- **GIVEN** a Go module with packages `./internal/crap` and `./internal/loader`
- **WHEN** the user runs `gaze analyze ./internal/crap ./internal/loader`
- **THEN** the output MUST include analysis results for functions from both packages

#### Scenario: Single package backward compatibility

- **GIVEN** a Go module with package `./internal/crap`
- **WHEN** the user runs `gaze analyze ./internal/crap`
- **THEN** the output MUST contain the same analysis results (same function names, same side effects, same exit code 0) as the current single-argument invocation
- **AND** existing tests MUST pass with only the mechanical change from `pkgPath` to `patterns`

#### Scenario: Function filter with multiple packages

- **GIVEN** a Go module with packages `./a` and `./b`, where `./a` contains `Foo` and `./b` contains `Bar`
- **WHEN** the user runs `gaze analyze --function Foo ./...`
- **THEN** the output MUST include only `Foo` from `./a` and MUST NOT include `Bar` from `./b`

#### Scenario: No functions found across all packages

- **GIVEN** a Go module with packages `./a` and `./b`, where `--function Baz` matches no function in either package
- **WHEN** the user runs `gaze analyze --function Baz ./...`
- **THEN** the command MUST return an error indicating the function was not found

#### Scenario: Classification with multi-package

- **GIVEN** packages `./a` and `./b` with functions producing side effects
- **WHEN** the user runs `gaze analyze --classify ./...`
- **THEN** classification results MUST include effects from both packages
- **AND** each effect's classification MUST use the correct target package context

#### Scenario: Include-unexported auto-detection per package

- **GIVEN** packages `./cmd/tool` (package main) and `./internal/lib` (library package)
- **WHEN** the user runs `gaze analyze ./...` without `--include-unexported`
- **THEN** unexported functions in `./cmd/tool` MUST be included automatically (main package auto-detection)
- **AND** only exported functions in `./internal/lib` MUST be included

#### Scenario: Invalid package pattern

- **GIVEN** a pattern that does not match any Go packages
- **WHEN** the user runs `gaze analyze ./nonexistent/...`
- **THEN** the command MUST return an error with an actionable message indicating no packages matched

### Requirement: Multi-package pattern support for quality

The `gaze quality` command MUST accept one or more package patterns as positional arguments. When a pattern resolves to multiple packages, the command MUST assess all matched packages and merge results into a single output.

#### Scenario: Wildcard pattern assesses all packages

- **GIVEN** a Go module with packages `./a` and `./b`, both with test files
- **WHEN** the user runs `gaze quality ./...`
- **THEN** the output MUST include quality reports for test-target pairs from both packages

#### Scenario: Package without tests is skipped gracefully

- **GIVEN** a Go module with packages `./a` (has tests) and `./b` (no tests)
- **WHEN** the user runs `gaze quality ./...`
- **THEN** the output MUST include quality reports for `./a` and MUST log a warning for `./b` being skipped

#### Scenario: Include-unexported auto-detection per package for quality

- **GIVEN** packages `./cmd/tool` (package main) and `./internal/lib` (library package), both with tests
- **WHEN** the user runs `gaze quality ./...` without `--include-unexported`
- **THEN** quality assessment for `./cmd/tool` MUST include unexported functions automatically
- **AND** quality assessment for `./internal/lib` MUST include only exported functions

### Requirement: Loader multi-package warning

When `loader.Load` is called with a pattern that resolves to more than one package, it MUST log a warning indicating the number of packages resolved and that only the first is used.

#### Scenario: Loader warns on multi-package resolution

- **GIVEN** a wildcard pattern `./...` that resolves to 5 packages
- **WHEN** `loader.Load("./...")` is called
- **THEN** a warning MUST be logged: "pattern resolved to N packages, using only the first; consider using resolvePackagePaths for multi-package analysis"
- **AND** the function MUST return the first package (existing behavior preserved)

### Requirement: Export ResolvePackagePaths

The `resolvePackagePaths` function in `internal/crap/contract.go` MUST be exported as `ResolvePackagePaths` so that `cmd/gaze` can call it for the `analyze` and `quality` commands without duplicating the pattern expansion logic.

#### Scenario: Exported function maintains existing behavior

- **GIVEN** a pattern slice `[]string{"./..."}` for a module with 3 packages
- **WHEN** `crap.ResolvePackagePaths(patterns, moduleDir)` is called
- **THEN** it MUST return 3 deduplicated, non-test-variant package paths
- **AND** the existing internal callers (`BuildContractCoverageFunc`) MUST continue to work unchanged

## MODIFIED Requirements

### Requirement: Analyze command argument handling

Previously: `gaze analyze` accepted exactly one positional argument (`cobra.ExactArgs(1)`).

The `analyze` command MUST accept one or more positional arguments (`cobra.MinimumNArgs(1)`). The `Use` string MUST be updated from `"analyze [package]"` to `"analyze [packages...]"`.

### Requirement: Quality command argument handling

Previously: `gaze quality` accepted exactly one positional argument (`cobra.ExactArgs(1)`).

The `quality` command MUST accept one or more positional arguments (`cobra.MinimumNArgs(1)`). The `Use` string MUST be updated from `"quality [package]"` to `"quality [packages...]"`.

### Requirement: README known limitations

Previously: README listed "Single package loading. The analyze command processes one package at a time."

This entry MUST be removed from the Known Limitations section.

### Requirement: CLI reference documentation

Previously: `docs/reference/cli/analyze.md` stated "Exactly one package argument is required."

The documentation MUST be updated to reflect multi-package support. Same for `docs/reference/cli/quality.md`.

## REMOVED Requirements

### Requirement: Single-package-only analyze behavior

The restriction that `gaze analyze` processes only one package is removed. Rationale: the single-package limitation was a known deficiency (#107), not a design constraint. The multi-package pattern is already established in `crap` and `report`.
