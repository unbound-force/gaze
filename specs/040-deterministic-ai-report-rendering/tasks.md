---

description: "Task list for 040-deterministic-ai-report-rendering"
---

# Tasks: Deterministic AI Report Rendering

**Input**: Design documents from `specs/040-deterministic-ai-report-rendering/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md),
[research.md](research.md) (R1–R18), [data-model.md](data-model.md),
[contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: REQUIRED. The spec mandates tests explicitly (FR-011,
FR-035–FR-038, SC-001–SC-005, SC-010). Within each story, write the
test tasks first and confirm they fail before implementing.

**Organization**: Tasks are grouped by user story. The plan's
phasing is used: US2 (the renderer and grading core) comes first
because US1 and US3 build on it.

## Format: `- [ ] T### [P?] [Story?] Description`

- **[P]**: can run in parallel (different files, no dependency on an
  incomplete task)
- **[Story]**: US1–US5, mapping to the user stories in spec.md
- Every task names its exact file path(s)

## Protected gates (apply to every task)

- The dogfood thresholds `--max-crapload=38 --max-gaze-crapload=6
  --min-contract-coverage=8` in `.github/workflows/test.yml` MUST NOT
  change. Neither may `-race -count=1`, the golangci-lint config,
  pinned action SHAs, the agent `tools:` frontmatter, or the
  coverage ratchets (`TestSC003_MappingAccuracy` and others).
- `.github/workflows/*.yml` and `.mega-linter.yml` are NOT modified
  by this feature (research R13, R15).
- Every new or changed function keeps cyclomatic complexity ≤ 10 and
  gets direct contract assertions on its returned values, strings, or
  errors, so it lands in Q1/Q2 and not in GazeCRAPload (R12).
- If a phase checkpoint shows a gate would fail, STOP and report the
  conflict. Never edit the gate.

---

## Phase 1: Setup (shared fixtures and baselines)

**Purpose**: Capture pre-change baselines, then create the shared
test fixtures every story uses.

- [x] T001 Capture the pre-change baselines and record them in `specs/040-deterministic-ai-report-rendering/baseline.md`. Record: (a) per-package statement coverage from `go test -race -count=1 -short -coverprofile="$TMPDIR/040-base.out" ./internal/aireport/... ./cmd/gaze/... ./internal/scaffold/... ./internal/crap/...` plus `go tool cover -func`; (b) `go build -o "$TMPDIR/gaze" ./cmd/gaze && "$TMPDIR/gaze" crap --format=json ./... > "$TMPDIR/040-baseline-crap.json"` with the CRAPload and GazeCRAPload values written into baseline.md; (c) the dogfood gate run `"$TMPDIR/gaze" report ./... --format=json --max-crapload=38 --max-gaze-crapload=6 --min-contract-coverage=8 --coverprofile=<profile from (a) run on ./...>` and its actual values; (d) the DE-004 baseline `wc -l -w -c .opencode/agents/gaze-reporter.md` (expected 528 lines / 2,965 words / 20,769 bytes) and chars÷4 (≈5,118 tokens)
- [x] T002 [P] Create `internal/aireport/testdata/golden/payload-full.json`, a full `ReportPayload` fixture in which every step succeeds. Fixed `metadata`: `module_path` `github.com/example/project`, `branch` `main`, `gaze_version` `v2.4.0`, `date` `2026-11-01`, and `sections` `["crap","quality","classification","health"]`. The data must exercise: ≥ 5 `worst_crap` rows covering all four fix strategies, a Q4 function, and a 0% coverage function; non-empty `recommended_actions` (> 5 entries, to exercise the cap); quadrant counts; `fix_strategy_counts`; `gaze_crapload` present; a quality summary with `total_tests > 0`, `worst_coverage_tests`, and gaps (> 10 for one test, to exercise the cap of 10); `ssa_degraded` true with one package; `skipped_tests > 0`; classification counts where `Ambiguous > Contractual + Incidental`; and docscan with ≥ 1 file. Include one function name containing `|`, one containing a backtick, one 200-rune name, and one CJK name (edge cases "Very wide tables" and "Wide characters")
- [x] T003 [P] Create `internal/aireport/testdata/golden/payload-partial.json`: the same metadata, a quality step error (`errors.quality` set, `quality` null), and `summary.gaze_crapload` absent (US2 AS-2/AS-3)
- [x] T004 [P] Create the opencode event-stream fixtures in `internal/aireport/testdata/opencode_streams/`, following the verified opencode 1.19.0 schema (research R1: JSON Lines with `type`, `timestamp`, `sessionID`, and a `part` carrying `id`/`messageID`/`type`/`text`/`synthetic`/`ignored`/`tool`/`state.status`/`reason`): `answer_only.jsonl` (one step whose text is the golden "answer A" JSON), `narration_then_answer.jsonl` (two narration steps with `tool_use` status `error`, then a final step with a valid answer), `incident.jsonl` (synthetic reconstruction per R11: three steps, each with English/Chinese narration `text` parts, `tool_use` events between steps, and a last step of Chinese-dominant narration with no JSON), `chinese_answer.jsonl` (a valid JSON shape whose `summary` is predominantly Chinese), `summary_too_long.jsonl` (a valid shape whose summary is 1,201 runes, for the rejected golden), `tool_refused.jsonl` (a `tool_use` with `state.status` `error`, then a valid answer), `tool_completed.jsonl` (a `tool_use` with `state.status` `completed`), `error_event.jsonl` (an `error` event `{name, data:{message}}`), and `not_json.txt` (plain-text narration, zero JSON events)
- [x] T005 Create the shared golden-test helpers in `internal/aireport/golden_helpers_test.go`: a test-only `-update` flag (`flag.Bool("update", false, ...)`), `loadPayloadFixture(t, name) *ReportPayload`, `assertGolden(t, name string, got string)` (writes when `-update` is set, otherwise compares bytes and prints a unified-diff hint), and `stripAISlots(s string) string` using the exact regex ``(?s)<!-- gaze:ai-(summary|recommendations) -->\n.*?<!-- /gaze:ai-\1 -->\n\n`` (research R6). Export them to `cmd/gaze` tests through a small exported test-support path, or duplicate the golden path constant there, so the CLI tests read the SAME golden files (SC-003)

**Checkpoint**: The baselines are recorded and the fixtures exist. No production code has changed.

---

## Phase 2: Foundational (blocking prerequisites)

**Purpose**: Types and plumbing that every story uses. No user story
work starts until this phase is complete.

