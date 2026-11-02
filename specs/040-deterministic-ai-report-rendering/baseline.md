# Baseline: 040-deterministic-ai-report-rendering

Implementation record for spec 040. T001 captured the pre-change
baselines below. Later tasks (T030, T044, T051, T067, T068, T076,
T078) compare against these values and append their results here.

## T001 capture context

- **Captured**: 2026-11-01
- **Branch**: `040-deterministic-ai-report-rendering`
- **HEAD**: `524a153ace183b5ed69a1430906cdd604bda7c38` (clean tree, no
  production code changes yet)
- **Toolchain**: `go version go1.26.5 darwin/arm64` (local; CI runs Go
  1.24 and 1.25)
- **Gaze binary**: built from HEAD with
  `go build -o "$TMPDIR/gaze" ./cmd/gaze` (dev build)

## (a) Per-package statement coverage

Command:

```bash
go test -race -count=1 -short -coverprofile="$TMPDIR/040-base.out" \
  ./internal/aireport/... ./cmd/gaze/... ./internal/scaffold/... ./internal/crap/...
go tool cover -func="$TMPDIR/040-base.out"
```

All four packages passed.

| Package | Statement coverage |
|---------|--------------------|
| `internal/aireport` | 84.7% |
| `cmd/gaze` | 77.3% |
| `internal/scaffold` | 90.1% |
| `internal/crap` | 94.8% |
| **Total (`go tool cover -func`, 4 packages)** | **85.5%** |

`go tool cover -func` listed 216 functions across the four packages.
The coverage ratchet (T078) is: none of these four package
percentages may fall.

## (b) `gaze crap --format=json ./...`

Command:

```bash
go build -o "$TMPDIR/gaze" ./cmd/gaze && \
  "$TMPDIR/gaze" crap --format=json ./... > "$TMPDIR/040-baseline-crap.json"
```

Exit 0. `gaze crap` generates its own coverage profile internally.

| Metric | Value |
|--------|-------|
| Total functions | 587 |
| **CRAPload** (CRAP ≥ 15) | **26** |
| **GazeCRAPload** (GazeCRAP ≥ 15) | **6** |
| CRAP threshold / GazeCRAP threshold | 15 / 15 |
| Avg complexity | 5.36 |
| Avg line coverage | 86.09% |
| Avg CRAP / Avg GazeCRAP | 6.39 / 7.31 |
| Avg contract coverage (CRAP summary) | 80.11% |
| Quadrants | Q1 92, Q2 1, Q3 5, Q4 1 |
| Fix strategies | decompose 19, add_tests 7 |

### CRAPload functions (26)

| CRAP | CC | Line cov | Strategy | Function | File |
|-----:|---:|---------:|----------|----------|------|
| 120.7 | 24 | 44.8% | decompose | `(*analyzer).walkExpr` | `internal/cognitive/cognitive.go` |
| 101.5 | 18 | 36.4% | decompose | `(*analyzer).walkExprStmt` | `internal/cognitive/cognitive.go` |
| 56.0 | 7 | 0.0% | add_tests | `writeChangeGateText` | `cmd/gaze/main.go` |
| 48.9 | 24 | 64.9% | decompose | `runCrap` | `cmd/gaze/main.go` |
| 42.0 | 6 | 0.0% | add_tests | `buildAIMapperFunc` | `cmd/gaze/main.go` |
| 32.1 | 30 | 86.8% | decompose | `runAnalyze` | `cmd/gaze/main.go` |
| 30.0 | 5 | 0.0% | add_tests | `newCrapCmd` | `cmd/gaze/main.go` |
| 30.0 | 5 | 0.0% | add_tests | `newReportCmd` | `cmd/gaze/main.go` |
| 29.4 | 11 | 46.7% | add_tests | `resolveSSAValueToObj` | `internal/quality/mapping.go` |
| 26.0 | 26 | 97.6% | decompose | `buildSummary` | `internal/crap/analyze.go` |
| 25.9 | 25 | 88.6% | decompose | `Assess` | `internal/quality/quality.go` |
| 24.2 | 21 | 80.6% | decompose | `detectASTPointerArgMutations` | `internal/analysis/mutation.go` |
| 21.0 | 21 | 97.5% | decompose | `funcSignature` | `internal/analysis/returns.go` |
| 20.0 | 20 | 96.4% | decompose | `hintForEffect` | `internal/quality/hints.go` |
| 19.8 | 15 | 72.2% | decompose | `findHelperCallInFuncDepth` | `internal/quality/mapping.go` |
| 18.9 | 18 | 86.2% | decompose | `loadConfig` | `cmd/gaze/main.go` |
| 18.5 | 15 | 75.0% | decompose | `stripFrontmatter` | `internal/aireport/prompt.go` |
| 18.0 | 18 | 100.0% | decompose | `checkQualityThresholds` | `cmd/gaze/main.go` |
| 17.5 | 17 | 87.9% | decompose | `matchInlineCall` | `internal/quality/mapping.go` |
| 17.3 | 17 | 89.7% | decompose | `findSSAFunction` | `internal/analysis/mutation.go` |
| 17.1 | 14 | 75.0% | add_tests | `ParseCoverProfile` | `internal/crap/coverage.go` |
| 17.0 | 17 | 100.0% | decompose | `BuildQualityFromMappings` | `internal/adapter/quality.go` |
| 16.1 | 16 | 91.7% | decompose | `(*analyzer).walkStmt` | `internal/cognitive/cognitive.go` |
| 16.0 | 16 | 100.0% | decompose | `printSummary` | `internal/scaffold/scaffold.go` |
| 15.7 | 10 | 61.5% | add_tests | `validateReportParams` | `cmd/gaze/main.go` |
| 15.7 | 15 | 85.7% | decompose | `runQuality` | `cmd/gaze/main.go` |

