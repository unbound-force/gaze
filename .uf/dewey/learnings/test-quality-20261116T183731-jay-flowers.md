---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-16T18:37:31Z
identity: test-quality-20261116T183731-jay-flowers
tier: draft
as_of: "gaze@745e597410ff162c7a3ae03ddce988e471b24e3f"
---

When a behavior's trigger set shrinks, pin BOTH sides in tests. In the gaze-command-tool-owned change, commands/gaze.md flipped from user-owned to tool-owned, so the gaze init summary's "use --force to overwrite" hint lost its only .opencode/ trigger. The test suite asserted the hint's absence where the old trigger used to fire (TestRun_SkipsExisting, flipped from a presence assertion to absence) AND asserted its presence on the surviving trigger (ci_test.go's TestRun_CI_SkipIfPresent pinning "1 file skipped (use --force to overwrite)" for the CI workflow — a different test file from the behavior's home tests). Presence-only or absence-only coverage lets the hint silently die or silently over-fire in the untested direction; the surviving-trigger pin matters most when the surviving trigger lives in a different subsystem's test file than the removed one.
