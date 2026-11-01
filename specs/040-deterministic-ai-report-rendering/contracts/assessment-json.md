# Contract: Machine-readable additions (`--format=json`)

Covers FR-010 and the write→read seam (FR-035 Layer 3, SC-005).
These fields are additive: every existing top-level key (`crap`,
`quality`, `classify`, `docscan`, `errors`) is unchanged.

## Shape

```json
{
  "metadata": {
    "module_path": "github.com/example/project",
    "branch": "main",
    "gaze_version": "v2.4.0",
    "date": "2026-11-01",
    "sections": ["crap", "quality", "classification", "health"]
  },
  "crap": { "...": "unchanged" },
  "quality": { "...": "unchanged" },
  "classify": { "...": "unchanged" },
  "docscan": { "...": "unchanged" },
  "errors": { "crap": null, "quality": null, "classify": null, "docscan": null },
  "assessment": {
    "rubric_version": "1",
    "scorecard": [
      {"dimension": "crapload", "label": "CRAPload", "value": 11.1, "grade": "C+",
       "severity": "moderate", "marker": "🟡",
       "detail": "24/216 functions (11.1%) at or above threshold 15"},
      {"dimension": "gaze_crapload", "label": "GazeCRAPload", "value": null, "grade": null,
       "severity": "unavailable", "marker": "⚪", "detail": "GazeCRAP unavailable"}
    ],
    "recommendations": [
      {"severity": "critical", "marker": "🔴", "source": "fix_strategy",
       "text": "Add tests for `processQueue` — 0.0% line coverage, complexity 8, CRAP 72.0 (internal/worker/queue.go:89).",
       "function": "processQueue", "file": "internal/worker/queue.go", "line": 89,
       "fix_strategy": "add_tests"}
    ]
  }
}
```

## Stability rules

- `scorecard` always has exactly 5 rows, in the order `crapload`,
  `gaze_crapload`, `avg_line_coverage`, `contract_coverage`,
  `complexity`.
- `grade` is one of `A, A-, B+, B, B-, C+, C, C-, D, F`, or `null`.
  `severity` is one of `low, moderate, critical, unavailable`.
  `marker` is the matching emoji.
- `rubric_version` changes whenever a boundary or template changes.
- `metadata.sections` lists the selected sections. A step whose
  section is not selected is `null`, and its `errors` entry is also
  `null`.
- The same `metadata` and `assessment` objects are included in the
  compact payload sent to AI adapters.

## Round-trip invariant (SC-005)

For each scorecard row, let **R** be the markdown row from
`RenderMarkdown(p)` whose `Dimension` cell equals `label`, and let
**J** be the JSON row with that `label`. Then:

- R's `Grade` cell = `J.marker + " " + J.grade`, or `⚪ N/A` when
  `J.grade == null`.
- R's `Details` cell = `J.detail`.

For each recommendation at index i, markdown line i under
`### Recommendations` = `"{i+1}. " + J.marker + " " + J.text`.

The test produces J with `--format=json` and R with
`--format=markdown`, both through `runReport` on the same injected
payload. It decodes J with `encoding/json` and parses R's table rows.