### GazeCRAPload functions (6)

GazeCRAPload is already **at the dogfood ceiling of 6**: any new
function that enters GazeCRAPload fails the gate. New functions must
stay out of this list (T030 / T044 / T078).

| GazeCRAP | CC | Quadrant | Function | File |
|---------:|---:|----------|----------|------|
| 48.3 | 8 | Q3 | `(*Session).Initialize` | `internal/adapter/session.go` |
| 42.0 | 6 | Q3 | `WriteStepSummary` | `internal/aireport/output.go` |
| 38.5 | 14 | Q3 | `EvaluateThresholds` | `internal/aireport/threshold.go` |
| 26.1 | 11 | Q3 | `(*Client).Call` | `internal/protocol/client.go` |
| 19.3 | 13 | Q3 | `Scan` | `internal/docscan/scanner.go` |
| 17.0 | 17 | Q4 | `BuildQualityFromMappings` | `internal/adapter/quality.go` |

Two of the six (`WriteStepSummary`, `EvaluateThresholds`) live in
`internal/aireport`, the package this spec modifies.

## (c) Dogfood gate (`gaze report`, thresholds 38/6/8)

Commands:

```bash
go test -race -count=1 -short -coverprofile="$TMPDIR/040-base-all.out" ./...
"$TMPDIR/gaze" report ./... --format=json \
  --max-crapload=38 --max-gaze-crapload=6 --min-contract-coverage=8 \
  --coverprofile="$TMPDIR/040-base-all.out"
```

The `./...` short suite passed (22 packages, exit 0, total statement
coverage 87.3%). The report exited 0 with no pipeline step errors
(`errors.crap/quality/classify/docscan` all `null`).

| Gate | Actual | Threshold | Result |
|------|-------:|----------:|--------|
| CRAPload | 24 | ≤ 38 | **PASS** |
| GazeCRAPload | 6 | ≤ 6 | **PASS** (at ceiling) |
| AvgContractCoverage | 43% (43.17) | ≥ 8 | **PASS** |

Additional report values: 587 functions; quadrants Q1 91, Q2 2, Q3 5,
Q4 1; fix strategies decompose 19, add_tests 5; quality summary
2,427 tests; CRAP-summary avg contract coverage 80.11%.

**Why CRAPload differs between (b) and (c) (26 vs 24)**: the profiles
differ. `gaze crap` generates its own profile; `gaze report` used the
`./...` `-short` profile. Only in (b): `writeChangeGateText`,
`buildAIMapperFunc`, `newCrapCmd`, `newReportCmd`,
`validateReportParams` (all `cmd/gaze`). Only in (c): `LoadModule`
(`internal/loader`), `resolveQualityDeps`, `runCRAPStep`
(`internal/aireport`). The GazeCRAPload set is identical in both
runs. Compare later runs against the same command they came from.

