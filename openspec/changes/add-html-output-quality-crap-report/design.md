## Context

The completed analyze HTML change added `internal/report.WriteHTML`, a narrowly embedded `html/template`, explicit command-level format opt-in, contextual escaping, deterministic rendering, and self-containment tests. It intentionally deferred quality, CRAP, and combined reports because their data models differ materially.

Today `gaze quality` dispatches typed `QualityReport` and `PackageSummary` values to `internal/quality`; `gaze crap` dispatches `Report` or `ComparisonResult` values to `internal/crap`; and `gaze report` assembles an `aireport.ReportPayload` before choosing JSON or AI-generated text. The confirmed execution contract requires the smallest change that gives these three commands self-contained HTML without changing text or JSON. The proposal's constitution alignment is preserved by producing standalone artifacts, retaining optional formats and machine-readable JSON, and keeping each renderer isolated behind `io.Writer` inputs.

## Goals / Non-Goals

### Goals
- Add explicit native HTML dispatch to quality, CRAP, and combined report commands.
- Render the data already available at each formatter boundary without changing analysis or scoring.
- Preserve nil and absent values so unavailable metrics are not presented as zero.
- Preserve quality degraded and empty-result diagnostics, CRAP baseline comparisons, and combined pipeline partial failures.
- Match the analyze formatter's contextual escaping, self-containment, deterministic output, and focused testing pattern.
- Keep existing text, JSON, thresholds, and gate ordering unchanged.

### Non-Goals
- Add HTML output to analyze (already delivered), self-check, classify, docscan, or any other command.
- Add new analysis, metrics, visualization data, interactivity, JavaScript, a dashboard, a server, or remote assets.
- Add a package, service, protocol, dependency, generalized HTML framework, shared universal report schema, snapshot framework, or fake system.
- Redesign existing text or JSON output, JSON schemas, AI prompts, or report pipeline orchestration.
- Perform unrelated formatter cleanup or refactoring.

## Decisions

### D1: Keep report-specific formatters in their existing packages

Add quality HTML rendering under `internal/quality`, CRAP and comparison HTML rendering under `internal/crap`, and combined HTML rendering under `internal/aireport`. Each formatter accepts its package's existing typed boundary values and writes to an `io.Writer`.

This is smaller than introducing a universal view model or a new presentation package. The templates may follow the analyze report's visual language, but report-specific structures remain independent because their semantics differ. No new cross-package abstraction is introduced.

### D2: Embed one dedicated template per report family

Each existing package embeds its narrowly scoped template with `embed.FS` and parses it with `html/template`. Quality uses one template for populated, empty, and degraded data. CRAP uses one template whose view model optionally includes comparison data. Combined report uses one template for the pipeline payload and step errors.

Templates contain complete documents and small, report-specific inline styles. They use semantic headings, tables, lists, and native `<details>` where useful. They contain no JavaScript, external URLs, fonts, images, stylesheets, or runtime filesystem reads.

### D3: Build deterministic presentation view models

Each formatter converts domain values into plain-string and scalar view models before template execution. Input ordering is preserved except where existing report behavior already defines a stable order. Map-backed quadrant and remediation counts are emitted in explicit enum order rather than ranged directly. Error values and all other dynamic data remain ordinary strings so `html/template` escapes them contextually.

Formatters do not cast dynamic values to `template.HTML`, `template.JS`, `template.CSS`, or similar trusted types. Preparation, template parsing, and template execution failures return operation-specific wrapped errors.

### D4: Render quality states from existing typed data

Add `quality.WriteHTML(w, reports, summary)`. Its view model represents each test-target pair and available gaps, hints, discarded returns, suggestions, ambiguous effects, unmapped assertions, and summary diagnostics. Nil or empty optional fields omit their sections. Empty output retains skipped-test, SSA-degraded, and reason data instead of falling through to text.

The formatter reuses the existing skipped-test display limit rather than creating a new truncation policy. It does not add or infer metrics.

### D5: Render normal and comparison CRAP through one report model

Add `crap.WriteHTML(w, report)` and `crap.WriteComparisonHTML(w, comparison)`. Both build the same top-level HTML view; the comparison entry point adds baseline summary and categorized function data. The CLI keeps its existing decision between normal and comparison writers, preserving baseline gate ordering.

Optional pointers for contract coverage, GazeCRAP, quadrants, reasons, and deltas remain optional in the view model. Maps use the existing canonical quadrant and fix-strategy ordering. The implementation does not alter score sorting or threshold rules.

### D6: Render combined HTML natively and bypass AI setup