- [x] T006 Implement `internal/aireport/sections.go`: the `Section` string type with constants `SectionCRAP="crap"`, `SectionQuality="quality"`, `SectionClassification="classification"`, `SectionHealth="health"`; an immutable `SectionSet` (unexported bitset); `ParseSections(raw []string) (SectionSet, error)`; `AllSections()`; `Has`; `List()` in canonical order crap, quality, classification, health; `StepSet struct{ CRAP, Quality, Classify, Docscan bool }`; `Steps()` with the mapping `crap→CRAP`, `quality→Quality`, `classification→Classify`, `health→CRAP+Quality+Docscan`; and `CheckThresholds(s SectionSet, cfg ThresholdConfig) error` (`MaxCrapload`/`MaxGazeCrapload` require CRAP, `MinContractCoverage` requires Quality). Validation rules quoted from data-model.md §1: "An empty list is an error, and so is an empty element"; unknown names produce exactly `invalid --sections value "x": valid sections are crap, quality, classification, health`; "Duplicates collapse. Flag order never changes render order."; the incompatible-threshold error is exactly `--<flag> requires the <step> analysis, which --sections=<list> does not run` (for example `--min-contract-coverage requires the quality analysis, which --sections=crap does not run`). GoDoc goes on every exported symbol
- [x] T007 [P] Write table-driven tests in `internal/aireport/sections_test.go`: valid single and multi lists, duplicates, reordered input giving canonical `List()`, an empty list, an empty element (`crap,,quality` and `""`), an unknown name with the exact error text, `Steps()` for every section and for `AllSections()`, and `CheckThresholds` for all three threshold flags against selected and unselected steps (exact error text). Target 100% statements
- [x] T008 Add the additive payload types in `internal/aireport/payload.go`: `ReportMetadata{ModulePath string \`json:"module_path"\`; Branch string \`json:"branch,omitempty"\`; GazeVersion string \`json:"gaze_version"\`; Date string \`json:"date"\` /* YYYY-MM-DD, UTC */; Sections []Section \`json:"sections"\` /* canonical order */}`, plus `ReportPayload.Metadata ReportMetadata \`json:"metadata"\`` and `ReportPayload.Assessment *Assessment \`json:"assessment"\``. Existing fields and tags stay unchanged (spec assumption "Machine-readable output"). Declare the assessment types in `internal/aireport/assess.go` (types only for now): `Grade` (closed set A, A-, B+, B, B-, C+, C, C-, D, F), `Severity` (`low`/`moderate`/`critical`/`unavailable`), `Assessment{RubricVersion "rubric_version"; Scorecard "scorecard" /* always 5 rows, fixed order */; Recommendations "recommendations" /* 0..8 */}`, `ScorecardRow{Dimension, Label, Value *float64, Grade *Grade, Severity, Marker, Detail}`, and `Recommendation{Severity, Marker, Source, Text, Function, File, Line, FixStrategy}` (omitempty on function/file/line/fix_strategy), with the exact JSON tags from data-model.md §4. Update `internal/aireport/payload_test.go` to assert that the existing keys are unchanged and that the new keys are present
- [x] T009 [P] Export `crap.ReadModulePath` (rename from `readModulePath`) in `internal/crap/coverage.go`, update every caller and test reference in `internal/crap/`, and add GoDoc. Behavior is unchanged (research R7)
- [x] T010 Extend `RunnerOptions` in `internal/aireport/runner.go` with `Sections SectionSet` (zero value = all sections) and `Metadata ReportMetadata`, and make `Run` copy `Metadata` (with `Sections` set to `Sections.List()`) onto the payload before any format path. Extend `validateRunnerOpts` to accept the formats text, json, html, and markdown, requiring an adapter only for text. Add unit cases to `internal/aireport/runner_test.go` (markdown accepted without an adapter; unknown format rejected; metadata copied onto the payload)

**Checkpoint**: `go build ./... && go test -race -count=1 -short ./internal/aireport/... ./internal/crap/...` passes.

---

## Phase 3: User Story 2 — The report looks the same every time for the same data (Priority: P1) 🎯 MVP foundation

**Goal**: Gaze renders the full markdown report deterministically from
the payload, computes grades and severities from rubric v1, and emits
the same assessment in JSON.

**Independent Test**: Render `payload-full.json` with three different
valid AI answers and with no AI, strip the AI slots, and confirm the
strings are byte-identical and match `report-full.md` (SC-002,
SC-003). Grades in `--format=json` equal the markdown scorecard
(SC-005).

### Tests for User Story 2 (write first, confirm they fail)

- [x] T011 [P] [US2] Write the rubric boundary matrix in `internal/aireport/grade_test.go` (SC-004, FR-011): for EVERY boundary in EVERY column of contracts/grade-rubric.md (CRAPload % ≤, GazeCRAPload % ≤, Avg Line Coverage % ≥, Contract Coverage % ≥, Avg Complexity ≤), one case exactly on the bound and one case one tenth (or one integer point for contract coverage) past it. Also cover: ratio rounding with `tenths = (2*n*1000 + total) / (2*total)` for n/total pairs on either side of `x.x5`; the displayed-value rule (`strconv.FormatFloat(v,'f',1,64)` re-parsed, for example 79.96 → "80.0" → B+); the F row; and the grade→severity→marker map for all 10 grades plus N/A (A/A-/B+→low 🟢; B/B-/C+/C→moderate 🟡; C-/D/F→critical 🔴; N/A→unavailable ⚪)
- [x] T012 [P] [US2] Write `internal/aireport/assess_test.go` using synthetic payloads: exactly 5 scorecard rows in the order `crapload`, `gaze_crapload`, `avg_line_coverage`, `contract_coverage`, `complexity`, with labels "CRAPload", "GazeCRAPload", "Avg Line Coverage", "Contract Coverage", "Complexity"; `rubric_version` "1"; the exact detail templates (`{n}/{total} functions ({r}%) at or above threshold {t}`, `{v}% average line coverage`, `{v}% average contract coverage across {tests} tests`, `{v} average cyclomatic complexity`, threshold formatted `%g`); the full N/A matrix with reasons `not selected`, `analysis failed`, `no functions analyzed`, `GazeCRAP unavailable`, `no tests analyzed`, with `value` and `grade` both null; recommendations: one case per fix strategy template, severity critical when `line_coverage == 0` or quadrant `Q4_Dangerous` and otherwise moderate, the cap of the first 5 `recommended_actions` in their given order (never re-ranked, FR-006), the join to `crap.scores[]` on (package, function, file, line), the condition items in the order `ssa_degraded` (moderate), `skipped_tests` (low), `ambiguous_classification` (low) with exact texts, and a maximum of 8 items
- [x] T013 [P] [US2] Write renderer unit tests in `internal/aireport/render_markdown_test.go`: the title for each section set (`🔍 Gaze Full Quality Report`, `🔍 Gaze CRAP Report`, `🔍 Gaze Quality Report`, `🔍 Gaze Quality Report (sections: a, b)`); the Branch segment omitted with its ` · ` separator when empty; unselected sections omitted; `---` separators; cell escaping (`|`→`\|` even inside code spans, newline→space, a backtick name in a double-backtick span, a CJK name, a 200-rune name never truncated); the failed-step callout `> ⚠️ <Step> analysis unavailable: <first line, ≤ 200 bytes>` with the heading kept (FR-004); `⚪ N/A` for an unavailable metric (FR-005); the SSA-degraded, zero-tests, and `Documentation: no files found.` lines; the closed-vocabulary scan (only 🔍 📊 🧪 🏷️ 🏥 🟢 🟡 🔴 ⚪ ⚠️ outside the AI slots, FR-002); AI slot placement (summary after metadata; recommendations after the deterministic recommendations, or at the end when `health` is not selected); the failure outcome rendering only the summary slot as `> ⚠️ AI narrative rejected: <detail>` / `> ⚠️ AI narrative unavailable: <detail>`; the slots omitted entirely for the zero `AIOutcome`; and exactly one trailing `\n`
- [x] T014 [US2] Write `TestRenderMarkdown_Golden` in `internal/aireport/render_markdown_golden_test.go` (depends on T005). It is table-driven over all 6 golden files: `report-full.md` (full payload, no outcome, all sections), `report-full-ai.md` (answer A), `report-full-rejected.md` (rejection `summary_length`), `report-crap.md` (sections crap), `report-quality.md` (sections quality), and `report-partial.md` (partial payload). Also write `TestRenderMarkdown_ByteIdentity` (SC-002): render `payload-full.json` with three different valid `AIAnswer`s and with no AI, apply `stripAISlots`, and assert 0 differing bytes across all four
- [x] T015 [P] [US2] Write `TestRubricDocMatchesCode` in `internal/aireport/grade_doc_test.go` (DE-007 Layer 2, rubric docs ≡ code). It parses the "Grading rubric" table in `docs/reference/cli/report.md` (path resolved relative to the module root) and asserts that every grade row and column equals the rubric v1 constants

### Implementation for User Story 2

