# Delta Spec: Extract Shared CLI Helpers

## ADDED Requirements

### Requirement: Format Validation Helper

The `internal/cliutil` package MUST export a `ValidateFormat` function
with signature `func ValidateFormat(format string) error`.

The function MUST return `nil` when `format` is `"text"` or `"json"`.

The function MUST return a non-nil error for any other value. The error
message MUST include the invalid format value and the list of valid
options, matching the existing message format:
`invalid format %q: must be 'text' or 'json'`.

#### Scenario: Valid format accepted

- **GIVEN** a format string of `"text"` or `"json"`
- **WHEN** `cliutil.ValidateFormat(format)` is called
- **THEN** the return value MUST be `nil`

#### Scenario: Invalid format rejected

- **GIVEN** a format string that is not `"text"` or `"json"` (e.g., `"csv"`, `""`)
- **WHEN** `cliutil.ValidateFormat(format)` is called
- **THEN** the return value MUST be a non-nil error containing the invalid value

---

### Requirement: JSON Capture Helper

The `internal/cliutil` package MUST export a `CaptureJSON` function
with signature `func CaptureJSON(fn func(w io.Writer) error) (json.RawMessage, error)`.

The function MUST execute `fn` with a `bytes.Buffer` and return the
buffer contents as `json.RawMessage` on success.

The function MUST propagate any error returned by `fn` without
modification, returning `nil` for the `json.RawMessage`.

#### Scenario: Successful JSON capture

- **GIVEN** a writer function that writes valid JSON to the provided `io.Writer`
- **WHEN** `cliutil.CaptureJSON(fn)` is called
- **THEN** the returned `json.RawMessage` MUST contain the bytes written by `fn`
- **AND** the returned error MUST be `nil`

#### Scenario: Writer function returns error

- **GIVEN** a writer function that returns a non-nil error
- **WHEN** `cliutil.CaptureJSON(fn)` is called
- **THEN** the returned `json.RawMessage` MUST be `nil`
- **AND** the returned error MUST be the error from `fn`

#### Scenario: Empty output

- **GIVEN** a writer function that writes zero bytes and returns `nil`
- **WHEN** `cliutil.CaptureJSON(fn)` is called
- **THEN** the returned `json.RawMessage` MUST be an empty `json.RawMessage`
- **AND** the returned error MUST be `nil`

---

### Requirement: Unit Tests for cliutil

The `internal/cliutil` package MUST have a `cliutil_test.go` file with
table-driven tests covering all scenarios listed above.

Tests MUST use the standard library `testing` package only — no
third-party assertion libraries.

#### Scenario: Full test coverage

- **GIVEN** the `internal/cliutil` package
- **WHEN** `go test ./internal/cliutil/...` is run
- **THEN** all tests MUST pass
- **AND** line coverage SHOULD be 100%

---

### Requirement: Main Package Auto-Detect Helper

`cmd/gaze/main.go` MUST contain a private function `autoDetectMainPkg`
that encapsulates the main-package detection and `includeUnexported`
auto-enable logic.

The function MUST call `loader.IsMainPkg(pkgPath)` and, when true, set
the pointed-to boolean to `true` and log an info message.

The function MUST NOT modify the boolean if `loader.IsMainPkg` returns
`false`.

The function MUST NOT modify the boolean if it is already `true`
(guard against redundant logging).

#### Scenario: Main package detected

- **GIVEN** `pkgPath` resolves to a `package main`
- **AND** the pointed-to boolean is `false`
- **WHEN** `autoDetectMainPkg(pkgPath, &includeUnexported)` is called
- **THEN** the pointed-to boolean MUST be set to `true`

#### Scenario: Non-main package

- **GIVEN** `pkgPath` does not resolve to a `package main`
- **WHEN** `autoDetectMainPkg(pkgPath, &includeUnexported)` is called
- **THEN** the pointed-to boolean MUST remain unchanged

#### Scenario: Already enabled

