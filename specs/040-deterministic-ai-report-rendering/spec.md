# Feature Specification: Deterministic AI Report Rendering

**Feature Branch**: `040-deterministic-ai-report-rendering`

**Created**: 2026-11-01

**Status**: Draft

**Input**: User description: "Deterministic AI report rendering —
Go renders every deterministic part of the gaze quality report
(layout, tables, emoji, grades, severity, metric wording); the AI
supplies only judgment slots as strict JSON that Go validates.
Prompted by the Gaze AI Report CI incident (test.yml run
41309423609) in which the job published only mixed English/Chinese
agent narration instead of a quality report, and still passed."
(Change name: deterministic-ai-report-rendering; tier routed by
/uf.unleash to speckit; no originating issue.)

## Background

### The incident

The Gaze AI Report CI job (`test.yml`, run 41309423609) published
only agent narration — mixed English and Chinese text produced by
the GLM-5 model configured through the `OPENCODE_CONFIG_CONTENT`
organization secret — instead of a quality report. The job passed.

Three root causes combined:

1. **Unfiltered model stream.** The opencode adapter runs the
   model in a plain-text streaming mode that emits every assistant
   text fragment, including the narration the model produces
   between tool calls. The whole stream is treated as the report.
2. **Wrong prompt for the job.** `gaze report` hands the model the
   interactive, tool-using `/gaze` agent prompt (the project-local
   `.opencode/agents/gaze-reporter.md` when present, otherwise an
   embedded copy). That prompt instructs the model to run gaze
   commands, read reference files, and hand-render tables, emoji
   and grades. The model explored the repository and never wrote a
   report.
3. **No output validation.** Any non-empty model output is accepted
   as a successful report, so the CI job cannot tell a report from
   noise.

### The principle

Report layout, emoji, tables, letter grades, severity markers, and
metric wording are fixed procedures with no judgment in them. Under
the determinism-first conventions (determinism pack DE-001 and
DE-003 — prescribed output must be rendered by executable code, not
by prose instructions to a model), these belong in Gaze itself.
The AI's job shrinks to the parts that genuinely need judgment: a
short narrative summary and a prioritized list of recommendations,
returned in a strict, machine-checkable shape that Gaze validates
before anything is published.

### Decisions already made by the project owner

These are settled and are not open questions for this spec:

- Deliver the full change, not only a prompt guard.
- Gaze computes grades and severity; the AI never does.
- Invalid AI output fails the job.
- An existing project-local `.opencode/agents/gaze-reporter.md` is
  ignored by `gaze report`, with a notice on the diagnostic stream.
- The CI model stays GLM-5; the fix must be robust to the model.
- The interactive `/gaze` flow is folded into this spec.
- The embedded copy of the reporter prompt inside the report
  pipeline, and the test that keeps it in sync, are removed.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - CI publishes a real report or fails loudly (Priority: P1)

A maintainer's CI pipeline runs `gaze report` with an AI adapter
on every push. Whatever the configured model does — narrates, mixes
languages, explores, or returns malformed output — the published
report either contains every deterministic section of the quality
report plus a validated AI narrative, or the job fails with a clear
reason. A job never passes while publishing narration in place of a
report.

**Why this priority**: This is the incident. A quality gate that
passes while publishing garbage gives false assurance, which is the
worst failure mode for a quality tool (Constitution Principle I:
Accuracy; Principle III: Actionable Output).

**Independent Test**: Run `gaze report` with an AI adapter
substituted by a fake that replays the incident output (several
narration fragments, mixed English and Chinese, no structured
answer). Verify the command exits non-zero and names the validation
failure. Then replay a well-formed structured answer and verify the
command exits zero and publishes the full report.

**Acceptance Scenarios**:

1. **Given** an AI adapter whose model emits several narration
   fragments followed by no structured answer, **When**
   `gaze report --ai=<adapter>` runs, **Then** the command exits
   non-zero and the error identifies that the AI output failed
   validation.
