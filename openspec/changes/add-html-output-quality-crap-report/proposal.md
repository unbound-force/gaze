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
- Existing text and JSON bytes, analysis/scoring behavior, thresholds, and JSON schemas remain unchanged.
- No browser application, JavaScript, external assets, third-party charting library, new service, protocol, package, generalized formatter framework, snapshot suite, or fake infrastructure is added.
- Because this is user-facing CLI behavior, the website documentation gate requires the user to create a tracking issue in `unbound-force/website` before the implementing PR is merged.

## Constitution Alignment

Assessed against the Unbound Force org constitution.

### I. Autonomous Collaboration

**Assessment**: PASS

Each HTML document is a self-describing artifact that can move from a local run to CI or review without runtime coupling. The change does not introduce coordination between heroes or services.

### II. Composability First

**Assessment**: PASS

HTML remains an optional CLI format implemented with the Go standard library. Each command remains independently usable, and text and JSON retain their existing standalone behavior.

### III. Observable Quality

**Assessment**: PASS

JSON remains the unchanged machine-parseable contract with existing provenance. HTML adds a deterministic human-readable representation, identifies the report type and Gaze version, and accurately distinguishes available, unavailable, degraded, and failed analysis data.

### IV. Testability

**Assessment**: PASS

Each formatter accepts an `io.Writer` and typed report data, enabling isolated tests without external services. Tests cover changed behavior through semantic structure, contextual escaping, deterministic bytes, self-containment, optional states, writer failures, and CLI dispatch while existing CI gates protect unchanged text and JSON paths.
<!-- scaffolded by uf v0.17.0 -->
