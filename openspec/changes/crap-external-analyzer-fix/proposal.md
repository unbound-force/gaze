## Why

`gaze crap --analyzer snake-eyes --language python .` fails immediately with `Error: no go.mod found`. The `newCrapCmd` function in `cmd/gaze/main.go:929` calls `loader.FindModuleRoot(cwd)` unconditionally before the `--analyzer` flag is even checked. The external analyzer dispatch happens later in `runCrap` at line 501 (`if p.analyzerFlag != ""`), so the command never reaches it.

The quality command already handles this correctly — `runQualityWithExternalAnalyzer` (line 1179) uses `os.Getwd()` instead of `FindModuleRoot`, and `runQuality` checks the `--analyzer` flag before resolving the module root.

This blocks external analyzer users entirely (Constitution Principle II: Minimal Assumptions — Gaze must not assume a Go module exists when analyzing non-Go projects). It also makes the `--analyzer` and `--language` flags on `gaze crap` effectively non-functional.

Ref: https://github.com/unbound-force/gaze/issues/250

## What Changes

Defer the `loader.FindModuleRoot` call from `newCrapCmd`'s `RunE` closure into `runCrap`, after the `--analyzer` flag check. When `--analyzer` is set, use `os.Getwd()` for `moduleDir` (matching the quality command pattern). When it's not set, call `FindModuleRoot` as before.

## Capabilities

### New Capabilities

- None

### Modified Capabilities

- `newCrapCmd` (`cmd/gaze/main.go`): No longer calls `loader.FindModuleRoot` in the `RunE` closure. Passes `cwd` (from `os.Getwd()`) as `moduleDir` in `crapParams`.
- `runCrap` (`cmd/gaze/main.go`): Now calls `loader.FindModuleRoot(p.moduleDir)` at the start of the Go-native path (after the `--analyzer` dispatch), replacing `p.moduleDir` with the resolved module root before downstream use.

### Removed Capabilities

- None

## Impact

- **`cmd/gaze/main.go`**: Two changes — `newCrapCmd`'s `RunE` closure simplified (remove `FindModuleRoot` call, pass `cwd` as `moduleDir`), and `runCrap` gains a `FindModuleRoot` call after the `--analyzer` check for the Go-native path.
- No signature changes to exported functions.
- No new dependencies.
- Fixes `gaze crap --analyzer <binary> --language <lang> .` for non-Go projects.
- Closes #250.

## Constitution Alignment

Assessed against the Gaze project constitution.

### I. Accuracy

**Assessment**: PASS

This fix does not change analysis behavior. It only changes when `FindModuleRoot` is called — deferred to the Go-native path where a Go module is actually required. The external analyzer path receives `cwd` (the user's working directory), which is the correct root for non-Go projects. All existing Go-native CRAP analysis continues to use the resolved module root.

### II. Minimal Assumptions

**Assessment**: PASS

This is the primary principle this fix addresses. The current code unconditionally assumes a Go module exists, violating Principle II for external analyzer users. After this fix, `FindModuleRoot` is only called when Go-specific analysis is requested — external analyzers operate without Go module assumptions.

### III. Actionable Output

**Assessment**: PASS

No change to output formats or content. The fix removes a premature error (`Error: no go.mod found`) that prevented users from reaching the external analyzer path entirely.

### IV. Testability

**Assessment**: PASS

The `crapParams` struct and `runCrap`/`runCrapWithExternalAnalyzer` functions remain fully testable via dependency injection. The fix moves `FindModuleRoot` into the tested function body rather than the untestable Cobra closure, improving testability. A new test verifies the external analyzer path no longer requires `FindModuleRoot`.
