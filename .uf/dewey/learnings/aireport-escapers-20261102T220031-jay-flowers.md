---
tag: aireport-escapers
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:00:31Z
identity: aireport-escapers-20261102T220031-jay-flowers
tier: draft
---

Assertions that compute the expected value by calling the helper under test (gaze render_markdown_test.go:174 checks strings.Contains(out, codeSpan(longName))) pin routing but not output — a regression in the helper mutates both sides identically and the test still passes. Escape and span helpers that enforce a documented byte contract need direct table-driven exact-value unit tests. For backtick code-span helpers the input classes are: plain content, interior single backtick, interior multi-backtick run, leading backtick, trailing backtick, and all-backticks — the delimiter must be longestRun+1, and CommonMark requires space padding when content starts or ends with a backtick. Golden fixtures rarely carry delimiter-adjacent shapes: on gaze PR #314 the only backtick fixture was an interior single backtick, leaving codeSpan's padding branch and 2+ run counting unreachable from every test until directly table-driven.
