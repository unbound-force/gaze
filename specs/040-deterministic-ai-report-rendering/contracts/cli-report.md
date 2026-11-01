# Contract: `gaze report` CLI surface

Covers FR-019, FR-022–FR-029, and FR-038. Flags that are not listed
here are unchanged.

## Flags

| Flag | Type | Default | Change |
|------|------|---------|--------|
| `--format` | string | `text` | Adds `markdown`. Valid values: `text`, `json`, `html`, `markdown`. |
| `--sections` | string slice (comma-separated) | `crap,quality,classification,health` | **New.** Selects which sections are analyzed and rendered. Applies to every format. |
| `--ai` | string | `""` | Required only for `text`. Ignored (never contacted) for `json`, `html`, and `markdown`. |

Help text: `--sections`: "report sections to analyze and render:
crap, quality, classification, health (default all)".

## Format behavior matrix

| Format | AI contacted | Stdout | Step summary | Thresholds | Notice for local `gaze-reporter.md` |
|--------|--------------|--------|--------------|------------|------|
| `text` | yes (`--ai` required) | Gaze-rendered markdown + AI slots (or rejection callout) | yes | yes | yes, when the file exists |
| `markdown` | no | Gaze-rendered markdown, no slots | yes | yes | no |
| `json` | no | payload JSON (+ `metadata`, `assessment`) | no (unchanged) | yes | no |
| `html` | no | HTML (unchanged) | no (unchanged) | yes | no |

## Pre-analysis validation (exit non-zero before any analysis)

| Condition | Error text (prefix-stable) |
|-----------|---------------------------|
| Unknown format | `invalid format "x": must be 'text', 'json', 'html', or 'markdown'` |
| Unknown section | `invalid --sections value "x": valid sections are crap, quality, classification, health` |
| Empty section list or element | `invalid --sections value "": valid sections are crap, quality, classification, health` |
| Incompatible threshold (FR-028) | `--<flag> requires the <step> analysis, which --sections=<list> does not run` |
| Text without `--ai` | unchanged |

## Diagnostic stream additions

- Ignore notice (FR-023). This is exactly one line, printed only in
  `text` format when `<moduleDir>/.opencode/agents/gaze-reporter.md`
  exists:
  `notice: ignoring .opencode/agents/gaze-reporter.md for gaze report (it is the interactive /gaze agent; gaze report uses built-in report-mode instructions)`
- On AI failure, the error is returned by the command (printed by
  Cobra). It is never the raw model output.

## Exit semantics (text format)

| AI outcome | Thresholds | Exit | Stdout |
|------------|-----------|------|--------|
| valid | pass/none | 0 | full report with AI slots |
| valid | fail | ≠0 | full report with AI slots |
| rejected or adapter error | pass/none | ≠0 (FR-019, edge "AI failure with thresholds passing") | deterministic report + rejection callout (FR-020) |
| rejected or adapter error | fail | ≠0; both errors joined with `errors.Join` | deterministic report + rejection callout |

Error text formats:
- `AI answer rejected (rule <rule-id>): <detail>`, as defined in
  [ai-answer.md](ai-answer.md).
- Adapter errors keep their existing text. The opencode extraction
  errors are listed in [opencode-adapter.md](opencode-adapter.md).

The zero-result gate is unchanged, except that it is skipped when
the CRAP step is not selected.

## Removed behavior

`gaze report` no longer reads `.opencode/agents/gaze-reporter.md`
or any other file as AI instructions (FR-022). The
`loading system prompt` error path is gone.
