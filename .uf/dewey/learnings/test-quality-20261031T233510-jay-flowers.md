---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-10-31T23:35:10Z
identity: test-quality-20261031T233510-jay-flowers
tier: draft
---

When a feature adds an opt-in validation gate (a fail-closed path behind a new flag), the tests the feature itself adds tend to verify the ungated path only with values that would PASS the gate — so the opt-OUT contract ("plain invocation never validates") ends up protected only incidentally, by pre-existing tests that happen to pass invalid values. In the gaze init-ci-scaffold review (2026-10-31), every new plain-init test used Version "v2.4.0" while the gating contract ("plain init runs never validate the version") was guarded solely by older tests using "dev"/""/"test" versions — protection a fixture-normalization cleanup could silently delete. Reviewer check: for every new opt-in flag with fail-closed validation, confirm at least one test added BY the feature asserts the ungated path succeeds with a value the gate would reject; that makes the additivity contract deliberately owned rather than borrowed from unrelated coverage.
