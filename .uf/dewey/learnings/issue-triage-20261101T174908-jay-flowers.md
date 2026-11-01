---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:49:08Z
identity: issue-triage-20261101T174908-jay-flowers
tier: draft
---

Feature requests that prescribe an implementation mechanism should be triaged on the goal, not the mechanism. Gaze issue #271 (branch coverage CI gate) prescribes custom Go AST instrumentation to track branch execution, but the project's Minimal Assumptions constitution principle (analysis must not require host-source annotation or restructuring) and the existing coverprofile consumption path (LineCoverageProvider, --coverprofile flag) favor cross-referencing statement-coverage source ranges against AST-enumerated branch arms — the issue's own prior triage comment flags this alternative. Triage should validate the what and route mechanism conflicts with constitution principles into the spec phase as design decisions; a prescribed-but-questionable mechanism does not invalidate an otherwise well-formed feature request.
