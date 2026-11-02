# gaze report

Run Gaze's CRAP, quality, classification, and docscan operations together to produce a text, markdown, JSON, or HTML report. Text output uses an external AI adapter. Gaze generates markdown, JSON, and HTML directly.

The formatted report is written to stdout and optionally appended to `$GITHUB_STEP_SUMMARY` for GitHub Actions Step Summary integration.

## Synopsis

```text
gaze report [packages] [flags]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `packages` | No | Go package patterns (e.g., `./...`, `./internal/...`). Defaults to `./...` when omitted. |

When no package arguments are provided, `./...` is used automatically (the entire module).

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | `string` | `text` | Output format: `text` (AI-formatted markdown), `markdown` (deterministic native markdown), `json` (raw analysis payload), or `html` (native combined report) |
| `--sections` | `string slice` | `crap,quality,classification,health` | Report sections to analyze and render: `crap`, `quality`, `classification`, `health`. Only the selected analyses run. |
| `--ai` | `string` | `""` | AI adapter: `claude`, `gemini`, `ollama`, or `opencode`. **Required in text mode.** |
| `--model` | `string` | `""` | Model name for the AI adapter. **Required for `ollama`**; optional for other adapters. |
| `--ai-timeout` | `duration` | `10m` | Maximum time to wait for the AI adapter to respond. Uses Go duration format (e.g., `5m`, `30s`, `2m30s`). |
| `--coverprofile` | `string` | `""` | Path to a pre-generated Go coverage profile. Skips the internal `go test -coverprofile` run. |
| `--analyzer` | `string` | `""` | External analyzer binary to use instead of Go-native analysis (e.g. `snake-eyes`). Discovered from `.gaze.yaml` or PATH when omitted. |
| `--language` | `string` | `""` | Target language for external analyzer discovery (e.g. `python`). |
| `--max-crapload` | `int` | not set | CI gate: fail if CRAPload exceeds N. Absent by default (no enforcement). |
| `--max-gaze-crapload` | `int` | not set | CI gate: fail if GazeCRAPload exceeds N. Absent by default (no enforcement). |
| `--min-contract-coverage` | `int` | not set | CI gate: fail if average contract coverage is below N%. Absent by default (no enforcement). |

### Threshold Semantics

The threshold flags (`--max-crapload`, `--max-gaze-crapload`, `--min-contract-coverage`) use pointer semantics internally: when a flag is not provided, no threshold is enforced. When explicitly set — even to `0` — the threshold is active. This means `--max-crapload=0` will fail if any function is in the CRAPload (i.e., zero tolerance).

## Configuration Interaction

The `gaze report` command does not directly read `.gaze.yaml`. The underlying analysis pipeline uses default classification thresholds. To customize thresholds, use `gaze crap` and `gaze quality` individually.

The `--coverprofile` flag is the key CI optimization — pass a coverage profile generated during your test step to avoid running tests twice.

## AI Adapter Details

| Adapter | Binary | System Prompt Delivery | Payload Delivery |
|---------|--------|----------------------|-----------------|
| `claude` | `claude` | Temp file via `-p` flag | stdin |
| `gemini` | `gemini` | `GEMINI.md` in temp dir | stdin |
| `ollama` | HTTP API | `system` field in JSON body | `prompt` field in JSON body |
| `opencode` | `opencode` | `.opencode/agents/gaze-headless.md` in temp dir via `--dir` | stdin |

For text output, Gaze validates that the adapter binary exists on `PATH` (or that the Ollama HTTP API is reachable) before analysis starts. An invalid binary produces a hard exit before any analysis runs.

## External Analyzer Mode

With `--analyzer` (or `--language` alone, when discovery resolves an analyzer), the report is built from the external analyzer's JSON-RPC data instead of Go-native analysis. Each selected section is gated independently, so `--sections=quality` alone still runs the quality analysis:

| Section | External analyzer source |
|---------|-------------------------|
| `crap` | `complexity` + `coverage` + `analyze` methods |
| `quality` | `test_mapping` — produces the same `quality_reports` / `quality_summary` shape `gaze quality --analyzer` emits |
| `classification` | the `classification_counts` derived from `test_mapping` results |
| `health` | CRAP, quality, and documentation (via the optional `doc_coverage` method) |

When the analyzer does not advertise `test_mapping`, or the call fails, Gaze does not fabricate the section. The rendered report shows a warning callout in its place:

```text
> ⚠️ Quality analysis unavailable: <reason>
```

The same applies to `> ⚠️ Classification analysis unavailable:` and `> ⚠️ Documentation analysis unavailable:`. When a quality threshold is configured, the report still fails the run; the report itself is always published first so the failure is visible.

## Native HTML Output

Select `--format=html` to generate a complete combined report without an AI adapter. In HTML mode, `--ai` is optional: Gaze does not validate or invoke an adapter, and it does not load an AI prompt.

The report includes typed summary metrics and separate CRAP, quality, classification, and documentation sections. Successful steps show their available pipeline data as formatted JSON. If a step fails, its section shows the recorded error while output from successful steps remains available. Metrics that were not computed are shown as unavailable rather than as measured zero values.

HTML output is a self-contained file with inline styling. It does not load scripts, stylesheets, fonts, images, or other network resources. Its content remains readable with JavaScript disabled.

## Deterministic Markdown Output

`--format=markdown` renders the report inside Gaze and never calls an AI adapter. Every byte outside the optional AI slots is derived from the analysis data, so identical data always produces an identical report. Only the sections selected by `--sections` are rendered, and no AI-written text appears.

```bash
gaze report ./... --format=markdown --sections=crap,quality
```

### Format behavior

| Format | AI adapter | Rendered by | AI text in output |
|--------|------------|-------------|-------------------|
| `text` | Required | Gaze renders the deterministic report; the adapter supplies only the summary and recommendations | Yes, inside the AI slots |
| `markdown` | Not used | Gaze | No |
| `json` | Not used | Gaze | No |
| `html` | Not used | Gaze | No |

All three native formats carry the same selected sections, the same grades, and the same deterministic recommendations. Only `text` asks the AI adapter for judgment, and only `text` lets the adapter's answer affect the exit code.

### Pre-analysis validation

`gaze report` rejects invalid input before any analysis runs:

- Unknown format: `invalid format "yaml": must be 'text', 'json', 'html', or 'markdown'`
- Unknown or empty section: `invalid --sections value "bogus": valid sections are crap, quality, classification, health`
- Threshold without its analysis: `--min-contract-coverage requires the quality analysis, which --sections=crap does not run` (the same form applies to `--max-crapload` and `--max-gaze-crapload` with the CRAP analysis)
- Text mode without an adapter: `--ai is required in text mode: ...`
- A `--coverprofile` path that does not exist or is a directory

## AI answer contract

In `text` mode the report-mode prompt (built into the binary) asks the adapter for exactly one JSON object:

```json
{
  "summary": "Plain-prose narrative ...",
  "recommendations": ["First action ...", "Second action ..."]
}
```

Gaze validates every adapter's output with the same rules, in this order, and rejects the answer with `AI answer rejected (rule <rule>): <detail>`:

| Rule | Rejects |
|------|---------|
| `empty_output` | No output at all |
| `surrounding_text` | Anything but one JSON object, optionally inside a single fenced block tagged `json` or untagged |
| `not_json` | Unparseable JSON |
| `multiple_values` | More than one JSON value |
| `not_object` | A top-level array, string, or number |
| `duplicate_field` | The same field twice |
| `unknown_field` | A field other than `summary` and `recommendations` |
| `missing_field` | Either field absent |
| `wrong_type` | `summary` not a string, or `recommendations` not an array of strings |
| `summary_length` | A summary shorter than 1 or longer than 1,200 characters |
| `recommendation_count` | Fewer than 1 or more than 5 recommendations |
| `recommendation_length` | An empty recommendation, one longer than 400 characters, or one containing a line break |
| `markup` | Markdown or HTML: headings, lists, tables, fences, blockquotes, thematic breaks, links, `www.` or `://` URLs, HTML tags or comments, or a carriage return |
| `non_english` | Non-Latin letters above 5% of the prose, no letters at all, or code spans carrying more non-Latin letters than the prose |