## (d) DE-004 baseline (`.opencode/agents/gaze-reporter.md`)

Command: `wc -l -w -c .opencode/agents/gaze-reporter.md`

| Metric | Value |
|--------|------:|
| Lines | 528 |
| Words | 2,965 |
| Bytes | 20,769 |
| Characters (UTF-8 decoded) | 20,472 |
| **≈ Tokens (characters ÷ 4)** | **≈ 5,118** |

The token figure uses decoded characters, matching T067's method
(`len(open(..., encoding='utf-8').read()) // 4`). Bytes ÷ 4 would
give ≈ 5,192; the gap is the multi-byte emoji in the prompt. All
values match the research.md / contracts expectations (528 / 2,965 /
20,769 / ≈ 5,118).

## Checkpoints

Each phase checkpoint records the gate values at that point. The
protected dogfood gates are CRAPload ≤ 38, GazeCRAPload ≤ 6, and
average complexity ≤ 8.

### Phase 3 checkpoint (T030)

- `go build ./...` — ok
- `go test -race -count=1 -short ./...` — every package ok
- `golangci-lint run` — 0 issues
- `gaze crap --format=json --test-short ./...` — CRAPload 24,
  GazeCRAPload 6, 672 functions, average complexity 5.03, average
  line coverage 87.0% (gates pass; GazeCRAPload at its unchanged
  limit of 6)
- New functions in `grade.go`, `assess.go`, and
  `render_markdown.go` are all CC ≤ 8 (requirement: ≤ 10). The six
  GazeCRAPload members are pre-existing: `BuildQualityFromMappings`,
  `(*Session).Initialize`, `WriteStepSummary`, `EvaluateThresholds`,
  `Scan`, and `(*Client).Call`.

### Phase 4 checkpoint (T044)

- `go build ./...` — ok
- `go test -race -count=1 -short ./...` — every package ok
- `golangci-lint run` — 0 issues
- `gaze crap --format=json --test-short ./...` — CRAPload 24,
  GazeCRAPload 6, 715 functions, average complexity 4.99 (gates
  pass; GazeCRAPload at its unchanged limit)
- New functions in `answer.go`, `opencode_events.go`, and the
  runner helpers are CC ≤ 10 (highest: `isListItem`,
  `markupViolation`, `containsHTMLMarkup`, `isThematicBreak`,
  `ParseAIAnswer`, `codeSpanLetterCounts` at 8–10).

### Phase 5 checkpoint (T051)

Gates:

- `go build ./...` — ok
- `go test -race -count=1 -short ./...` — every package ok
  (`cmd/gaze` 414s, `internal/quality` 499s, `internal/aireport`
  13.7s)
- `golangci-lint run` — 0 issues
- `.github/scripts/test-validate-module-major.sh` — exit 0
  (5 cases)
- `.github/scripts/test-validate-workflow-floor.sh` — exit 0
  (14 cases)
- `gaze crap --format=json --test-short ./...` — CRAPload 24,
  GazeCRAPload 6, 721 functions, average complexity 4.97, average
  line coverage 87.02%, average contract coverage 82.06% (gates
  pass; GazeCRAPload at its unchanged limit of 6)
- The GazeCRAPload members are exactly the six pre-existing
  functions listed under Phase 3; none of the Phase 5 additions
  (`runCRAPInto`, `runQualityInto`, `runClassifyInto`,
  `runDocscanInto`, `withDefaultSteps`, `sectionSteps`,
  `runXInto` helpers, `runExternalReportCRAP` selection branch)
  appear.

Quickstart scenarios 2–4 (SC-009), run with the binary built by
`go build -o "$TMPDIR/gaze" ./cmd/gaze`:

Scenario 2 — deterministic report with no AI:

```text
$ gaze report --format=markdown ./...   # exit=0, 13m45s
$ gaze report --format=markdown ./...   # exit=0, 112 lines
$ diff /tmp/r1.md /tmp/r2.md && echo IDENTICAL
IDENTICAL
$ md5 /tmp/r1.md /tmp/r2.md
0b9d252bc6333688d2bf9ab4e1ef1a86  /tmp/r1.md
0b9d252bc6333688d2bf9ab4e1ef1a86  /tmp/r2.md
```

