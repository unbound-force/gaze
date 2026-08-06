## Context

`cmd/gaze/main.go` (1,953 lines) contains four duplicated patterns
confirmed during triage of GitHub issue #199:

1. Format validation if-statement — 4 identical copies
2. JSON capture function — exact clone across `cmd/gaze` and `internal/aireport`
3. Main-package auto-detect block — 2 identical 3-line blocks
4. `loadGazeConfigBestEffort` — local copy that duplicates `config.LoadFromDir`

All four are pure structural duplication with no semantic variation between
occurrences. The refactor is safe to execute mechanically.

## Goals / Non-Goals

### Goals
- Create `internal/cliutil` package with `ValidateFormat` and `CaptureJSON`
- Remove `captureReportJSON` from `cmd/gaze/main.go`
- Remove `captureJSON` from `internal/aireport/runner.go`
- Remove `loadGazeConfigBestEffort` from `cmd/gaze/main.go`; replace 3
  call sites with `config.LoadFromDir`
- Extract `autoDetectMainPkg` as a private helper within `cmd/gaze/main.go`
- All existing tests pass without modification
- No behaviour changes

### Non-Goals
- Module root discovery consolidation (contextual, low ROI)
- Format dispatch switch consolidation (not duplication — each handles
  different types)
- Any new features or flag changes
- Changes to test fixtures or JSON schema

## Decisions

### D1: `internal/cliutil` for shared utilities

`ValidateFormat` and `CaptureJSON` are placed in a new `internal/cliutil`
package rather than `internal/report`. `internal/report` owns output
formatters; a generic I/O capture helper and format validator are a
different concern. `cliutil` is honest about what it contains and can
absorb future CLI-layer utilities without stretching an existing package's
cohesion. See proposal for full tradeoff analysis.

**Package API:**
```go
package cliutil

// ValidateFormat returns an error if format is not "text" or "json".
func ValidateFormat(format string) error

// CaptureJSON executes fn into a buffer and returns the result as
// json.RawMessage. Returns an error if fn fails.
func CaptureJSON(fn func(w io.Writer) error) (json.RawMessage, error)
```

### D2: `autoDetectMainPkg` stays private in `cmd/gaze`

The auto-detect block imports `internal/loader` (for `IsMainPkg`) and
calls the package-level logger. Placing it in `cliutil` would require
`cliutil` to import `internal/loader` and the logger, making it less pure.
Since only 2 call sites exist (both in `cmd/gaze`), a private helper in
`cmd/gaze/main.go` is the right scope.

**Helper signature:**
```go
func autoDetectMainPkg(pkgPath string, includeUnexported *bool)
```

### D3: `loadGazeConfigBestEffort` replaced by `config.LoadFromDir`

`config.LoadFromDir` already exists and is semantically identical.
`internal/aireport/runner_steps.go` and `internal/provider/goprovider/contract.go`
already use it. The local copy in `cmd/gaze/main.go` is deleted; its 3 call
sites call `config.LoadFromDir` directly. No wrapper needed.

### D4: `captureJSON` deleted from `internal/aireport/runner.go`

The function is an internal implementation detail with no callers outside
`runner.go`. After migrating the one call site to `cliutil.CaptureJSON`,
the local copy is deleted. The `cmd/gaze` copy (`captureReportJSON`) is
also deleted and its call site updated.

### D5: Test coverage for `internal/cliutil`

`ValidateFormat` and `CaptureJSON` are pure functions with no external
dependencies. Unit tests in `internal/cliutil/cliutil_test.go` cover:
- `ValidateFormat`: valid inputs ("text", "json"), invalid input, error
  message format
- `CaptureJSON`: success path, error propagation, empty output

## Risks / Trade-offs

**Risk**: `internal/aireport` now depends on `internal/cliutil`.
This is a new dependency edge. Mitigation: `cliutil` has no internal
imports, so it cannot create a cycle. The dependency graph is acyclic.

**Risk**: A caller of `captureJSON` in `internal/aireport` is missed.
Mitigation: grep confirms a single call site in `runner.go` before deletion.

**Trade-off**: `autoDetectMainPkg` is not in `cliutil`, so it cannot be
unit-tested in isolation. Accepted: it is 3 lines, the logic is trivial,
and the behaviour is covered by existing integration tests.
