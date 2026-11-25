---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-16T19:35:14Z
identity: test-quality-20261116T193514-jay-flowers
tier: draft
as_of: "gaze@5f11936038eeb8a4537c3b74dbcce858972ae567"
---

Use field-survey data verbatim as regression fixtures (demonstrated in gaze PR #356, TestRun_HealsDuplicateGazeCommandMarkers): when a consumer survey documents the exact defective state observed in the wild (the dev + v1.8.0 duplicate version-marker pair in stale commands/gaze.md copies), reproduce that exact shape — old frontmatter, both observed marker lines, stale body — rather than a synthetic approximation. The regression test then pins the real-world defect class the change claims to heal, not just the mechanism, and the fixture doubles as documentation of the incident evidence.