The report opens with `# 🔍 Gaze Full Quality Report`, carries all
four selected sections, a 5-row scorecard, and no `gaze:ai-`
markers. Two consecutive runs over identical data are
byte-identical (SC-002/SC-009).

Scenario 3 — section selection:

| Command | Exit | Observed |
|---------|-----:|----------|
| `report --format=markdown --sections=crap ./internal/crap` | 0 | title `# 🔍 Gaze CRAP Report`, exactly one `## ` section |
| `report --format=markdown --sections=quality ./internal/quality` | 0 | title `# 🔍 Gaze Quality Report`, exactly one `## ` section |
| `report --format=markdown --sections=bogus ./...` | 1 | `invalid --sections value "bogus": valid sections are crap, quality, classification, health` |
| `report --format=markdown --sections=crap --min-contract-coverage=50 ./...` | 1 | `--min-contract-coverage requires the quality analysis, which --sections=crap does not run` — rejected before any analysis |

Scenario 4 — thresholds and step summary in markdown mode:

```text
$ GITHUB_STEP_SUMMARY=/tmp/summary040.md gaze report --format=markdown \
    --max-crapload=0 ./internal/crap
# exit=1, report printed to stdout
# stderr: CRAPload: 2/0 (FAIL)
$ grep -c "Gaze Full Quality Report" "$GITHUB_STEP_SUMMARY"
1
```

The rendered report is written to stdout and appended verbatim to
`$GITHUB_STEP_SUMMARY` (byte-identical to stdout), and the failing
threshold sets a non-zero exit without suppressing the report
(US3 AS-2/AS-3, FR-029).

### Phase 7 checkpoint (T068)

The OpenCode TUI cannot be driven from the build environment, so the
verification ran the exact command sequence the rewritten agent
documents in `.opencode/agents/gaze-reporter.md` "Run the Report",
against a binary built from this branch
(`go build -o "${TMPDIR:-/tmp}/gaze040" ./cmd/gaze`), with
`OPENCODE_CONFIG_CONTENT` unset to prove no AI adapter is required.

| `/gaze` mode | Command | Exit | Time | Title | Sections |
|--------------|---------|-----:|-----:|-------|----------|
| full | `gaze040 report --format=markdown ./internal/aireport` | 0 | 1m2.5s | `# 🔍 Gaze Full Quality Report` | 4: `## 📊 CRAP Summary`, `## 🧪 Quality Summary`, `## 🏷️ Classification Summary`, `## 🏥 Overall Health Assessment` |
| crap | `gaze040 report --format=markdown --sections=crap ./internal/crap` | 0 | 8.1s | `# 🔍 Gaze CRAP Report` | 1: `## 📊 CRAP Summary` |
| quality | `gaze040 report --format=markdown --sections=quality ./internal/quality` | 0 | 4.9s | `# 🔍 Gaze Quality Report` | 1: `## 🧪 Quality Summary` |

Every run printed `Project: github.com/unbound-force/gaze/v2` and the
selected sections only, so the agent shows the Gaze-rendered report
verbatim and adds the judgment-only `## Assessment` section described
in its prompt (SC-007). The agent prompt deltas are recorded under
DE-004: 138 lines / 774 words / 5,459 bytes / about 1,364 tokens,
against 528 lines and about 5,118 tokens before, both inside the
160-line / 1,600-token target.

## DE-004

Measured after T061/T062 rewrote `.opencode/agents/gaze-reporter.md` and its
byte-identical scaffold mirror.

Command: `wc -l -w -c .opencode/agents/gaze-reporter.md` plus a Python
character count.

| Metric | Before | After | Reduction |
|--------|-------:|------:|----------:|
| Lines | 528 | 138 | 73.9% |
| Words | 2,965 | 774 | 73.9% |
| Bytes | 20,769 | 5,459 | 73.7% |
| Characters (UTF-8 decoded) | 20,472 | 5,457 | 73.3% |
| **≈ Tokens (characters ÷ 4)** | **≈ 5,118** | **≈ 1,364** | **73.3%** |

