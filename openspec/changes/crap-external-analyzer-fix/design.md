## Context

`newCrapCmd` (`cmd/gaze/main.go:924-957`) builds a `crapParams` struct in its `RunE` closure and passes it to `runCrap`. At line 929, it unconditionally calls `loader.FindModuleRoot(cwd)`, which fails with `"no go.mod found"` when run from a non-Go project directory. This blocks the external analyzer path (`runCrapWithExternalAnalyzer`, line 588) which does not need a Go module.

The quality command handles this correctly: `runQuality` (line 1097) checks `p.analyzerFlag != ""` at line 1106 before resolving `moduleDir` at line 1111, and `runQualityWithExternalAnalyzer` (line 1179) uses `os.Getwd()` directly at line 1180. The crap command should follow the same pattern.

## Goals / Non-Goals

### Goals

- Defer `loader.FindModuleRoot` until after the `--analyzer` flag is checked, so external analyzer users are not blocked by Go module requirements
- Use `os.Getwd()` as `moduleDir` for the external analyzer path, matching the quality command pattern
- Add a test verifying the external analyzer path does not call `FindModuleRoot`
- Close #250

### Non-Goals

- Removing the `moduleDir` field from `crapParams` entirely (it is used by the Go-native path and passed through to `crap.Analyze` and `resolveBaselineAndCompare`)
- Refactoring to match the quality command's exact structure (quality resolves `moduleDir` inside `runQuality`/`runQualityWithExternalAnalyzer` rather than in the params struct — the crap command can achieve the same fix with less churn by resolving in `runCrap`)

## Decisions

### D1: Move FindModuleRoot into runCrap, after the --analyzer check

In `newCrapCmd`'s `RunE` closure, replace `loader.FindModuleRoot(cwd)` with just `os.Getwd()`. Pass `cwd` as `crapParams.moduleDir`. In `runCrap`, after the `if p.analyzerFlag != ""` dispatch (line 501), call `loader.FindModuleRoot(p.moduleDir)` to resolve the actual Go module root for the Go-native path.

This minimizes the diff while fixing the bug. The external analyzer path (`runCrapWithExternalAnalyzer`) receives `cwd` unchanged, which is the correct behavior — external analyzers work from the user's current directory, not a Go module root.

### D2: runCrapWithExternalAnalyzer uses moduleDir as-is

`runCrapWithExternalAnalyzer` already uses `p.moduleDir` at two sites: `initExternalSession` (line 590) and `crap.Analyze` (line 602). After D1, `p.moduleDir` is `cwd` (the user's working directory). This is correct and matches the quality command pattern where `runQualityWithExternalAnalyzer` calls `os.Getwd()` at line 1180.

No changes needed inside `runCrapWithExternalAnalyzer`.

### D3: Error handling for FindModuleRoot stays in runCrap

`FindModuleRoot` can fail when there is no `go.mod`. This error is now returned from `runCrap` instead of from the Cobra closure. The error message and wrapping remain identical: `fmt.Errorf("finding module root: %w", err)`. From the user's perspective, the error is the same — it just only occurs when Go-native analysis is requested.

## Coverage Strategy

Three unit tests in `cmd/gaze/main_test.go` covering all three spec scenarios:

1. **External analyzer bypass** (Scenario 1): Call `runCrap` with `analyzerFlag` set, `moduleDir` set to a directory without `go.mod`. Assert the error is non-nil, does NOT contain `"finding module root"`, and DOES contain an analyzer-related error. Proves the external analyzer path bypasses `FindModuleRoot`.

2. **FindModuleRoot failure** (Scenario 3): Call `runCrap` without `analyzerFlag`, `moduleDir` set to a directory without `go.mod`. Assert the error contains `"finding module root"`. Proves the error wrapping is preserved in the new call site.

3. **Go-native regression** (Scenario 2): Existing tests that call `runCrap` with valid Go module directories must continue to pass unchanged. The new `FindModuleRoot` call inside `runCrap` is idempotent when given a module root directory. This relies on `FindModuleRoot` being idempotent when called with a directory that is already a module root (it returns the same path).

## Risks / Trade-offs

### R1: FindModuleRoot moves from Cobra closure to runCrap

Previously, `FindModuleRoot` ran before any `runCrap` logic (e.g., `crap.DefaultOptions()`, provider construction). After this fix, `FindModuleRoot` runs inside `runCrap`, after options are already built. This is safe because `FindModuleRoot` is a pure filesystem lookup with no dependencies on options or providers — it only needs a starting directory path.

### R2: moduleDir in crapParams changes meaning

Before: `moduleDir` always contained the resolved Go module root. After: `moduleDir` contains `cwd` and is resolved to the module root inside `runCrap` for the Go-native path only. Any test code constructing `crapParams` directly with a `moduleDir` value continues to work — the value is overwritten inside `runCrap` for Go-native analysis. Tests using `analyzerFlag` get the passed `moduleDir` unchanged, which is the intended behavior.
