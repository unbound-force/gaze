---
tag: aireport-escaping
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:00:28Z
identity: aireport-escaping-20261102T220028-jay-flowers
tier: draft
---

Golden fixtures containing internal-delimiter content create an illusion of boundary coverage for delimiter-escaping helpers. On gaze PR #314, the golden payloads' parse`raw`Literal function name pins codeSpan's delimiter-widening for internal single backticks (golden bytes plus an independent literal in TestRenderMarkdown_CellEscaping), but the CommonMark space-padding branch — taken only when content starts or ends with a backtick — remains unreachable because no fixture in the package has that shape, and the only test-side codeSpan call (render_markdown_test.go:174) builds its expected substring by calling codeSpan itself, which is tautological and cannot detect a regression. Escaping helpers need direct table-driven unit tests asserting exact literal outputs per boundary shape (leading delimiter, trailing delimiter, internal run, multi-run, all-delimiter content); "the fixture contains a delimiter somewhere" is not "the fixture exercises the boundary".
