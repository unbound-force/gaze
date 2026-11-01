# Research: Deterministic AI Report Rendering

**Feature**: `040-deterministic-ai-report-rendering`
**Date**: 2026-11-01
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

Each entry follows the Decision / Rationale / Alternatives format.
Entries R1–R3 resolve the items the spec explicitly deferred to
planning (FR-014, FR-016, the grade-rubric and validation-bound
assumptions). The rest settle design questions found while reading
the code.

---

## R1: `opencode run --format json` event stream (FR-016) — VERIFIED

**How it was verified**: The installed opencode CLI is
`/opt/homebrew/Cellar/opencode/1.19.0/bin/opencode`
(`opencode --version` → `1.19.0`). That is the same version the
reusable workflow pins (`opencode-version` default `"1.19.0"` in
`.github/workflows/reusable_gaze_ai_report.yml`, installed through
`npm install -g opencode-ai@1.19.0`). The binary is Bun-compiled, so
its JavaScript source is embedded. `opencode run --help` confirms the
flag (`--format  format: default (formatted) or json (raw JSON
events)`). The `run` handler and the part schemas were read straight
from the embedded source (`strings` on the binary).

**Verified facts (opencode 1.19.0)**:

1. **Line framing.** In `--format json` mode every event is written
   as `process.stdout.write(JSON.stringify({type, timestamp: Date.now(),
   sessionID, ...extra}) + EOL)`. That makes the stream JSON Lines:
   one object per line.
