# Implementation Plan: Deterministic AI Report Rendering

**Branch**: `040-deterministic-ai-report-rendering` | **Date**: 2026-11-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/040-deterministic-ai-report-rendering/spec.md`
**Incident**: Gaze AI Report CI job, `test.yml` run 41309423609 (no originating issue)

## Summary

Gaze will render every deterministic part of the human-readable
quality report itself: layout, tables, emoji, grades, severities, and
metric wording. The AI is confined to two judgment slots (`summary`,
`recommendations`). It returns them as strict JSON, and Gaze
validates that JSON before publishing.

Technical approach:

1. **Renderer and grading.** A pure markdown renderer
   (`RenderMarkdown`) and a published grade rubric (`Assess`) are
   added to `internal/aireport`. The payload gains additive
   `metadata` and `assessment` JSON objects.
2. **New format and flag.** `--format=markdown` (no AI) and
   `--sections` (crap, quality, classification, health) are added.
3. **One built-in prompt for every adapter.** A small embedded
   report-mode prompt replaces the scaffolded `/gaze` prompt in the
   pipeline. The local `.opencode/agents/gaze-reporter.md` is ignored,
   with a notice. All four adapters' output passes through one
   deterministic validator (`ParseAIAnswer`, 14 rules). A rejection
   fails the job and still publishes the deterministic report.
4. **opencode adapter hardening.** It switches to
   `--format json` and extracts only the final step's text. It runs a
   unique `gaze-headless` agent with `mode: primary` and
   `permission: {"*": deny}`, and it fails closed if a tool actually
   executes. The event schema and permission keys were **verified
   against opencode 1.19.0** (research R1/R2).
5. **`/gaze` agent rewrite.** The agent becomes a thin consumer of
   `gaze report --format=markdown`. It shows the output verbatim and
   adds only judgment. The 528-line prompt is cut to about 160 lines,
   and the file becomes tool-owned (FR-034, option A).

## Technical Context

**Language/Version**: Go 1.25+ (per the `go.mod` directive)
**Primary Dependencies**: Standard library only (`encoding/json`,
`strings`, `strconv`, `unicode`, `embed`, `errors`). The existing
internal packages are `crap`, `taxonomy`, `docscan/apidoc`, `adapter`,
`scaffold`, and `cliutil`. There are no new modules.
**Storage**: N/A. The pipeline is ephemeral; the only files written
are the existing temp agent dir (opencode adapter) and the step
summary.
**Testing**: The standard `testing` package with `-race -count=1`.
Golden files live under `internal/aireport/testdata/golden/` and are
regenerated with a test-only `-update` flag. Fake adapter binaries
(`fake_opencode` replay mode) and the existing `FakeAdapter` are
used.
**Target Platform**: CLI binary (darwin/linux × amd64/arm64), GitHub
Actions consumers
**Project Type**: Single Go module CLI
**Performance Goals**: Rendering is O(rows) and well under 50 ms;
analysis cost is unchanged. `--sections` lets crap-only or
quality-only runs skip unneeded analysis steps (FR-027).
**Constraints**:
- The output is byte-deterministic for a given payload (FR-003).
- The compact AI payload stays bounded (the assessment adds ≤ 13
  small objects).
- Dogfood CI gates 38/6/8 must not change, and there is about zero
  GazeCRAPload headroom (R12).
- Agent `tools:` frontmatter must not change (protected gate).
**Scale/Scope**: About 9 new production files and 8 modified ones;
about 1,300 lines of production Go and about 2,000 lines of tests and
fixtures. Two agent and reference markdown pairs are rewritten, and
5 docs pages are updated.

All former NEEDS CLARIFICATION items (FR-014 permission keys, FR-016
event format, rubric boundaries, validation bounds) are resolved in
[research.md](research.md) R1–R4.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design (below).*

### Pre-Design Check

| Principle | Status | Evidence |
|-----------|--------|----------|
| **I. Accuracy** | ✅ PASS | Fixes a false pass: the CI gate published narration and exited 0. Metrics are rendered verbatim from analysis data, never re-thresholded (FR-006), and unavailable data shows N/A, never a proxy (FR-005). The incident has an automated regression test (FR-036/SC-001). |
| **II. Minimal Assumptions** | ✅ PASS | No new requirements on host projects. `--format=markdown` needs no AI provider. External-analyzer projects keep working, with parity extended to the report pipeline (R9). The pinned opencode version is an explicit, documented assumption, enforced by a fail-closed extractor. |
| **III. Actionable Output** | ✅ PASS | Deterministic recommendations name a function, file:line, metric, and fix strategy. Output is identical across runs and adapters (FR-003). Grades appear in JSON as a stable contract (FR-010). Validation errors name the violated rule (FR-019). |
| **IV. Testability** | ✅ PASS | The renderer, rubric, validator, and event extractor are pure functions. The CLI uses the existing `runXxx(params)` pattern with injected `analyzeFunc` and clock. The coverage strategy is specified below. |

**Gate Result**: PASS. No violations.

### Post-Design Re-Check

| Principle | Status | Evidence |
|-----------|--------|----------|
| I | ✅ PASS | The grade is computed from the displayed value (R3), so the shown number and the grade cannot disagree. The zero-result gate is fixed so a quality-only selection does not trip it falsely (R8). A completed tool call fails closed (R1). |
| II | ✅ PASS | The JSON additions are additive. HTML is unchanged. The `--ai-mapper` interface is unchanged (R14). |
| III | ✅ PASS | The golden files are the human-readable contract. JSON↔markdown round-trip invariants are defined in [contracts/assessment-json.md](contracts/assessment-json.md). |
| IV | ✅ PASS | All three DE-007 seam layers have concrete tests (see Seam Verification). No new function is untestable without external services. |

## Coverage Strategy (Constitution Principle IV — mandatory)

| Unit (new or changed) | Test kind | Contract asserted | Target |
|-----------------------|-----------|-------------------|--------|
| `ParseAIAnswer` + rule helpers (`answer.go`) | Unit, table-driven, two cases per rule (violate / nearest pass) | Returned `AIAnswer` or `*AnswerRejection.Rule`; no raw echo in `Detail` | 100% of rule branches; ≥ 95% statements |
| Rubric (`grade.go`) | Unit, table-driven boundary matrix (SC-004) | `Grade`, `Severity`, `Marker`, `Detail` per input | 100% statements |
| `Assess` + recommendations (`assess.go`) | Unit, synthetic payloads | Scorecard rows (5, ordered), recommendation order, cap, templates, N/A reasons | ≥ 95% |
| `RenderMarkdown` + section renderers (`render_markdown.go`) | Golden (6 files) + targeted unit (escaping, title, callouts, vocabulary scan, slot placement) | Exact output bytes | ≥ 90% |
| `ParseSections` / `CheckThresholds` (`sections.go`) | Unit | Errors and step sets | 100% |
| `extractFinalText` + `OpenCodeAdapter.Format` | Unit on fixture bytes + integration via the `fake_opencode` replay | Returned text and errors | ≥ 95% |
| `Run` / `runTextPath` / new `runMarkdownPath`, step selection in `runProductionPipeline`, `checkZeroResultGate` | Unit via `AnalyzeFunc` + `FakeAdapter` + `pipelineStepFuncs` | stdout and step-summary bytes, joined errors, steps invoked | ≥ 90% of changed functions |
| `CompactForAI` additions | Existing compact tests extended | `metadata` / `assessment` keys present and bounded | No regression |
| `runReport` / `validateReportParams` / notice / metadata (cmd/gaze) | CLI unit via `reportParams` with `analyzeFunc`, `now`, buffers | Exit errors, notice line, CLI golden, JSON↔markdown round trip | ≥ 85% of changed functions |
| External quality in `runExternalReportCRAP` | Integration with the existing protocol fake analyzer | Quality JSON shape, `AvgContractCoverage`, classification counts, degradation errors | ≥ 85% |
| `isToolOwned`, embedded asset sync, example↔golden | Existing scaffold tests updated + new `TestExampleReportMatchesGolden` | Ownership table; byte equality | 100% |
| Prompt and agent content | `TestReportModePrompt_*`, `TestGazeReporterAgent_NoRenderingRules` | Required and forbidden content | n/a (content tests) |

**Ratchets and gates**:
- Package coverage for `internal/aireport`, `cmd/gaze`, and
  `internal/scaffold` must not fall below the pre-change baseline.
  The baseline is captured in the first task.
- `TestSC003_MappingAccuracy` and the other existing ratchets must
  stay unchanged.
- Dogfood thresholds stay **38/6/8, unchanged**.
- Every new function keeps cyclomatic complexity ≤ 10 and is
  contract-asserted, so it lands in Q1/Q2 and not in
  GazeCRAPload (R12).
- `gaze crap --format=json ./...` is compared against the baseline at
  each phase checkpoint. If a gate would fail, work **stops and the
  conflict is reported**; the gate is never edited.

**Test tiers**: Everything above is unit or integration level and
runs under `-short`. The fake binaries are built per test. No e2e
test is added. SC-009 (the real repository run) is a quickstart
scenario. `TestRunSelfCheck` is unaffected.

## Seam Verification (DE-007 — all three layers in this increment)

| Layer | Seam | Test |
|-------|------|------|
| **1. Registration surface** | `--format=markdown` and `--sections` registered on the Cobra command and forwarded through `reportParams` → `RunnerOptions` | `TestNewReportCmd_FlagsRegistered` (flag existence, defaults, help text); `TestRunReport_SectionsForwarded`; `TestValidateReportParams_FormatMarkdown` |
| | The adapter structured-answer path is wired: every adapter's output → `ParseAIAnswer` | `TestRunTextPath_ValidatesEveryAdapterOutput` (FakeAdapter); `TestOpenCodeAdapter_Replay_*` (fake binary) |
| | Report-mode prompt registered (embedded, not scaffolded) | `TestReportModePrompt_*`, scaffold manifest still 9 (`TestEmbeddedAssetsMatchSource`) |
| **2. Documented output shape** | Shared golden markdown | `TestRenderMarkdown_Golden` (renderer) + `TestRunReport_Golden` (CLI) on the same files (SC-003) |
| | Rubric docs ≡ code | `TestRubricDocMatchesCode` parses the `docs/reference/cli/report.md` rubric table |
| | Agent example ≡ real output | `TestExampleReportMatchesGolden` |
| **3. Write→read round trip** | Grades and recommendations written to JSON, read back, equal to what the markdown shows | `TestRunReport_AssessmentRoundTrip` (SC-005) per [contracts/assessment-json.md](contracts/assessment-json.md) |
| | Payload metadata written by the CLI, read by the renderer | Covered by the CLI golden (fixed clock, branch env) |

## Project Structure

### Documentation (this feature)

```text
specs/040-deterministic-ai-report-rendering/
├── spec.md
├── plan.md                    # This file
├── research.md                # Phase 0 (R1–R18; opencode 1.19.0 verified)
├── data-model.md              # Phase 1
├── quickstart.md              # Phase 1
├── contracts/
│   ├── cli-report.md          # flags, format matrix, errors, exit semantics, notice
│   ├── ai-answer.md           # strict JSON shape + 14 validation rules
│   ├── opencode-adapter.md    # invocation, agent file, event extraction, fake binary
│   ├── grade-rubric.md        # rubric v1, severity map, deterministic recommendations
│   ├── markdown-report.md     # layout, slots, callouts, golden files
│   ├── assessment-json.md     # additive JSON + round-trip invariant
│   └── agents-and-prompts.md  # report-mode prompt, /gaze rewrite, DE-004, ownership
├── checklists/requirements.md
└── tasks.md                   # Phase 2 (/speckit.tasks — not created here)
```

### Source Code (repository root)

```text
cmd/gaze/
  main.go                      # report cmd: --format=markdown, --sections, metadata+clock,
                               #   ignore notice, remove LoadPrompt, external quality step
  main_test.go                 # CLI wiring, CLI golden, round trip, notice tests