- [x] T016 [US2] Implement rubric v1 in `internal/aireport/grade.go`: `rubricVersion = "1"`; the boundary tables from contracts/grade-rubric.md as package-level read-only data (one table per dimension, scanned from A downward, inclusive, F otherwise); `displayTenths(v float64) int` (format with `'f',1` and re-parse); `ratioTenths(n, total int) int`; `gradeLowerIsBetter` and `gradeHigherIsBetter` helpers; `severityForGrade`; and `markerForSeverity`. Keep each function at CC ≤ 10
- [x] T017 [US2] Implement the scorecard in `internal/aireport/assess.go`: `Assess(p *ReportPayload) Assessment`, with one builder function per dimension (`crapLoadRow`, `gazeCRAPLoadRow`, `lineCoverageRow`, `contractCoverageRow`, `complexityRow`). Each decodes only the data it needs from `p.CRAP`/`p.Quality` and `p.Summary`, applies the N/A rules (step not selected per `p.Metadata.Sections`, step failed, `total_functions == 0`, `gaze_crapload` absent, `AvgContractCoverage == nil` or `total_tests == 0`), and produces `Detail` with the exact templates. Values are verbatim from the analysis data (FR-006)
- [x] T018 [US2] Implement the deterministic recommendations in `internal/aireport/assess.go`: `fixStrategyRecommendations` (the first 5 `recommended_actions`, joined to `crap.scores[]` for `line_coverage`/`quadrant`, with one text template per strategy exactly as in contracts/grade-rubric.md, and `function`/`file`/`line`/`fix_strategy` set) and `conditionRecommendations` (`ssa_degraded`, `skipped_tests`, `ambiguous_classification`, in that order, with exact texts), each split so that CC ≤ 10
- [x] T019 [US2] Implement the renderer skeleton in `internal/aireport/render_markdown.go`: the `AIOutcome{Answer *AIAnswer; FailureNote string}` type (with `AIAnswer` declared here or in a stub `answer.go` that holds only the struct), `RenderMarkdown(p *ReportPayload, outcome AIOutcome) (string, error)` (pure: no I/O, no clock; an empty `p.Metadata.Sections` renders all sections), `writeTitle`, `writeMetadata`, `writeAISummarySlot` (the `<!-- gaze:ai-summary -->` … `<!-- /gaze:ai-summary -->` block with a `### Summary` heading, followed by one blank line), `writeAIRecommendationsSlot` (`<!-- gaze:ai-recommendations -->` … with a `### AI-Prioritized Recommendations` numbered list), `writeCallout`, and the cell helpers `codeSpan(name)` (double-backtick when the name contains a backtick) and `escapeCell(s)` (`|`→`\|`, newlines→space)
- [x] T020 [US2] Implement the CRAP section in `internal/aireport/render_markdown.go` (one function per table, CC ≤ 10 each): `writeCRAPSection` (the `## 📊 CRAP Summary` metric table with total functions, average complexity, average line coverage, average CRAP score, CRAPload with threshold, and GazeCRAPload with threshold or `⚪ N/A`), `writeWorstCRAPTable` (`### Top 5 Worst CRAP Scores` in `worst_crap` order), `writeQuadrantTable` (`### GazeCRAP Quadrant Distribution` with the four fixed labels, markers, and meanings from contracts/markdown-report.md), and `writeRemediationTable` (`### Remediation Breakdown`). Floats use `%.1f` and percentages carry `%`
- [x] T021 [US2] Implement the quality section in `internal/aireport/render_markdown.go`: `writeQualitySection` (`## 🧪 Quality Summary` metric table: tests analyzed, average contract coverage, over-specifications, assertion detection confidence, skipped tests), `writeWorstTestsTable` (`### Worst Tests by Contract Coverage` in `worst_coverage_tests` order), `writeGapsTable` (`### Coverage Gaps in Worst Tests`, per-test order, capped at 10), the SSA-degraded callout `> ⚠️ SSA analysis degraded for {k} package(s): {list}`, and the zero-tests callout `> ⚠️ Quality analysis found 0 tests for the selected packages.`
- [x] T022 [US2] Implement the classification section in `internal/aireport/render_markdown.go`: `writeClassificationSection` (`## 🏷️ Classification Summary` with Contractual / Ambiguous / Incidental counts and `%.1f%%` shares from `p.Summary`)
- [x] T023 [US2] Implement the health section in `internal/aireport/render_markdown.go`: `writeHealthSection` (`## 🏥 Overall Health Assessment`), `writeScorecardTable` (`### Summary Scorecard`, `| Dimension | Grade | Details |`, with the Grade cell `Marker + " " + Grade` or `⚪ N/A` and the Details cell = `Detail`, reading ONLY `p.Assessment` so the markdown and JSON cannot diverge), the documentation line (`Documentation: {n} files scanned.` / `Documentation: no files found.`), and `writeRecommendationsList` (`### Recommendations`, lines `"{i+1}. {Marker} {Text}"`, or `No recommendations — no functions at or above the CRAP threshold.`)
- [x] T024 [US2] Wire assessment and markdown into `internal/aireport/runner.go`: compute `payload.Assessment = &Assess(payload)` once in `Run`, after analysis and before the format switch, so it is present for json, html, markdown, and text; add `runMarkdownPath(payload, opts)`, which renders with the zero `AIOutcome`, writes to stdout, appends to `$GITHUB_STEP_SUMMARY` through the existing step-summary writer, and evaluates thresholds the same way the text path does (FR-029). Keep `Run` at CC ≤ 10 by extending the existing format dispatch. Add tests to `internal/aireport/runner_test.go`: the markdown path writes stdout and the step summary, a threshold failure still prints the report and returns an error, and the JSON output includes `assessment`
- [x] T025 [P] [US2] Add `metadata` and `assessment` to the compact AI payload in `internal/aireport/compact.go`, and extend `internal/aireport/compact_test.go` to assert both keys are present, the scorecard has 5 rows, there are ≤ 8 recommendations, and the existing compact size-budget test still passes
- [x] T026 [US2] Generate the 6 golden files in `internal/aireport/testdata/golden/` (`report-full.md`, `report-full-ai.md`, `report-full-rejected.md`, `report-crap.md`, `report-quality.md`, `report-partial.md`) with `go test ./internal/aireport -run Golden -update`. Review each by hand against contracts/markdown-report.md (layout, vocabulary, escaping, slots, trailing newline), then re-run without `-update` to confirm a pass. Answer A is the same JSON that `opencode_streams/answer_only.jsonl` carries
- [x] T027 [US2] Add the CLI surface in `cmd/gaze/main.go`: `cliutil.ValidateFormat(p.format, "html", "markdown")` with error text `invalid format "x": must be 'text', 'json', 'html', or 'markdown'`; the `--sections` string-slice flag with default `crap,quality,classification,health` and help text "report sections to analyze and render: crap, quality, classification, health (default all)"; `ParseSections` and `CheckThresholds` run in `validateReportParams` BEFORE any analysis (FR-028); `reportParams` additions `sections []string`, `now func() time.Time` (default `time.Now`), `gazeVersion string`, and `analyzeFunc` (forwarded to `RunnerOptions.AnalyzeFunc`); and a `buildReportMetadata` helper (module path from `crap.ReadModulePath(moduleDir)` with a `filepath.Base(moduleDir)` fallback; branch from `GITHUB_HEAD_REF`, then `GITHUB_REF_NAME`, otherwise omitted, with no shelling out to git; UTC `YYYY-MM-DD` date from `now`; sections in canonical order), forwarded through `RunnerOptions.Metadata`/`Sections`. Keep `runReport` CC ≤ 10 by extracting helpers
- [x] T028 [US2] Write CLI tests in `cmd/gaze/main_test.go` (DE-007 Layer 1 and Layer 3): `TestNewReportCmd_FlagsRegistered` (`--sections` exists with its default and help text; `--format` help lists markdown); `TestValidateReportParams_FormatMarkdown` (accepted without `--ai`; an unknown format gives the exact error); `TestRunReport_Golden` (injected `analyzeFunc` returning `payload-full.json`/`payload-partial.json`, a fixed `now` of 2026-11-01, branch env set; output compared byte-for-byte with the SAME `internal/aireport/testdata/golden/report-full.md`, `report-crap.md`, `report-quality.md`, `report-partial.md`, per SC-003); and `TestRunReport_AssessmentRoundTrip` (SC-005: run `--format=json` and `--format=markdown` on the same injected payload, decode J with `encoding/json`, parse the markdown scorecard rows and the `### Recommendations` lines, and assert `Grade cell == marker+" "+grade` (or `⚪ N/A` when grade is null), `Details == detail`, and `line i == "{i+1}. "+marker+" "+text` for every row and item)
- [x] T029 [US2] Add the "Grading rubric" section to `docs/reference/cli/report.md`: `rubric_version` "1", the boundary table exactly as in contracts/grade-rubric.md (the format `TestRubricDocMatchesCode` parses), the displayed-value and integer-tenths rules, the N/A conditions, the grade→severity→marker map, the scorecard detail templates, and the deterministic recommendation templates and severities. Confirm T015 passes
- [x] T030 [US2] Phase checkpoint: run `go build ./... && go test -race -count=1 -short ./...` and `golangci-lint run`; rebuild gaze and compare `gaze crap --format=json ./...` CRAPload and GazeCRAPload against `baseline.md` (T001); confirm that the new functions in `grade.go`, `assess.go`, and `render_markdown.go` are CC ≤ 10 and none appear in GazeCRAPload. If the gate would exceed 38/6/8, STOP and report

