---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-02T21:02:14Z
identity: test-quality-20261102T210214-jay-flowers
tier: draft
---

Escaping-coverage audits — and validation of a reviewer's coverage claim — must map assertions per escaper and per replacer pair, not per test. On gaze PR #314 (feedback item 2), the reviewer claimed proseEscaper's newline flattening was "never asserted" because TestRenderMarkdown_HostileDataEscaped only checks bracket/angle escaping; in fact TestSanitizeProse_Markup's table case already pins sanitizeProse("one\ntwo\r\nthree\rfour") == "one two three four" — one exact-output case simultaneously locks LF/CR/CRLF flattening AND the CRLF-before-CR/LF replacer ordering, because a reordered replacer emits two spaces for CRLF and fails exact equality. The claim held only for headerEscaper: 4 of 12 pairs covered (TestRenderMarkdown_MetadataEscaped), with backslash/backtick/asterisk/underscore/pipe and all three newline pairs unasserted and golden fixtures carrying newline-free metadata, despite the doc comment promising no-newline injection. Audit rules: (1) one exact-output case per multi-byte sequence ("a\r\nb" -> "a b") pins both flattening and pair ordering, while kitchen-sink token-absence checks on a single hostile input verify only a subset of pairs; (2) check a coverage claim against the whole test file, not the single test the reviewer anchors on; (3) verify which pipeline stage actually transforms the bytes — boundedFirstLine truncated the hostile callout's newline before sanitizeProse ever ran, so that path exercised no flattening at all.
