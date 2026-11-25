---
tag: test-design
author: jay-flowers
category: pattern
created_at: 2026-11-15T06:02:22Z
identity: test-design-20261115T060222-jay-flowers
tier: draft
as_of: "gaze@bff29c8a934970cf9f74962fe0a8b3f3a9b326a6"
evidence:
  - "cmd/gaze/main_test.go#TestRunCrap_ZeroMatchWarningSuppressedOnEmptySide"
---

When a warning or guard condition is a conjunction of boolean clauses (e.g., a zero-match warning firing only when both the baseline and current report are non-empty AND zero deltas were produced), pin every clause with its own negative test — one subtest per clause that must independently suppress the behavior. Dropping any single clause from the implementation then fails a named test instead of silently widening the warning to edge cases like empty reports.
