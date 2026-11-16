---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-15T06:48:16Z
identity: test-quality-20261115T064816-jay-flowers
tier: draft
as_of: "gaze@7965494232385e920cd63a717a79469c8b7eac5e"
---

Match-signal assertions for path-rewrite features: when testing a normalization/rewrite end-to-end, asserting the notice or rewrite count alone cannot distinguish a correct rewrite from a plausible-but-wrong one (a wrong rewrite still prints the notice and can exit 0 below thresholds). Also assert the matched entity appears in the output and the zero-match warning stays silent — the pattern gaze's TestRunCrap_BaselineRootThreading uses (matched function "Foo" present in stdout, "matched zero functions" absent from stderr), which converts a smoke assertion into a correctness check of the rewrite itself.
