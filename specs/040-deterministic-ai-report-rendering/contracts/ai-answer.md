# Contract: AI answer shape and validation

Covers FR-012, FR-015, FR-017–FR-021. Implemented by the pure
function `ParseAIAnswer(raw string) (AIAnswer, error)`. All four
adapters' `Format` output passes through it.

## Accepted shape

Exactly one JSON object with exactly these two members:

```json
{
  "summary": "Plain-English narrative, 1–1200 characters.",
  "recommendations": [
    "Decompose `runQuality` — complexity 14 keeps CRAP at 31.2 regardless of coverage.",
    "Add tests for `parseConfig` — 0% line coverage on a complexity-5 function."
  ]
}
```

## Validation pipeline (first failing rule wins; order is normative)

| # | Rule ID | Check | Detail text (example) |
|---|---------|-------|-----------------------|
| 1 | `empty_output` | `TrimSpace(raw) == ""` | `AI returned no output` |
| 2 | `surrounding_text` | After trimming: the text either starts with `{` and ends with `}`, or it is exactly one fenced block (an opening line of ```` ``` ```` or ```` ```json ````, a body, and a closing ```` ``` ```` line). Anything else, including prose before or after a single fence, fails. | `answer must be a single JSON object (optionally in one code fence) with no surrounding text` |
| 3 | `not_json` | The body is not valid JSON. | `answer is not valid JSON` |
| 4 | `multiple_values` | Any non-whitespace token follows the first JSON value. | `answer contains more than one JSON value` |
| 5 | `not_object` | The top-level value is not an object. | `answer must be a JSON object` |
| 6 | `duplicate_field` | A member name appears twice (detected by token walk). | `field "summary" appears more than once` |
| 7 | `unknown_field` | A member other than `summary` or `recommendations` is present. | `unknown field "notes"` |
| 8 | `missing_field` | A required member is absent. | `missing field "recommendations"` |
| 9 | `wrong_type` | `summary` is not a string, `recommendations` is not an array of strings, or a value is `null`. | `field "recommendations" must be an array of strings` |
| 10 | `summary_length` | Runes of `TrimSpace(summary)` ∉ [1, 1200]. | `summary exceeds length bound (1342 > 1200 characters)` / `summary is empty` |
| 11 | `recommendation_count` | Count ∉ [1, 5]. | `recommendation count 7 outside bounds 1–5` |
| 12 | `recommendation_length` | Runes of `TrimSpace(item)` ∉ [1, 400], or the item contains `\n` or `\r`. | `recommendation 3 exceeds length bound (512 > 400 characters)` |
| 13 | `markup` | Structural markup per research R5: HTML/comment (`<` followed by a letter, `/`, or `!`), `](`, or a line starting with `#`, `>`, `\|`, a list marker, a fence, or a line made only of `-`, `=`, `*`, `_`. | `summary contains disallowed markup (heading)` |
| 14 | `non_english` | Per field: letters outside inline code spans; non-Latin letters × 100 > 5 × letters, or letters == 0. | `non-English text ratio above limit in summary (37% > 5%)` |

Rules:
- The detail is bounded to ≤ 200 bytes. It contains counts, field
  names, and indices only, never content from the answer (FR-019).
- No truncation, translation, trimming of over-long items, or retry
  (FR-021). The only normalization is `TrimSpace`, used for bound
  checks. Accepted values are stored trimmed.
- The constants `maxSummaryRunes=1200`, `minRecs=1`, `maxRecs=5`,
  `maxRecRunes=400`, and `maxNonLatinPercent=5` are unexported
  package constants. They are documented here and in
  `docs/reference/cli/report.md`.

## Error type

`*AnswerRejection{Rule, Detail}`.
`Error()` = `AI answer rejected (rule <Rule>): <Detail>`.
Callers use `errors.As` to read the rule for the rejection callout.

## Test obligations

Two table-driven cases per rule: a minimal violating input and a
nearest passing input (for example 1200 vs 1201 runes, 5 vs 6
recommendations, and exactly 5% vs just above 5% non-Latin). Also
covered: a fenced answer accepted, a fence plus trailing prose
rejected, a backtick-wrapped CJK identifier accepted, and emoji
ignored.
