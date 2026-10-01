## ADDED Requirements

### Requirement: HTML Format Selection

The `gaze quality`, `gaze crap`, and `gaze report` commands MUST accept `html` as a value for `--format` and MUST route successful analysis data to their native HTML formatters. `gaze report --format=html` MUST NOT require, validate, or invoke an AI adapter. Commands outside this change's scope MUST continue to reject `html` unless they independently support it.

#### Scenario: User selects quality HTML
- **GIVEN** a package that produces quality analysis data
- **WHEN** the user runs `gaze quality --format=html <package>`
- **THEN** the command MUST write a complete quality HTML document to standard output
- **AND** threshold evaluation MUST retain its existing behavior

#### Scenario: User selects CRAP HTML
- **GIVEN** a package that produces CRAP analysis data
- **WHEN** the user runs `gaze crap --format=html <package>`
- **THEN** the command MUST write a complete CRAP HTML document to standard output
- **AND** baseline and threshold evaluation MUST retain their existing behavior

#### Scenario: User selects combined HTML without an AI adapter
- **GIVEN** package patterns accepted by the combined report pipeline
- **WHEN** the user runs `gaze report --format=html <package>` without `--ai`
- **THEN** the command MUST write a complete combined HTML document to standard output
- **AND** it MUST NOT load an AI prompt or invoke an AI adapter

#### Scenario: Unsupported command selects HTML
- **GIVEN** a Gaze command outside this change's scope that does not support HTML
- **WHEN** the user selects `--format=html`
- **THEN** the command MUST return an invalid-format error
- **AND** it MUST NOT silently emit text or JSON

### Requirement: Complete Self-Contained Documents

Each HTML formatter MUST write a complete document containing a doctype, an `<html>` root, document metadata, an identifying title, and a body. Required styling MUST be inline, and the document MUST NOT reference external `<link>` elements, `<script src>` elements, remote images, CSS imports, fonts, stylesheets, or other network resources. Core content visibility MUST NOT depend on JavaScript.

#### Scenario: Report is opened offline
- **GIVEN** HTML produced by any command in this change
- **WHEN** the document is opened without network access or JavaScript
- **THEN** all report content MUST remain readable and navigable
- **AND** the report MUST remain styled as a single file

### Requirement: Quality Result Representation

Quality HTML MUST represent each available test-target report and its package summary. It MUST show the test and target identities, contract coverage, over-specification, and assertion-detection confidence. When present, it MUST show gaps and hints, discarded returns, remediation suggestions, ambiguous effects, unmapped assertions and reasons, SSA degradation, skipped-test diagnostics, and lowest-coverage tests.

#### Scenario: Populated quality report is rendered
- **GIVEN** quality results containing mappings, gaps, diagnostics, and a package summary
- **WHEN** the quality HTML formatter renders them
- **THEN** every test-target report MUST appear under its associated identity
- **AND** each available metric, gap, hint, and diagnostic MUST be represented without changing its meaning

#### Scenario: Optional quality data is absent
- **GIVEN** quality results without optional gaps, suggestions, ambiguous effects, or unmapped assertions
- **WHEN** the quality HTML formatter renders them
- **THEN** the document MUST remain valid
- **AND** it MUST NOT fabricate placeholder findings or unavailable metrics

#### Scenario: Quality analysis is empty or degraded
- **GIVEN** no resolved test-target reports and a summary that records skipped tests, SSA degradation, or an unavailable-analysis reason
- **WHEN** the quality HTML formatter renders the result
- **THEN** the document MUST clearly state that no test-target pairs were resolved
- **AND** it MUST show each available reason or diagnostic accurately

### Requirement: CRAP Result Representation

CRAP HTML MUST represent every available function score and the report summary. It MUST show function identity, source location, complexity, line coverage, CRAP score, and available contract coverage, GazeCRAP, quadrant, diagnostic reason, and fix strategy. It MUST show available aggregate metrics, quadrant counts, remediation counts, recommended actions, worst offenders, and SSA degradation diagnostics.

#### Scenario: Populated CRAP report is rendered
- **GIVEN** CRAP results containing scores, quadrant data, remediation guidance, and summary metrics
- **WHEN** the CRAP HTML formatter renders them
- **THEN** every function score MUST be represented
- **AND** available summaries and recommended actions MUST be represented without treating unavailable GazeCRAP values as zero

