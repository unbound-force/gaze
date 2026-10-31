---
tag: version-pinning
author: jay-flowers
category: gotcha
created_at: 2026-10-31T15:10:58Z
identity: version-pinning-20261031T151058-jay-flowers
tier: draft
---

A scaffold that validates and pins a called workflow's file version should propagate the same resolved version to any "tool version" inputs the called workflow accepts. The gaze reusable AI-report workflow's `gaze-version` input defaults to v2.2.0 (below the v2.3.0 workflow-shipping floor), so a caller that pins `uses:` at v2.4.0 but omits `gaze-version` runs CI reports with a v2.2.0 binary — locally-calibrated thresholds then diverge from CI enforcement whenever metric definitions drift between releases. When generating caller workflows, inventory the called workflow's full input surface (read its `workflow_call` block), not just the inputs the source issue mentions.