2. **Given** an AI adapter whose model returns a structured answer
   whose text is predominantly Chinese, **When** the report runs,
   **Then** the answer is rejected as non-English and the command
   exits non-zero.
3. **Given** an AI adapter whose model returns a valid structured
   answer, **When** the report runs, **Then** the published report
   contains every deterministic section rendered by Gaze, plus the
   model's summary and recommendations in their designated slots,
   and the command exits zero (subject to threshold gates).
4. **Given** an AI adapter whose model emits narration before a
   final, valid structured answer, **When** the report runs,
   **Then** only the final answer is used, no narration appears in
   the published report, and the command exits zero.
5. **Given** the AI output is rejected, **When** the job fails,
   **Then** the failure message states which validation rule was
   violated (for example "summary exceeds length bound" or
   "non-English text ratio above limit") without echoing the full
   rejected output.

---

### User Story 2 - The report looks the same every time for the same data (Priority: P1)

A developer compares two report runs on the same commit, or a
reviewer reads reports across many PRs. Titles, section order,
tables, emoji, quadrant labels, letter grades, severity markers,
and metric wording are identical for identical analysis data,
regardless of which AI adapter or model was used, and regardless
of whether an AI was used at all.

**Why this priority**: Comparable output across runs is a
constitutional requirement (Principle III). Deterministic rendering
is also the mechanism that makes User Story 1 possible: once Gaze
renders the report, the AI can be confined to a small, checkable
slot.

**Independent Test**: Render the report twice from the same fixed
analysis payload with two different fake AI answers. Verify every
byte outside the two AI slots is identical, and that the output
matches a shared golden report file.

**Acceptance Scenarios**:

1. **Given** a fixed analysis payload, **When** the report is
   rendered, **Then** it matches the shared golden report
   byte-for-byte outside the AI narrative slots.
2. **Given** a payload where one analysis step failed (for example
   quality analysis), **When** the report is rendered, **Then**
   that section is replaced by a warning callout naming the failure
   and all other sections render normally.
3. **Given** a payload where a metric is unavailable (for example
   GazeCRAPload could not be computed), **When** the report is
   rendered, **Then** the metric displays "N/A" with a neutral
   marker and is never estimated or substituted.
4. **Given** a fixed payload, **When** grades are computed,
   **Then** each scorecard dimension receives the same letter
   grade and severity marker every time, according to the
   published rubric.

---

### User Story 3 - A deterministic report with no AI at all (Priority: P2)

A developer — or the interactive `/gaze` agent — wants the full
human-readable quality report without configuring any AI provider.
They run `gaze report --format=markdown` and get the complete
Gaze-rendered report (all sections, scorecard, grades, worst
offenders, deterministic recommendations derived from fix
strategies), with no AI narrative slot.

**Why this priority**: It unlocks the `/gaze` rewrite (User Story
4), gives CI users a zero-secret report option, and is the
foundation the AI path builds on. It is P2 because the incident fix
(P1) can ship with AI-backed rendering alone.

**Independent Test**: Run `gaze report --format=markdown ./...` on
a fixture module with no AI adapter configured. Verify it exits
zero, prints a complete markdown report, and the output matches the
golden report with the narrative slots omitted.

**Acceptance Scenarios**:

1. **Given** no `--ai` flag, **When** the user runs
   `gaze report --format=markdown`, **Then** the full deterministic
   report is printed to standard output and no AI provider is
   contacted.
2. **Given** `--format=markdown` and a threshold flag, **When** a
   threshold fails, **Then** the report is still printed and the
   command exits non-zero, as in the other formats.
3. **Given** `--format=markdown` and `GITHUB_STEP_SUMMARY` set,
   **When** the report completes, **Then** the rendered report is
   appended to the step summary.
4. **Given** `--format=markdown --sections=crap`, **When** the
   report runs, **Then** only the CRAP section (and the header) is
   rendered and only the analysis needed for it is performed.

---

### User Story 4 - The interactive /gaze experience keeps working (Priority: P2)

