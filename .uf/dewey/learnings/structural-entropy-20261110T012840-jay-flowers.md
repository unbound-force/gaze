---
tag: structural-entropy
author: jay-flowers
category: gotcha
created_at: 2026-11-10T01:28:40Z
identity: structural-entropy-20261110T012840-jay-flowers
tier: draft
---

Entropy-review calibration (gaze PR #326, vibe-check v0.3.0): adding a single stdlib import (`errors` for a sentinel error) to a package with Ca=2/Ce=21 yields ΔCe +1, ΔInstability +0.0036, ΔDistance −0.0036 (toward the main sequence), and metrics.DecideVerdict reports COMMENT ("materiality: non-zero metric shifts below REQUEST_CHANGES thresholds") rather than APPROVE. Expect COMMENT — not APPROVE — from `vibe-check diff` whenever a PR adds any import, even when the change is architecturally neutral (guard clause inside an existing function, no new types, no new internal edges). Test-file-only changes in other packages (cmd/gaze, internal/aireport) produce zero deltas — vibe-check's package design metrics exclude _test.go files. When the diff payload reports unreliable:false and no gate fires, map the tool's COMMENT to the council's APPROVE WITH ADVISORIES rather than a clean APPROVE.
