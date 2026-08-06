## Why

`cmd/gaze/main.go` has grown to 1,953 lines and contains four duplicated
patterns that increase maintenance burden and risk of divergence. These
patterns were identified during triage of GitHub issue #199. Extracting
them into shared helpers reduces duplication, improves testability of
the CLI layer, and makes individual functions easier to read and reason
about. This is a pure refactor — no behaviour changes.

## What Changes

1. **`validateFormat`** — a new helper in `internal/cliutil` replacing
   four identical if-statement validations spread across `runAnalyze`,
   `runCrap`, `runQuality`, and `runSelfCheck`.

2. **`CaptureJSON`** — a new exported function in `internal/cliutil`
   replacing the exact code clone between `cmd/gaze/main.go:captureReportJSON`
   (line 1794) and `internal/aireport/runner.go:captureJSON` (line 342).
   The local copies in both packages are deleted; both import from `cliutil`.

3. **`autoDetectMainPkg`** — a private helper in `cmd/gaze/main.go`
   replacing two identical 3-line blocks in `runAnalyze` (line 269) and
   `runQuality` (line 1126). Stays in `cmd/gaze` because it imports
   `internal/loader` and logs, making it unsuitable for `cliutil`.

4. **`loadGazeConfigBestEffort` removal** — the local copy in
   `cmd/gaze/main.go` (lines 800–809) is deleted and its 3 call sites
   (lines 657, 730, 781) are replaced with `config.LoadFromDir`, which
   already exists and is already used in `internal/aireport/runner_steps.go`
   and `internal/provider/goprovider/contract.go`.

## Capabilities

### New Capabilities
- `internal/cliutil.ValidateFormat(format string) error`: validates that a
  format string is one of `"text"` or `"json"`, returning a formatted error
  if not.
- `internal/cliutil.CaptureJSON(fn func(w io.Writer) error) (json.RawMessage, error)`:
  executes a writer function into a buffer and returns the result as
  `json.RawMessage`.

### Modified Capabilities
- `cmd/gaze/main.go` — format validation call sites updated; JSON capture
  call site updated; main-package auto-detect extracted to private helper;
  `loadGazeConfigBestEffort` removed.
- `internal/aireport/runner.go` — `captureJSON` removed; replaced with
  import of `internal/cliutil.CaptureJSON`.

### Removed Capabilities
- `cmd/gaze/main.go:captureReportJSON` — deleted; replaced by `cliutil.CaptureJSON`.
- `cmd/gaze/main.go:loadGazeConfigBestEffort` — deleted; replaced by `config.LoadFromDir`.
- `internal/aireport/runner.go:captureJSON` — deleted; replaced by `cliutil.CaptureJSON`.

## Impact

- New package: `internal/cliutil` (new file `internal/cliutil/cliutil.go`)
- Modified: `cmd/gaze/main.go` (4 patterns updated)
- Modified: `internal/aireport/runner.go` (1 function removed, import added)
- No changes to public API, JSON output, CLI flags, or test fixtures
- All existing tests must pass without modification

## Constitution Alignment

Assessed against the Gaze constitution (`.specify/memory/constitution.md`).

### I. Accuracy

**Assessment**: PASS

This change does not alter any analysis logic, detection behaviour, or
output content. It is a structural refactor of the CLI layer only.
All observable outputs remain identical.

### II. Minimal Assumptions

**Assessment**: PASS

No new assumptions are introduced. `internal/cliutil` contains only
pure utility functions with no project-specific behaviour. The format
validation values (`"text"`, `"json"`) are the same values already
enforced at every call site.

### III. Actionable Output

**Assessment**: PASS

Output is unchanged. The refactor only affects how validation and
capture are structured internally — no user-visible output changes.

### IV. Testability

**Assessment**: PASS

Extracting `ValidateFormat` and `CaptureJSON` into `internal/cliutil`
makes them directly unit-testable in isolation. The private
`autoDetectMainPkg` helper in `cmd/gaze` is small enough to be verified
through existing integration tests. Coverage ratchets are unaffected.
