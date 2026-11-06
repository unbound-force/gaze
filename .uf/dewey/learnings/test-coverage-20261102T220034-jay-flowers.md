---
tag: test-coverage
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:00:34Z
identity: test-coverage-20261102T220034-jay-flowers
tier: draft
---

A representative pathological fixture protects only the exact input shape it carries. On gaze PR #314, the golden payload's backtick-bearing name "parse`raw`Literal" pinned codeSpan's delimiter-widening branch at three layers (the render-level assertion at render_markdown_test.go:177 plus five golden reports checked at renderer and CLI level), while adjacent branches of the same helper — CommonMark space padding for leading/trailing backticks and widening for multi-backtick runs — stayed fully dark: no fixture exhibits those shapes, and the one direct test call (codeSpan(longName) inside a containment assertion) exercised only the plain path. A reviewer's "no direct unit tests" claim can be literally true while the primary behavior is triple-protected, so audit which input shape each fixture carries before endorsing or scoping the gap. Close the asymmetry with a per-helper table-driven unit test asserting exact output strings for every documented rule branch (codeSpan("a`b") == "``a`b``", codeSpan("`x") == "`` `x ``", a double-backtick run widening to triple delimiters) in the internal-package test file when the helper is unexported; prefer this over extending shared golden payloads, which churns every golden file to cover one branch.
