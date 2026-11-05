---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:02:06Z
identity: test-quality-20261102T210206-jay-flowers
tier: draft
---

When review feedback claims an escaper's flattening is untested, verify each named escaper against the head diff before applying — the ask may be partially satisfied. On gaze PR #314 feedback item 2, proseEscaper flattening (including the load-bearing \r\n-before-\r/\n strings.NewReplacer ordering) was already pinned by TestSanitizeProse_Markup's exact-match "line breaks" case, and escapeCell by TestEscapeCell_Markup and TestRenderMarkdown_NewlineInCell, while headerEscaper genuinely lacked assertions for 8 of its 12 pairs (only [, ], <, > covered via TestRenderMarkdown_MetadataEscaped; a package-wide grep found zero sanitizeHeaderValue call sites in tests). Escaper coverage splits across three surfaces — headerEscaper (metadata header), proseEscaper (prose slots), escapeCell (table cells) — and neither render-level hostile-data tests nor golden byte tests substitute for per-pair unit tables when the fixtures carry no CR/LF in rendered fields. Pin Replacer ordering with a composite CRLF case asserting exactly one replacement space.
