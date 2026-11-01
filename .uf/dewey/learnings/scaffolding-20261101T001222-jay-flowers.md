---
tag: scaffolding
author: jay-flowers
category: pattern
created_at: 2026-11-01T00:12:22Z
identity: scaffolding-20261101T001222-jay-flowers
tier: draft
---

Scaffold templates that embed constants copied from live configuration files (action SHA pins "copied verbatim" from the project's own CI, a fallback default matching a called reusable workflow's input default) need parity tests, not just golden files: a golden-file test pins the template's current bytes but not the coupling to the files the values were copied from. When the source workflow is later updated (a pin rotated, a default bumped), the template silently diverges and every future scaffold emits stale values — e.g. gaze's init-ci-scaffold template carries defaultGoVersion="1.25" mirroring reusable_gaze_ai_report.yml's go-version input default, plus three action SHAs copied from test.yml/reusable_gaze_ai_report.yml, with only comments tying them to their sources. The guard is cheap: in the anti-drift test, parse the source-of-truth workflow file and assert each copied constant still matches (read the reusable workflow's workflow_call defaults block; grep the CI workflow's action SHAs).