**Checkpoint**: `gaze report --format=markdown` and `--format=json` produce the deterministic report and assessment. US2 is independently testable.

---

## Phase 4: User Story 1 — CI publishes a real report or fails loudly (Priority: P1) 🎯 MVP

**Goal**: Every adapter's output passes one strict validator. The
opencode adapter extracts only the final answer and runs tool-free.
A rejection still publishes the deterministic report and exits
non-zero.

**Independent Test**: Replay `incident.jsonl` through `gaze report
--ai=opencode` with the fake binary and confirm a non-zero exit with
rule `not_json`. Replay `answer_only.jsonl` and confirm exit 0 with
the full report and AI slots.

### Tests for User Story 1 (write first, confirm they fail)

- [x] T031 [P] [US1] Write `TestParseAIAnswer_*` in `internal/aireport/answer_test.go`: two table-driven cases (a minimal violation and the nearest pass) for each of the 14 rules, in normative order (`empty_output`, `surrounding_text`, `not_json`, `multiple_values`, `not_object`, `duplicate_field`, `unknown_field`, `missing_field`, `wrong_type`, `summary_length`, `recommendation_count`, `recommendation_length`, `markup`, `non_english`), asserting `errors.As` → `*AnswerRejection` with the exact `Rule`, a `Detail` of ≤ 200 bytes containing no answer content, and `Error()` == `AI answer rejected (rule <Rule>): <Detail>`. The bounds pinned exactly: 1200 vs 1201 summary runes, 5 vs 6 recommendations, 400 vs 401 recommendation runes, exactly 5% vs just above 5% non-Latin, and `letters == 0` rejected. Also: one fenced answer (```` ``` ```` and ```` ```json ````) accepted, a fence plus trailing prose rejected, a backtick-wrapped CJK identifier accepted, emoji ignored, a `\n` inside a recommendation rejected, each markup form (`<a`, `</`, `<!`, `](`, `#`, `>`, `|`, `- `, `1. `, `1) `, a fence, `---`/`===`/`***`/`___`) rejected, inline emphasis and code accepted, and accepted values stored trimmed
- [x] T032 [P] [US1] Write `TestExtractFinalText_*` in `internal/aireport/opencode_events_test.go`, unit tests over each fixture's bytes from T004: `answer_only` → the answer; `narration_then_answer` → only the last step's text; `tool_refused` → the answer (status `error` tolerated); `tool_completed` → the error `opencode executed tool "<tool>" although the gaze-headless agent denies all tools`; `error_event` → `opencode reported an error: <name>: <message>` (redacted via `redactEnvValues` and truncated to 512 bytes); `not_json.txt` → `opencode produced no JSON events (expected --format json event stream; pinned opencode-ai 1.19.0)`; `incident` → the Chinese-dominant last-step narration text (not an error; rejection happens in the validator). Synthetic cases: no `step_finish` (fall back to the last text `messageID`), duplicate part `id` (last write wins, first-seen position kept), `synthetic`/`ignored` parts dropped, `\r\n` framing, a blank final text → `opencode produced no final assistant text (FR-009)`
- [x] T033 [P] [US1] Write `TestReportModePrompt_*` in `internal/aireport/prompt_test.go` (new tests appended; the legacy `LoadPrompt` tests are removed in US5): the prompt is non-empty with no leading `---`; it contains `"summary"`, `"recommendations"`, `1200`, `400`, `5`, `Do not use tools`, and `English`; the example answer embedded in it passes `ParseAIAnswer` (guards against prompt and validator drift); it is ≤ 40 lines; and no file under `internal/scaffold/assets/` contains the prompt text (FR-013)

### Implementation for User Story 1