internal/aireport/
  sections.go        (new)     # Section, SectionSet, ParseSections, StepSet, CheckThresholds
  grade.go           (new)     # rubric v1 tables, grade/severity/marker, displayed-value parsing
  assess.go          (new)     # Assess(): scorecard + deterministic recommendations
  render_markdown.go (new)     # RenderMarkdown + per-section/per-table renderers, cell escaping
  answer.go          (new)     # AIAnswer, AnswerRejection, ParseAIAnswer (14 rules)
  opencode_events.go (new)     # openCodeEvent/part, extractFinalText
  prompt.go                    # REWRITE: embed assets/report-mode-prompt.md; ReportModePrompt()
  assets/report-mode-prompt.md (new)
  assets/agents/gaze-reporter.md (DELETE)
  adapter_opencode.go          # --format json, gaze-headless agent file, extractFinalText
  runner.go                    # Sections/Metadata opts, markdown path, AI outcome, gate fix,
                               #   step selection; SystemPrompt opt removed
  payload.go                   # + Metadata, + Assessment fields
  compact.go                   # + metadata/assessment in compact payload
  *_test.go                    # new + updated tests; prompt_test.go rewritten
  testdata/golden/             (new) payload-*.json, report-*.md
  testdata/opencode_streams/   (new) *.jsonl fixtures
  testdata/fake_opencode/main.go # replay mode