The target was ≤ 160 lines and ≈ 1,600 tokens; the rewritten agent is
138 lines and ≈ 1,364 tokens, so no further tightening was needed.

What moved out of the prompt: the grade rubric and boundary table, the
quadrant and remediation tables, the emoji/severity vocabulary, the
documentation-weight scoring model, and every instruction to recompute
metrics. The body now resolves the binary, parses the mode, detects the
analyzer, runs `gaze report --format=markdown`, shows the output verbatim,
and adds one `## Assessment` section of judgment. The rendered layout and
grades come from Go (`internal/aireport/render_markdown.go` and
`assess.go`), so the prompt no longer restates them.

## Seam verification

DE-007 requires all three seam layers to be green before the increment is
complete. The verified command is:

```bash
go test -race -count=1 -short -v ./internal/aireport/... ./cmd/gaze/... ./internal/scaffold/... \
  -run 'FlagsRegistered|FormatMarkdown|SectionsForwarded|ValidatesEveryAdapterOutput|OpenCodeAdapter_Replay|ReportModePrompt|EmbeddedAssetsMatchSource|Golden|ByteIdentity|RubricDocMatchesCode|ExampleReportMatchesGolden|AssessmentRoundTrip|Incident'
```

Result: `exit 0`, 23 passing tests, 0 failures. Pass list:

| Layer | Test | What it proves |
|-------|------|----------------|
| 1 — registration surface | `TestNewReportCmd_FlagsRegistered` | `--sections` exists with the four-section default; `--format` help lists `markdown` |
| 1 | `TestValidateReportParams_FormatMarkdown` | `markdown` is accepted without `--ai`; unknown formats and incompatible thresholds produce the documented errors before analysis |
| 1 | `TestRunReport_SectionsForwarded` | the parsed section set reaches `RunnerOptions` |
| 1 | `TestReportModePrompt_Content`, `_ExampleAnswerParses`, `_Length`, `_NotScaffolded` | the embedded prompt exists, is bounded, parses through the validator, and is never scaffolded |
| 1 | `TestEmbeddedAssetsMatchSource` | the nine scaffold assets remain byte-identical to `.opencode/` |
| 1 | `TestOpenCodeAdapter_Replay_NarrationThenAnswer`, `_AnswerOnly`, `_ToolRefused`, `_ToolCompletedFailsClosed`, `_ErrorEventFails`, `_NotJSONFails`, `TestOpenCodeAdapter_AgentFileIsToolFree` | the adapter runs the tool-free `gaze-headless` agent and extracts only the final answer (these subprocess tests skip under `-short` and pass in the full run: `go test -race -count=1 -run 'TestOpenCodeAdapter' ./internal/aireport/`) |
| 2 — documented output shape | `TestRenderMarkdown_Golden` | the renderer reproduces all six golden reports |
| 2 | `TestRunReport_Golden` | the CLI reproduces the same golden files through `runReport` (SC-003) |
| 2 | `TestRenderMarkdown_ByteIdentity` | identical data renders identical bytes outside the AI slots (SC-002) |
| 2 | `TestRubricDocMatchesCode` | the published grading-rubric table equals the rubric v1 constants |
| 2 | `TestExampleReportMatchesGolden` | the scaffolded example report embeds the golden report byte for byte |
| 3 — write→read round-trip | `TestRunReport_AssessmentRoundTrip` | every markdown scorecard grade and recommendation line equals the JSON `assessment` (SC-005) |
| regression | `TestIncident_NarrationReplay_ExitsNonZero`, `TestIncident_ChineseAnswer_ExitsNonZero`, `TestRunReport_IncidentReplay_ExitsNonZero` | the incident stream and a non-English answer exit non-zero while still publishing the deterministic report (SC-001) |

The runner validator seam is covered by `TestRun_TextPath_RejectedAnswer_PublishesReportAndFails`,
`TestRun_TextPath_AdapterError_PublishesUnavailable`, and
`TestRun_TextPath_JoinsAIAndThresholdErrors`, which pass in
`go test -race -count=1 -run 'TestRun_TextPath' ./internal/aireport/`.

## CI parity gate (T077)

Commands were derived from `.github/workflows/test.yml` and
`.github/workflows/mega-linter.yml`, then run locally without changing any
workflow, threshold, or script.