#### Scenario: CRAP report has no functions
- **GIVEN** a CRAP report with no function scores and no active threshold gate
- **WHEN** the CRAP HTML formatter renders it
- **THEN** the output MUST remain a complete HTML document
- **AND** it MUST clearly state that no functions were analyzed

### Requirement: Baseline Comparison Representation

When a CRAP baseline comparison exists, CRAP HTML MUST represent its pass or fail result, aggregate counts, regressions, improvements, new functions, new-function violations, removed functions, unchanged count, and available CRAP and GazeCRAP deltas. When no baseline comparison exists, the formatter MUST omit the comparison section rather than imply a pass or fail result.

#### Scenario: Baseline comparison is available
- **GIVEN** a baseline comparison with regressions, improvements, new functions, and removed functions
- **WHEN** the CRAP comparison HTML formatter renders it
- **THEN** the comparison summary and each available categorized function MUST be represented
- **AND** baseline, current, and delta values MUST remain associated with the correct function

#### Scenario: Baseline comparison is absent
- **GIVEN** a CRAP report produced without a configured baseline
- **WHEN** the CRAP HTML formatter renders it
- **THEN** the document MUST contain the normal CRAP results
- **AND** it MUST NOT render a baseline pass or fail status

### Requirement: Combined Report Representation

Combined report HTML MUST render the existing pipeline payload natively. It MUST identify CRAP, quality, classification, and documentation sections; show the typed combined summary metrics; represent each successful step's available output; and show step-specific errors for failed steps. It MUST preserve partial-failure behavior so one failed step does not hide successful step output.

#### Scenario: All combined report steps succeed
- **GIVEN** a combined payload containing CRAP, quality, classification, and docscan results
- **WHEN** the combined HTML formatter renders it
- **THEN** the document MUST show the combined summary and four labeled result sections
- **AND** each section MUST represent its available pipeline data

#### Scenario: Combined report has a partial failure
- **GIVEN** a combined payload where one step failed and the remaining steps succeeded
- **WHEN** the combined HTML formatter renders it
- **THEN** the failed step's section MUST show its recorded error
- **AND** successful step data MUST remain present

#### Scenario: Combined metric is unavailable
- **GIVEN** a combined payload where GazeCRAP or contract coverage was not computed
- **WHEN** the combined HTML formatter renders it
- **THEN** the metric MUST be identified as unavailable or omitted according to the section design
- **AND** it MUST NOT be displayed as a measured zero

### Requirement: Contextual Escaping

The formatters MUST use Go's `html/template` package. All source-derived, analyzer-derived, diagnostic, and error values MUST pass through contextual escaping and MUST NOT be converted to trusted raw template content types.

#### Scenario: Report data contains HTML markup
- **GIVEN** report values containing `<script>`, quotes, ampersands, tags, or attribute-breaking characters
- **WHEN** any HTML formatter renders the data
- **THEN** those values MUST appear as inert escaped text
- **AND** no data-derived element, style rule, URL, or executable script MUST be introduced

### Requirement: Deterministic Rendering

Each formatter MUST produce byte-identical output when called repeatedly with the same ordered input. The output MUST NOT contain timestamps, random identifiers, environment-derived values, or nondeterministic map iteration.

#### Scenario: Identical input is rendered twice
- **GIVEN** identical ordered report data
- **WHEN** an HTML formatter renders them in two independent calls
- **THEN** the resulting byte sequences MUST be identical

### Requirement: Rendering Failures

If an HTML formatter cannot write its document, it MUST return an operation-specific error and MUST NOT substitute text, JSON, or a fabricated successful HTML document.

#### Scenario: The output writer fails
- **GIVEN** an output writer that returns an error
- **WHEN** an HTML formatter writes a document
- **THEN** the formatter MUST return an error identifying the failed HTML rendering operation

### Requirement: Existing Format Compatibility

Adding HTML output MUST NOT alter the accepted behavior of existing `text` and `json` output paths. HTML output MUST NOT change analysis, scoring, classification, thresholds, baseline gate ordering, partial-failure handling, or JSON schemas.

#### Scenario: Existing formats are selected
- **GIVEN** input supported before this change
- **WHEN** the user selects `--format=text` or `--format=json`
- **THEN** the command MUST use its existing formatter and output contract unchanged

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
<!-- scaffolded by uf v0.17.0 -->
