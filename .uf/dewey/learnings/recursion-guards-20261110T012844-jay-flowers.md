---
tag: recursion-guards
author: jay-flowers
category: decision
created_at: 2026-11-10T01:28:44Z
identity: recursion-guards-20261110T012844-jay-flowers
tier: draft
---

Recursion in self-analyzing test suites (tests that run the tool's own coverage generation against the tool's own module) must be prevented at the single choke point that spawns the subprocess, not by distributing skip conventions across every self-analyzing test. In gaze, generateCoverProfile returns the exported ErrNestedCoverageRun sentinel when GAZE_COVERAGE_RUN is already non-empty, making unbounded recursion structurally impossible; per-test skip guards under the same variable then serve only to keep nested runs clean and fast. A forgotten guard degrades to a bounded test failure with partial-profile recovery instead of an infinite process chain (a 2026-11-09 incident left 23 processes and ~6.9 GB RSS before this fix).
