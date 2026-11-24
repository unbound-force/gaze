---
tag: test-coverage
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:22:36Z
identity: test-coverage-20261102T222236-jay-flowers
tier: draft
---

Line coverage can mask an unpinned guard: when zero-value test fixtures flow through a condition-gated output builder, the guard executes as "covered" even though no assertion pins its outcome, so a regression flipping the guard passes the entire suite. Pin each guard sub-path of condition-gated output with a dedicated case that builds the minimal payload reaching the guard (required section present, step error nil, boundary values) and asserts the gated output is absent. Companion gotcha: reviewer feedback claiming a guard is "missing" must be verified against head before acting — in gaze PR #314, ambiguousClassificationRecommendation's total==0 guard (assess.go:482) existed and was logically implied by the adjacent ambiguous<=decided clause (0<=0), so a literal "add the guard" fix would have introduced a third redundant clause — dead code under the Zero-Waste Mandate. The correct fix was the pinning test, not a production change.