A developer types `/gaze`, `/gaze crap ./internal/store`, or
`/gaze quality ./pkg/api` in their AI coding assistant. They still
get a quality report in the chat, in all three modes. The report
body is now the Gaze-rendered markdown shown verbatim; the agent
adds only judgment on top: a narrative, prioritized
recommendations, and evaluation of assertions Gaze could not map
mechanically.

**Why this priority**: The `/gaze` agent is the same 528-line
prompt that caused the incident when reused by CI. Rewriting it to
consume the deterministic renderer removes the duplicated prose
rubric and keeps the two experiences consistent. P2 because the CI
fix does not depend on it.

**Independent Test**: Invoke the rewritten agent's documented
command sequence (binary resolution, then
`gaze report --format=markdown` with the mode's sections) on a
fixture module and confirm the agent's instructions reference the
Gaze-rendered output verbatim and contain no grade rubric, emoji
templates, table templates, or document scoring weights.

**Acceptance Scenarios**:

1. **Given** the user runs `/gaze` (full mode), **When** the agent
   runs, **Then** it shows the full Gaze-rendered report verbatim
   and adds a narrative and prioritized recommendations.
2. **Given** `/gaze crap <pattern>`, **When** the agent runs,
   **Then** it requests only the CRAP section for that pattern.
3. **Given** `/gaze quality <pattern>`, **When** the agent runs,
   **Then** it requests only the quality section for that pattern
   and may evaluate unmapped assertions as an added judgment
   section.
4. **Given** the rewritten agent prompt, **When** it is compared
   with the 528-line original, **Then** the change record reports
   the line and approximate token deltas (DE-004).
5. **Given** the project uses an external analyzer, **When** the
   agent runs, **Then** analyzer and language selection continue to
   work as they do today.

---

### User Story 5 - A locally customized reporter prompt does not silently change CI (Priority: P3)

A consumer project contains its own `.opencode/agents/gaze-reporter.md`
(scaffolded by `gaze init` and possibly customized). When CI runs
`gaze report --ai=...`, that file no longer influences the report
pipeline, and the run prints a notice explaining that it was
ignored, so nobody is surprised that a customization had no effect.

**Why this priority**: Required for correctness of User Story 1
(the interactive prompt must never be fed to the report pipeline
again), but the notice itself is low-risk polish.

**Independent Test**: Run `gaze report --ai=<fake>` in a directory
containing `.opencode/agents/gaze-reporter.md`. Verify the
diagnostic stream contains the ignore notice, the AI receives only
the built-in report-mode instructions, and the file's contents
appear nowhere in what is sent to the AI.

**Acceptance Scenarios**:

1. **Given** a working directory containing
   `.opencode/agents/gaze-reporter.md`, **When**
   `gaze report --ai=<adapter>` runs, **Then** a single notice on
   the diagnostic stream states that the file is ignored for
   `gaze report`.
2. **Given** the same directory, **When** `gaze report
   --format=json`, `--format=html`, or `--format=markdown` runs,
   **Then** no AI instructions are loaded and no notice is
   required.
3. **Given** no such file exists, **When** the report runs,
   **Then** no notice is printed.

---

### Edge Cases

- **Narration then answer**: the model emits narration and tool
  chatter, then a valid final answer. Only the final assistant
  answer is considered; earlier fragments are discarded.
- **Answer wrapped in a code fence**: the model wraps its JSON in
  a single fenced code block. A single surrounding fence is
  tolerated; any other surrounding text causes rejection.
- **Multiple JSON objects or trailing prose** in the final answer:
  rejected.
- **Unknown or missing fields** in the answer: missing required
  fields are rejected; unknown extra fields are rejected (strict
  schema).
- **Empty strings, whitespace-only summary, or zero
  recommendations**: rejected by the bounds.
- **Too many recommendations** or an over-long summary or
  recommendation: rejected (no silent truncation — truncating would
  hide a model misbehaving).
- **Mixed-language text**: a small number of non-Latin characters
  (for example a function name, an emoji, or a quoted identifier)
  is allowed; text whose non-English ratio exceeds the limit is
  rejected.
- **Markup injection**: recommendation or summary text containing
  headings, tables, raw HTML, or links that could restructure the
  rendered report is neutralized or rejected so the AI cannot alter
  report layout.
- **Model attempts tool use**: the report-mode agent is configured
  with all tools denied; any tool attempt is refused by the runtime
  and the run still must produce a valid answer or fail validation.
- **Model returns nothing**: treated as a validation failure (no
  change to the current non-zero exit).
- **Timeout**: the existing adapter timeout still applies and
  fails the job.
- **AI failure with thresholds passing**: the command still exits
  non-zero; a valid quality score does not mask a broken report.
- **AI failure with thresholds failing**: both failures are
  reported; exit remains non-zero.
- **Partial analysis failure**: sections whose analysis failed
  render as warning callouts; the AI receives the same partial
  payload and the summary may mention it; grades for dimensions
  without data are "N/A".
- **All analysis steps failed**: the report renders a header and
  warning callouts only; the zero-result gate and existing error
  behavior are unchanged.
- **Section selection with an incompatible threshold**: for
  example `--sections=crap --min-contract-coverage=50` asks to gate
  on data that will not be computed. This is rejected up front with
  a clear error rather than passing or failing silently.
- **Unknown section name** in `--sections`: rejected with the list
  of valid names.
- **`--ai` combined with `--format=json`, `--format=html`, or
  `--format=markdown`**: the AI is not invoked, matching current
  behavior for json and html.
- **Very wide tables**: function names longer than the column
  budget are rendered without breaking table structure.
- **Wide characters in function names** (external analyzers): do
  not corrupt table rendering.

## Requirements *(mandatory)*

### Functional Requirements

#### Deterministic rendering

- **FR-001**: Gaze MUST render the human-readable report itself,
  from the analysis payload alone, including: title and metadata
  header; CRAP summary and quadrant distribution; worst offenders
  with fix strategies; quality summary; classification summary;
  documentation/health scorecard; and deterministic
  recommendations derived from fix strategies.
- **FR-002**: The rendered report MUST use a closed, documented
  vocabulary of section markers and severity emoji, carried over
  from the current report style (🔍 📊 🧪 🏷️ 🏥 and 🟢 🟡 🔴 ⚪).
- **FR-003**: For identical analysis payloads and identical AI
  answers, the rendered report MUST be byte-identical across runs.
  Outside the AI slots it MUST be byte-identical regardless of AI
  adapter, model, or whether any AI was used.
- **FR-004**: When an analysis step failed, its section MUST be
  replaced by a warning callout naming the failure; other sections
  MUST still render.
- **FR-005**: When a metric is unavailable, it MUST be shown as
  "N/A" with the neutral marker; Gaze MUST NOT estimate,
  substitute, or recompute a proxy value.
- **FR-006**: Rendered metric values (CRAPload and its threshold,
  GazeCRAPload, average contract coverage, quadrant counts, worst
  offender scores and strategies) MUST be taken verbatim from the
  analysis data, never re-thresholded or re-ranked.

#### Grades and severity

- **FR-007**: Gaze MUST compute a letter grade for each health
  scorecard dimension from a published, documented rubric using
  the grade scale A, A-, B+, B, B-, C+, C, C-, D, F.
- **FR-008**: Each grade MUST map to a severity marker using the
  existing mapping: A, A-, B+ → 🟢; B, B-, C+, C → 🟡; C-, D, F → 🔴;
  unavailable → ⚪ "N/A".
- **FR-009**: Each deterministic recommendation MUST carry a
  severity derived by Gaze from the analysis data (for example
  zero-coverage or Q4 functions → critical; decomposition or
  coverage gaps → moderate; optional improvements → low).
- **FR-010**: Grades and severities MUST also appear in the
  machine-readable report output as a documented, stable contract
  so automation reads the same values humans see.
- **FR-011**: The rubric MUST be covered by table-driven tests
  that pin every grade boundary.

#### AI judgment slots

- **FR-012**: When an AI adapter is used, the AI MUST be asked
  only for judgment: a short narrative summary and a prioritized
  list of recommendations, in English, returned as a strict
  structured answer with exactly the fields `summary` (text) and
  `recommendations` (ordered list of text items).
- **FR-013**: The instructions given to the AI for `gaze report`
  MUST be a dedicated, small, report-mode prompt built into Gaze.
  It MUST tell the AI that the analysis data arrives on its input,
  that it must not use tools, and that it must answer with the
  structured shape only. This prompt MUST NOT be scaffolded into
  consumer projects.
- **FR-014**: The report-mode agent definition handed to the
  opencode runtime MUST deny all tool use, using the permission
  settings supported by the opencode version Gaze pins in CI
  (to be verified during planning).
- **FR-015**: Every adapter (claude, gemini, ollama, opencode)
  MUST return the AI's answer in the same structured shape and
  pass it through the same validator.
- **FR-016**: The opencode adapter MUST read the model's output as
  structured events and extract only the final assistant answer,
  discarding intermediate narration and tool events. The event
  format of the pinned opencode release MUST be verified during
  planning.
- **FR-017**: The AI's summary and recommendations MUST be placed
  into designated slots in the Gaze-rendered report; the AI MUST
  NOT be able to alter any other part of the report (headings,
  tables, grades, metrics).

#### Validation

- **FR-018**: Gaze MUST validate the AI answer deterministically
  before rendering, checking: exact schema (required fields
  present, no extra fields, correct types); summary length within
  bounds; recommendation count within bounds; each recommendation
  length within bounds; and non-English character ratio below a
  limit.
- **FR-019**: When validation fails, `gaze report` MUST exit
  non-zero with an error naming the violated rule, so CI fails.
  The error MUST NOT reproduce the full rejected output.
- **FR-020**: When validation fails, Gaze SHOULD still emit the
  deterministic portion of the report (with a callout stating the
  AI narrative was rejected) to standard output and the step
  summary, so the failed job still shows the quality data.
- **FR-021**: Gaze MUST NOT silently truncate, translate, or
  repair an invalid answer. Retrying the AI on validation failure
  is out of scope.

#### Prompt sources and notices

- **FR-022**: `gaze report` MUST NOT read the project-local
  `.opencode/agents/gaze-reporter.md` (or any other local prompt
  file) as AI instructions.
- **FR-023**: When `.opencode/agents/gaze-reporter.md` exists in
  the working directory and an AI adapter is used, `gaze report`
  MUST print one notice on the diagnostic stream stating the file
  is ignored for `gaze report`. A validated override mechanism is
  out of scope.
- **FR-024**: The embedded copy of the reporter prompt used by the
  report pipeline, and its scaffold-sync test, MUST be removed. The
  scaffolded `/gaze` agent and its `.opencode/` mirror remain, the
  scaffold manifest stays at 9 files, and the existing
  scaffold-sync test stays green.

#### Markdown output and section selection

- **FR-025**: `gaze report` MUST accept `--format=markdown`, which
  prints the Gaze-rendered report without contacting any AI
  provider. In this format the AI narrative slots are omitted.
- **FR-026**: `gaze report` MUST accept a section selection flag
  (`--sections`) listing which report sections to analyze and
  render, defaulting to all sections. It MUST support at least the
  equivalents of today's `/gaze` modes: full, CRAP only, and
  quality only.
- **FR-027**: Only the analysis needed for the selected sections
  SHOULD be performed.
- **FR-028**: A threshold flag that depends on a section not
  selected MUST be rejected before analysis with a clear error.
- **FR-029**: Threshold evaluation, the zero-result gate, and the
  step-summary write MUST behave the same for `--format=markdown`
  as for the AI-backed text format.

#### Interactive /gaze agent

- **FR-030**: The `/gaze` agent prompt (and its byte-identical
  scaffold copy) MUST be rewritten to: resolve the gaze binary
  (unchanged order); detect analyzer and language (unchanged); run
  `gaze report --format=markdown` with the sections for the chosen
  mode (and the JSON format when it needs data for judgment); show
  the rendered report verbatim; and add only judgment — narrative,
  prioritized recommendations, and unmapped-assertion evaluation.
- **FR-031**: The rewritten agent prompt MUST NOT contain prose
  scoring rules, the grade rubric, emoji or table templates, or
  document-classification weights. It SHOULD keep one or two real
  output examples (DE-002).
- **FR-032**: The change record MUST report the before/after line
  count and approximate token count of the agent prompt (528 lines
  before) (DE-004).
- **FR-033**: The `/gaze` command's user-facing modes (full,
  `crap`, `quality`) and package-pattern argument MUST keep
  working.
