---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-02T21:02:08Z
identity: test-quality-20261102T210208-jay-flowers
tier: draft
---

Multi-pair strings.NewReplacer escapers have load-bearing argument order whenever one pattern is a prefix of another: the "\r\n" pair must precede "\r" and "\n" in the argument list or CRLF flattens to two spaces instead of one. Pin each escaper's full pair contract with one exact-match table-driven test on its wrapper function (e.g. sanitizeHeaderValue("A\r\nB") == "A B") — a single table catches dropped pairs and silent reorderings alike — and pair it with a render-level assertion that hostile multi-line input produces no interior newline in the output. When a feedback item claims an escaping behavior is "never asserted", verify the claim against the whole test file and trace the fed input's actual path before accepting it: on gaze PR #314, TestSanitizeProse_Markup:251 already asserted proseEscaper flattening and CRLF ordering (sanitizeProse("one\ntwo\r\nthree\rfour") == "one two three four") while the feedback item called it unasserted, and the multi-line callout the reviewer cited is pre-truncated by a first-line bound before the flattening replacer ever runs.
