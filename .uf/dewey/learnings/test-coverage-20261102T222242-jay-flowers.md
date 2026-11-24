---
tag: test-coverage
author: jay-flowers
category: pattern
created_at: 2026-11-02T22:22:42Z
identity: test-coverage-20261102T222242-jay-flowers
tier: draft
---

When a guard clause suppresses via multiple conditions, each condition needs a test where IT is the deciding branch with every other precondition satisfied — a test that omits the feature through a different mechanism silently masks the gap. In gaze PR #314 (internal/aireport/assess_test.go), TestAssess_ConditionRecommendationsOmitted carried tallies 9/3/3 meant to exercise tally-based suppression of the ambiguous_classification recommendation, but also passed SectionCRAP-only, so ambiguousClassificationRecommendation returned at the section-absent guard and the tallies were dead data. No test constructed classification-present + nonzero + ambiguous ≤ decided (the golden fixtures render only the emit path, 9 > 6), so an ||→&& mutation of the guard at assess.go:482 survived every test targeting it. Note also that incidental protection diagnoses poorly: a spurious zero-tally emission would trip TestAssess_FixStrategyTemplates' exact-count assertion with the misleading message "recommendations = 2, want 1". Pattern: table-drive guard tests per suppress sub-case (zero total, below-threshold nonzero, tie), and when one test omits a feature via two mechanisms, split it or the second mechanism's branch stays unpinned.