- [x] T034 [US1] Implement `internal/aireport/answer.go`: `AIAnswer{Summary string \`json:"summary"\`; Recommendations []string \`json:"recommendations"\`}`, `AnswerRejection{Rule, Detail string}` with `Error()`, and `ParseAIAnswer(raw string) (AIAnswer, error)` (pure), applying the 14 rules in the normative order of contracts/ai-answer.md. Constants `maxSummaryRunes=1200`, `minRecs=1`, `maxRecs=5`, `maxRecRunes=400`, `maxNonLatinPercent=5`. Use one helper per rule group (`unwrapFence`, `checkSingleValue`, `checkMembers` using a `json.Decoder` token walk for duplicates, `checkBounds`, `checkMarkup`, `nonEnglishPercent` counting `unicode.IsLetter` runes outside inline code spans with `!unicode.Is(unicode.Latin, r)` as non-English, in integer math `nonLatin*100 > 5*letters`), each at CC ≤ 10. No truncation, translation, repair, or retry (FR-021)
- [x] T035 [US1] Create the embedded report-mode prompt `internal/aireport/assets/report-mode-prompt.md` (no frontmatter, ≤ 40 lines, the content items 1–8 of contracts/agents-and-prompts.md §A, including one complete example answer) and add `//go:embed assets/report-mode-prompt.md` plus `ReportModePrompt() string` with GoDoc to `internal/aireport/prompt.go`
- [x] T036 [US1] Rewrite the text path in `internal/aireport/runner.go`: remove `RunnerOptions.SystemPrompt`; `runTextPath` always calls `opts.Adapter.Format(ctx, ReportModePrompt(), compact)`; every adapter's output goes through `ParseAIAnswer` (FR-015); build the `AIOutcome` (adapter error → `FailureNote: "AI narrative unavailable: <short adapter error>"`; rejection → `"AI narrative rejected: <rule detail>"`; success → `Answer`); ALWAYS render and write stdout and the step summary (FR-020); return `errors.Join(aiErr, thresholdErr)` so an AI failure exits non-zero even when thresholds pass. Extract `buildAIOutcome` and `publishReport` helpers to keep CC ≤ 10. Remove the `SystemPrompt` usage in `cmd/gaze/main.go` (stop calling `aireport.LoadPrompt`; the function itself is deleted in US5) and update `internal/aireport/runner_test.go` / `internal/aireport/adapter_test.go` expectations that referenced a caller-supplied prompt
- [x] T037 [P] [US1] Implement `internal/aireport/opencode_events.go`: the unexported `openCodeEvent`/`openCodePart` structs with the JSON tags from data-model.md §7 and the pure `extractFinalText(stdout []byte) (string, error)` implementing contracts/opencode-adapter.md "Extraction" steps 1–5, split into `parseEvents`, `handleEvent`, and `selectFinalText` helpers (CC ≤ 10 each). Never fall back to raw stdout
- [x] T038 [US1] Harden `internal/aireport/adapter_opencode.go`: write `<tmpDir>/.opencode/agents/gaze-headless.md` (file 0600, dirs 0700) with the frontmatter exactly `description: Gaze headless judgment call (no tools)` / `mode: primary` / `permission:` / `  "*": deny`, followed by the system prompt; invoke `opencode run --dir <tmpDir> --agent gaze-headless --format json [-m <model>] ""` with the payload on stdin (separate argv strings, no shell); and pass stdout through `extractFinalText`, appending `formatStderrSuffix(stderr)` on the no-final-text error. The `--ai-mapper=opencode` path (`buildAIMapperFunc`) inherits the tool-free agent and final-text extraction unchanged
- [x] T039 [US1] Add replay mode to `internal/aireport/testdata/fake_opencode/main.go`: when `FAKE_OPENCODE_STREAM=<file>` is set, assert the `run` subcommand, `--format json`, `--agent gaze-headless`, and that `<--dir>/.opencode/agents/gaze-headless.md` exists containing `mode: primary` and `"*": deny` (exit 2 with a message otherwise); drain stdin; copy the file to stdout verbatim. When `FAKE_OPENCODE_CAPTURE_AGENT=<path>` is set, copy the agent file there for inspection (used by the US5 test). Keep the existing `--exit-error` / `--empty-output` modes (empty output now yields "no JSON events")
- [x] T040 [US1] Update the integration tests in `internal/aireport/adapter_opencode_test.go` (the fake binary built per test and put on PATH): `TestOpenCodeAdapter_Replay_NarrationThenAnswer` (returns only the final answer), `_AnswerOnly`, `_ToolRefused` (succeeds), `_ToolCompleted` (fails closed), `_ErrorEvent` (fails, with redaction applied), `_NotJSON` (fails "no JSON events"), and an argv/agent-file assertion test (0600 perms, exact frontmatter). Update the existing tests that expected `--format default`
- [x] T041 [US1] Write `TestRunTextPath_ValidatesEveryAdapterOutput` and related tests in `internal/aireport/runner_test.go`, using `FakeAdapter` (DE-007 Layer 1: adapter → validator wiring): a valid answer → slots rendered, nil error; a rejected answer → stdout and step summary contain the deterministic report plus `> ⚠️ AI narrative rejected:`, the error is `*AnswerRejection`, no raw output is echoed; an adapter error → the `AI narrative unavailable` callout plus an error; thresholds failing plus an AI failure → `errors.Join` containing both; `FakeAdapter.Calls[0].SystemPrompt == ReportModePrompt()`
- [x] T042 [US1] Write the incident regression tests (FR-036, SC-001) in `internal/aireport/incident_test.go`: `TestIncident_NarrationReplay_ExitsNonZero` runs `Run` in text format with the real `OpenCodeAdapter` against the fake binary replaying `incident.jsonl` and an injected `AnalyzeFunc` returning `payload-full.json`. It asserts a non-nil error whose `errors.As` rule is `not_json`, that stdout contains every deterministic section heading plus the rejection callout, and that stdout contains NO narration text and NO CJK code point outside the callout. `TestIncident_ChineseAnswer_ExitsNonZero` replays `chinese_answer.jsonl` and asserts rule `non_english`. Add the CLI-level counterpart `TestRunReport_IncidentReplay_ExitsNonZero` in `cmd/gaze/main_test.go` (`runReport` with `--ai=opencode`, an injected `analyzeFunc`, and the fake binary on PATH) asserting a non-nil error naming the rule
- [x] T043 [US1] Extend `TestRunReport_Golden` in `cmd/gaze/main_test.go` with the AI outcomes (SC-003, CLI level): `--ai=opencode` plus the fake binary replaying `answer_only.jsonl` → matches `report-full-ai.md` with a nil error; replaying `summary_too_long.jsonl` → matches `report-full-rejected.md` with an error naming rule `summary_length`
- [x] T044 [US1] Phase checkpoint: `go build ./... && go test -race -count=1 -short ./...`, `golangci-lint run`, and the `gaze crap` comparison against `baseline.md` (CC ≤ 10 for `answer.go`, `opencode_events.go`, `runner.go` helpers). STOP and report if a gate would fail

**Checkpoint**: The incident replay fails loudly, valid answers publish the full report, and narration never appears. US1 is independently testable.

---

## Phase 5: User Story 3 — A deterministic report with no AI at all (Priority: P2)

**Goal**: `--format=markdown` and `--sections` run only the analysis
needed. Thresholds, the step summary, and the zero-result gate
behave as they do in the text format.

**Independent Test**: `gaze report --format=markdown ./...` on a
fixture module with no AI configured exits 0 and matches
`report-full.md` (narrative slots omitted). `--sections=crap` runs
only the CRAP step.

### Tests for User Story 3 (write first, confirm they fail)

- [x] T045 [P] [US3] Write step-selection tests in `internal/aireport/pipeline_internal_test.go` (via `pipelineStepFuncs`): for each `StepSet` derived from crap, quality, classification, health, and all, assert that only the selected step funcs are invoked, that unselected payload fields AND their `errors` entries stay nil, and that the contract-coverage builder is invoked only when CRAP runs
- [x] T046 [P] [US3] Write zero-result gate tests in `internal/aireport/runner_test.go`: `--sections=quality --min-contract-coverage=…` with `TotalFunctions == 0` does NOT trip the gate (research R8); the CRAP-selected behavior is unchanged (the existing gate tests still pass unmodified)
- [x] T047 [P] [US3] Write CLI tests in `cmd/gaze/main_test.go` (FR-038): `TestRunReport_SectionsForwarded` (`RunnerOptions.Sections` and `metadata.sections` match the flag); `--sections=bogus` → the exact error, `analyzeFunc` never called; `--sections=` and `--sections=crap,,quality` → the empty-element error; `--sections=crap --min-contract-coverage=50` → `--min-contract-coverage requires the quality analysis, which --sections=crap does not run` before analysis; `--format=markdown --max-crapload=0` → the report printed AND a non-nil error (US3 AS-2); `GITHUB_STEP_SUMMARY` set to a temp file → the rendered report appended (US3 AS-3); `--format=markdown --ai=opencode` → no adapter constructed or contacted; `--format=json --sections=crap` → `quality`, `classify`, `docscan` and their `errors` entries are `null`

### Implementation for User Story 3

- [x] T048 [US3] Make `runProductionPipeline` in `internal/aireport/runner.go` accept a `StepSet` (from `opts.Sections.Steps()`), skip unselected steps (leaving both the payload field and the error nil), and build the contract-coverage provider only when `StepSet.CRAP` is set. Keep CC ≤ 10, extracting a `runSelectedSteps` helper if needed
- [x] T049 [US3] Apply `checkZeroResultGate` in `internal/aireport/runner.go` only when the CRAP step was selected (FR-029, SC-011); its behavior is otherwise unchanged
- [x] T050 [US3] Honor section selection on the external-analyzer path in `cmd/gaze/main.go` (`runExternalReportCRAP` and its caller): skip the CRAP-dependent work when CRAP is not selected, and record unselected steps as nil
- [x] T051 [US3] Phase checkpoint: run the full short suite, golangci-lint, and the `gaze crap` comparison; run quickstart scenarios 2–4 (`go build -o "$TMPDIR/gaze" ./cmd/gaze`, then `report --format=markdown ./...` twice with a `diff` showing identical output, the section variants, and thresholds plus the step summary) and record the results in `specs/040-deterministic-ai-report-rendering/baseline.md` under "Checkpoints" (SC-009)

**Checkpoint**: The zero-secret deterministic report works end to end. US3 is independently testable.

---

## Phase 6: User Story 5 — A locally customized reporter prompt does not silently change CI (Priority: P3)

**Goal**: `gaze report` never reads the local
`.opencode/agents/gaze-reporter.md`, prints one notice in text
format when that file exists, and the embedded prompt copy and its
sync test are gone.

