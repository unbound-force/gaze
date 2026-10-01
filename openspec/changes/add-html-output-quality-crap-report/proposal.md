## Why

Gaze's quality and risk commands currently provide terminal text and JSON, but neither format gives reviewers a portable, human-readable artifact suited to CI uploads and stakeholder review. The completed `gaze analyze` HTML foundation demonstrates that Gaze can provide deterministic, safely escaped, self-contained reports without network resources or new dependencies.

This change implements issue #261 with the smallest user-visible outcome: users can generate self-contained HTML from `gaze quality`, `gaze crap`, and `gaze report` while existing text and JSON behavior remains unchanged.

## What Changes

- Add `html` as an accepted `--format` value for `gaze quality`, `gaze crap`, and `gaze report` only.
- Render quality results, CRAP results, and the combined report pipeline as complete HTML documents using Go's `html/template` package and inline CSS.
- Render combined report HTML natively from the pipeline payload without requiring or invoking an AI adapter.
- Represent available summaries, diagnostics, gaps, scores, quadrants, remediation guidance, baseline comparisons, classification, docscan data, and partial-step errors without inventing unavailable values.
- Preserve contextual escaping, deterministic rendering, and the no-external-resource guarantees established by `gaze analyze` HTML output.
- Add focused formatter and CLI tests for populated, empty, degraded, optional-data, partial-failure, escaping, and format-selection behavior.
- Update command documentation to describe HTML output.

## Capabilities

### New Capabilities
- `quality-crap-report-html-output`: Produces complete, deterministic, self-contained HTML documents for quality, CRAP, and combined report results.

### Modified Capabilities
- None. No existing OpenSpec capability defines these commands' output-format contracts.

### Removed Capabilities
- None.

## Impact

- `internal/quality/`: quality HTML formatter, embedded template, and focused tests.
- `internal/crap/`: CRAP HTML formatter, embedded template, and focused tests.
- `internal/aireport/`: native combined HTML rendering and runner dispatch tests.
- `cmd/gaze/`: explicit HTML validation and dispatch for the three commands.
- `docs/reference/cli/` and `README.md`: supported-format documentation and examples.
- Existing text and JSON behavior, analysis/scoring behavior, thresholds, and JSON schemas remain unchanged.
- No browser application, JavaScript, external assets, third-party charting library, new service, protocol, package, generalized formatter framework, snapshot suite, or fake infrastructure is added.
- Because this is user-facing CLI behavior, the website documentation gate requires the user to create a tracking issue in `unbound-force/website` before the implementing PR is merged.

## Constitution Alignment

Assessed against the Gaze constitution.

### I. Accuracy

**Assessment**: PASS

HTML renders existing typed analysis data without changing analysis or scoring and distinguishes unavailable, degraded, and failed results from measured values.

### II. Minimal Assumptions

**Assessment**: PASS

HTML is an optional CLI format implemented with the Go standard library. It requires no source annotations, external services, network resources, or new dependencies.

### III. Actionable Output

**Assessment**: PASS

The reports retain the existing test, target, function, diagnostic, and remediation details in a portable human-readable document while JSON remains the machine-readable format.

### IV. Testability

**Assessment**: PASS

Each formatter accepts an `io.Writer` and existing report data, enabling isolated unit tests. Existing command seams cover HTML dispatch, and existing CI gates protect unchanged text and JSON paths without adding a new ratchet or test framework.
<!-- scaffolded by uf v0.17.0 -->
