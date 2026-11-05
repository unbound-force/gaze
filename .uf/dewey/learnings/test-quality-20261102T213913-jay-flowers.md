---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:39:13Z
identity: test-quality-20261102T213913-jay-flowers
tier: draft
---

When assessing test-gap feedback, audit both the fixture inventory and the table-driven synthetic cases before endorsing the gap verbatim. On gaze PR #314 (feedback item 4), the reviewer's claim that no fixture exercises \r\n line endings was literally true, but TestExtractFinalText_Synthetic's "crlf framing is tolerated" case already drove CRLF-framed valid lines through the same extractFinalText path a fixture would use (readStream only supplies bytes) — the plain-CRLF regression risk was covered, and only the junk+CRLF combination plus fixture provenance remained open. The accurate disposition — accept on the mixed-junk core and fold the \r\n element into the mixed fixture — required reading the synthetic test table, not just the fixture directory. Fixture-level and synthetic coverage must both be checked before a coverage gap is confirmed.
