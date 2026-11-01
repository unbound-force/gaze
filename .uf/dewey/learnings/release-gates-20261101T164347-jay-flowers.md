---
tag: release-gates
author: jay-flowers
category: pattern
created_at: 2026-11-01T16:43:47Z
identity: release-gates-20261101T164347-jay-flowers
tier: draft
---

A containment test that guards a gate's silent-skip path by duplicating the gate's extraction grammar (grep/sed) verifies the repository's state but not the gate's liveness on that repository — the two grammar copies can be updated independently, leaving a drift path where the gate permanently skips while CI stays green. Close it with a canary invocation: run the gate against the real repository with an input guaranteed to take a non-skip branch and assert that branch's exit status. For gaze's validate-workflow-floor.sh, `-- v0.0.1 <repo-root>` is deterministic (below any floor the repository can carry, exiting 1 before any tag-dependent branch), so asserting exit 1 proves the validator found the real constant; a stale regex would exit 0 via the legacy skip. Pattern observed in PR #309's test-validate-workflow-floor.sh (design D7): repo invariants plus a synthetic-repo matrix contain every drift path except a partial update that touches the test's regex but not the validator's.