- **FR-034**: The scaffolded `.opencode/agents/gaze-reporter.md`
  MUST become tool-owned: `agents/gaze-reporter.md` is added to
  `isToolOwned`, so `gaze init` overwrites it when its content
  differs from the embedded copy and reports it as updated, the
  same way it handles the reference files and
  `gaze-test-generator.md`. Consumer projects then receive the
  rewrite automatically. Local customizations to that file are
  overwritten, and release notes and docs MUST say so.

#### Tests

- **FR-035**: Tests MUST cover the three DE-007 seam layers: the
  registration surface (the new format value, the section flag, and
  the adapters' structured-answer path are wired into the CLI and
  runner); the documented output shape (a shared golden markdown
  report used by renderer and CLI tests); and a write→read round
  trip (grades emitted in machine-readable output read back to the
  same values the rendered report shows).
- **FR-036**: An incident regression test MUST replay multi-part
  narration plus predominantly Chinese output through the opencode
  adapter path and assert it is rejected and the command exits
  non-zero.
- **FR-037**: The fake opencode test binary MUST be able to emit a
  multi-event stream (narration events, tool events, final answer)
  so adapter extraction is tested against realistic output.
- **FR-038**: CLI wiring tests MUST cover `--format=markdown`,
  `--sections` (valid, invalid, incompatible-threshold), and the
  ignore notice.

