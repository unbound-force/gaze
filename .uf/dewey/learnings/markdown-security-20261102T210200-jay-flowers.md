---
tag: markdown-security
author: jay-flowers
category: pattern
created_at: 2026-11-02T21:02:00Z
identity: markdown-security-20261102T210200-jay-flowers
tier: draft
---

Multi-pattern strings.NewReplacer escapers need per-pair table-driven assertions, because render-level hostile-data tests assert only a token subset and coverage is asymmetric across pairs: gaze PR #314's TestRenderMarkdown_MetadataEscaped pinned 4 of headerEscaper's 12 pairs, leaving backslash/backtick/asterisk/underscore/pipe and all three newline-flattening pairs deletable without a test failure. Pair ORDER is load-bearing when one pattern is a prefix of another — \r\n must precede \r or CRLF flattens to two spaces — and only an assertion that a CRLF input yields exactly one space catches a reorder. Corollary for feedback triage: verify "never asserted" claims against the frozen head before implementing — PR #314 feedback item 2 asked to assert proseEscaper flattening, but TestSanitizeProse_Markup already pinned sanitizeProse("one\ntwo\r\nthree\rfour") == "one two three four" (ordering included); only the headerEscaper half was a real gap. Also trace which defense actually processes a given input: the hostile-data test's multi-line callout never reached proseEscaper because boundedFirstLine truncates at the first \r/\n first.
