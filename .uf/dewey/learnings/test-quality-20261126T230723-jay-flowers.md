---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-26T23:07:23Z
identity: test-quality-20261126T230723-jay-flowers
tier: draft
as_of: "gaze@6cfd16ef8c1d55fbe397eb7a1126b9261eebdd00"
---

When contract tests pin a shell-assembled command line (e.g. a GitHub Actions step that conditionally builds --flag arguments from env vars), pin the assembly GUARDS as well as the assembled values. In gaze's reusable_gaze_ai_report.yml tests (TestCIWorkflow_ReusableExternalAnalyzerInputs), asserting `if [ -n "$ANALYZER" ]; then` and `if [ -z "$ANALYZER" ] && [ -n "$COVERPROFILE" ]; then` — not just `ANALYZER_ARG="--analyzer=$ANALYZER"` — catches a dropped guard that would pass `--analyzer=` on every Go-mode run and silently degrade to Go-native analysis only because the CLI happens to treat empty flags as absent. Without the guard pinned, the test green-lights a behavior change hidden behind an unpinned cross-component coupling (workflow shell ↔ CLI flag parsing). Same principle applies to pinning `exit 1` alongside each `::error::` message in gate steps: dropping the exit converts an enforced gate into an annotation-only silent pass (gaze issues #101/#108/#116 class).