Treat `html` like `json` for adapter preflight: it does not require `--ai`, construct an adapter, load a system prompt, or write AI-formatted output. `aireport.Run` still executes the same production pipeline and zero-result gate, then dispatches HTML to a native `runHTMLPath`. Threshold evaluation runs after a successful HTML write, matching the other output paths.

The combined formatter uses `ReportPayload.Summary` for typed headline metrics. It presents each successful raw step payload in a labeled, escaped, formatted `<pre>` section and each failed step's recorded error in the same section. This is the smallest complete representation of all pipeline data and avoids duplicating quality and CRAP parsing/rendering inside `aireport` or adding reusable fragment abstractions. JSON formatting of raw step values MUST use deterministic `encoding/json` indentation and fail with context if a non-empty successful payload is malformed.

### D7: Extend command validation explicitly

Call `cliutil.ValidateFormat(format, "html")` only from `runQuality` and `runCrap`, following analyze's existing opt-in. Report preflight adds explicit `text`, `json`, and `html` validation because it currently defaults unknown runner formats to text. Dispatch switches add explicit HTML cases; default branches remain existing text behavior only after validation.

The report CLI treats only `text` as AI-backed. JSON and HTML skip adapter and prompt setup. External-analyzer paths use the same typed outputs and therefore receive HTML without a separate rendering implementation.

### D8: Test semantic behavior without snapshots

Formatter tests use synthetic existing domain models and assert complete document structure, required semantic sections, accurate optional-state behavior, contextual escaping, no external resources, deterministic bytes, and writer-error propagation. They do not add golden snapshots, pixel assertions, generalized HTML test harnesses, broad ratchets, or new fake systems.

CLI tests verify format acceptance and correct writer dispatch for normal, empty/degraded, comparison, external-analyzer-compatible, and no-AI combined paths. Existing text and JSON tests remain the regression boundary for unchanged behavior; this change does not add byte-for-byte tests for paths it does not modify.

### D9: Update user-facing documentation and required tracking

Update the README output-format summary and the quality, CRAP, and report CLI references with HTML examples, offline behavior, and the report command's no-AI behavior. Before the implementing PR is merged, the user must create the website documentation tracking issue required by repository governance; workspace policy prevents the agent from creating it directly.

## Coverage Strategy

- Unit-test all three report-family `WriteHTML` entry points with representative populated data and the empty, degraded, unavailable, comparison, or partial-failure states applicable to each formatter.
- Add adversarial strings to names, locations, descriptions, hints, analyzer data, and step errors to prove contextual escaping.
- Verify each document has a doctype, identifying title, semantic section labels, inline styling, no executable script, and no external resource references.
- Render identical input twice and compare complete byte sequences.
- Use an erroring `io.Writer` to verify template execution errors propagate with context.
- Integration-test HTML format selection in `runQuality`, `runCrap`, and `runReport` with existing dependency-injection seams and fixtures; do not create a generalized harness.
- Verify report HTML runs without an adapter or prompt and retains threshold evaluation and partial-failure output.
- Retain existing text and JSON tests unchanged as non-regression coverage.
- Do not add a feature-specific e2e suite: formatter unit tests and command integration tests cover the changed boundaries, while the repository's existing e2e and coverage ratchets continue unchanged.
- During implementation, derive and run the exact CI-equivalent build, race-enabled tests, and lint commands from `.github/workflows/` before completion.

## Risks / Trade-offs

### R1: Dedicated templates repeat some visual conventions

Small CSS similarities are accepted to avoid a generalized framework that the three different data models do not require. If a later issue demonstrates costly drift, shared styling can be proposed separately through the mandatory scope-expansion gate.

### R2: Combined step payloads are less tailored than dedicated reports

The combined document provides typed headline metrics and readable, escaped JSON for complete per-step data. This meets the confirmed outcome without duplicating three formatter stacks or adding fragment abstractions. Richer combined visualizations can be handled by a follow-up issue if users need them.

### R3: HTML can misrepresent absent pointer metrics as zero

View models carry explicit availability booleans or optional strings for GazeCRAP and contract coverage. Tests cover unavailable values and reject zero substitution.

### R4: Dynamic data could inject markup

All dynamic values remain untrusted plain data passed through `html/template`. Adversarial tests cover element, attribute, URL, and script-like input.

### R5: Large combined payloads produce large HTML files

The combined report intentionally preserves full pipeline data for offline review. Native `<details>` sections keep the page navigable without discarding information or introducing JavaScript.
<!-- scaffolded by uf v0.17.0 -->
