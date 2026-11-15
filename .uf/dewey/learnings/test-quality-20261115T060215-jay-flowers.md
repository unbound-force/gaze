---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-15T06:02:15Z
identity: test-quality-20261115T060215-jay-flowers
tier: draft
as_of: "gaze@bff29c8a934970cf9f74962fe0a8b3f3a9b326a6"
evidence:
  - "cmd/gaze/main_test.go#TestCrapCmd_BaselineRootFlag"
  - "cmd/gaze/main_test.go#TestRunCrap_BaselineRootInertWithModuleRelativeBaseline"
  - "cmd/gaze/main_test.go#TestRunCrap_BaselineRootThreading"
---

CLI tests that inject flag values through a params struct bypass the cobra registration layer and the flag-set inertness scenarios. When a change adds a user-facing CLI flag, three test guards are needed: (1) a Flags().Lookup registration test asserting existence, default value, and usage (gaze repo precedent: TestCrapCmd_TestShortFlag), (2) at least one test with the flag actually set against the inert/no-op state the spec's inertness scenario describes (e.g. module-relative baseline or no baseline for gaze's --baseline-root), and (3) the params-struct threading test. Without (1), removing the flag registration keeps the suite green while the documented flag disappears; without (2), a future change that consults the flag on the inert path regresses the documented MUST silently — the spec scenario's WHEN clause (flag set) is never executed, so it cannot fail.