#### Documentation

- **FR-039**: User-facing documentation MUST be updated: the CI
  integration guide (what the AI now supplies, validation failure
  behavior, `--format=markdown` as a no-secret option), the README,
  the `gaze report` CLI reference page, and AGENTS.md (Recent
  Changes entry; removal of the second File Sync Invariants row).
- **FR-040**: A documentation issue MUST be filed in
  `unbound-force/website` before the implementing PR merges (new
  flag and format, changed report behavior).

### Key Entities

- **Analysis payload**: the combined result of the CRAP, quality,
  classification, and documentation steps, including per-step
  errors. The single input to rendering, grading, and the AI.
- **Rendered report**: the Gaze-produced markdown document; fixed
  sections, closed emoji vocabulary, warning callouts for missing
  data, and up to two AI slots.
- **Health scorecard**: one row per dimension (for example
  CRAPload, GazeCRAPload, average line coverage, contract coverage,
  complexity) with metric detail, letter grade, and severity
  marker.
- **Grade rubric**: the documented mapping from metric values to
  letter grades, plus the fixed grade-to-severity mapping.
- **Deterministic recommendation**: a recommendation Gaze derives
  from fix strategies and worst offenders, with Gaze-assigned
  severity.
- **AI answer**: the structured judgment returned by the model —
  `summary` and ordered `recommendations` — accepted only after
  validation.
