---
tag: deterministic-rendering
author: jay-flowers
category: pattern
created_at: 2026-11-02T21:02:02Z
identity: deterministic-rendering-20261102T210202-jay-flowers
tier: draft
---

Exact-output table rows are the mechanism that pins strings.NewReplacer pair ordering: removing or misordering the CRLF pair (which must precede the lone \\r and \\n pairs) produces a detectable double space, so a want-string like "one two three four" against input "one\ntwo\r\nthree\rfour" both documents and enforces the CRLF-collapses-to-one-space contract. Verified on gaze PR #314: TestSanitizeProse_Markup and TestEscapeCell_Markup already pin proseEscaper and escapeCell this way, so reviewer feedback claiming proseEscaper flattening was "never asserted" was wrong — only headerEscaper's 8 remaining pairs (of 12) were genuinely uncovered. Corollary for feedback assessment: before accepting a "never asserted" claim, enumerate every test that touches the package-level var; and for multi-pair Replacers, prefer per-pair exact-output table rows over presence/absence token checks.
