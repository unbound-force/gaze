---
tag: test-conventions
author: jay-flowers
category: pattern
created_at: 2026-11-10T02:01:14Z
identity: test-conventions-20261110T020114-jay-flowers
tier: draft
---

In Go test code, repeating an environment-variable literal that production centralizes as a constant can be deliberate contract pinning rather than drift. When the variable name is a documented external interface (gaze's GAZE_COVERAGE_RUN is named in AGENTS.md, README, and three CLI reference pages), literal-pinned guards (os.Getenv/t.Setenv) fail loudly if the production constant's value changes, while tests referencing an exported constant would follow the change silently and mask interface drift. External test packages (package foo_test) cannot see the unexported constant at all, and exporting a symbol only for tests grows the public API beyond the change record's declared capabilities. The residual typo risk of a literal guard is contained when the invariant is also enforced at the production spawn seam — gaze's generateCoverProfile refuses nested generation (ErrNestedCoverageRun), so a missing or broken skip degrades to a bounded, loud single-level failure instead of unbounded recursion. Decide extraction by a recorded site-count threshold: gaze's PR #326 review recorded the trigger at the 6th site (export goprovider.CoverageRunEnv or extract a shared skip helper), and the author's decline of the earlier refactor was accepted as consistent with the documented house pattern.