internal/crap/coverage.go      # export ReadModulePath (was readModulePath)
internal/scaffold/scaffold.go  # isToolOwned += "agents/gaze-reporter.md"
internal/scaffold/scaffold_test.go # ownership expectations; TestExampleReportMatchesGolden

.opencode/agents/gaze-reporter.md        ≡ internal/scaffold/assets/agents/gaze-reporter.md   (rewrite)
.opencode/references/example-report.md   ≡ internal/scaffold/assets/references/example-report.md (real golden)
.opencode/commands/gaze.md               ≡ internal/scaffold/assets/commands/gaze.md (wording only)

docs/reference/cli/report.md   # --format=markdown, --sections, validation, Grading rubric section
docs/guides/ci-integration.md  # what the AI supplies, failure behavior, no-secret markdown option
docs/guides/ai-reports.md      # report-mode prompt, opencode json events / gaze-headless agent
docs/guides/opencode-integration.md, docs/reference/cli/init.md # gaze-reporter now tool-owned
README.md, AGENTS.md           # Recent Changes; remove File Sync Invariants row 2
```

**Structure Decision**: Everything lives in the existing package
layout. The renderer, rubric, and validator go in `internal/aireport`
because they consume `ReportPayload` (a separate package would create
an import cycle) and sit beside the existing native HTML renderer.
There is no new package and no new dependency. `.mega-linter.yml`
and the CI workflows are not modified (R13, R15).

## Design Decisions (summary; rationale in research.md)

| ID | Decision | Ref |
|----|----------|-----|
| D1 | Final opencode answer = text parts of the last `step_finish` message; fail closed on a completed tool or an `error` event | R1 |
| D2 | Unique `gaze-headless` agent, `mode: primary`, `permission: {"*": deny}` | R2 |
| D3 | Grade on the displayed one-decimal value; integer-tenths ratios; rubric v1 | R3 |
| D4 | Per-field non-English ratio; reject markup, never escape or repair | R4, R5 |
| D5 | AI slots delimited by HTML-comment markers; omitted entirely with no AI | R6 |
| D6 | Metadata captured into the payload; no Go version line; injectable clock | R7 |
| D7 | Sections map to steps; incompatible thresholds rejected up front; zero-result gate only when CRAP runs | R8 |
| D8 | External-analyzer quality and classification wired into the report pipeline for `/gaze` parity | R9 |
| D9 | `AIAdapter` interface unchanged; one validator in the runner for all adapters | R14 |
| D10 | Report-mode prompt embedded under `internal/aireport/assets/`, never scaffolded | R15 |
| D11 | `/gaze` agent: verbatim render + judgment only; frontmatter tool access unchanged; tool-owned | R16 |
| D12 | On AI failure, still publish the deterministic report with a callout, then exit non-zero (`errors.Join` with threshold errors) | contracts/cli-report.md |

## Phasing (input to /speckit.tasks)

1. **Setup**: capture coverage and CRAP baselines; add the golden
   payload fixtures.
2. **US2 core (P1)**: sections, rubric, assessment, renderer, golden
   files, `--format=markdown`, JSON additions, round trip. This is
   the foundation US1 and US3 depend on.
3. **US1 (P1)**: validator, report-mode prompt, runner text path with
   outcome, opencode JSON events and agent, fake replay, incident
   regression.
4. **US3 (P2)**: section-driven step selection, threshold
   compatibility, step summary, zero-result gate fix (CLI surface).
5. **US5 (P3)**: remove `LoadPrompt`, the embedded copy, and the sync
   test; add the ignore notice; update AGENTS.md row.
6. **US4 (P2)**: external quality parity; `/gaze` agent and
   example-report rewrite; tool-owned; DE-004 measurement.
7. **Polish**: docs (FR-039), website issue (FR-040), CI parity run,
   review council.
8. **Follow-up (tracked, not implemented here)**: a post-release
   bump of the reusable workflow's `gaze-version` so the dogfood job
   runs this code (SC-008, R13). The DE-005 collapse of binary
   resolution and analyzer detection is a candidate.

## Risks

| Risk | Mitigation |
|------|-----------|
| GazeCRAPload dogfood ceiling (6) has about zero headroom | CC ≤ 10 per function, contract assertions on every new function, per-phase gaze comparison, stop-and-report on breach (R12) |
| opencode event schema changes after 1.19.0 | Fail-closed extractor naming the pinned version; fixtures pin the schema (R1) |
| Name collision or subagent fallback runs the build agent with tools | Unique name, explicit `mode: primary`, fail-closed on a completed tool (R2) |
| Dogfood keeps running v2.2.0 until release + bump | Documented as a post-release verification and tracked follow-up (R13) |
| Overwriting consumer customizations of `gaze-reporter.md` | Human-approved (FR-034 option A); docs and release notes state it |

## Complexity Tracking

No constitution violations, so this section is intentionally empty.