On rejection the run exits non-zero even when every threshold passes, and the deterministic report (plus a `> ⚠️ AI narrative rejected:` callout) is still printed and appended to the step summary. An adapter failure renders `> ⚠️ AI narrative unavailable:` instead. Raw adapter output is never echoed.

### Ignore notice

`gaze report` always uses its built-in report-mode prompt. When `<moduleDir>/.opencode/agents/gaze-reporter.md` exists (the interactive `/gaze` agent), Gaze prints a one-line notice to stderr that the file is ignored. The file never affects `gaze report`, and `gaze report` never modifies it.

## JSON output

The JSON payload carries the analysis sections plus two report-level objects:

- `metadata`: `module_path`, optional `branch`, `gaze_version`, `date` (`YYYY-MM-DD`, UTC), and `sections` in canonical order.
- `assessment`: `rubric_version` (`"1"`), a five-row `scorecard` (`dimension`, `label`, `value`, `grade`, `severity`, `marker`, `detail`), and up to eight `recommendations` (`severity`, `marker`, `source`, `text`, plus optional `function`, `file`, `line`, `fix_strategy`).

The Overall Health markdown section renders `assessment`, so the same data drives both formats. Unselected or failed steps are `null`, and the matching `errors` entry explains why.

## Grading rubric

The report's Overall Health section and the JSON payload's `assessment` field use rubric version `1`. A metric grades against the value shown in the report: floating-point metrics are compared at one decimal place, and CRAPload and GazeCRAPload are compared as integer tenths of a percent before being displayed with one decimal. Bounds are inclusive: the first bound a value satisfies, scanning from `A` downward, wins, and a value that satisfies none grades `F`.

