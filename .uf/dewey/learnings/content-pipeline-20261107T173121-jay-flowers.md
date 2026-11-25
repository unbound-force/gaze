---
tag: content-pipeline
author: jay-flowers
category: decision
created_at: 2026-11-07T17:31:21Z
identity: content-pipeline-20261107T173121-jay-flowers
tier: draft
---

Curator Website Documentation Gate precedent (gaze PR #317 dogfood-ai-report-from-source): an optional input added to a reusable GitHub Actions workflow (.github/workflows/reusable_gaze_ai_report.yml) is CI/CD-exempt — no unbound-force/website issue — when three conditions hold: (a) the input default keeps existing consumer behavior byte-identical (build-from-source defaults false; the release-install step is unchanged), (b) the input is documented in the repo's own consumer-contract doc (docs/guides/ci
