---
tag: recursion-guard
author: jay-flowers
category: gotcha
created_at: 2026-11-09T23:16:45Z
identity: recursion-guard-20261109T231645-jay-flowers
tier: draft
---

A marker environment variable set in child processes prevents nothing by itself — propagation without a refusal check is a half-built recursion guard. In gaze, buildGoTestCmd already set GAZE_COVERAGE_RUN=1 (with dedup and passing unit tests) and runGoTestCoverage's doc comment even claimed 'recursive subprocess prevention is handled by the GAZE_COVERAGE_RUN environment variable', yet generateCoverProfile never consulted it, so self-analysis recursion (SC005 in non-short mode) remained unbounded until the e
