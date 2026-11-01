---
tag: release-gates
author: jay-flowers
category: pattern
created_at: 2026-11-01T15:59:29Z
identity: release-gates-20261101T155929-jay-flowers
tier: draft
---

When a release/CI validator has a legitimate silent-skip path (e.g. legacy trees lacking a governance constant like gaze's minWorkflowTag), the skip becomes a permanent fail-open bypass the moment the marker relocates, is renamed, or the extraction regex drifts — the gate keeps exiting green while enforcing nothing. Containment pattern (gaze validate-workflow-floor design D7): pair the gate with a CI test script that asserts repository invariants FIRST — exactly one constant declaration matching the gate's grep grammar, the value parses canonically, companion files exist — and only then runs the synthetic-repo exit-status matrix. This converts silent permanent skip into loud CI failure in the same change that causes the drift. Defense in depth: an independent Go test (ci_test.go) can pin the floor value too, so the grep, the test script, and the Go test all guard one value.