| Step (as CI runs it) | Result |
|----------------------|--------|
| `.github/scripts/test-validate-module-major.sh` | exit 0 |
| `.github/scripts/test-validate-workflow-floor.sh` | exit 0 |
| `go build ./...` | exit 0 |
| `go test -race -count=1 -short -timeout 15m -coverprofile=coverage.out ./...` | exit 0 (22 packages ok, no failures) |
| Gaze threshold check: `go build -o gaze ./cmd/gaze` then `./gaze report ./... --format=json --coverprofile=coverage.out --max-crapload=38 --max-gaze-crapload=6 --min-contract-coverage=8 > /dev/null` | exit 0 (gates 38/6/8 unchanged) |
| `go test -race -count=1 -run 'TestRunSelfCheck' -timeout 30m ./cmd/gaze/...` | exit 0 (1456s) |
| `golangci-lint run` | exit 0, 0 issues |
| `yamllint .` | not runnable locally (the pyenv shim cannot resolve a Python for the tool); CI runs it in the MegaLinter container |
| `markdownlint` | not installed locally; CI runs it in the MegaLinter container |
| `gitleaks detect` | 2 findings, both pre-existing: `internal/aireport/subprocess_test.go:107` and `:168` (generic-api-key on obviously fake `test-secret-...` constants). The file is byte-identical to `upstream/main` (`git diff --stat upstream/main -- internal/aireport/subprocess_test.go` is empty), so the findings are not branch-caused. |

The dogfood gate therefore passes on this branch with unchanged thresholds:
CRAPload 23 of 38, GazeCRAPload 6 of 6, contract coverage 40% of 8.

## Final comparison

Measurements after all phases, from the CI-parity run described above
(`go test -race -count=1 -short -timeout 15m -coverprofile=coverage.out ./...`
plus `gaze report ./... --format=json --coverprofile=coverage.out`).

### Package coverage (statement-weighted from the fresh profile)

| Package | Baseline (T001) | Final | Change |
|---------|----------------:|------:|-------:|
| `internal/aireport` | 84.7% | 90.2% | +5.5 |
| `cmd/gaze` | 77.3% | 78.4% | +1.1 |
| `internal/scaffold` | 90.1% | 90.1% | 0.0 |
| `internal/crap` | 94.8% | 94.8% | 0.0 |
| module total | — | 88.1% | — |

No package lost coverage. Coverage ratchets are untouched: the full
short-mode suite (including `TestSC003_MappingAccuracy` at its 85.0%
floor) passes with no expected value changed.

### CRAP and GazeCRAP

| Metric | Baseline (T001) | Final | Protected gate |
|--------|----------------:|------:|---------------:|
| CRAPload | 26 | **23** | ≤ 38 |
| GazeCRAPload | 6 | **6** | ≤ 6 |
| Functions analyzed | — | 724 | — |
| Average complexity | — | 4.9 | ≤ 8 (rubric) |
| Average line coverage | — | 87.3% | — |
| Average contract coverage | — | 40% (3,088 tests) | ≥ 8 |

The six GazeCRAPload members are the same pre-existing functions recorded
at baseline: `adapter.(*Session).Initialize` (48.3), `aireport.WriteStepSummary`
(42), `aireport.EvaluateThresholds` (38.5), `protocol.(*Client).Call` (26.1),
`docscan.Scan` (19.3), and `adapter.BuildQualityFromMappings` (17). No
function added by this change entered GazeCRAPload.

### Complexity of new code

The six new `internal/aireport` files (`sections.go`, `grade.go`, `assess.go`,
`render_markdown.go`, `answer.go`, `opencode_events.go`) contribute 109
functions. The highest cyclomatic complexity among them is 10
(`isListItem`); none exceeds the CC ≤ 10 rule. The highest among the new
`cmd/gaze` report helpers is 6 (`runExternalReportQuality`).

### Rubric assessment of the final tree

`assessment` in the gate payload: rubric version `1`; CRAPload 3.2 (`A-`, 🟢),
GazeCRAPload 0.8 (`A`, 🟢), average line coverage 87.3 (`A-`, 🟢), contract
coverage 40 (`B-`, 🟡), complexity 4.9 (`B+`, 🟢); six deterministic
recommendations.
