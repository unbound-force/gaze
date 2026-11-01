# Contract: Report-mode prompt and rewritten `/gaze` agent

Covers FR-013, FR-030 through FR-034, and SC-006/SC-007.

## A. Report-mode prompt (`internal/aireport/assets/report-mode-prompt.md`)

This prompt is embedded in the binary and is never scaffolded. It has
no frontmatter. Target length is 40 lines or fewer. The required
content is listed below, and `TestReportModePrompt_*` asserts each
item.

1. Role: write only the judgment portion of a Gaze quality report.
   Gaze renders every table, grade, metric, and deterministic
   recommendation itself.
2. Input: the analysis payload arrives as JSON in the user message
   (stdin). `assessment` holds Gaze's final grades and
   recommendations. Never recompute grades, thresholds, counts, or
   rankings, and never invent numbers.
3. **Do not use tools, read files, or run commands.**
4. Answer in **English**.
5. Respond with **exactly one JSON object and nothing else**:
   `{"summary": "...", "recommendations": ["...", ...]}`.
6. Bounds and markup, stated with the same numbers as
   [ai-answer.md](ai-answer.md): summary 1–1200 characters of plain
   prose with no headings, lists, tables, links, or HTML;
   recommendations 1–5 single-line items of 400 characters or fewer,
   most important first, each starting with an action verb, naming a
   function or package in backticks, and citing at least one metric
   from the input.
7. Wrap code identifiers in backticks.
8. One complete example answer.

`TestReportModePrompt_*` checks:

- The prompt is non-empty and has no leading `---`.
- It contains the strings `"summary"` and `"recommendations"`, the
  literals `1200`, `400`, and `5`, and the phrases `Do not use tools`
  and `English`.
- The example answer inside it passes `ParseAIAnswer`, which guards
  against prompt and validator drift.
- No file under `internal/scaffold/assets/` contains the prompt
  text.

## B. Rewritten `/gaze` agent (`.opencode/agents/gaze-reporter.md` ≡ `internal/scaffold/assets/agents/gaze-reporter.md`)

**Frontmatter**: unchanged keys and values (`mode: subagent`; `tools:`
read/bash true, write/edit/webfetch false). Tool-access settings are
protected gates. The `description` text may be reworded.

**Body sections** (ordered):

1. **Role.** Show Gaze's rendered report verbatim and add only
   judgment.
2. **Binary resolution.** The three methods, unchanged in order.
3. **Mode parsing.** Unchanged: `crap`, `quality`, otherwise full; the
   default pattern is `./...`.
4. **Analyzer and language detection.** Unchanged logic.
5. **Run the report:**
   - full: `<gaze> report --format=markdown [--analyzer <cmd>] [--language <lang>] <pattern>`
   - crap: the same command with `--sections=crap`
   - quality: the same command with `--sections=quality`
6. **Show the output verbatim.** Do not edit, re-order, re-grade, or
   re-format any part of it.
7. **Judgment to add** under a `## Assessment` heading, after the
   verbatim report:
   - a narrative of at most one paragraph;
   - at most 5 prioritized recommendations that cite metrics from the
     report;
   - in quality mode, **unmapped-assertion evaluation** when
     `gaze quality --format=json <pattern>` shows `unmapped_assertions`:
     read the source and report semantically mapped assertions as
     suggestions at confidence 50 (the existing semantics), without
     changing the rendered coverage numbers;
   - optional, full mode only: **document-informed ambiguity notes**
     that consult `.opencode/references/doc-scoring-model.md` and list
     ambiguous effects the docs suggest are contractual. These are
     suggestions only and never change rendered counts.
8. **Example.** Point to `.opencode/references/example-report.md`,
   which contains the real rendered golden report plus one judgment
   section (DE-002).
9. **Errors and degradation.** If the binary is missing, show the
   install hints. If the command fails, show its error and suggest a
   fix. Do not analyze code manually.

**Forbidden content (FR-031)**, asserted by
`TestGazeReporterAgent_NoRenderingRules`, which scans the scaffold
copy:

- the grade scale table and grade-to-emoji mapping;
- the emoji vocabulary table;
- table templates (any line starting with `| Metric |`, `| Quadrant |`,
  or `| Dimension |`);
- the quadrant label list;
- the scoring weights (`±`);
- "Scoring Consistency Rules".

**DE-004 record (FR-032)**: the baseline is 528 lines, 2,965 words,
20,769 bytes, about 5,118 tokens (characters ÷ 4). The target is
160 lines or fewer and about 1,600 tokens or fewer. The measured
after-values go in the PR description and in AGENTS.md Recent
Changes.

**Ownership (FR-034)**: add `agents/gaze-reporter.md` to
`isToolOwned`. Update `scaffold_test.go`, which today expects the
file to be user-owned. Docs must state that local customizations are
overwritten on `gaze init` (`docs/guides/opencode-integration.md`,
`docs/reference/cli/init.md`, the release notes in the PR
description).

## C. `references/example-report.md`

This file is tool-owned and mirrored. It contains:

1. One fenced `markdown` block equal **byte-for-byte** to
   `internal/aireport/testdata/golden/report-full.md`. The
   `TestExampleReportMatchesGolden` test enforces this as a Layer 2
   seam.
2. One short example of the agent's `## Assessment` judgment
   section.

## D. `commands/gaze.md`

Only the wording that says the agent "interprets the JSON output"
changes, to "shows the Gaze-rendered report and adds judgment".
Usage, modes, and examples stay byte-for-byte the same. The embedded
copy stays identical.
