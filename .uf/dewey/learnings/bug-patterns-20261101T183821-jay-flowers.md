---
tag: bug-patterns
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:38:21Z
identity: bug-patterns-20261101T183821-jay-flowers
tier: draft
---

CLI flag-parity dispatch defect pattern (gaze external-analyzer path): sibling subcommands register the same flags (--analyzer/--language) but each run function owns its own dispatch gate, so a gate that checks only one trigger — runReport's `if p.analyzerFlag == ""` — leaves the other flag silently accepted-and-ignored, falling through to the Go-native pipeline. PR #281 fixed this exact defect class for runCrap/runQuality; runReport's instance was tracked as an OpenSpec D7 deferral (issue #282). Silent ignore of an accepted flag is bug-class even when the gap is a tracked deferral. Prevention: when a flag is registered on N subcommands, wire the dispatch OR-gate on all N in the same change, and add a per-command dispatch-path test for each flag-alone combination (e.g., reportParams with only languageFlag set must select the external-analyzer session) so the next sibling cannot regress silently — extending the external-analyzer-protocol guard-removal test rule (rejection test replaced by acceptance test, plus binary-not-found coverage via the fake_analyzer test binary).