**Independent Test**: Run `gaze report --ai=<fake>` in a directory
containing `.opencode/agents/gaze-reporter.md` with sentinel
content. Confirm exactly one notice line on stderr and that the
sentinel is absent from the instructions sent to the AI.

### Tests for User Story 5 (write first, confirm they fail)

- [x] T052 [P] [US5] Write notice tests in `cmd/gaze/main_test.go`: `TestRunReport_IgnoreNotice_TextFormat` (a temp module dir with `.opencode/agents/gaze-reporter.md` containing a sentinel; `--ai=opencode` with the fake binary in replay mode plus `FAKE_OPENCODE_CAPTURE_AGENT`; stderr contains exactly one line `notice: ignoring .opencode/agents/gaze-reporter.md for gaze report (it is the interactive /gaze agent; gaze report uses built-in report-mode instructions)`; the captured agent file does NOT contain the sentinel and DOES contain the report-mode prompt); `_NoNoticeForJSONHTMLMarkdown` (zero notice lines for each of the three formats); `_NoNoticeWhenAbsent` (zero notice lines)

### Implementation for User Story 5

- [x] T053 [US5] Implement the ignore notice in `cmd/gaze/main.go`: in `text` format only, when `<moduleDir>/.opencode/agents/gaze-reporter.md` exists (`os.Stat`), print the exact notice line once to stderr before running the AI (research R17), in a dedicated `printIgnoredPromptNotice(moduleDir string, stderr io.Writer)` helper
- [x] T054 [US5] Remove the legacy prompt plumbing: delete `LoadPrompt`, `stripFrontmatter`, and `defaultPromptRaw` (and the old `//go:embed` of `assets/agents/gaze-reporter.md`) from `internal/aireport/prompt.go`; delete the file `internal/aireport/assets/agents/gaze-reporter.md` (and the now-empty `internal/aireport/assets/agents/` dir); delete `TestLoadPrompt_*`, `TestStripFrontmatter*`, and `TestEmbeddedPromptMatchesScaffold` from `internal/aireport/prompt_test.go` (FR-024). Confirm that `internal/aireport/assets/` still exists (it holds `report-mode-prompt.md`), so the `.mega-linter.yml` exclusion stays valid and is NOT edited
- [x] T055 [US5] Remove row 2 (`internal/scaffold/assets/agents/gaze-reporter.md` ↔ `internal/aireport/assets/agents/gaze-reporter.md`, enforced by `TestEmbeddedPromptMatchesScaffold`) from the File Sync Invariants table in `AGENTS.md`. The 9-file manifest list stays unchanged
- [x] T056 [US5] Orphan sweep (SC-010): run `rg -n "LoadPrompt|stripFrontmatter|TestEmbeddedPromptMatchesScaffold|aireport/assets/agents|loading system prompt"` over the repo, excluding `specs/` and `openspec/changes/archive/`, and fix every live reference (Go code, `docs/guides/ai-reports.md`, `docs/reference/cli/report.md`, `README.md`, `AGENTS.md`). Then run `go test -race -count=1 -short ./internal/scaffold/... ./internal/aireport/...` and confirm `TestEmbeddedAssetsMatchSource` passes with the manifest still at 9 files

**Checkpoint**: The interactive prompt can no longer reach the report pipeline. US5 is independently testable.

---

## Phase 7: User Story 4 — The interactive /gaze experience keeps working (Priority: P2)

**Goal**: `/gaze` (full, crap, quality) becomes a thin consumer of
`gaze report --format=markdown`. It shows the output verbatim and
adds only judgment. The agent file becomes tool-owned. External
analyzers keep working.

**Independent Test**: Run the rewritten agent's documented command
sequence on this repository for all three modes; confirm the prompt
holds no rubric, emoji, or table templates and that its line and
token deltas are recorded.

### Tests for User Story 4 (write first, confirm they fail)

- [x] T057 [P] [US4] Write external-analyzer quality parity tests in `cmd/gaze/main_test.go` using the existing protocol fake analyzer (`internal/protocol/testdata/fake_analyzer/`): when the Quality step is selected and the analyzer advertises `test_mapping`, the `quality` payload matches the JSON shape `gaze quality` emits, `Summary.AvgContractCoverage` is set, and classification counts are derived from `summary.ClassificationCounts`; when `test_mapping` is unavailable or errors, `errors.quality` is set, so the rendered section is a warning callout (US4 AS-5, research R9)
- [x] T058 [P] [US4] Update the ownership tests in `internal/scaffold/scaffold_test.go` (FR-034): `TestIsToolOwned` now expects `agents/gaze-reporter.md` → true (`commands/gaze.md` stays false); add `TestRun_OverwritesCustomizedGazeReporter` (scaffold, append `# custom` to `.opencode/agents/gaze-reporter.md`, re-run `Run` without `--force`, and assert the file equals the embedded copy and is listed in `Result.Updated`); update any existing expectation that treated the file as skip-if-present (also check `cmd/gaze/main_test.go` init tests)
- [x] T059 [P] [US4] Write content tests in `internal/scaffold/scaffold_test.go`: `TestGazeReporterAgent_NoRenderingRules` (scan `internal/scaffold/assets/agents/gaze-reporter.md` and fail on the grade scale table or grade-to-emoji mapping, the emoji vocabulary table, any line starting with `| Metric |`, `| Quadrant |`, or `| Dimension |`, the quadrant label list, `±`, or "Scoring Consistency Rules"; assert the frontmatter `mode: subagent` and `tools:` values are unchanged; assert it references `gaze report --format=markdown` and `--sections=crap` / `--sections=quality`); and `TestExampleReportMatchesGolden` (the single fenced `markdown` block in `internal/scaffold/assets/references/example-report.md` equals `internal/aireport/testdata/golden/report-full.md` byte for byte; DE-007 Layer 2)

### Implementation for User Story 4

