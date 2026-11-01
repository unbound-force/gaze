---
tag: documentation
author: jay-flowers
category: gotcha
created_at: 2026-11-01T19:34:15Z
identity: documentation-20261101T193415-jay-flowers
tier: draft
---

Gaze spec-time docs planning: enumeration-based doc file lists plus removal-targeted orphan sweeps (grepping removed symbol names) miss docs-tree contract sites made stale by a change. Spec 040 would have missed docs/architecture/extending.md:189, docs/architecture/overview.md:20/:100, docs/reference/cli/report.md:51, docs/reference/cli/docscan.md:3. Reliable check: grep the docs tree for changed CONTRACT PHRASES (format-family enumerations, adapter invocation strings, agent data-flow framing like 'gaze-reporter'), and record an explicit N/A decision for every docs/ file deliberately excluded.
