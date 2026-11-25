---
tag: assertion-depth
author: jay-flowers
category: pattern
created_at: 2026-11-15T06:02:27Z
identity: assertion-depth-20261115T060227-jay-flowers
tier: draft
as_of: "gaze@bff29c8a934970cf9f74962fe0a8b3f3a9b326a6"
evidence:
  - "cmd/gaze/main_test.go#TestRunCrap_BaselineRootThreading"
---

A presence assertion on output that appears regardless of the condition under test is non-discriminating: asserting a function name appears in report stdout does not prove a baseline matched, because the current report prints that name whether or not the baseline matched. Pair such assertions with the absence of the negative signal (e.g., no zero-match warning on stderr) or with a condition-specific marker (a comparison section in text output, an unchanged-count in JSON), so the test fails on plausible-but-wrong rewrites rather than passing on mere output production.
