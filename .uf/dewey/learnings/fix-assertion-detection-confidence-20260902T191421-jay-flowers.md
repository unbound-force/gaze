---
tag: fix-assertion-detection-confidence
author: jay-flowers
category: gotcha
created_at: 2026-09-02T19:14:21Z
identity: fix-assertion-detection-confidence-20260902T191421-jay-flowers
tier: draft
---

When fixing AssertionDetectionConfidence for external analyzers (issue #251), the correct integration point is buildExternalQualityReports in cmd/gaze/main.go, not runQualityStep in internal/aireport/runner_steps.go. The gaze report pipeline (runProductionPipeline) does not use external analyzers — it always uses goprovider. The gaze quality --analyzer path bypasses quality.Assess entirely and constructs QualityReport entries directly in buildExternalQualityReports. This was the most common spec review finding across all 5 Divisor reviewers in Round 1, and fixing it early saved significant implementation rework.
