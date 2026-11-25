---
tag: aireport-escapers
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:00:39Z
identity: aireport-escapers-20261102T220039-jay-flowers
tier: draft
---

On gaze PR #314 (spec 040 deterministic rendering), codeSpan/longestBacktickRun in internal/aireport/assess.go implement CommonMark code-span rules whose edge branches are unreachable from fixture-driven coverage: backticks are invalid in Go identifiers, so realistic fixture names exercise only the plain span, and the backtick-bearing fixture (parse`raw`Literal) carries internal backticks only — the space-padding branch for leading/trailing backticks and the run>=2 triple-delimiter path never execute under any test. Scope coverage claims and fixes per backtick-position class (none / internal run / leading / trailing / all-backticks), extending the per-replacer-pair audit rule from the aireport-escapers learnings. Two adjacent gotchas: (1) TestRenderMarkdown_CellEscaping asserts its 200-rune name via strings.Contains(out, codeSpan(longName)) — an expectation computed with the function under test pins truncation-absence, not span shape; use exact literals. (2) A deliberately implemented, contract-justified branch that no test can reach is indistinguishable from dead code under the zero-waste mandate; when the branch enforces a normative byte contract (specs/040 contracts/markdown-report.md Rule 8), the resolution is a direct table-driven unit test per input class, not branch deletion.