- [x] T060 [US4] Extend the external report path in `cmd/gaze/main.go` (research R9): when the Quality step is selected and the analyzer supports `test_mapping`, call `providers.SideEffects.AllResults()`, `adapter.FetchTestMappings`, and `adapter.BuildQualityFromMappings`, serialize them in the `gaze quality` JSON shape, set `Summary.AvgContractCoverage`, and derive the classification counts; on unavailability or error, record the step error using the existing `handleQualityNoTestMapping` / `handleQualityTestMappingError` messages. Split this into `runExternalReportQuality` and `externalClassificationCounts` helpers (CC ≤ 10, each contract-asserted by T057) because `cmd/gaze` counts against the GazeCRAPload ceiling (R12)
- [x] T061 [US4] Rewrite `.opencode/agents/gaze-reporter.md` per contracts/agents-and-prompts.md §B: the frontmatter keys and values are UNCHANGED (`mode: subagent`; `tools:` read/bash true, write/edit/webfetch false; only `description` may be reworded); the ordered body sections are Role, Binary resolution (three methods in unchanged order), Mode parsing (unchanged), Analyzer/language detection (unchanged), Run the report (full / `--sections=crap` / `--sections=quality`, with `[--analyzer <cmd>] [--language <lang>] <pattern>`), Show output verbatim, Judgment to add under `## Assessment` (≤ 1 narrative paragraph; ≤ 5 prioritized recommendations citing metrics; quality-mode unmapped-assertion evaluation via `gaze quality --format=json <pattern>` at confidence 50 as suggestions; optional full-mode document-informed ambiguity notes consulting `.opencode/references/doc-scoring-model.md`, which never change rendered counts), Example (pointing to `.opencode/references/example-report.md`), and Errors/degradation. No rubric, emoji or table templates, or `±` weights (FR-031). Target ≤ 160 lines and ≤ ~1,600 tokens
- [x] T062 [US4] Mirror the rewrite byte-identically to `internal/scaffold/assets/agents/gaze-reporter.md` (`cp .opencode/agents/gaze-reporter.md internal/scaffold/assets/agents/gaze-reporter.md`); do not reformat either copy independently
- [x] T063 [P] [US4] Rewrite `.opencode/references/example-report.md` to contain one fenced `markdown` block equal byte-for-byte to `internal/aireport/testdata/golden/report-full.md` plus one short example `## Assessment` judgment section (DE-002), then mirror it byte-identically to `internal/scaffold/assets/references/example-report.md`
- [x] T064 [P] [US4] Change only the wording "interprets the JSON output" to "shows the Gaze-rendered report and adds judgment" in `.opencode/commands/gaze.md`, keeping usage, modes, and examples byte-for-byte; mirror the change to `internal/scaffold/assets/commands/gaze.md`
- [x] T065 [US4] Add `"agents/gaze-reporter.md"` to the exact-match list in `isToolOwned` in `internal/scaffold/scaffold.go` and update its GoDoc comment (which today lists `agents/gaze-reporter.md` as user-owned) to say that only `commands/gaze.md` keeps skip-if-present behavior (FR-034)
- [x] T066 [US4] Run `go test -race -count=1 -short ./internal/scaffold/... ./cmd/gaze/...` and confirm that `TestEmbeddedAssetsMatchSource` (manifest still 9), `TestIsToolOwned`, `TestRun_OverwritesCustomizedGazeReporter`, `TestGazeReporterAgent_NoRenderingRules`, and `TestExampleReportMatchesGolden` pass
- [x] T067 [US4] Measure the DE-004 after-values with `wc -l -w -c .opencode/agents/gaze-reporter.md` and `python3 -c "print(len(open('.opencode/agents/gaze-reporter.md',encoding='utf-8').read())//4)"`, and record before/after lines, words, bytes, and ≈tokens and the percentage reduction in `specs/040-deterministic-ai-report-rendering/baseline.md` under "DE-004". If the result exceeds 160 lines or ~1,600 tokens, tighten the prompt (T061/T062) before proceeding (SC-006)
- [x] T068 [US4] Verify SC-007 manually (quickstart §8): in OpenCode on this repository run `/gaze`, `/gaze crap ./internal/crap`, and `/gaze quality ./internal/quality`; confirm each shows the Gaze-rendered report verbatim followed by `## Assessment`, and record the outcome in `baseline.md` under "Checkpoints"