- **Validation result**: pass, or the first violated rule with a
  short reason.
- **Section selection**: the set of report sections requested; it
  determines which analyses run, which sections render, and which
  thresholds are permitted.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Replaying the incident output (run 41309423609:
  multi-part narration, mixed English/Chinese, no structured
  answer) through `gaze report` produces a non-zero exit 100% of
  the time.
- **SC-002**: Across a fixed payload rendered with at least three
  different valid AI answers and with no AI, every byte outside the
  AI slots is identical (0 differing bytes).
- **SC-003**: The rendered report for the shared fixture matches
  the golden report exactly, in both renderer-level and CLI-level
  tests.
- **SC-004**: Every grade boundary in the rubric is pinned by at
  least one test case on each side of the boundary.
- **SC-005**: For each scorecard dimension, the grade shown in the
  rendered report equals the grade in the machine-readable output
  in 100% of round-trip test cases.
- **SC-006**: The rewritten `/gaze` agent prompt is substantially
  shorter than the 528-line original, with the before/after line
  and token deltas recorded; it contains zero grade rubric tables,
  emoji templates, or scoring-weight rules.
- **SC-007**: All three `/gaze` modes (full, `crap`, `quality`)
  still produce a report on the Gaze repository itself.
- **SC-008**: The dogfood CI job on the Gaze repository, still
  using GLM-5, either publishes a full deterministic report with a
  validated narrative or fails; in no run does it pass while
  publishing a report missing the deterministic sections.
