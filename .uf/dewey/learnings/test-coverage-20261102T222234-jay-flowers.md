---
tag: test-coverage
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:22:34Z
identity: test-coverage-20261102T222234-jay-flowers
tier: draft
---

A guard clause crossed incidentally by tests that never assert its outcome provides zero regression protection. In gaze's deterministic assessment code, the compound guard `total == 0 || ambiguous <= decided` (internal/aireport/assess.go, ambiguousClassificationRecommendation) is mutually redundant at the zero boundary — either clause alone catches all-zero tallies — so tests that execute zero-tally payloads (TestAssess_ScorecardOrderAndLabels uses ReportSummary{}) but assert only scorecard fields leave the boundary completely unpinned; a regression breaking both clauses would emit a spurious "0 of 0 effects (0.0%)" recommendation with no failing test. Pattern: for every guard clause in deterministic rendering code, add one table case that reaches past the preceding guards and asserts the no-output outcome by stable identifier (e.g., recommendation Source == "ambiguous_classification" absent), never by full-text matching.
