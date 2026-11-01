---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:16:42Z
identity: issue-triage-20261101T181642-jay-flowers
tier: draft
---

Rename-followup issue triage (from gaze#244, /review-council → /uf.review-council): always verify the issue's claimed file list against current HEAD before validating — stale-reference issues age quickly because later doc rewrites and tooling rescaffolds fix checklist items in place (3 of 4 files in #244 were already clean two months after filing; only the constitution, whose amendment protocol requires a PATCH version bump plus a Sync Impact Report entry, still carried the old name — procedurally heavy governance items outlive mechanical renames). Verification technique: a literal grep for the old unprefixed form is a complete discriminator because '/review-council' is not a substring of '/uf.review-council'; expect same-named-entity false positives (a sibling repo named 'review-council' matched the acceptance grep via a Dewey sources.yaml path — config to adjudicate, not a stale command reference). Partially-stale checklists downgrade scope accuracy, not validity, when the acceptance criterion is grep-based and self-correcting.