2. **Event types emitted** (from the `run` handler's `message.part.updated`
   loop, filtered to the run's own session ID):

   | `type` | Extra field | Emitted when |
   |--------|-------------|--------------|
   | `step_start` | `part` (StepStartPart) | a model step begins |
   | `text` | `part` (TextPart) | a text part completes (`part.time.end` is set) |
   | `tool_use` | `part` (ToolPart) | a tool part reaches `state.status` `completed` or `error` |
   | `step_finish` | `part` (StepFinishPart) | a model step ends |
   | `reasoning` | `part` (ReasoningPart) | only with `--thinking`, which Gaze never passes |
   | `error` | `error` (`{name, data: {message}}`) | a `session.error` for the run's session |

3. **Part schema.** Every part has the base fields `id`,
   `sessionID`, and `messageID`. A `TextPart` adds `type: "text"`,
   `text`, an optional `synthetic`, an optional `ignored`, an optional
   `time: {start, end?}`, and optional `metadata`. A
   `StepFinishPart` adds `type: "step-finish"`, `reason` (string),
   `cost`, and `tokens`. A `ToolPart` carries `tool` and
   `state.status`.
4. **Narration is separate from the answer.** Narration the model
   writes between tool calls arrives as separate `text` events on
   earlier steps. In `--format default` the adapter's current mode
   prints every completed text part, which is incident root cause #1.
5. **stdin.** When stdin is not a TTY, `run` reads all of stdin and
   combines it with the positional message. The current
   `"" + stdin` invocation keeps working.
6. **Exit code.** `run` sets `process.exitCode = 1` when a session
   error was observed. `runSubprocess` already turns that into a Go
   error.

**Decision — extraction algorithm** (normative version in
[contracts/opencode-adapter.md](contracts/opencode-adapter.md)):

- Parse stdout line by line. Skip blank lines. A line that is not a
  JSON object is ignored but counted. If the whole stream produced no
  valid events, extraction fails ("no JSON events" — this catches a
  format drift where the CLI prints plain text again).
- An `error` event fails the call. The error name and message are
  passed through `redactEnvValues` and truncated to
  `maxAdapterStderrBytes`.
- A `tool_use` event whose `part.state.status == "completed"` fails
  the call **closed**: "opencode executed tool <name> although
  report-mode denies all tools". A `status == "error"` tool event
  (a refused call) is tolerated (spec edge case "Model attempts tool
  use").
- **Final answer** = the `text` parts whose `part.messageID` equals
  the `messageID` of the **last `step_finish` event**. If there is no
  `step_finish`, use the `messageID` of the last `text` event. Parts
  are deduplicated by `part.id` (last write wins) and joined with `\n`
  in first-seen order. Parts marked `synthetic` or `ignored` are
  dropped.
- If no text part remains, extraction fails ("opencode produced no
  final assistant text"). Then `formatStderrSuffix` runs, as it does
  today.

**Rationale**: `messageID` is the only field that ties text to a
model step. Selecting on the last step discards narration from
earlier steps, and the event schema guarantees the field exists.
Failing closed on a completed tool call means a broken permission
setup can never pass silently.

**Fallbacks if the schema drifts in a future opencode release**: The
extractor fails with a clear message that names the pinned version.
It never falls back to treating raw stdout as the answer, because
that would reintroduce the incident. The fake-binary fixtures (R10)
are modeled on the verified 1.19.0 schema, so a pin bump is a
deliberate, reviewed change.

**Alternatives considered**:
- *Keep `--format default` and take the last paragraph*: rejected.
  Paragraph heuristics cannot tell a final answer from narration.
- *`opencode serve` + SDK*: rejected. It needs a server lifecycle
  and port management, which is much more machinery for the same
  data.
- *Pass `--pure` (no external plugins)*: rejected. Provider-auth
  plugins can be load-bearing in consumer configurations, and the
  validator is the backstop anyway.

---

## R2: Agent frontmatter that denies all tools (FR-014) — VERIFIED

**Verified facts (opencode 1.19.0, embedded config schema and agent
loader)**:

1. `PermissionConfig` is either an action string (`"ask" | "allow" |
   "deny"`), normalized to `{"*": action}`, or an object with the
   known keys `read, edit, glob, grep, list, bash, task,
   external_directory, todowrite, question, webfetch, websearch, lsp,
   doom_loop, skill` plus arbitrary extra keys (MCP tools). `"*"` is a
   wildcard, and the last matching rule wins.
2. A markdown or config agent's permission is built as
   `merge(defaults, top-level config permission, agent permission)`.
   The agent's own rules come **last**, so they override a
   permissive top-level `permission` in `OPENCODE_CONFIG_CONTENT`.
3. The native no-tool agents (`compaction`, `title`, `summary`) use
   exactly `{"*": "deny"}`. opencode computes a disabled-tool set
   from the ruleset (`Permission.disabled`) and leaves those tools
   out of skill and MCP instructions.
4. `run` auto-rejects any `permission.asked` request unless `--auto`
   is passed, and Gaze never passes it.
5. **Hazard:** `run --agent <name>` prints a warning and **falls back
   to the default `build` agent, with full tools,** when the named
   agent is not found or has `mode: subagent`. Custom agents default
   to `mode: "all"`. When agent definitions share a name, their fields
   merge, so a global `gaze-reporter` subagent (the scaffolded `/gaze`
   agent installed globally, for example) could otherwise make
   `mode` resolve to `subagent`.

**Decision**: The adapter writes
`<tmp>/.opencode/agents/gaze-headless.md` with this frontmatter:

```yaml
---
description: Gaze headless judgment call (no tools)
mode: primary
permission:
  "*": deny
---
```

It invokes `opencode run --dir <tmp> --agent gaze-headless --format
json [-m <model>] ""`. The name `gaze-headless` is unique, so it does
not collide with the scaffolded `/gaze` agent `gaze-reporter`.
Explicit `mode: primary` guards against the subagent fallback. The
fail-closed `tool_use` check in R1 covers any residual fallback.
The same agent serves every `OpenCodeAdapter.Format` call, so the
`--ai-mapper=opencode` path also runs tool-free and extracts only the
final answer.

**Alternatives considered**: `permission: deny` (the string
shorthand) is equivalent, but the explicit map matches the native
agents and reads more clearly. A `steps: 1` limit was rejected
because its behavior when the limit is reached was not verified. The
legacy `tools:` map was rejected because it is deprecated in favor of
`permission`.

---

## R3: Grade rubric boundaries (FR-007, FR-011, SC-004)

**Decision**: Five dimensions, graded on the **displayed one-decimal
value**. The display string is produced with
`strconv.FormatFloat(v, 'f', 1, 64)` and the grade is computed from
that same string re-parsed, so the grade can never disagree with the
number a reader sees. Ratio dimensions use integer tenths:
`tenths = (2*n*1000 + total) / (2*total)`. Boundaries are inclusive.
The full table is in [contracts/grade-rubric.md](contracts/grade-rubric.md),
`rubric_version: "1"`:

| Grade | CRAPload % / GazeCRAPload % (≤) | Avg line coverage % (≥) | Contract coverage % (≥) | Avg complexity (≤) |
|-------|------|------|------|------|
| A  | 2.0 | 90.0 | 80 | 3.0 |
| A- | 4.0 | 85.0 | 70 | 4.0 |
| B+ | 6.0 | 80.0 | 60 | 5.0 |
| B  | 8.0 | 75.0 | 50 | 6.0 |
| B- | 10.0 | 70.0 | 40 | 7.0 |
| C+ | 12.5 | 65.0 | 30 | 8.0 |
| C  | 15.0 | 60.0 | 20 | 9.0 |
| C- | 20.0 | 55.0 | 15 | 10.0 |
| D  | 30.0 | 45.0 | 10 | 12.0 |
| F  | otherwise | otherwise | otherwise | otherwise |

The ratio denominator is `crap.summary.total_functions` for both
CRAPload and GazeCRAPload. A dimension is N/A when its step failed or
was not selected, when `total_functions == 0` (CRAP dimensions), when
`gaze_crapload` is absent (GazeCRAPload), or when the quality summary
reports zero tests (contract coverage).

**Rationale**: Ratios keep grades comparable across project sizes
(Principle III). The table reproduces the grades in the current
canonical example (`references/example-report.md`): 24/216 = 11.1%
→ C+, and average complexity 6.2 → B-. Integer and displayed-value
arithmetic removes float-boundary flakiness, so every boundary can be
pinned exactly (SC-004).

**Alternatives considered**: Absolute counts were rejected because
they are not comparable across project sizes. Letting the AI grade
was rejected by an owner decision ("Gaze computes grades").
Exact-float comparison was rejected because it allows a grade that
disagrees with the shown value.

---

## R4: AI answer validation bounds (FR-018) — final values

**Decision** (normative in [contracts/ai-answer.md](contracts/ai-answer.md)):
summary 1–1,200 Unicode code points after `TrimSpace`; 1–5
recommendations; each recommendation 1–400 code points after
`TrimSpace` and a single line. Non-English limit is evaluated **per
field**: count `unicode.IsLetter` runes outside inline code spans
(`` `...` ``). Count as non-English every letter not in
`unicode.Latin`. Reject when `nonLatin*100 > 5*letters`, and reject
when a field has `letters == 0`. Digits, punctuation, symbols, and
emoji are not letters and are ignored.

**Rationale**: These are the spec defaults. Evaluating per field
stops a single Chinese recommendation from hiding behind a long
English summary, which a combined ratio would allow. Integer math
avoids float edges.

**Alternatives**: A combined ratio was rejected for that reason.
Language-detection libraries were rejected because they add a new
dependency and are nondeterministic across versions.

---

## R5: Markup neutralization (FR-017, spec edge case "Markup injection")

**Decision**: **Reject** structural markup instead of escaping it,
so FR-021's "no silent repair" holds. Reject rules: raw HTML or
comments (`<` followed by a letter, `/`, or `!`); link or image
syntax (`](`); any line whose first non-space characters are `#`,
`>`, `|`, a list marker (`-`/`*`/`+` followed by a space, or digits
followed by `.` or `)` and a space), a code fence (```` ``` ```` or
`~~~`), or a line made only of `-`, `=`, `*`, or `_` (a thematic
break or setext underline). Recommendations must not contain a
newline. Inline emphasis and inline code are allowed. AI text is
rendered only inside delimited slots (R6), as paragraphs and as
numbered list items Gaze writes itself.

**Rationale**: With these rules, accepted text can only produce
inline content inside Gaze-owned blocks. It cannot add headings,
tables, or HTML, or close a slot early.

**Alternatives**: Backslash-escaping everything was rejected because
it silently rewrites the answer and is fragile across markdown
dialects.

---

## R6: AI slot delimitation and byte-identity testing (FR-003, SC-002)

**Decision**: Each AI slot is a block wrapped in HTML comment markers:
`<!-- gaze:ai-summary -->` … `<!-- /gaze:ai-summary -->` and
`<!-- gaze:ai-recommendations -->` … `<!-- /gaze:ai-recommendations -->`,
followed by one blank line. With no AI (`--format=markdown`) the
whole block, markers included, is omitted. A test helper strips
`(?s)<!-- gaze:ai-(summary|recommendations) -->\n.*?<!-- /gaze:ai-\1 -->\n\n`
and asserts zero differing bytes. When validation fails, the summary
slot carries the rejection callout instead of AI text (FR-020).

**Rationale**: Comment markers are invisible in GitHub-rendered step
summaries and give tests an exact slot boundary. The validator
rejects `<!` in AI text (R5), so the AI cannot forge a marker.

---

## R7: Report metadata determinism (FR-003)

**Finding**: The current format prints Branch, Go version, and Date.
Read from the environment at render time, those values break "byte
identical for identical payloads".

**Decision**: The CLI captures metadata once into the payload as
`metadata` (module path, gaze version, branch, UTC date, sections),
and the renderer reads only the payload. Branch comes from
`GITHUB_HEAD_REF`, then `GITHUB_REF_NAME`, and is otherwise omitted.
Gaze does not shell out to git. The date comes from an injectable
clock. The Go version line is dropped: the build toolchain version is
misleading as a "project Go" value. Module path comes from `go.mod`
(the existing `crap.readModulePath`, exported as
`crap.ReadModulePath`). External-analyzer projects without `go.mod`
fall back to `filepath.Base(moduleDir)`.

**Alternatives**: Rendering `time.Now()` was rejected because it
breaks golden files and FR-003.

---

## R8: Sections → analysis steps and the zero-result gate (FR-026–FR-028)

**Decision**: Sections are `crap`, `quality`, `classification`, and
`health`; the default is all four. Required steps: `crap→{CRAP}`,
`quality→{Quality}`, `classification→{Classify}`,
`health→{CRAP, Quality, Docscan}`. Thresholds require steps:
`--max-crapload` and `--max-gaze-crapload` require CRAP;
`--min-contract-coverage` requires Quality. A threshold whose step is
not in the selected set is rejected before analysis.

**Finding**: `checkZeroResultGate` (runner.go) assumes the CRAP step
ran. With `--sections=quality --min-contract-coverage=…`,
`Errors.CRAP == nil` and `TotalFunctions == 0` would trip the gate
falsely. **Decision**: Apply the gate only when the CRAP step was
selected. Its behavior is otherwise unchanged (FR-029, SC-011).

**Decision**: `--sections` applies to every format. JSON output
records `metadata.sections`, and unselected steps stay `null` with a
`null` error.

---

## R9: External-analyzer parity for the `/gaze` rewrite (US4 AS-5, FR-033)

**Finding**: Today `/gaze quality` on an external-analyzer project
runs `gaze quality --analyzer …`, which builds quality reports with
`adapter.BuildQualityFromMappings`. The report pipeline's external
path (`runExternalReportCRAP`) **skips** quality and classification
("skipped: external analyzer mode"). A rewrite that only calls
`gaze report --format=markdown` would therefore regress those modes.

**Decision**: Extend the external report path. When the Quality step
is selected and the analyzer advertises `test_mapping`, call the
existing `providers.SideEffects.AllResults()`,
`adapter.FetchTestMappings`, and `adapter.BuildQualityFromMappings`,
and serialize them in the same JSON shape `gaze quality` emits. Fill
`Summary.AvgContractCoverage` and derive classification counts from
`summary.ClassificationCounts`. When `test_mapping` is unavailable or
fails, record the step error, as `handleQualityNoTestMapping` /
`handleQualityTestMappingError` already describe. That renders as a
warning callout.

**Rationale**: This reuses existing functions; nothing new is added
to the protocol. **Risk**: it adds functions to `cmd/gaze`, which
counts against the dogfood GazeCRAPload ceiling (R12).

---

## R10: Fake opencode test binary (FR-037)

**Decision**: `internal/aireport/testdata/fake_opencode` gains an
env-driven replay mode. When `FAKE_OPENCODE_STREAM=<path>` is set, it
verifies `--format json` and the `gaze-headless` agent file (including
the `"*": deny` frontmatter), then copies the fixture JSONL to stdout.
The existing flags (`--exit-error`, `--empty-output`) stay. Fixtures
live in `internal/aireport/testdata/opencode_streams/`:
`incident.jsonl` (multi-step narration, mixed English/Chinese, no
JSON answer), `narration_then_answer.jsonl`,
`answer_only.jsonl`, `tool_refused.jsonl`, `tool_completed.jsonl`,
`error_event.jsonl`, and `no_events.txt`. Each fixture uses the
verified R1 schema.

---

## R11: Incident narration replay without the original text

**Finding**: The CI log for run 41309423609 is not in the
repository. **Decision**: `incident.jsonl` is a synthetic
reconstruction. It has three steps; each step carries narration
`text` parts mixing English and Chinese, with `tool_use` events
between steps. The last step is Chinese-dominant narration with no
JSON. The incident's essential properties are reproduced: multi-part,
mixed-language, and no structured answer. FR-036 and SC-001 assert a
non-zero exit with rule `not_json`. A second fixture wraps a valid
JSON shape around Chinese text to assert rule `non_english`.

---

## R12: CI gate headroom (SC-011, protected gates)

**Finding**: A prior learning (init-ci-scaffold) recorded the dogfood
GazeCRAPload at its ceiling (`max-gaze-crapload: 6`, zero headroom).
The dogfood thresholds in `.github/workflows/test.yml` are 38/6/8.
These are protected gate values, and this plan does not change them.

**Decision**: Every new function keeps cyclomatic complexity ≤ 10,
and its contract is asserted directly: tests check returned strings,
errors, and structs, not only line coverage. Large renderers are
split into one function per section and per table, following the
`crap.WriteText` decomposition precedent. A baseline `gaze crap
--format=json ./...` is captured before Phase 2 and compared at each
phase checkpoint. If a gate would fail, implementation stops and
reports it; the gate is not adjusted.

---

## R13: Dogfood CI binary version (SC-008) — FINDING

**Finding**: `reusable_gaze_ai_report.yml` installs gaze with
`go install …@${{ inputs.gaze-version }}` (default `v2.2.0`), and the
dogfood job in `test.yml` does not pass `gaze-version`. The PR's code
therefore **does not run** in the dogfood job until a release
containing it exists and the workflow's `gaze-version` is bumped.
Until then, v2.2.0's `LoadPrompt` keeps reading the repository's
`.opencode/agents/gaze-reporter.md`, which this change rewrites. The
dogfood behaves no better and no worse than today.

**Decision**: SC-008 is a **post-release verification** item. The
spec assumption keeps workflow invocation unchanged, so this spec
does not modify CI workflows. A follow-up change, CI config with its
own OpenSpec, bumps the reusable workflow's `gaze-version` default
(and the dogfood input) to the first release containing this
feature. That follow-up is recorded in tasks.md as a tracked
follow-up issue, not an implementation task, and the PR description
must state it.

---

## R14: Other adapters (FR-015)

**Decision**: The `AIAdapter.Format` interface is unchanged: it
returns the model's final text. Claude (`-p`), Gemini, and Ollama
(`/api/generate`, non-streaming) already return only the final
answer text. The opencode adapter now does too (R1). The runner
passes every adapter's output through the same `ParseAIAnswer`
validator. The interface is also used by `--ai-mapper`
(`buildAIMapperFunc`), and keeping it unchanged leaves that path
unaffected apart from the opencode final-text improvement.

**Alternatives**: Ollama `format: "json"` was rejected because it
would break the `--ai-mapper` plain-ID response. Claude
`--output-format json` was rejected as unnecessary; the validator
already covers it. Residual risk: CLIs other than opencode may run
their own tools in headless mode. The validator is the backstop for
every adapter.

---

## R15: Report-mode prompt location (FR-013, FR-024)

**Decision**: Embed the new prompt with
`//go:embed assets/report-mode-prompt.md` in
`internal/aireport/prompt.go`. The prompt has no frontmatter; the
opencode adapter adds its own. Delete
`internal/aireport/assets/agents/gaze-reporter.md`,
`LoadPrompt`, `stripFrontmatter`, and `TestEmbeddedPromptMatchesScaffold`,
and remove the second File Sync Invariants row from AGENTS.md. The
`.mega-linter.yml` exclusion `internal/aireport/assets/` stays valid
because the directory still exists, so no CI config change is
needed. The prompt is not under `internal/scaffold/assets/`, so it is
never scaffolded (FR-013).

---

## R16: `/gaze` agent rewrite, references, and ownership (FR-030–FR-034)

**Decisions**:

- **Commands.** Full mode runs `gaze report --format=markdown
  [--analyzer/--language] <pattern>`. Crap mode adds
  `--sections=crap`, and quality mode adds `--sections=quality`. The
  agent runs `gaze quality --format=json <pattern>` only to evaluate
  unmapped assertions; the JSON-format call FR-030 permits for
  judgment data.
- **Tool access.** The frontmatter `tools:` block (read/bash true;
  write/edit/webfetch false) and `mode: subagent` stay **unchanged**.
  Agent tool-access settings are a protected gate category.
- **`references/example-report.md`** (tool-owned, in the manifest)
  is rewritten to contain the real Gaze-rendered golden report plus
  one example of the agent's added judgment sections (DE-002). A test
  asserts that its fenced report block equals the no-AI golden file
  (a Layer 2 seam).
- **`references/doc-scoring-model.md`** stays in the 9-file manifest
  (FR-024 / SC-010 pin the count). To keep it from becoming an
  orphan, the rewritten agent consults it only for an optional,
  clearly labeled judgment note: which ambiguous effects the docs
  suggest are contractual. Those notes are suggestions; the agent
  never changes rendered counts. The weights stay in the reference
  file, not in the prompt (FR-031).
- **Ownership.** `agents/gaze-reporter.md` is added to `isToolOwned`
  (FR-034, the human-chosen option A). `commands/gaze.md` stays
  user-owned; only its description wording changes, with the
  embedded copy kept byte-identical.
- **DE-005 residual.** Binary resolution and analyzer detection stay
  in the prompt because FR-030 requires that order unchanged.
  Collapsing them into a single intent-level tool is noted as a
  follow-up candidate, outside this spec's scope.

**DE-004 baseline (measured 2026-11-01)**: `.opencode/agents/gaze-reporter.md`
is 528 lines, 2,965 words, 20,769 bytes, and 20,472 characters, which
is about 5,118 tokens (characters ÷ 4; no tokenizer is installed, so
this is a stated approximation). The target for the rewrite is ≤ 160
lines and ≤ about 1,600 tokens, a reduction of at least 70%. Actual
after-values are measured with the same `wc -l -w -c` / chars÷4
method and recorded in the PR description and the AGENTS.md Recent
Changes entry (FR-032).

---

## R17: Notice location for a local `gaze-reporter.md` (FR-023)

**Decision**: In `--format=text`, the CLI checks
`<moduleDir>/.opencode/agents/gaze-reporter.md`, where `moduleDir`
is the module root. That equals the working directory in CI and
whenever `--analyzer` is set, and it is the path the removed
`LoadPrompt` used to read. If the file exists, the CLI prints exactly
one notice line to stderr. No notice is printed for json, html, or
markdown formats.

---

## R18: Markdown width

**Decision**: Markdown output is exempt from the 80-column
text-report check, because table rows cannot wrap. Table cells
escape `|` as `\|` (even inside code spans, per GFM) and replace
newlines with spaces. Function names render inside code spans, so
names like `(*T).M` cannot trigger emphasis. A name containing a
backtick uses a double-backtick span. Wide or CJK characters need no
special handling because markdown tables are not column-aligned.
