---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:22:44Z
identity: test-quality-20261102T222244-jay-flowers
tier: draft
---

A table-driven "omitted" test can pass at the wrong guard and provide false coverage confidence. In gaze PR #314, TestAssess_ConditionRecommendationsOmitted set tallies implying ambiguous<decided, but ran with sections=SectionCRAP, so the section-absent guard returned before the tally comparison executed — the tallies were dead weight and every disjunct of `total == 0 || ambiguous <= decided` (internal/aireport/assess.go:482) stayed unpinned, including the zero-tally and ambiguous==decided boundaries. When pinning a compound short-circuit guard, enumerate each disjunct as its own table case with all preceding guards satisfied (section present; step error nil; zero tallies; equality edge; below-threshold), because coverprofile statement coverage cannot show which || operand fired — short-circuit right operands are not separate basic blocks.