- **SC-009**: `gaze report --format=markdown` completes on the
  Gaze repository with no AI provider configured and no network
  access to any model provider.
- **SC-010**: The scaffold manifest remains 9 files and the
  existing scaffold-sync test passes; the removed sync test and
  embedded prompt copy leave no orphaned references.
- **SC-011**: Existing CI quality gates (coverage ratchets,
  CRAPload and GazeCRAPload limits, lint) pass without any gate
  value being changed.

## Assumptions

- **Grade rubric values**: the current prompt names the grade
  scale and the grade-to-emoji mapping but leaves the metric-to-
  grade boundaries to the model. The concrete boundaries per
  dimension will be chosen during planning, written into a single
  rubric reference, and pinned by tests. The rubric is versioned
  with Gaze; changing a boundary is a deliberate, reviewed change.
- **Validation bounds defaults** (final values set in planning,
  then fixed and documented): summary 1–1,200 characters;
  1–5 recommendations, matching today's "Top 5 Prioritized
  Recommendations"; each recommendation 1–400 characters;
  non-English ratio limit of 5% of letter characters, counting
  CJK and other non-Latin scripts as non-English while ignoring
  digits, punctuation, emoji, and code identifiers.
- **English only**: the report-mode answer must be English. No
  localization is in scope.
- **No retries**: a single AI attempt per run. Retry policy is
  deferred.
- **Narrative placement**: the AI summary appears near the top of
  the report and the AI recommendations appear alongside the
  deterministic recommendations in the health section; exact
  placement is decided in planning and fixed by the golden file.
- **Deterministic recommendations remain**: Gaze's own
  fix-strategy recommendations are always rendered; AI
  recommendations add prioritization judgment, they do not replace
  them.
- **Machine-readable output**: grades are added to the JSON
  output additively; existing fields are unchanged.
- **HTML output**: the existing HTML renderer is unchanged by this
  spec, other than optionally sharing grade computation.
- **Override path deferred**: a schema-validated way for consumers
  to customize report-mode instructions is out of scope.
- **CI workflows**: the reusable AI-report workflow and the
  dogfood job keep their current invocation and thresholds; they
  gain correct failure behavior from the CLI change alone.
- **CI model**: GLM-5 remains configured through the organization
  secret; robustness comes from structured extraction and
  validation, not from a model change.
- **opencode version**: the event format and tool-permission keys
  are those of the opencode release pinned by the reusable
  workflow (opencode-ai 1.19.0 per the incident context); both must
  be verified against that release during planning.
- **Scope of /gaze command file**: the `/gaze` command's
  user-facing usage stays the same; only wording that describes how
  the agent produces the report may change, keeping its embedded
  copy byte-identical.

## Clarifications

### Session 2026-11-01

- Q: Should the scaffolded `.opencode/agents/gaze-reporter.md` become tool-owned (overwritten on `gaze init` when it differs), so consumers receive the rewrite automatically? -> A: Yes. Option A, make it tool-owned (add `agents/gaze-reporter.md` to `isToolOwned`; overwrite-on-diff; document that local customizations are overwritten). (human-answered during /uf.unleash clarify)
