---
tag: review-dispatch
author: jay-flowers
category: pattern
created_at: 2026-11-02T19:40:17Z
identity: review-dispatch-20261102T194017-jay-flowers
tier: draft
---

In manifest-delivery review mode (RD-FR-032), embedded mirror pairs such as `.opencode/<file>` ↔ `internal/scaffold/assets/<file>` can be verified byte-identical without retrieving either file's diff content: compare the `index <old>..<new>` blob hash lines of the two files' manifest chunks — identical new-hash git blobs prove byte identity. This works because git blob hashes are content-addressed. Used on PR #314 to confirm all three changed mirror pairs (gaze-reporter.md, commands/gaze.md, references/example-report.md) satisfy the File Sync Invariants enforced by TestEmbeddedAssetsMatchSource, at zero content-retrieval cost.
