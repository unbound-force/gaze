# Quickstart: Report Voice Refinement

**Feature**: `010-report-voice-refinement`

## What This Feature Does

Updates the gaze-reporter agent prompt to produce clinical, emoji-free reports with concise formatting, word-based grades, a prominent risk matrix, and a "Bottom line:" closing summary.

## Files Changed

| File | Change |
|------|--------|
| `.opencode/agents/gaze-reporter.md` | Rewrite Output Format, update mode sections, add Example Output |
| `internal/scaffold/assets/agents/gaze-reporter.md` | Byte-identical copy of above |

## How to Verify

After implementation, run `/gaze` on any Go project and check:

1. **No emojis** — scan output for Unicode emoji characters. Should find zero.
2. **Title format** — first line reads `Gaze Health Report — <project-name>`
3. **Metadata format** — second line reads `Package: <pattern> | Date: <date>`
4. **Right-aligned numerics** — CRAP, Complexity, Coverage columns in tables
5. **Word grades** — Overall Grade uses Poor/Fair/Good/Strong/Excellent
6. **Risk matrix** — Priority/Function/Risk/Why table in health assessment
7. **Bottom line** — report ends with "Bottom line: ..." paragraph
8. **No empty sections** — if quality data is unavailable, no Quality Summary header appears
9. **Terse interpretations** — each interpretation after a table is one sentence, max ~25 words

## How to Test on Different Projects

```bash
# On the gaze project itself (healthy codebase):
/gaze

# On a project with poor test coverage (if available):
# cd /path/to/other-project && /gaze

# CRAP mode only:
/gaze crap

# Quality mode only:
/gaze quality
```

## Key Design Decisions

- **Rules + Example**: The prompt uses both explicit formatting rules and a concrete example output. Models follow examples more reliably than abstract rules alone (see research.md R2).
- **Omit over placeholder**: Empty sections are dropped entirely, not rendered with "N/A" or warning blockquotes. A single line at the end notes which analyses were unavailable.
- **Grade scale**: Poor → Fair → Good → Strong → Excellent. Maps to intuitive quality levels without requiring letter-grade decoding.