**Checkpoint**: All three `/gaze` modes work through the deterministic renderer; consumers receive the rewrite on `gaze init`.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [x] T069 [P] Update `docs/reference/cli/report.md` (beyond the rubric section from T029): `--format=markdown`, `--sections` (values, default, help text), the format behavior matrix, pre-analysis validation errors, the AI answer shape and its 14 validation rules with bounds (1–1200 summary characters, 1–5 recommendations, 1–400 characters each, 5% non-Latin limit), exit semantics (an AI failure exits non-zero even when thresholds pass; the deterministic report plus rejection callout are still published), the ignore notice, and the `metadata`/`assessment` JSON additions
- [x] T070 [P] Update `docs/guides/ci-integration.md` (FR-039): what the AI now supplies (summary and recommendations only, as validated JSON), validation-failure behavior (the job fails, the step summary still shows the deterministic report), `--format=markdown` as a no-secret option, and a note that the reusable workflow gains this behavior only once its `gaze-version` points at a release containing it
- [x] T071 [P] Update `README.md`: the `gaze report` section gains `--format=markdown` and `--sections`, a description of deterministic rendering plus AI judgment slots, and validation failing the run
- [x] T072 [P] Update `docs/guides/ai-reports.md`: the built-in report-mode prompt replaces the local agent file (which is now ignored, with a notice), the opencode adapter's `--format json` event extraction and the tool-free `gaze-headless` agent, and the shared validator for all adapters
- [x] T073 [P] Update `docs/guides/opencode-integration.md` and `docs/reference/cli/init.md`: `agents/gaze-reporter.md` is now tool-owned and overwritten on `gaze init` when it differs, so local customizations are overwritten (FR-034); refresh the `/gaze` description (verbatim render plus judgment)
- [x] T074 Add an AGENTS.md "Recent Changes" entry for `040-deterministic-ai-report-rendering` in `AGENTS.md`: new files (`sections.go`, `grade.go`, `assess.go`, `render_markdown.go`, `answer.go`, `opencode_events.go`, `assets/report-mode-prompt.md`, golden and stream fixtures), removals (`LoadPrompt`, the embedded prompt copy, `TestEmbeddedPromptMatchesScaffold`, File Sync row 2), `--format=markdown` and `--sections`, rubric v1, the gaze-headless agent, `gaze-reporter.md` being tool-owned, the DE-004 line and token deltas from T067 (528 lines / ≈5,118 tokens → measured values), and the website issue number from T079. Also add the Active Technologies line for 040 (Go 1.25+, standard library only)
- [x] T075 GoDoc and comment audit: every new or changed exported symbol in `internal/aireport/` (`Section`, `SectionSet`, `ParseSections`, `AllSections`, `StepSet`, `CheckThresholds`, `ReportMetadata`, `Assessment`, `ScorecardRow`, `Recommendation`, `Grade`, `Severity`, `Assess`, `RenderMarkdown`, `AIOutcome`, `AIAnswer`, `AnswerRejection`, `ParseAIAnswer`, `ReportModePrompt`), `crap.ReadModulePath`, and the package doc comments has GoDoc; non-obvious decisions cite their research entry (R1–R18)
- [x] T076 DE-007 seam verification (all three layers, FR-035): run and confirm each named test exists and passes. **Layer 1 — registration surface / input-schema parity**: `TestNewReportCmd_FlagsRegistered`, `TestValidateReportParams_FormatMarkdown`, `TestRunReport_SectionsForwarded` (flag → `reportParams` → `RunnerOptions` → `metadata.sections`), `TestRunTextPath_ValidatesEveryAdapterOutput` and `TestOpenCodeAdapter_Replay_*` (every adapter's output → `ParseAIAnswer`), and `TestReportModePrompt_*` plus `TestEmbeddedAssetsMatchSource` (prompt embedded, not scaffolded; manifest 9). **Layer 2 — documented output shape via golden markdown**: `TestRenderMarkdown_Golden` and `TestRunReport_Golden` reading the SAME `internal/aireport/testdata/golden/report-*.md` files (SC-003), `TestRenderMarkdown_ByteIdentity` (SC-002), `TestRubricDocMatchesCode`, and `TestExampleReportMatchesGolden`. **Layer 3 — write→read round trip**: `TestRunReport_AssessmentRoundTrip` (grades and recommendations written to `--format=json`, decoded, equal to the markdown, per SC-005). Command: `go test -race -count=1 -short -v ./internal/aireport/... ./cmd/gaze/... ./internal/scaffold/... -run 'FlagsRegistered|FormatMarkdown|SectionsForwarded|ValidatesEveryAdapterOutput|OpenCodeAdapter_Replay|ReportModePrompt|EmbeddedAssetsMatchSource|Golden|ByteIdentity|RubricDocMatchesCode|ExampleReportMatchesGolden|AssessmentRoundTrip|Incident'`; record the pass list in `baseline.md` under "Seam verification"
- [x] T077 CI parity gate (AGENTS.md Technical Guardrails): read `.github/workflows/test.yml` and `.github/workflows/mega-linter.yml` and run the SAME commands locally, deriving them from those files (do not rely on this list if the workflows differ): `.github/scripts/test-validate-module-major.sh`; `.github/scripts/test-validate-workflow-floor.sh`; `go build ./...`; `go test -race -count=1 -short -timeout 15m -coverprofile=coverage.out ./...`; the "Gaze threshold check" step exactly as written in test.yml (thresholds 38/6/8, UNCHANGED) using the fresh `coverage.out`; the e2e suite `go test -race -count=1 -run 'TestRunSelfCheck' -timeout 30m ./cmd/gaze/...`; `golangci-lint run`; and, where installed, the MegaLinter linters (markdownlint on the changed docs, yamllint, gitleaks). Any failure blocks completion. If a protected gate fails, STOP and report rather than editing it
- [x] T078 Coverage ratchet and CRAP comparison: compare the per-package coverage of `internal/aireport`, `cmd/gaze`, `internal/scaffold`, and `internal/crap` and the `gaze crap --format=json ./...` CRAPload and GazeCRAPload against `baseline.md` (T001); confirm no package coverage fell, `TestSC003_MappingAccuracy` and the other ratchets are unchanged, and every new function has CC ≤ 10. Record the final numbers in `baseline.md`. On a regression, fix it by adding tests or decomposing, never by changing a gate
- [x] T079 File the website documentation issue (FR-040) before the PR merges: `gh issue create --repo unbound-force/website --title "docs: gaze report deterministic rendering, --format=markdown and --sections" --body "<what changed: Gaze renders every report section, grade, and severity itself; the AI supplies only a validated summary and recommendations; invalid AI output fails the job; new --format=markdown (no AI provider) and --sections flags; the local .opencode/agents/gaze-reporter.md is ignored by gaze report; gaze init now overwrites gaze-reporter.md (tool-owned). Pages: gaze report CLI reference, CI integration guide, /gaze agent docs, gaze init docs>"`. Record the issue number in the AGENTS.md entry (T074) and the PR description
- [x] T080 File the SC-008 follow-up tracking issue (not implemented in this spec; research R13): `gh issue create --repo unbound-force/gaze --title "ci: bump reusable_gaze_ai_report gaze-version to the first release with deterministic report rendering (spec 040 SC-008)" --body "<the dogfood Gaze AI Report job installs gaze@v2.2.0 via the reusable workflow's gaze-version default, so spec 040's code does not run there until a release ships it and the default/dogfood input is bumped (its own OpenSpec change). After the bump, verify quickstart §10: the job publishes the full deterministic report with validated AI slots or fails, and never passes with narration. Also note the DE-005 candidate: collapsing /gaze binary resolution and analyzer detection into one intent-level tool.>"`; record the issue number in the PR description
- [x] T081 Run `/uf.review-council` on the final diff and resolve every REQUEST CHANGES finding until all reviewers APPROVE (AGENTS.md Council Governance Protocol). Re-run T077 after any fix. Make no further code changes between APPROVE and PR submission
- [x] T082 Prepare the PR description: summary, the incident reference (test.yml run 41309423609), the DE-004 before/after line and token deltas (FR-032), the release note that `gaze init` now overwrites customized `.opencode/agents/gaze-reporter.md` (FR-034), the release note that `gaze report` ignores the local agent file, the website issue (T079), and the SC-008 follow-up issue (T080), stating that the dogfood job will not exercise this code until a release plus a `gaze-version` bump

---

## Follow-up (tracked, NOT implemented in this spec)

- **SC-008 post-release verification** (research R13): the dogfood
  `Gaze AI Report` job runs `gaze@v2.2.0` from the reusable
  workflow's `gaze-version` default. This feature's behavior reaches
  CI only after (1) a release containing it is cut and (2) a
  separate CI-config change (its own OpenSpec) bumps
  `reusable_gaze_ai_report.yml`'s `gaze-version` default and the
  dogfood input. Then verify quickstart §10. This is tracked by the
  issue filed in T080. This spec does not modify any workflow file.
- **DE-005 candidate**: collapse `/gaze` binary resolution and
  analyzer detection into a single intent-level tool (research R16).
  This is out of scope here.

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1 Setup** has no dependencies.
- **Phase 2 Foundational** depends on Phase 1 (fixtures) and blocks
  every story.
- **Phase 3 US2 (P1)** depends on Phase 2. It is the foundation US1
  and US3 build on (renderer, assessment, golden files, CLI format).
- **Phase 4 US1 (P1)** depends on US2 (`RenderMarkdown`, `AIOutcome`,
  golden files).
- **Phase 5 US3 (P2)** depends on US2 (markdown path, `--sections`
  flag). It is independent of US1 and can run in parallel with it.
- **Phase 6 US5 (P3)** depends on US1 (T036 removes the `LoadPrompt`
  call and T039 adds the capture mode to the fake binary).
- **Phase 7 US4 (P2)** depends on US2 (golden `report-full.md` for the
  example) and US3 (`--sections` for crap and quality modes). T060
  depends on T050.
- **Phase 8 Polish** depends on all stories. T081 and T082 come last.

### Story completion order

```text
Setup → Foundational → US2 ─┬─► US1 ─► US5 ─┐
                            └─► US3 ────────┴─► US4 ─► Polish
```

### Within each story

- Write the tests first and confirm they fail, then implement.
- Types come before functions, functions before wiring, and wiring
  before the CLI.
- The phase checkpoint (T030, T044, T051) must pass before the next
  phase starts.
- `render_markdown.go` tasks T019–T023 are sequential (same file).
  `assess.go` tasks T017–T018 are sequential.
- `cmd/gaze/main.go` tasks (T027, T036 CLI part, T050, T053, T060) are
  sequential across stories (same file). The `cmd/gaze/main_test.go`
  tasks are likewise sequential.

---

## Parallel Examples

### Setup

```text
T002 payload-full.json  ‖  T003 payload-partial.json  ‖  T004 opencode_streams/*
```

### Foundational

```text
T007 sections_test.go  ‖  T009 crap.ReadModulePath   (after T006 / independent)
```

### User Story 2

```text
T011 grade_test.go  ‖  T012 assess_test.go  ‖  T013 render_markdown_test.go  ‖  T015 grade_doc_test.go
then T016 grade.go → T017/T018 assess.go → T019–T023 render_markdown.go
T025 compact.go can run alongside T019–T023
```

### User Story 1

```text
T031 answer_test.go  ‖  T032 opencode_events_test.go  ‖  T033 prompt_test.go
T034 answer.go  ‖  T037 opencode_events.go  ‖  T035 report-mode-prompt.md
then T036 runner.go → T038 adapter_opencode.go → T039 fake_opencode → T040–T043
```

### User Story 3

```text
T045 pipeline_internal_test.go  ‖  T046 runner_test.go (gate)  ‖  T047 main_test.go
```

### User Story 4

```text
T057 main_test.go  ‖  T058/T059 scaffold_test.go  ‖  T063 example-report.md  ‖  T064 gaze.md
```

### Polish

```text
T069 report.md  ‖  T070 ci-integration.md  ‖  T071 README.md  ‖  T072 ai-reports.md  ‖  T073 opencode-integration.md + init.md
```

---

## Implementation Strategy

### MVP first (US2 + US1, both P1)

1. Complete Phase 1 Setup and Phase 2 Foundational.
2. Complete Phase 3 (US2): the deterministic renderer, rubric, and
   golden files. Validate with `TestRenderMarkdown_Golden` and the
   round trip.
3. Complete Phase 4 (US1): the validator, report-mode prompt, and
   opencode hardening. Validate with the incident regression.
4. **STOP and validate**: the incident can no longer pass CI. This is
   the minimum shippable fix.

### Incremental delivery

1. Add US3 (the zero-secret `--format=markdown` with `--sections`).
2. Add US5 (remove the legacy prompt plumbing and add the notice).
3. Add US4 (the `/gaze` rewrite, tool ownership, external parity, and
   DE-004).
4. Polish: docs, website issue, CI parity, review council, PR.

### Notes

- `[P]` tasks touch different files and have no dependency on an
  incomplete task.
- Mark each task `[x]` immediately after completing it (AGENTS.md Task
  Completion Bookkeeping).
- Completed spec artifacts (spec.md, plan.md, research.md,
  data-model.md, contracts/) are immutable during implementation.
  Only tasks.md checkboxes and `baseline.md` (implementation record)
  change.
