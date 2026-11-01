---
tag: input-validation
author: jay-flowers
category: pattern
created_at: 2026-11-01T19:34:13Z
identity: input-validation-20261101T193413-jay-flowers
tier: draft
---

When a spec adds strict validation to one input channel of a published artifact (e.g., gaze spec 040's validator for AI-generated report text), it must enumerate EVERY other data source rendered into the same artifact and apply source-appropriate sanitization — the design is only as strong as the least-validated channel. In spec 040, env-derived metadata (branch name from GITHUB_HEAD_REF/GITHUB_REF_NAME) rendered verbatim into the markdown header is an injection surface ([]()<!- allowed in ref names). Checklist: for every byte position in the published output, name its origin (model output, env var, filesystem, config, analysis data) and the control applied (trusted, escaped, validated).