| Grade | CRAPload ≤ | GazeCRAPload ≤ | Avg Line Coverage ≥ | Contract Coverage ≥ | Avg Complexity ≤ |
|-------|-----------:|---------------:|--------------------:|--------------------:|-----------------:|
| A | 2.0 | 2.0 | 90.0 | 80 | 3.0 |
| A- | 4.0 | 4.0 | 85.0 | 70 | 4.0 |
| B+ | 6.0 | 6.0 | 80.0 | 60 | 5.0 |
| B | 8.0 | 8.0 | 75.0 | 50 | 6.0 |
| B- | 10.0 | 10.0 | 70.0 | 40 | 7.0 |
| C+ | 12.5 | 12.5 | 65.0 | 30 | 8.0 |
| C | 15.0 | 15.0 | 60.0 | 20 | 9.0 |
| C- | 20.0 | 20.0 | 55.0 | 15 | 10.0 |
| D | 30.0 | 30.0 | 45.0 | 10 | 12.0 |
| F | > 30.0 | > 30.0 | < 45.0 | < 10 | > 12.0 |

A metric is unavailable (`⚪ N/A`, no value, no grade) when its analysis did not run or produced nothing usable:

- CRAPload, Avg Line Coverage, and Avg Complexity: the `crap` section is not selected, the CRAP analysis failed, or no functions were analyzed.
- GazeCRAPload: the same conditions, or the analysis produced no GazeCRAP data.
- Contract Coverage: the `quality` section is not selected, the quality analysis failed, no contract coverage was computed, or no tests were analyzed.

Each grade maps to a severity and a marker:

| Grade | Severity | Marker |
|-------|----------|--------|
| A, A-, B+ | `low` | 🟢 |
| B, B-, C+, C | `moderate` | 🟡 |
| C-, D, F | `critical` | 🔴 |
| unavailable | `unavailable` | ⚪ |

The scorecard's Details column uses these templates:

- CRAPload and GazeCRAPload: `{n}/{total} functions ({ratio}%) at or above threshold {threshold}`
- Avg Line Coverage: `{value}% average line coverage`
- Contract Coverage: `{value}% average contract coverage across {tests} tests`
- Avg Complexity: `{value} average cyclomatic complexity`
- Unavailable: the reason — `not selected`, `analysis failed`, `no functions analyzed`, `GazeCRAP unavailable`, or `no tests analyzed`

Deterministic recommendations describe the first five recommended actions, in the order the analysis produced them. A function with 0% line coverage or a Q4 Dangerous quadrant is `critical`; every other one is `moderate`.

- `add_tests`: Add tests for `{function}` — {coverage}% line coverage, complexity {complexity}, CRAP {crap} ({file}:{line}).
- `add_assertions`: Strengthen assertions for `{function}` — tests execute it without verifying its contracts ({file}:{line}).
- `decompose`: Decompose `{function}` — complexity {complexity} keeps CRAP at {crap} even at full coverage ({file}:{line}).
- `decompose_and_test`: Decompose `{function}`, then add tests — complexity {complexity}, {coverage}% line coverage, CRAP {crap} ({file}:{line}).

Three condition recommendations follow, in this order: SSA degradation (`moderate`), skipped tests (`low`), and ambiguous side effects that outnumber the decided ones (`low`). The assessment carries at most eight recommendations.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_STEP_SUMMARY` | When set, the formatted report is appended to this file path for GitHub Actions Step Summary display. Symlink protection (`O_NOFOLLOW`) is applied. |

## Examples

### AI-formatted report with Claude

```bash
gaze report ./... --ai=claude
```

### JSON-only output (no AI required)

```bash
gaze report ./... --format=json > report.json
```

In JSON mode, the `--ai` flag is not required. The raw analysis payload is written directly to stdout.

### Native HTML report written to a file

```bash
gaze report ./... --format=html > gaze-report.html
```

No `--ai` flag is needed. Open `gaze-report.html` directly in a browser; the file does not require network access.

### CI integration with thresholds

```bash
# Generate coverage during test step
go test -race -count=1 -coverprofile=coverage.out ./...

# Run report with pre-generated coverage and quality gates
gaze report ./... \
  --ai=opencode \
  --coverprofile=coverage.out \
  --max-crapload=35 \
  --max-gaze-crapload=5 \
  --min-contract-coverage=8
```

### Using Ollama (local model)

```bash
gaze report ./... --ai=ollama --model=llama3.2
```

The `--model` flag is required for Ollama and specifies which model to use for formatting.

## See Also

- [CI Integration](../../guides/ci-integration.md) — full GitHub Actions workflow setup
- [AI Reports](../../guides/ai-reports.md) — adapter setup and configuration
- [Scoring](../../concepts/scoring.md) — understanding the metrics in the report
- [`gaze crap`](crap.md) — standalone CRAP analysis
- [`gaze quality`](quality.md) — standalone quality analysis
- [`gaze docscan`](docscan.md) — standalone document scanning