- **GIVEN** the pointed-to boolean is already `true`
- **WHEN** `autoDetectMainPkg(pkgPath, &includeUnexported)` is called
- **THEN** the function MUST NOT modify the boolean (no-op)

---

## MODIFIED Requirements

### Requirement: Format Validation in CLI Commands

Previously: Each of `runAnalyze`, `runCrap`, `runQuality`, and
`runSelfCheck` contained an inline if-statement for format validation.

The four inline format validation blocks MUST be replaced with calls to
`cliutil.ValidateFormat`. The error message format and the set of valid
values MUST remain identical to the current behaviour.

#### Scenario: Existing commands reject invalid format

- **GIVEN** a user invokes `gaze analyze --format=csv`
- **WHEN** the command runs
- **THEN** the error message MUST match the current format:
  `invalid format "csv": must be 'text' or 'json'`

---

### Requirement: JSON Capture in CLI and Report Pipeline

Previously: `cmd/gaze/main.go` defined `captureReportJSON` and
`internal/aireport/runner.go` defined `captureJSON` — identical
implementations under different names.

Both local definitions MUST be removed. All call sites MUST use
`cliutil.CaptureJSON` instead.

`internal/aireport/runner.go` MUST import `internal/cliutil` and call
`cliutil.CaptureJSON` where it previously called `captureJSON`.

`cmd/gaze/main.go` MUST import `internal/cliutil` and call
`cliutil.CaptureJSON` where it previously called `captureReportJSON`.

#### Scenario: Report pipeline JSON capture unchanged

- **GIVEN** a `gaze report` invocation producing JSON output
- **WHEN** the report pipeline runs
- **THEN** the JSON output MUST be byte-identical to the pre-refactor output

---

### Requirement: Config Loading in CLI

Previously: `cmd/gaze/main.go` defined `loadGazeConfigBestEffort`
(lines 800–809), byte-identical to `config.LoadFromDir`.

The local `loadGazeConfigBestEffort` function MUST be deleted. The
three call sites (in `initExternalSession`, `resolveBaselinePath`,
`loadAndCompare`) MUST be replaced with `config.LoadFromDir`.

#### Scenario: Config loading behaviour unchanged

- **GIVEN** a `.gaze.yaml` file exists in the module root
- **WHEN** any of the three call sites executes
- **THEN** the returned `*config.GazeConfig` MUST be identical to the
  pre-refactor result

#### Scenario: Missing config file

- **GIVEN** no `.gaze.yaml` file exists
- **WHEN** any of the three call sites executes
- **THEN** `config.DefaultConfig()` MUST be returned (no error)

---

### Requirement: Main Package Auto-Detect in CLI Commands

Previously: `runAnalyze` (line 269) and `runQuality` (line 1126)
contained identical 3-line blocks for main-package auto-detection.

Both inline blocks MUST be replaced with calls to `autoDetectMainPkg`.
Observable behaviour (logging, flag mutation) MUST remain identical.

#### Scenario: Analyze detects main package

- **GIVEN** `gaze analyze ./cmd/myapp` where `cmd/myapp` is `package main`
- **WHEN** `runAnalyze` executes
- **THEN** unexported functions MUST be included in the analysis output

---

## REMOVED Requirements

### Requirement: `captureReportJSON` in `cmd/gaze/main.go`

Removed because it is an exact duplicate of functionality now provided
by `cliutil.CaptureJSON`. Keeping it would violate DRY and risk
divergence.

### Requirement: `captureJSON` in `internal/aireport/runner.go`

Removed because it is an exact duplicate of functionality now provided
by `cliutil.CaptureJSON`. The `internal/aireport` package MUST import
`internal/cliutil` instead.

### Requirement: `loadGazeConfigBestEffort` in `cmd/gaze/main.go`

Removed because it is byte-identical to `config.LoadFromDir`, which
already exists and is used by other packages. Keeping a local copy
serves no purpose.
